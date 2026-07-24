package provider

import (
	"go/types"
	"strings"
	"testing"
)

func TestCatalogRejectsAliasDuplicateOutput(t *testing.T) {
	root := writeModule(t, map[string]string{
		"go.mod": "module example.com/aliasduplicates\n\ngo 1.23.0\n",
		"providers.go": `package aliasduplicates

type Original struct{}
type Alias = Original
type Distinct Original

// @Bean
func OriginalProvider() Original { panic("provider body must not execute") }

// @Bean
func AliasProvider() Alias { panic("provider body must not execute") }

// @Bean
func DistinctProvider() Distinct { panic("provider body must not execute") }
`,
	})
	program, resolved := loadAndResolve(t, root, ".")
	catalog := buildQuiet(t, program, resolved)
	providers := catalog.Providers()
	original := providerByName(providers, "OriginalProvider")
	alias := providerByName(providers, "AliasProvider")
	distinct := providerByName(providers, "DistinctProvider")
	if original == nil || alias == nil || distinct == nil {
		t.Fatalf("providers = %#v", providers)
	}
	if !types.Identical(original.Output, alias.Output) {
		t.Fatalf("alias output %q is not identical to original output %q", alias.OutputTypeID, original.OutputTypeID)
	}
	if original.OutputTypeID == alias.OutputTypeID {
		t.Fatalf("fixture does not exercise distinct alias display IDs: both are %q", original.OutputTypeID)
	}
	if types.Identical(original.Output, distinct.Output) {
		t.Fatalf("distinct named output %q unexpectedly conflicts with %q", distinct.OutputTypeID, original.OutputTypeID)
	}

	diagnostics := catalog.Diagnostics()
	if len(diagnostics) != 1 {
		t.Fatalf("diagnostics = %v, want one alias duplicate diagnostic", diagnosticStrings(diagnostics))
	}
	message := diagnostics[0].Error()
	for _, expected := range []string{
		"exact type example.com/aliasduplicates.Alias",
		"example.com/aliasduplicates.AliasProvider",
		"example.com/aliasduplicates.OriginalProvider",
		"qualifiers and implicit interface bindings are not supported",
	} {
		if !strings.Contains(message, expected) {
			t.Fatalf("diagnostic %q missing %q", message, expected)
		}
	}
	if strings.Contains(message, "DistinctProvider") {
		t.Fatalf("distinct named provider incorrectly entered alias duplicate diagnostic: %q", message)
	}
}
