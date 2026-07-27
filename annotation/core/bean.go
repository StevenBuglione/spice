package core

import "github.com/StevenBuglione/spice/annotation/sdk"

// Bean marks a package-level provider function.
//
// Spice derives dependencies and output from exact Go type identity. A
// provider may additionally return lifecycle.Cleanup and error. Cleanup is
// registered immediately after construction and runs in reverse order during
// rollback or shutdown. Interface outputs require an explicit adapter
// provider; assignability is never guessed.
//
//	// @spice.import { Bean } from "github.com/StevenBuglione/spice/annotation/core"
//	// @Bean
//	func NewStore(config Config) (*Store, lifecycle.Cleanup, error)
func Bean() sdk.Definition {
	return sdk.Definition{
		Name:    "core.Bean",
		Summary: "Declares an exact-type dependency provider.",
		Targets: []sdk.Target{sdk.TargetFunction},
		Examples: []sdk.Example{{
			Title: "Provider",
			Code:  "// @Bean\nfunc NewStore(config Config) (*Store, error)",
		}},
		Compatibility: sdk.Compatibility{
			Since:        "0.1.0",
			MinimumSpice: "0.1.0",
		},
		Implementation: sdk.Implementation{
			Tool:     "github.com/StevenBuglione/spice/cmd/spice-annotation-core",
			Handler:  "core/bean",
			Protocol: sdk.ProtocolV1Alpha1,
			Source: sdk.Symbol{
				Package: "github.com/StevenBuglione/spice/internal/annotationcore",
				Name:    "BeanHandler",
			},
		},
	}
}
