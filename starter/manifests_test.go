package starter_test

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/StevenBuglione/spice/starter"
	"github.com/StevenBuglione/spice/starter/oauth2client"
	"github.com/StevenBuglione/spice/starter/oidc"
	"github.com/StevenBuglione/spice/starter/otel"
	"github.com/StevenBuglione/spice/starter/postgres"
)

func TestShippedStarterManifests(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name         string
		manifest     func() starter.Manifest
		entrypoints  []any
		capabilities []string
		dependencies []starter.Dependency
	}{
		{
			name:         "oauth2client",
			manifest:     oauth2client.Manifest,
			entrypoints:  []any{oauth2client.NewClient},
			capabilities: []string{"security.oauth2-client-credentials"},
			dependencies: []starter.Dependency{
				{Module: "golang.org/x/oauth2", Version: "v0.36.0", License: "BSD-3-Clause"},
			},
		},
		{
			name:         "oidc",
			manifest:     oidc.Manifest,
			entrypoints:  []any{oidc.Discover, oidc.NewResourceServer},
			capabilities: []string{"security.oidc-resource-server"},
			dependencies: []starter.Dependency{
				{
					Module:  "github.com/coreos/go-oidc/v3",
					Version: "v3.20.0",
					License: "Apache-2.0",
				},
			},
		},
		{
			name:         "otel",
			manifest:     otel.Manifest,
			entrypoints:  []any{otel.NewHTTPObserver},
			capabilities: []string{"observability.metrics", "observability.tracing"},
			dependencies: []starter.Dependency{
				{
					Module:  "go.opentelemetry.io/otel",
					Version: "v1.43.0",
					License: "Apache-2.0",
				},
				{
					Module:  "go.opentelemetry.io/otel/metric",
					Version: "v1.43.0",
					License: "Apache-2.0",
				},
				{
					Module:  "go.opentelemetry.io/otel/trace",
					Version: "v1.43.0",
					License: "Apache-2.0",
				},
			},
		},
		{
			name:         "postgres",
			manifest:     postgres.Manifest,
			entrypoints:  []any{postgres.Open},
			capabilities: []string{"data.postgresql", "data.sql"},
			dependencies: []starter.Dependency{
				{Module: "github.com/jackc/pgx/v5", Version: "v5.10.0", License: "MIT"},
			},
		},
	}
	seenIDs := make(map[string]struct{}, len(tests))
	for _, test := range tests {
		id := test.manifest().Spec().ID
		if _, duplicate := seenIDs[id]; duplicate {
			t.Fatalf("starter ID %q is duplicated", id)
		}
		seenIDs[id] = struct{}{}
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			manifest := test.manifest()
			spec := manifest.Spec()
			if spec.Schema != starter.Schema ||
				spec.Version != "0.1.0-dev" ||
				spec.Module != "github.com/StevenBuglione/spice" ||
				spec.SpiceAPI != starter.APIVersion ||
				spec.MinimumGo != "1.26" ||
				spec.License != "Apache-2.0" ||
				spec.Activation.Mode != starter.ActivationExplicitConstructor {
				t.Fatalf("Manifest() = %#v", spec)
			}
			if len(spec.Activation.EntryPoints) != len(test.entrypoints) {
				t.Fatalf(
					"entrypoints = %#v, compile-time references = %d",
					spec.Activation.EntryPoints,
					len(test.entrypoints),
				)
			}
			if !slices.Equal(spec.Capabilities, test.capabilities) ||
				!slices.Equal(spec.Dependencies, test.dependencies) {
				t.Fatalf(
					"capabilities=%#v dependencies=%#v",
					spec.Capabilities,
					spec.Dependencies,
				)
			}
			if len(spec.Annotations) != 0 ||
				len(spec.ApplicationFeatures) != 0 ||
				len(manifest.Definitions()) != 0 {
				t.Fatal("constructor starter unexpectedly declares annotation activation")
			}
			if err := manifest.Compatible(starter.APIVersion, "go1.26.5"); err != nil {
				t.Fatalf("Compatible() error = %v", err)
			}
			content, err := manifest.JSON()
			if err != nil {
				t.Fatalf("JSON() error = %v", err)
			}
			if _, err := starter.Parse(content); err != nil {
				t.Fatalf("Parse(JSON()) error = %v", err)
			}
			review := filepath.Join("..", filepath.FromSlash(spec.Review))
			if info, err := os.Stat(review); err != nil || info.IsDir() {
				t.Fatalf("review %q is not a file: info=%v err=%v", review, info, err)
			}
		})
	}
}
