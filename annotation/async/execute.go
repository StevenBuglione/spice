// Package async defines the canonical descriptor for generated asynchronous
// method execution.
package async

import "github.com/StevenBuglione/spice/annotation/sdk"

// Execute marks a provider-owned method for generated asynchronous execution.
//
// Spice validates the exact receiver, context, request, and error signature.
// Generated code owns cancellation, panic containment, bounded execution, and
// observability; the annotation never creates a hidden global worker pool.
//
//	// @import { Execute } from "github.com/StevenBuglione/spice/annotation/async"
//	// @Execute
//	func (*Mailer) Deliver(context.Context, Message) error
func Execute() sdk.Definition {
	return sdk.Definition{
		Name:    "async.Execute",
		Summary: "Declares a generated asynchronous method boundary.",
		Targets: []sdk.Target{sdk.TargetMethod},
		Examples: []sdk.Example{{
			Title: "Asynchronous method",
			Code:  "// @Execute\nfunc (*Mailer) Deliver(context.Context, Message) error",
		}},
		Compatibility: sdk.Compatibility{
			Since:        "0.1.0",
			MinimumSpice: "0.1.0",
		},
		Implementation: sdk.Implementation{
			Tool:     "github.com/StevenBuglione/spice/cmd/spice-annotation-core",
			Handler:  "async/execute",
			Protocol: sdk.ProtocolV1Alpha1,
			Source: sdk.Symbol{
				Package: "github.com/StevenBuglione/spice/internal/annotationcore",
				Name:    "AsyncExecuteHandler",
			},
		},
	}
}
