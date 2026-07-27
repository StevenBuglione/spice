// Package management defines canonical descriptors for generated management
// endpoints.
package management

import "github.com/StevenBuglione/spice/annotation/sdk"

// Enable selects the management endpoints generated for one application.
//
// Exposure is explicit and allowlisted. Sensitive values such as
// configuration secrets are redacted. Management routes participate in the
// same module ownership, authorization, metrics, and graceful shutdown model
// as application routes.
//
//	// @import { Enable } from "github.com/StevenBuglione/spice/annotation/management"
//	// @Enable(expose=["health", "liveness", "readiness", "info"])
func Enable() sdk.Definition {
	return sdk.Definition{
		Name:    "management.Enable",
		Summary: "Selects generated management endpoints for an application.",
		Targets: []sdk.Target{sdk.TargetFunction},
		Arguments: []sdk.Argument{{
			Name:             "expose",
			Kinds:            []sdk.Kind{sdk.KindList},
			ListElementKinds: []sdk.Kind{sdk.KindString},
			AllowedValues:    []string{"health", "liveness", "readiness", "info", "metrics", "configprops", "modules"},
			Description:      "Explicit management endpoint identifiers to expose.",
			Required:         true,
		}},
		Examples: []sdk.Example{{
			Title: "Health endpoints",
			Code:  "// @Enable(expose=[\"health\", \"liveness\", \"readiness\"])",
		}},
		Compatibility: sdk.Compatibility{
			Since:        "0.1.0",
			MinimumSpice: "0.1.0",
		},
		Implementation: sdk.Implementation{
			Tool:     "github.com/StevenBuglione/spice/cmd/spice-annotation-core",
			Handler:  "management/enable",
			Protocol: sdk.ProtocolV1Alpha1,
			Source: sdk.Symbol{
				Package: "github.com/StevenBuglione/spice/internal/annotationcore",
				Name:    "ManagementEnableHandler",
			},
		},
	}
}
