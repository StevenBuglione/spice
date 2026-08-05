package starter_test

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/spice-framework/spice/starter"
	grpcstarter "github.com/spice-framework/spice/starter/grpc"
	kafkastarter "github.com/spice-framework/spice/starter/kafka"
	mysqlstarter "github.com/spice-framework/spice/starter/mysql"
	"github.com/spice-framework/spice/starter/oauth2client"
	"github.com/spice-framework/spice/starter/oidc"
	"github.com/spice-framework/spice/starter/otel"
	redisstarter "github.com/spice-framework/spice/starter/redis"
	websocketstarter "github.com/spice-framework/spice/starter/websocket"
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
			name:     "websocket",
			manifest: websocketstarter.Manifest,
			entrypoints: []any{
				websocketstarter.NewHandler,
				websocketstarter.Dial,
			},
			capabilities: []string{
				"web.websocket.client",
				"web.websocket.server",
			},
			dependencies: []starter.Dependency{{
				Module:  "github.com/coder/websocket",
				Version: "v1.8.15",
				License: "ISC",
			}},
			activation: starter.ActivationExplicitConstructor,
		},
		{
			name:     "grpc",
			manifest: grpcstarter.Manifest,
			entrypoints: []any{
				grpcstarter.OpenServer,
				grpcstarter.OpenClient,
			},
			capabilities: []string{
				"rpc.grpc.client",
				"rpc.grpc.server",
			},
			dependencies: []starter.Dependency{{
				Module:  "google.golang.org/grpc",
				Version: "v1.82.1",
				License: "Apache-2.0",
			}},
			activation: starter.ActivationExplicitConstructor,
		},
		{
			name:     "kafka",
			manifest: kafkastarter.Manifest,
			entrypoints: []any{
				kafkastarter.Open,
				kafkastarter.OpenConsumer,
			},
			capabilities: []string{
				"messaging.kafka.consumer-group",
				"messaging.kafka.producer",
			},
			dependencies: []starter.Dependency{{
				Module:  "github.com/twmb/franz-go",
				Version: "v1.21.0",
				License: "BSD-3-Clause",
			}},
			activation: starter.ActivationExplicitConstructor,
		},
		{
			name:         "mysql",
			manifest:     mysqlstarter.Manifest,
			entrypoints:  []any{mysqlstarter.Open},
			capabilities: []string{"data.mysql", "data.sql"},
			dependencies: []starter.Dependency{
				{
					Module:  "github.com/go-sql-driver/mysql",
					Version: "v1.10.0",
					License: "MPL-2.0",
				},
			},
			activation: starter.ActivationExplicitConstructor,
		},
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
				"observability.module-events",
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
			name:         "redis",
			manifest:     redisstarter.Manifest,
			entrypoints:  []any{redisstarter.Open},
			capabilities: []string{"cache.redis", "data.redis"},
			dependencies: []starter.Dependency{
				{
					Module:  "github.com/redis/go-redis/v9",
					Version: "v9.21.0",
					License: "BSD-2-Clause",
				},
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
				spec.Module != "github.com/spice-framework/spice" ||
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
