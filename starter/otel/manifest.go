package otel

import spicestarter "github.com/StevenBuglione/spice/starter"

// Manifest returns OpenTelemetry starter compatibility and review metadata.
func Manifest() spicestarter.Manifest {
	return spicestarter.Must(spicestarter.Spec{
		Schema:    spicestarter.Schema,
		ID:        "github.com/StevenBuglione/spice/starter/otel",
		Version:   "0.1.0-dev",
		Module:    "github.com/StevenBuglione/spice",
		SpiceAPI:  spicestarter.APIVersion,
		MinimumGo: "1.26",
		License:   "Apache-2.0",
		Review:    "docs/dependency-reviews/opentelemetry-go.md",
		Activation: spicestarter.Activation{
			Mode: spicestarter.ActivationExplicitConstructor,
			EntryPoints: []spicestarter.EntryPoint{
				{
					Package: "github.com/StevenBuglione/spice/starter/otel",
					Symbol:  "NewHTTPObserver",
				},
			},
		},
		Capabilities: []string{"observability.metrics", "observability.tracing"},
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
