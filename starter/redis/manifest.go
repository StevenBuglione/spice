package redis

import spicestarter "github.com/spice-framework/spice/starter"

// Manifest returns Redis starter compatibility and review metadata.
func Manifest() spicestarter.Manifest {
	return spicestarter.Must(spicestarter.Spec{
		Schema:    spicestarter.Schema,
		ID:        "github.com/spice-framework/spice/starter/redis",
		Version:   "0.1.0-dev",
		Module:    "github.com/spice-framework/spice",
		SpiceAPI:  spicestarter.APIVersion,
		MinimumGo: "1.26",
		License:   "Apache-2.0",
		Review:    "docs/dependency-reviews/go-redis.md",
		Activation: spicestarter.Activation{
			Mode: spicestarter.ActivationExplicitConstructor,
			EntryPoints: []spicestarter.EntryPoint{
				{
					Package: "github.com/spice-framework/spice/starter/redis",
					Symbol:  "Open",
				},
			},
		},
		Capabilities: []string{"cache.redis", "data.redis"},
		Dependencies: []spicestarter.Dependency{
			{
				Module:  "github.com/redis/go-redis/v9",
				Version: "v9.21.0",
				License: "BSD-2-Clause",
			},
		},
	})
}
