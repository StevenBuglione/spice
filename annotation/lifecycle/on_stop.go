package lifecycle

import "github.com/StevenBuglione/spice/annotation/sdk"

// OnStop marks a provider-owned method that runs during graceful shutdown.
//
// The exact method signature is func(context.Context) error. Callbacks run in
// reverse dependency order before construction cleanups. Stop is idempotent
// and uses the caller-owned shutdown context.
//
//	// @import { OnStop } from "github.com/StevenBuglione/spice/annotation/lifecycle"
//	// @OnStop
//	func (*Server) Stop(context.Context) error
func OnStop() sdk.Definition {
	return sdk.Definition{
		Name:    "lifecycle.OnStop",
		Summary: "Declares a reverse-dependency application stop callback.",
		Targets: []sdk.Target{sdk.TargetMethod},
		Examples: []sdk.Example{{
			Title: "Stop callback",
			Code:  "// @OnStop\nfunc (*Server) Stop(context.Context) error",
		}},
		Compatibility: sdk.Compatibility{
			Since:        "0.1.0",
			MinimumSpice: "0.1.0",
		},
		Implementation: sdk.Implementation{
			Tool:     "github.com/StevenBuglione/spice/cmd/spice-annotation-core",
			Handler:  "lifecycle/on-stop",
			Protocol: sdk.ProtocolV1Alpha1,
			Source: sdk.Symbol{
				Package: "github.com/StevenBuglione/spice/internal/annotationcore",
				Name:    "OnStopHandler",
			},
		},
	}
}
