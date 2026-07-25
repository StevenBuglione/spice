package provider

import (
	"fmt"
	"testing"
)

func TestCatalogTreatsFirstCleanupResultAsProvidedOutput(t *testing.T) {
	tests := []struct {
		name         string
		declarations string
		signature    string
	}{
		{
			name:      "canonical",
			signature: "(life.Cleanup, life.Cleanup, error)",
		},
		{
			name: "aliases",
			declarations: `type OutputAlias = life.Cleanup
	type MetadataAlias = life.Cleanup
	type ErrorAlias = error`,
			signature: "(OutputAlias, MetadataAlias, ErrorAlias)",
		},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			root := writeModule(t, map[string]string{
				"go.mod": "module github.com/StevenBuglione/spice\n\ngo 1.23.0\n",
				"lifecycle/cleanup.go": `package lifecycle
import "context"
type Cleanup func(context.Context) error
`,
				"app/providers.go": fmt.Sprintf(`package app

import life "github.com/StevenBuglione/spice/lifecycle"

%s

// @Bean
func Provider() %s { panic("provider and cleanup bodies must not execute") }
`, test.declarations, test.signature),
			})

			program, resolved := loadAndResolve(t, root, "./app")
			catalog := buildQuiet(t, program, resolved)
			if diagnostics := catalog.Diagnostics(); len(diagnostics) != 0 {
				t.Fatalf("Build() diagnostics = %v", diagnosticStrings(diagnostics))
			}
			providers := catalog.Providers()
			if len(providers) != 1 {
				t.Fatalf("Providers() = %#v, want one provider", providers)
			}
			item := providers[0]
			if !item.ReturnsCleanup || !item.ReturnsError {
				t.Fatalf("flags cleanup=%v error=%v, want both true", item.ReturnsCleanup, item.ReturnsError)
			}
			if !isCleanup(item.Output) {
				t.Fatalf("first result was not retained as cleanup-typed output: %s", item.OutputTypeID)
			}
		})
	}
}
