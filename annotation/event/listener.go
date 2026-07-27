package event

import "github.com/StevenBuglione/spice/annotation/sdk"

// Listener marks a provider-owned method as a typed event listener.
//
// Spice matches the method payload to an exact topic payload type and orders
// listeners by order, module, and symbol identity. The generated invocation
// propagates context and errors without a global event bus.
//
//	// @import { Listener } from "github.com/StevenBuglione/spice/annotation/event"
//	// @Listener(order=10)
func Listener() sdk.Definition {
	return sdk.Definition{
		Name:    "event.Listener",
		Summary: "Declares an ordered typed application event listener.",
		Targets: []sdk.Target{sdk.TargetMethod},
		Arguments: []sdk.Argument{{
			Name:        "order",
			Kinds:       []sdk.Kind{sdk.KindInteger},
			Description: "Stable listener order; lower values run first.",
			Default:     "0",
		}},
		Examples: []sdk.Example{{
			Title: "Ordered listener",
			Code:  "// @Listener(order=10)",
		}},
		Compatibility: sdk.Compatibility{
			Since:        "0.1.0",
			MinimumSpice: "0.1.0",
		},
		Implementation: sdk.Implementation{
			Tool:     "github.com/StevenBuglione/spice/cmd/spice-annotation-core",
			Handler:  "event/listener",
			Protocol: sdk.ProtocolV1Alpha1,
			Source: sdk.Symbol{
				Package: "github.com/StevenBuglione/spice/internal/annotationcore",
				Name:    "EventListenerHandler",
			},
		},
	}
}
