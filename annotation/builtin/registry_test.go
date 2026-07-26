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
		kinds        []annotation.Kind
		required     bool
		positional   bool
		repeatable   bool
	}{
		{name: "Application", targets: []annotation.Target{annotation.TargetFunction}},
		{name: "Bean", targets: []annotation.Target{annotation.TargetFunction}},
		{name: "Configuration", targets: []annotation.Target{annotation.TargetType}, argumentName: "prefix", kinds: []annotation.Kind{annotation.KindString}},
		{name: "Controller", targets: []annotation.Target{annotation.TargetType}, argumentName: "prefix", kinds: []annotation.Kind{annotation.KindString}},
		{name: "Get", targets: []annotation.Target{annotation.TargetMethod}, argumentName: "path", kinds: []annotation.Kind{annotation.KindString}, required: true, positional: true},
		{name: "Module", targets: []annotation.Target{annotation.TargetPackage}, argumentName: "allowedDependencies", kinds: []annotation.Kind{annotation.KindList}},
		{name: "NamedInterface", targets: []annotation.Target{annotation.TargetPackage}, argumentName: "name", kinds: []annotation.Kind{annotation.KindString}, required: true, positional: true, repeatable: true},
		{name: "OnStart", targets: []annotation.Target{annotation.TargetMethod}},
		{name: "OnStop", targets: []annotation.Target{annotation.TargetMethod}},
		{name: "management.Enable", targets: []annotation.Target{annotation.TargetFunction}, argumentName: "expose", kinds: []annotation.Kind{annotation.KindList}, required: true},
		{name: "observability.Logging", targets: []annotation.Target{annotation.TargetFunction}},
		{name: "Post", targets: []annotation.Target{annotation.TargetMethod}, argumentName: "path", kinds: []annotation.Kind{annotation.KindString}, required: true, positional: true},
		{name: "security.Authorize", targets: []annotation.Target{annotation.TargetMethod}},
		{name: "Service", targets: []annotation.Target{annotation.TargetType}},
	}

	registry := Registry()
	definitions := registry.Definitions()
	if len(definitions) != len(tests) {
		t.Fatalf("len(Definitions()) = %d, want %d", len(definitions), len(tests))
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			definition, ok := registry.Lookup(test.name)
			if !ok {
				t.Fatalf("Lookup(%q) did not find definition", test.name)
			}
			if got := definition.Targets.Values(); !reflect.DeepEqual(got, test.targets) {
				t.Fatalf("%s targets = %#v, want %#v", test.name, got, test.targets)
			}
			if definition.Repeatable != test.repeatable {
				t.Fatalf("%s repeatable = %t, want %t", test.name, definition.Repeatable, test.repeatable)
			}
			if test.name == "security.Authorize" {
				assertAuthorizationDefinition(t, definition)
				return
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
			if argument.Name != test.argumentName || argument.Required != test.required || argument.Positional != test.positional {
				t.Fatalf("%s argument = %#v", test.name, argument)
			}
			if !reflect.DeepEqual(argument.Kinds, test.kinds) {
				t.Fatalf("%s argument kinds = %#v, want %#v", test.name, argument.Kinds, test.kinds)
			}
			if test.name == "management.Enable" &&
				!reflect.DeepEqual(argument.ListElementKinds, []annotation.Kind{annotation.KindString}) {
				t.Fatalf("%s list element kinds = %#v", test.name, argument.ListElementKinds)
			}
		})
	}
}

func assertAuthorizationDefinition(
	t *testing.T,
	definition annotation.Definition,
) {
	t.Helper()
	if len(definition.Arguments) != 4 {
		t.Fatalf(
			"security.Authorize arguments = %#v, want 4",
			definition.Arguments,
		)
	}
	want := []struct {
		name string
		kind annotation.Kind
	}{
		{name: "authenticated", kind: annotation.KindBoolean},
		{name: "anyRoles", kind: annotation.KindList},
		{name: "allRoles", kind: annotation.KindList},
		{name: "allScopes", kind: annotation.KindList},
	}
	for index, expected := range want {
		argument := definition.Arguments[index]
		if argument.Name != expected.name ||
			!reflect.DeepEqual(argument.Kinds, []annotation.Kind{expected.kind}) {
			t.Fatalf(
				"security.Authorize argument %d = %#v, want %s %s",
				index,
				argument,
				expected.name,
				expected.kind,
			)
		}
		if expected.kind == annotation.KindList &&
			!reflect.DeepEqual(
				argument.ListElementKinds,
				[]annotation.Kind{annotation.KindString},
			) {
			t.Fatalf(
				"security.Authorize argument %d list kinds = %#v",
				index,
				argument.ListElementKinds,
			)
		}
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

func TestRegistryLifecycleHooks(t *testing.T) {
	for _, name := range []string{"OnStart", "OnStop"} {
		definition, ok := Registry().Lookup(name)
		if !ok || !reflect.DeepEqual(definition.Targets.Values(), []annotation.Target{annotation.TargetMethod}) || definition.Repeatable || len(definition.Arguments) != 0 {
			t.Fatalf("%s definition = %#v, want argument-free non-repeatable method marker", name, definition)
		}
	}
}
