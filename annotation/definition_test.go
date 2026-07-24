package annotation

import (
	"reflect"
	"strings"
	"testing"
)

func TestNewTargetSetRejectsUnknownTarget(t *testing.T) {
	t.Parallel()
	if _, err := NewTargetSet(Target("unsupported")); err == nil {
		t.Fatal("NewTargetSet() did not reject unknown target")
	}
}

func TestTargetSetValuesAreDeterministic(t *testing.T) {
	t.Parallel()
	set := Targets(TargetMethod, TargetPackage, TargetType)
	want := []Target{TargetPackage, TargetType, TargetMethod}
	if got := set.Values(); !reflect.DeepEqual(got, want) {
		t.Fatalf("Values() = %#v, want %#v", got, want)
	}
}

func TestRegistryRejectsInvalidDefinitions(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		definitions []Definition
		message     string
	}{
		{name: "missing name", definitions: []Definition{{Targets: Targets(TargetType)}}},
		{name: "missing targets", definitions: []Definition{{Name: "Controller"}}},
		{name: "duplicate definition", definitions: []Definition{{Name: "Controller", Targets: Targets(TargetType)}, {Name: "Controller", Targets: Targets(TargetType)}}},
		{name: "missing argument name", definitions: []Definition{{Name: "Controller", Targets: Targets(TargetType), Arguments: []ArgumentDefinition{{Kinds: []Kind{KindString}}}}}},
		{name: "missing argument kind", definitions: []Definition{{Name: "Controller", Targets: Targets(TargetType), Arguments: []ArgumentDefinition{{Name: "prefix"}}}}},
		{name: "ambiguous positional metadata", message: "multiple positional arguments", definitions: []Definition{{
			Name: "Route", Targets: Targets(TargetMethod), Arguments: []ArgumentDefinition{
				{Name: "path", Kinds: []Kind{KindString}, Positional: true},
				{Name: "method", Kinds: []Kind{KindString}, Positional: true},
			},
		}}},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			_, err := NewRegistry(test.definitions...)
			if err == nil {
				t.Fatal("NewRegistry() did not reject invalid definitions")
			}
			if test.message != "" && !strings.Contains(err.Error(), test.message) {
				t.Fatalf("error = %q, want %q", err, test.message)
			}
		})
	}
}

func TestRegistryReturnsDefensiveCopies(t *testing.T) {
	t.Parallel()
	registry, err := NewRegistry(Definition{Name: "Controller", Targets: Targets(TargetType), Arguments: []ArgumentDefinition{{Name: "prefix", Kinds: []Kind{KindString}}}})
	if err != nil {
		t.Fatal(err)
	}
	first, _ := registry.Lookup("Controller")
	first.Arguments[0].Name = "changed"
	first.Arguments[0].Kinds[0] = KindInteger
	second, _ := registry.Lookup("Controller")
	if second.Arguments[0].Name != "prefix" || second.Arguments[0].Kinds[0] != KindString {
		t.Fatalf("registry state mutated: %#v", second)
	}
}

func TestRegistryDefinitionsAreSorted(t *testing.T) {
	t.Parallel()
	registry, err := NewRegistry(Definition{Name: "Service", Targets: Targets(TargetType)}, Definition{Name: "Controller", Targets: Targets(TargetType)})
	if err != nil {
		t.Fatal(err)
	}
	definitions := registry.Definitions()
	if got := []string{definitions[0].Name, definitions[1].Name}; !reflect.DeepEqual(got, []string{"Controller", "Service"}) {
		t.Fatalf("Definitions() names = %#v", got)
	}
}
