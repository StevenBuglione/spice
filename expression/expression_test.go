package expression

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"testing"
)

func TestCompileAndEvaluateTypedExpression(t *testing.T) {
	t.Parallel()
	program, err := Compile(
		`authenticated && (hasRole("admin") || subject == "owner") && issuer != "blocked"`,
		testSchema(),
	)
	if err != nil {
		t.Fatalf("Compile() error = %v", err)
	}
	if program.Source() == "" {
		t.Fatal("Source() is empty")
	}

	inputs := testInputs(true, "owner", "issuer", []string{"reader"})
	allowed, err := program.Evaluate(context.Background(), inputs)
	if err != nil || !allowed {
		t.Fatalf("Evaluate(owner) = %t, %v", allowed, err)
	}
	inputs = testInputs(true, "other", "issuer", []string{"admin"})
	allowed, err = program.Evaluate(context.Background(), inputs)
	if err != nil || !allowed {
		t.Fatalf("Evaluate(admin) = %t, %v", allowed, err)
	}
	inputs = testInputs(true, "other", "issuer", []string{"reader"})
	allowed, err = program.Evaluate(context.Background(), inputs)
	if err != nil || allowed {
		t.Fatalf("Evaluate(reader) = %t, %v", allowed, err)
	}
}

func TestEvaluateShortCircuitsFunctions(t *testing.T) {
	t.Parallel()
	for _, source := range []string{
		`false && hasRole("never")`,
		`true || hasRole("never")`,
	} {
		program, err := Compile(source, testSchema())
		if err != nil {
			t.Fatalf("Compile(%q) error = %v", source, err)
		}
		inputs := testInputs(false, "", "", nil)
		inputs.Functions[0] = func(context.Context, []Value) (Value, error) {
			return Value{}, errors.New("must not run")
		}
		if _, err := program.Evaluate(context.Background(), inputs); err != nil {
			t.Fatalf("Evaluate(%q) error = %v", source, err)
		}
	}
}

func TestCompileRejectsInvalidExpressions(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		source   string
		contains string
	}{
		{name: "empty", source: "", contains: "source is required"},
		{name: "whitespace", source: " true", contains: "surrounding whitespace"},
		{name: "unknown", source: "missing", contains: "unknown symbol"},
		{name: "function value", source: "hasRole", contains: "requires parentheses"},
		{name: "unknown function", source: "missing()", contains: "unknown function"},
		{name: "bad argument count", source: "hasRole()", contains: "argument count"},
		{name: "bad argument type", source: "hasRole(true)", contains: "argument type"},
		{name: "bad logical type", source: `subject && true`, contains: "Boolean operands"},
		{name: "bad equality type", source: `subject == true`, contains: "same type"},
		{name: "non Boolean result", source: `subject`, contains: "result must be Boolean"},
		{name: "assignment", source: `authenticated = true`, contains: "unexpected token"},
		{name: "property traversal", source: `principal.subject == "x"`, contains: "unexpected token"},
		{name: "unterminated string", source: `subject == "x`, contains: "unterminated string"},
		{name: "missing parenthesis", source: `(authenticated`, contains: "requires )"},
		{name: "trailing", source: `authenticated true`, contains: "trailing token"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			_, err := Compile(test.source, testSchema())
			if err == nil || !strings.Contains(err.Error(), test.contains) {
				t.Fatalf("Compile() error = %v, want %q", err, test.contains)
			}
			compileErr, ok := errors.AsType[*CompileError](err)
			if !ok || compileErr.Offset < 0 {
				t.Fatalf("Compile() error type = %T", err)
			}
		})
	}
}

func TestCompileRejectsInvalidSchemas(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		schema Schema
	}{
		{name: "invalid variable", schema: Schema{Variables: []Variable{{Name: "bad.name", Kind: Boolean}}}},
		{name: "reserved variable", schema: Schema{Variables: []Variable{{Name: "true", Kind: Boolean}}}},
		{name: "invalid variable type", schema: Schema{Variables: []Variable{{Name: "item"}}}},
		{name: "duplicate variable", schema: Schema{Variables: []Variable{{Name: "item", Kind: Boolean}, {Name: "item", Kind: Boolean}}}},
		{name: "variable function collision", schema: Schema{Variables: []Variable{{Name: "item", Kind: Boolean}}, Functions: []FunctionSpec{{Name: "item", Result: Boolean}}}},
		{name: "invalid function result", schema: Schema{Functions: []FunctionSpec{{Name: "check"}}}},
		{name: "invalid function parameter", schema: Schema{Functions: []FunctionSpec{{Name: "check", Parameters: []Kind{invalidKind}, Result: Boolean}}}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			if _, err := Compile("true", test.schema); err == nil {
				t.Fatal("Compile() error = nil")
			}
		})
	}
}

func TestEvaluateRejectsInvalidInputsAndCancellation(t *testing.T) {
	t.Parallel()
	program, err := Compile(`authenticated && hasRole("admin")`, testSchema())
	if err != nil {
		t.Fatalf("Compile() error = %v", err)
	}
	valid := testInputs(true, "subject", "issuer", []string{"admin"})
	//nolint:staticcheck // The public contract deliberately rejects nil contexts.
	if _, evaluateErr := program.Evaluate(nil, valid); evaluateErr == nil ||
		!strings.Contains(evaluateErr.Error(), "context is nil") {
		t.Fatalf("Evaluate(nil context) error = %v", evaluateErr)
	}
	tests := []struct {
		name     string
		program  Program
		inputs   Inputs
		contains string
	}{
		{name: "invalid program", inputs: valid, contains: "program is invalid"},
		{name: "variable count", program: program, inputs: Inputs{Functions: valid.Functions}, contains: "variable count"},
		{name: "variable type", program: program, inputs: Inputs{Variables: []Value{Text("wrong"), Text("subject"), Text("issuer")}, Functions: valid.Functions}, contains: "wrong type"},
		{name: "function count", program: program, inputs: Inputs{Variables: valid.Variables}, contains: "function count"},
		{name: "nil function", program: program, inputs: Inputs{Variables: valid.Variables, Functions: []Function{nil, valid.Functions[1]}}, contains: "is nil"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			_, evaluateErr := test.program.Evaluate(context.Background(), test.inputs)
			if evaluateErr == nil || !strings.Contains(evaluateErr.Error(), test.contains) {
				t.Fatalf("Evaluate() error = %v, want %q", evaluateErr, test.contains)
			}
		})
	}

	canceled, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := program.Evaluate(canceled, valid); !errors.Is(err, context.Canceled) {
		t.Fatalf("Evaluate(canceled) error = %v", err)
	}
}

func TestEvaluateRejectsFunctionFailuresAndWrongResults(t *testing.T) {
	t.Parallel()
	program, err := Compile(`hasRole("admin")`, testSchema())
	if err != nil {
		t.Fatalf("Compile() error = %v", err)
	}
	inputs := testInputs(true, "subject", "issuer", nil)
	inputs.Functions[0] = func(context.Context, []Value) (Value, error) {
		return Value{}, errors.New("lookup failed")
	}
	if _, err := program.Evaluate(context.Background(), inputs); err == nil ||
		!strings.Contains(err.Error(), "lookup failed") {
		t.Fatalf("Evaluate(function failure) error = %v", err)
	}
	inputs.Functions[0] = func(context.Context, []Value) (Value, error) {
		return Text("wrong"), nil
	}
	if _, err := program.Evaluate(context.Background(), inputs); err == nil ||
		!strings.Contains(err.Error(), "wrong type") {
		t.Fatalf("Evaluate(wrong result) error = %v", err)
	}
}

func TestValueAccessors(t *testing.T) {
	t.Parallel()
	if value, ok := Bool(true).BooleanValue(); !ok || !value {
		t.Fatalf("BooleanValue() = %t, %t", value, ok)
	}
	if _, ok := Bool(true).StringValue(); ok {
		t.Fatal("StringValue(Boolean) matched")
	}
	if value, ok := Text("value").StringValue(); !ok || value != "value" {
		t.Fatalf("StringValue() = %q, %t", value, ok)
	}
	if _, ok := Text("value").BooleanValue(); ok {
		t.Fatal("BooleanValue(String) matched")
	}
}

func ExampleCompile() {
	program, err := Compile(`enabled && hasLabel("stable")`, Schema{
		Variables: []Variable{{Name: "enabled", Kind: Boolean}},
		Functions: []FunctionSpec{{
			Name:       "hasLabel",
			Parameters: []Kind{String},
			Result:     Boolean,
		}},
	})
	if err != nil {
		fmt.Println(err)
		return
	}
	allowed, err := program.Evaluate(context.Background(), Inputs{
		Variables: []Value{Bool(true)},
		Functions: []Function{func(_ context.Context, arguments []Value) (Value, error) {
			label, _ := arguments[0].StringValue()
			return Bool(label == "stable"), nil
		}},
	})
	fmt.Println(allowed, err)
	// Output: true <nil>
}

func FuzzCompile(f *testing.F) {
	for _, source := range []string{
		`authenticated`,
		`authenticated && hasRole("admin")`,
		`subject == "owner" || issuer != "blocked"`,
		`!(authenticated && hasScope("write"))`,
		`principal.subject == "owner"`,
	} {
		f.Add(source)
	}
	f.Fuzz(func(t *testing.T, source string) {
		program, err := Compile(source, testSchema())
		if err != nil {
			return
		}
		if _, err := program.Evaluate(
			context.Background(),
			testInputs(true, "subject", "issuer", []string{"admin"}),
		); err != nil {
			t.Fatalf("Evaluate() error = %v", err)
		}
	})
}

func BenchmarkEvaluate(b *testing.B) {
	program, err := Compile(
		`authenticated && (hasRole("admin") || subject == "owner")`,
		testSchema(),
	)
	if err != nil {
		b.Fatalf("Compile() error = %v", err)
	}
	inputs := testInputs(true, "owner", "issuer", []string{"reader"})
	b.ReportAllocs()
	for b.Loop() {
		if _, err := program.Evaluate(context.Background(), inputs); err != nil {
			b.Fatalf("Evaluate() error = %v", err)
		}
	}
}

func testSchema() Schema {
	return Schema{
		Variables: []Variable{
			{Name: "authenticated", Kind: Boolean},
			{Name: "subject", Kind: String},
			{Name: "issuer", Kind: String},
		},
		Functions: []FunctionSpec{
			{Name: "hasRole", Parameters: []Kind{String}, Result: Boolean},
			{Name: "hasScope", Parameters: []Kind{String}, Result: Boolean},
		},
	}
}

func testInputs(authenticated bool, subject, issuer string, roles []string) Inputs {
	return Inputs{
		Variables: []Value{Bool(authenticated), Text(subject), Text(issuer)},
		Functions: []Function{
			func(_ context.Context, arguments []Value) (Value, error) {
				role, _ := arguments[0].StringValue()
				return Bool(slices.Contains(roles, role)), nil
			},
			func(_ context.Context, _ []Value) (Value, error) {
				return Bool(false), nil
			},
		},
	}
}
