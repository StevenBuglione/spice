// Package lifecycle defines canonical descriptors for generated application
// lifecycle callbacks.
package lifecycle

import "github.com/StevenBuglione/spice/annotation/sdk"

// OnStart marks a provider-owned method that runs after dependency
// construction and before the application becomes ready.
//
// The exact method signature is func(context.Context) error. Callbacks start
// dependency-first. A failure rolls back already-started callbacks and
// constructed providers in reverse deterministic order.
//
//	// @spice.import { OnStart } from "github.com/StevenBuglione/spice/annotation/lifecycle"
//	// @OnStart
//	func (*Server) Start(context.Context) error
func OnStart() sdk.Definition {
	return sdk.Definition{
		Name:    "lifecycle.OnStart",
		Summary: "Declares a dependency-ordered application start callback.",
		Targets: []sdk.Target{sdk.TargetMethod},
		Examples: []sdk.Example{{
			Title: "Start callback",
			Code:  "// @OnStart\nfunc (*Server) Start(context.Context) error",
		}},
		Compatibility: sdk.Compatibility{
			Since:        "0.1.0",
			MinimumSpice: "0.1.0",
		},
		Implementation: sdk.Implementation{
			Tool:     "github.com/StevenBuglione/spice/cmd/spice-annotation-core",
			Handler:  "lifecycle/on-start",
			Protocol: sdk.ProtocolV1Alpha1,
			Source: sdk.Symbol{
				Package: "github.com/StevenBuglione/spice/internal/annotationcore",
				Name:    "OnStartHandler",
			},
		},
	}
}
