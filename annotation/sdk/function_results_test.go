package sdk

import (
	"maps"
	"strconv"
	"strings"
	"testing"
)

func TestFunctionResultFactsRoundTrip(t *testing.T) {
	t.Parallel()
	input := []FunctionResultFact{
		{
			TypeID:             "example.com/app.ToolAlias",
			CanonicalTypeID:    "example.com/tool.Tool",
			Kind:               GoTypeInterface,
			NamedOriginPackage: "example.com/tool",
			NamedOriginName:    "Tool",
		},
		{
			TypeID:          "*example.com/app.Concrete",
			CanonicalTypeID: "*example.com/app.Concrete",
			Kind:            GoTypePointer,
		},
	}
	facts, err := EncodeFunctionResultFacts(input)
	if err != nil {
		t.Fatalf("EncodeFunctionResultFacts() error = %v", err)
	}
	wantFacts := map[string]string{
		"go.function.results.count":                  "2",
		"go.function.results.0.type_id":              "example.com/app.ToolAlias",
		"go.function.results.0.canonical_type_id":    "example.com/tool.Tool",
		"go.function.results.0.kind":                 "interface",
		"go.function.results.0.named_origin.package": "example.com/tool",
		"go.function.results.0.named_origin.name":    "Tool",
		"go.function.results.1.type_id":              "*example.com/app.Concrete",
		"go.function.results.1.canonical_type_id":    "*example.com/app.Concrete",
		"go.function.results.1.kind":                 "pointer",
	}
	if !maps.Equal(facts, wantFacts) {
		t.Fatalf("encoded facts = %#v, want %#v", facts, wantFacts)
	}
	facts["symbol_kind"] = "function"
	decoded, present, err := DecodeFunctionResultFacts(facts)
	if err != nil || !present || !equalFunctionResultFacts(decoded, input) {
		t.Fatalf(
			"DecodeFunctionResultFacts() = %#v, %t, %v",
			decoded,
			present,
			err,
		)
	}
	invocationDecoded, invocationPresent, invocationErr := (Invocation{
		Facts: facts,
	}).FunctionResultFacts()
	if invocationErr != nil || !invocationPresent ||
		!equalFunctionResultFacts(invocationDecoded, input) {
		t.Fatalf(
			"Invocation.FunctionResultFacts() = %#v, %t, %v",
			invocationDecoded,
			invocationPresent,
			invocationErr,
		)
	}
}

func TestFunctionResultFactsDistinguishAbsentAndZeroResults(t *testing.T) {
	t.Parallel()
	for _, facts := range []map[string]string{
		nil,
		{},
		{"symbol_kind": "function"},
		{"future.unrelated.fact": "accepted"},
	} {
		decoded, present, err := DecodeFunctionResultFacts(facts)
		if err != nil || present || decoded != nil {
			t.Fatalf(
				"DecodeFunctionResultFacts(%v) = %#v, %t, %v",
				facts,
				decoded,
				present,
				err,
			)
		}
	}
	facts, err := EncodeFunctionResultFacts(nil)
	if err != nil {
		t.Fatalf("EncodeFunctionResultFacts(nil) error = %v", err)
	}
	decoded, present, err := DecodeFunctionResultFacts(facts)
	if err != nil || !present || len(decoded) != 0 {
		t.Fatalf(
			"zero-result decode = %#v, %t, %v",
			decoded,
			present,
			err,
		)
	}
}

func TestFunctionResultFactsRejectMalformedReservedFacts(t *testing.T) {
	t.Parallel()
	valid := map[string]string{
		FunctionResultCountFact:                                    "1",
		functionResultPrefix(0) + functionResultTypeIDField:        "string",
		functionResultPrefix(0) + functionResultCanonicalTypeField: "string",
		functionResultPrefix(0) + functionResultKindField:          "basic",
	}
	tests := []struct {
		name    string
		mutate  func(map[string]string)
		message string
	}{
		{
			name: "missing count",
			mutate: func(facts map[string]string) {
				delete(facts, FunctionResultCountFact)
			},
			message: "require a count",
		},
		{
			name: "noncanonical count",
			mutate: func(facts map[string]string) {
				facts[FunctionResultCountFact] = "01"
			},
			message: "canonical non-negative",
		},
		{
			name: "count over bound",
			mutate: func(facts map[string]string) {
				facts[FunctionResultCountFact] = strconv.Itoa(
					MaximumFunctionResultFacts + 1,
				)
			},
			message: "exceeds",
		},
		{
			name: "unknown reserved field",
			mutate: func(facts map[string]string) {
				facts[functionResultPrefix(0)+"future"] = "value"
			},
			message: "unknown",
		},
		{
			name: "noncanonical index",
			mutate: func(facts map[string]string) {
				facts[FunctionResultFactNamespace+"00.type_id"] = "string"
			},
			message: "invalid index",
		},
		{
			name: "index outside count",
			mutate: func(facts map[string]string) {
				facts[functionResultPrefix(1)+functionResultTypeIDField] = "string"
			},
			message: "exceeds result count",
		},
		{
			name: "missing required fact",
			mutate: func(facts map[string]string) {
				delete(
					facts,
					functionResultPrefix(0)+functionResultCanonicalTypeField,
				)
			},
			message: "requires type ID",
		},
		{
			name: "unknown kind",
			mutate: func(facts map[string]string) {
				facts[functionResultPrefix(0)+functionResultKindField] = "magic"
			},
			message: "unsupported",
		},
		{
			name: "untrimmed type",
			mutate: func(facts map[string]string) {
				facts[functionResultPrefix(0)+functionResultTypeIDField] = " string"
			},
			message: "trimmed",
		},
		{
			name: "oversized value",
			mutate: func(facts map[string]string) {
				facts[functionResultPrefix(0)+functionResultTypeIDField] = strings.Repeat("x", MaximumFunctionResultTypeIDBytes+1)
			},
			message: "exceeds",
		},
		{
			name: "origin package without name",
			mutate: func(facts map[string]string) {
				facts[functionResultPrefix(0)+functionResultOriginPackageField] = "example.com/type"
			},
			message: "requires a name",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			facts := maps.Clone(valid)
			test.mutate(facts)
			_, _, err := DecodeFunctionResultFacts(facts)
			if err == nil || !strings.Contains(err.Error(), test.message) {
				t.Fatalf("decode error = %v, want %q", err, test.message)
			}
		})
	}
}

func TestEncodeFunctionResultFactsRejectsInvalidValues(t *testing.T) {
	t.Parallel()
	tooMany := make(
		[]FunctionResultFact,
		MaximumFunctionResultFacts+1,
	)
	if _, err := EncodeFunctionResultFacts(tooMany); err == nil ||
		!strings.Contains(err.Error(), "exceeds") {
		t.Fatalf("too many results error = %v", err)
	}
	invalid := []FunctionResultFact{{
		TypeID:             "example.com/Type",
		CanonicalTypeID:    "example.com/Type",
		Kind:               GoTypeStruct,
		NamedOriginPackage: "example.com/type",
	}}
	if _, err := EncodeFunctionResultFacts(invalid); err == nil ||
		!strings.Contains(err.Error(), "requires a name") {
		t.Fatalf("invalid result error = %v", err)
	}
}

func equalFunctionResultFacts(
	left []FunctionResultFact,
	right []FunctionResultFact,
) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}
