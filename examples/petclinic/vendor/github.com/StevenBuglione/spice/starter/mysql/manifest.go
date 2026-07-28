package mysql

import spicestarter "github.com/StevenBuglione/spice/starter"

// Manifest returns MySQL starter compatibility and review metadata.
func Manifest() spicestarter.Manifest {
	return spicestarter.Must(spicestarter.Spec{
		Schema:    spicestarter.Schema,
		ID:        "github.com/StevenBuglione/spice/starter/mysql",
		Version:   "0.1.0-dev",
		Module:    "github.com/StevenBuglione/spice",
		SpiceAPI:  spicestarter.APIVersion,
		MinimumGo: "1.26",
		License:   "Apache-2.0",
		Review:    "docs/dependency-reviews/go-sql-driver-mysql.md",
		Activation: spicestarter.Activation{
			Mode: spicestarter.ActivationExplicitConstructor,
			EntryPoints: []spicestarter.EntryPoint{
				{
					Package: "github.com/StevenBuglione/spice/starter/mysql",
					Symbol:  "Open",
				},
			},
		},
		Capabilities: []string{
			"data.mysql",
			"data.sql",
		},
		Dependencies: []spicestarter.Dependency{
			{
				Module:  "github.com/go-sql-driver/mysql",
				Version: "v1.10.0",
				License: "MPL-2.0",
			},
		},
	})
}
