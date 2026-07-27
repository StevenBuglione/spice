// Package schedule defines canonical descriptors for generated scheduled
// execution.
package schedule

import "github.com/StevenBuglione/spice/annotation/sdk"

// FixedDelay marks a provider-owned method for fixed-delay execution.
//
// Delay is measured after one invocation completes. Generated scheduling owns
// cancellation, panic containment, overlap prevention, graceful shutdown, and
// observations. Duration values use time.ParseDuration syntax.
//
//	// @spice.import { FixedDelay } from "github.com/StevenBuglione/spice/annotation/schedule"
//	// @FixedDelay(delay="5m", initialDelay="30s")
func FixedDelay() sdk.Definition {
	return sdk.Definition{
		Name:    "schedule.FixedDelay",
		Summary: "Declares a generated fixed-delay scheduled method.",
		Targets: []sdk.Target{sdk.TargetMethod},
		Arguments: []sdk.Argument{
			{
				Name:        "delay",
				Kinds:       []sdk.Kind{sdk.KindString},
				Description: "Required positive time.ParseDuration delay.",
				Required:    true,
			},
			{
				Name:        "initialDelay",
				Kinds:       []sdk.Kind{sdk.KindString},
				Description: "Optional non-negative delay before the first run.",
				Default:     "0s",
			},
			{
				Name:        "continueOnError",
				Kinds:       []sdk.Kind{sdk.KindBoolean},
				Description: "Whether another run is scheduled after an error.",
				Default:     "false",
			},
		},
		Examples: []sdk.Example{{
			Title: "Inventory refresh",
			Code:  "// @FixedDelay(delay=\"5m\", initialDelay=\"30s\")",
		}},
		Compatibility: sdk.Compatibility{
			Since:        "0.1.0",
			MinimumSpice: "0.1.0",
		},
		Implementation: sdk.Implementation{
			Tool:     "github.com/StevenBuglione/spice/cmd/spice-annotation-core",
			Handler:  "schedule/fixed-delay",
			Protocol: sdk.ProtocolV1Alpha1,
			Source: sdk.Symbol{
				Package: "github.com/StevenBuglione/spice/internal/annotationcore",
				Name:    "FixedDelayHandler",
			},
		},
	}
}
