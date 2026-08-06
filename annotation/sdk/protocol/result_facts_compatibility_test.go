package protocol

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/spice-framework/spice/annotation/sdk"
)

func TestFunctionResultFactsRemainV1Alpha2WireCompatible(t *testing.T) {
	t.Parallel()
	facts, err := sdk.EncodeFunctionResultFacts([]sdk.FunctionResultFact{{
		TypeID:             "example.com/app.ToolAlias",
		CanonicalTypeID:    "example.com/tool.Tool",
		Kind:               sdk.GoTypeInterface,
		NamedOriginPackage: "example.com/tool",
		NamedOriginName:    "Tool",
	}})
	if err != nil {
		t.Fatalf("EncodeFunctionResultFacts() error = %v", err)
	}
	current := AnalyzeParams{
		Descriptor: sdk.Symbol{
			Package: "example.com/annotation",
			Name:    "Tool",
		},
		Invocation: sdk.Invocation{
			DescriptorPackage: "example.com/annotation",
			DescriptorSymbol:  "Tool",
			CanonicalName:     "fixture.Tool",
			Declaration: sdk.Declaration{
				Target:      sdk.TargetFunction,
				SymbolID:    "fixture.Tool",
				Name:        "NewTool",
				PackagePath: "example.com/app",
				TypeID:      "func() example.com/app.ToolAlias",
			},
			Facts: facts,
		},
	}
	wire, err := json.Marshal(current)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	var legacy legacyAnalyzeParams
	decoder := json.NewDecoder(bytes.NewReader(wire))
	decoder.DisallowUnknownFields()
	if decodeErr := decoder.Decode(&legacy); decodeErr != nil {
		t.Fatalf("legacy strict decode error = %v\n%s", decodeErr, wire)
	}
	if legacy.Invocation.Facts[sdk.FunctionResultCountFact] != "1" {
		t.Fatalf("legacy facts = %#v", legacy.Invocation.Facts)
	}

	legacy.Invocation.Facts = nil
	legacyWire, err := json.Marshal(legacy)
	if err != nil {
		t.Fatalf("legacy json.Marshal() error = %v", err)
	}
	var decoded AnalyzeParams
	if decodeErr := json.Unmarshal(legacyWire, &decoded); decodeErr != nil {
		t.Fatalf("current decode of legacy payload error = %v", decodeErr)
	}
	results, present, err := decoded.Invocation.FunctionResultFacts()
	if err != nil || present || results != nil {
		t.Fatalf(
			"legacy result facts = %#v, %t, %v",
			results,
			present,
			err,
		)
	}
}

type legacyAnalyzeParams struct {
	Descriptor sdk.Symbol       `json:"descriptor"`
	Invocation legacyInvocation `json:"invocation"`
}

type legacyInvocation struct {
	DescriptorPackage string                   `json:"descriptor_package"`
	DescriptorSymbol  string                   `json:"descriptor_symbol"`
	CanonicalName     string                   `json:"canonical_name"`
	Arguments         []sdk.InvocationArgument `json:"arguments,omitempty"`
	Declaration       sdk.Declaration          `json:"declaration"`
	Facts             map[string]string        `json:"facts,omitempty"`
}
