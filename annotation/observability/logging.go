// Package observability defines canonical descriptors for generated
// observability features.
package observability

import "github.com/StevenBuglione/spice/annotation/sdk"

// Logging enables structured application and module lifecycle logging.
//
// Generated observations use slog-compatible structured records and preserve
// application, module, component, operation, and phase fields. No global
// logger is installed; the logger remains an explicit dependency.
//
//	// @spice.import { Logging } from "github.com/StevenBuglione/spice/annotation/observability"
//	// @Logging
func Logging() sdk.Definition {
	return sdk.Definition{
		Name:    "observability.Logging",
		Summary: "Enables generated structured lifecycle logging.",
		Targets: []sdk.Target{sdk.TargetFunction},
		Examples: []sdk.Example{{
			Title: "Structured logging",
			Code:  "// @Logging",
		}},
		Compatibility: sdk.Compatibility{
			Since:        "0.1.0",
			MinimumSpice: "0.1.0",
		},
		Implementation: sdk.Implementation{
			Tool:     "github.com/StevenBuglione/spice/cmd/spice-annotation-core",
			Handler:  "observability/logging",
			Protocol: sdk.ProtocolV1Alpha1,
			Source: sdk.Symbol{
				Package: "github.com/StevenBuglione/spice/internal/annotationcore",
				Name:    "ObservabilityLoggingHandler",
			},
		},
	}
}
