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
		activation   starter.ActivationMode
		annotation   string
		feature      string
		requirements []string
	}{
		{
			name:         "oauth2client",
			manifest:     oauth2client.Manifest,
			entrypoints:  []any{oauth2client.NewClient},
			capabilities: []string{"security.oauth2-client-credentials"},
			dependencies: []starter.Dependency{
				{Module: "golang.org/x/oauth2", Version: "v0.36.0", License: "BSD-3-Clause"},
			},
			activation: starter.ActivationExplicitConstructor,
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
			activation: starter.ActivationExplicitConstructor,
		},
		{
			name:        "otel",
			manifest:    otel.Manifest,
			entrypoints: []any{otel.NewHTTPObserver},
			capabilities: []string{
				"observability.http-server",
				"observability.metrics",
				"observability.tracing",
			},
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
			activation: starter.ActivationExplicitAnnotation,
			annotation: "otel.Enable",
			feature:    "observability.http-server",
			requirements: []string{
				"http.serve-mux",
			},
		},
		{
			name:         "postgres",
			manifest:     postgres.Manifest,
			entrypoints:  []any{postgres.Open},
			capabilities: []string{"data.postgresql", "data.sql", "migration.postgresql"},
			dependencies: []starter.Dependency{
				{Module: "github.com/jackc/pgx/v5", Version: "v5.10.0", License: "MIT"},
			},
			activation: starter.ActivationExplicitConstructor,
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
				spec.Activation.Mode != test.activation {
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
			if test.annotation == "" {
				if len(spec.Annotations) != 0 ||
					len(spec.ApplicationFeatures) != 0 ||
					len(manifest.Definitions()) != 0 {
					t.Fatal("constructor starter unexpectedly declares annotation activation")
				}
			} else if len(spec.Annotations) != 1 ||
				spec.Annotations[0].Name != test.annotation ||
				len(spec.ApplicationFeatures) != 1 ||
				spec.ApplicationFeatures[0].Annotation != test.annotation ||
				spec.ApplicationFeatures[0].Capability != test.feature ||
				!slices.Equal(
					spec.ApplicationFeatures[0].Requirements,
					test.requirements,
				) ||
				len(manifest.Definitions()) != 1 {
				t.Fatalf(
					"annotation activation = annotations %#v features %#v definitions %#v",
					spec.Annotations,
					spec.ApplicationFeatures,
					manifest.Definitions(),
				)
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
