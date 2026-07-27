package core

import (
	"context"

	"github.com/StevenBuglione/spice/annotation/coretool"
	"github.com/StevenBuglione/spice/annotation/sdk"
)

// Service classifies a type as an application service for architecture,
// documentation, navigation, and observability metadata.
//
// Service deliberately does not construct the type or register a hidden
// container entry. Add an explicit @Bean provider when the service must
// participate in dependency injection. This separation keeps initialization,
// dependencies, errors, and cleanup visible as ordinary Go.
//
//	// @import { Bean, Service } from "github.com/StevenBuglione/spice/annotation/core"
//	// @Service
//	type Orders struct{}
//
//	// @Bean
//	func NewOrders() *Orders
func Service() sdk.Definition {
	return sdk.Definition{
		Name:    "core.Service",
		Summary: "Classifies a type as an explicit application service.",
		Targets: []sdk.Target{sdk.TargetType},
		Examples: []sdk.Example{{
			Title: "Service stereotype with explicit provider",
			Code:  "// @Service\ntype Orders struct{}\n\n// @Bean\nfunc NewOrders() *Orders",
		}},
		Compatibility: sdk.Compatibility{
			Since:        "0.1.0",
			MinimumSpice: "0.1.0",
		},
		Implementation: sdk.Implementation{
			Tool:     coretool.Path,
			Handler:  ServiceHandler,
			Protocol: sdk.ProtocolV1Alpha2,
		},
	}
}

// ServiceHandler contributes explicit service stereotype semantics.
func ServiceHandler(
	_ context.Context,
	invocation sdk.Invocation,
) (sdk.Result, error) {
	if err := invocation.RequireDescriptor(
		"github.com/StevenBuglione/spice/annotation/core",
		"Service",
	); err != nil {
		return sdk.Result{}, err
	}
	if _, err := sdk.BindArguments(invocation, ""); err != nil {
		return sdk.Result{}, err
	}
	return sdk.OneContribution(sdk.Contribution{
		Kind: sdk.ContributionStereotype,
		Stereotype: &sdk.StereotypeContribution{
			Role: "service",
		},
	})
}
