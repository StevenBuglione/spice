package builtin

import (
	"reflect"
	"testing"

	"github.com/StevenBuglione/spice/annotation"
)

func TestRegistryContainsBuiltInDefinitions(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		targets      []annotation.Target
		argumentName string
		required     bool
	}{
		{name: "Application", targets: []annotation.Target{annotation.TargetFunction}},
		{name: "Configuration", targets: []annotation.Target{annotation.TargetType}},
		{name: "Controller", targets: []annotation.Target{annotation.TargetType}, argumentName: "prefix"},
		{name: "Get", targets: []annotation.Target{annotation.TargetMethod}, argumentName: "path", required: true},
		{name: "Post", targets: []annotation.Target{annotation.TargetMethod}, argumentName: "path", required: true},
		{name: "Service", targets: []annotation.Target{annotation.TargetType}},
	}

	registry := Registry()
	definitions := registry.Definitions()
	if len(definitions) != len(tests) {
		t.Fatalf("len(Definitions()) = %d, want %d", len(definitions), len(tests))
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			definition, ok := registry.Lookup(test.name)
			if !ok {
				t.Fatalf("Lookup(%q) did not find definition", test.name)
			}
			if got := definition.Targets.Values(); !reflect.DeepEqual(got, test.targets) {
				t.Fatalf("%s targets = %#v, want %#v", test.name, got, test.targets)
			}
			if definition.Repeatable {
				t.Fatalf("%s unexpectedly repeatable", test.name)
			}
			if test.argumentName == "" {
				if len(definition.Arguments) != 0 {
					t.Fatalf("%s arguments = %#v, want none", test.name, definition.Arguments)
				}
				return
			}
			if len(definition.Arguments) != 1 {
				t.Fatalf("%s argument count = %d, want 1", test.name, len(definition.Arguments))
			}
			argument := definition.Arguments[0]
			if argument.Name != test.argumentName || argument.Required != test.required {
				t.Fatalf("%s argument = %#v", test.name, argument)
			}
			if !reflect.DeepEqual(argument.Kinds, []annotation.Kind{annotation.KindString}) {
				t.Fatalf("%s argument kinds = %#v", test.name, argument.Kinds)
			}
		})
	}
}

func TestRegistryCallsAreIndependent(t *testing.T) {
	t.Parallel()

	first := Registry()
	definition, ok := first.Lookup("Controller")
	if !ok {
		t.Fatal("Controller definition missing")
	}
	definition.Arguments[0].Name = "mutated"

	second := Registry()
	fresh, ok := second.Lookup("Controller")
	if !ok {
		t.Fatal("Controller definition missing from fresh registry")
	}
	if fresh.Arguments[0].Name != "prefix" {
		t.Fatalf("fresh registry was affected by previous mutation: %#v", fresh)
	}
}
