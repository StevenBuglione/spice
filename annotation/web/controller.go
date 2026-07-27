// Package web defines canonical descriptors for generated net/http adapters.
package web

import (
	"context"

	"github.com/StevenBuglione/spice/annotation/coretool"
	"github.com/StevenBuglione/spice/annotation/sdk"
)

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
			Tool:     coretool.Path,
			Handler:  ControllerHandler,
			Protocol: sdk.ProtocolV1Alpha2,
		},
	}
}

// ControllerHandler contributes generated net/http controller semantics.
func ControllerHandler(
	_ context.Context,
	invocation sdk.Invocation,
) (sdk.Result, error) {
	if err := invocation.RequireDescriptor(
		"github.com/StevenBuglione/spice/annotation/web",
		"Controller",
	); err != nil {
		return sdk.Result{}, err
	}
	arguments, err := sdk.BindArguments(invocation, "", "prefix")
	if err != nil {
		return sdk.Result{}, err
	}
	prefix, err := arguments.String("prefix", false)
	if err != nil {
		return sdk.Result{}, err
	}
	return sdk.OneContribution(sdk.Contribution{
		Kind: sdk.ContributionController,
		Controller: &sdk.ControllerContribution{
			Prefix: prefix,
		},
	})
}
