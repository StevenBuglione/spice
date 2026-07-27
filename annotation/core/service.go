package core

import "github.com/StevenBuglione/spice/annotation/sdk"

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
			Tool:     "github.com/StevenBuglione/spice/cmd/spice-annotation-core",
			Handler:  "core/service",
			Protocol: sdk.ProtocolV1Alpha1,
			Source: sdk.Symbol{
				Package: "github.com/StevenBuglione/spice/internal/annotationcore",
				Name:    "ServiceHandler",
			},
		},
	}
}
