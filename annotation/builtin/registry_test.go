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
		positional   bool
	}{
		{name: "Application", targets: []annotation.Target{annotation.TargetFunction}},
		{name: "Configuration", targets: []annotation.Target{annotation.TargetType}},
		{name: "Controller", targets: []annotation.Target{annotation.TargetType}, argumentName: "prefix"},
		{name: "Get", targets: []annotation.Target{annotation.TargetMethod}, argumentName: "path", required: true, positional: true},
		{name: "Post", targets: []annotation.Target{annotation.TargetMethod}, argumentName: "path", required: true, positional: true},
		{name: "Service", targets: []annotation.Target{annotation.TargetType}},
	}
	registry := Registry()
	if len(registry.Definitions()) != len(tests) {
		t.Fatalf("definitions = %d, want %d", len(registry.Definitions()), len(tests))
	}
	for _, test := range tests {
		definition, ok := registry.Lookup(test.name)
		if !ok {
			t.Fatalf("missing %s", test.name)
		}
		if got := definition.Targets.Values(); !reflect.DeepEqual(got, test.targets) {
			t.Fatalf("%s targets = %#v", test.name, got)
		}
		if test.argumentName == "" {
			if len(definition.Arguments) != 0 {
				t.Fatalf("%s arguments = %#v", test.name, definition.Arguments)
			}
			continue
		}
		argument := definition.Arguments[0]
		if argument.Name != test.argumentName || argument.Required != test.required || argument.Positional != test.positional {
			t.Fatalf("%s argument = %#v", test.name, argument)
		}
		if !reflect.DeepEqual(argument.Kinds, []annotation.Kind{annotation.KindString}) {
			t.Fatalf("%s kinds = %#v", test.name, argument.Kinds)
		}
	}
}

func TestRegistryCallsAreIndependent(t *testing.T) {
	t.Parallel()
	first := Registry()
	definition, _ := first.Lookup("Controller")
	definition.Arguments[0].Name = "mutated"
	second := Registry()
	fresh, _ := second.Lookup("Controller")
	if fresh.Arguments[0].Name != "prefix" {
		t.Fatalf("fresh registry mutated: %#v", fresh)
	}
}
