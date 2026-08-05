package otel

import (
	"github.com/spice-framework/spice/annotation"
	spicestarter "github.com/spice-framework/spice/starter"
)

// Manifest returns OpenTelemetry starter compatibility and review metadata.
func Manifest() spicestarter.Manifest {
	return spicestarter.Must(spicestarter.Spec{
		Schema:    spicestarter.Schema,
		ID:        "github.com/spice-framework/spice/starter/otel",
		Version:   "0.1.0-dev",
		Module:    "github.com/spice-framework/spice",
		SpiceAPI:  spicestarter.APIVersion,
		MinimumGo: "1.26",
		License:   "Apache-2.0",
		Review:    "docs/dependency-reviews/opentelemetry-go.md",
		Activation: spicestarter.Activation{
			Mode: spicestarter.ActivationExplicitAnnotation,
			EntryPoints: []spicestarter.EntryPoint{
				{
					Package: "github.com/spice-framework/spice/starter/otel",
					Symbol:  "NewHTTPObserver",
				},
			},
		},
		Capabilities: []string{
			"observability.http-server",
			"observability.metrics",
			"observability.module-events",
			"observability.tracing",
		},
		Annotations: []spicestarter.AnnotationSpec{
			{
				Name:    "otel.Enable",
				Targets: []annotation.Target{annotation.TargetFunction},
			},
		},
		ApplicationFeatures: []spicestarter.FeatureSpec{
			{
				Annotation: "otel.Enable",
				Capability: "observability.http-server",
				EntryPoints: []spicestarter.EntryPoint{
					{
						Package: "github.com/spice-framework/spice/starter/otel",
						Symbol:  "NewHTTPObserver",
					},
				},
				Requirements: []string{"http.serve-mux"},
			},
		},
		Dependencies: []spicestarter.Dependency{
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
	})
}
