// Package core defines the canonical descriptors for Spice's application and
// dependency-injection annotations.
package core

import "github.com/StevenBuglione/spice/annotation/sdk"

// Application marks the package-level function that defines a Spice
// application target.
//
// The function remains ordinary valid Go. Spice inspects its exact parameter
// types as application roots and never executes its body during analysis.
// Argument-free package-main markers use automatic local-module discovery.
// Generated NewApplication, Start, Stop, and Run code owns construction,
// rollback, lifecycle ordering, and shutdown; no runtime reflection or global
// service locator is introduced.
//
// Use one explicit import in every file that declares the marker:
//
//	// @import { Application } from "github.com/StevenBuglione/spice/annotation/core"
//	// @Application
//	func main() {}
func Application() sdk.Definition {
	return sdk.Definition{
		Name:    "core.Application",
		Summary: "Defines a compile-time Spice application target.",
		Targets: []sdk.Target{sdk.TargetFunction},
		Examples: []sdk.Example{{
			Title: "Package-main application",
			Code:  "// @Application\nfunc main() {}",
		}},
		Compatibility: sdk.Compatibility{
			Since:        "0.1.0",
			MinimumSpice: "0.1.0",
		},
		Implementation: sdk.Implementation{
			Tool:     "github.com/StevenBuglione/spice/cmd/spice-annotation-core",
			Handler:  "core/application",
			Protocol: sdk.ProtocolV1Alpha1,
			Source: sdk.Symbol{
				Package: "github.com/StevenBuglione/spice/internal/annotationcore",
				Name:    "ApplicationHandler",
			},
		},
	}
}
