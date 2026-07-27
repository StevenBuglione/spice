// Package web defines canonical descriptors for generated net/http adapters.
package web

import "github.com/StevenBuglione/spice/annotation/sdk"

// Controller marks a provider-owned type whose annotated methods are exposed
// through generated net/http adapters.
//
// The receiver must have an exact provider. Spice validates method signatures,
// request DTO bindings, response handling, authorization, and route conflicts
// before generating ordinary inspectable Go. Prefix is an optional absolute
// route path shared by the controller's methods.
//
//	// @import { Controller } from "github.com/StevenBuglione/spice/annotation/web"
//	// @Controller(prefix="/orders")
//	type HTTPController struct{}
func Controller() sdk.Definition {
	return sdk.Definition{
		Name:    "web.Controller",
		Summary: "Declares a generated net/http controller.",
		Targets: []sdk.Target{sdk.TargetType},
		Arguments: []sdk.Argument{{
			Name:        "prefix",
			Kinds:       []sdk.Kind{sdk.KindString},
			Description: "Optional absolute route prefix.",
		}},
		Examples: []sdk.Example{{
			Title: "Controller",
			Code:  "// @Controller(prefix=\"/orders\")\ntype HTTPController struct{}",
		}},
		Compatibility: sdk.Compatibility{
			Since:        "0.1.0",
			MinimumSpice: "0.1.0",
		},
		Implementation: sdk.Implementation{
			Tool:     "github.com/StevenBuglione/spice/cmd/spice-annotation-core",
			Handler:  "web/controller",
			Protocol: sdk.ProtocolV1Alpha1,
			Source: sdk.Symbol{
				Package: "github.com/StevenBuglione/spice/internal/annotationcore",
				Name:    "ControllerHandler",
			},
		},
	}
}
