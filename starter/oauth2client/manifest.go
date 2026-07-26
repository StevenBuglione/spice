package oauth2client

import spicestarter "github.com/StevenBuglione/spice/starter"

// Manifest returns OAuth2 client-credentials compatibility and review metadata.
func Manifest() spicestarter.Manifest {
	return spicestarter.Must(spicestarter.Spec{
		Schema:    spicestarter.Schema,
		ID:        "github.com/StevenBuglione/spice/starter/oauth2client",
		Version:   "0.1.0-dev",
		Module:    "github.com/StevenBuglione/spice",
		SpiceAPI:  spicestarter.APIVersion,
		MinimumGo: "1.26",
		License:   "Apache-2.0",
		Review:    "docs/dependency-reviews/oauth2.md",
		Activation: spicestarter.Activation{
			Mode: spicestarter.ActivationExplicitConstructor,
			EntryPoints: []spicestarter.EntryPoint{
				{
					Package: "github.com/StevenBuglione/spice/starter/oauth2client",
					Symbol:  "NewClient",
				},
			},
		},
		Capabilities: []string{"security.oauth2-client-credentials"},
		Dependencies: []spicestarter.Dependency{
			{
				Module:  "golang.org/x/oauth2",
				Version: "v0.36.0",
				License: "BSD-3-Clause",
			},
		},
	})
}
