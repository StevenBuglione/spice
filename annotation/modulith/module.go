// Package modulith defines canonical descriptors for compile-time application
// module boundaries.
package modulith

import "github.com/StevenBuglione/spice/annotation/sdk"

// Module marks package documentation as an application-module root.
//
// The root package is the module's default public API. Descendant packages are
// internal unless explicitly exposed by NamedInterface. Spice validates
// cross-module import edges, allowed dependencies, cycles, and unassigned
// packages before generation.
//
//	// @spice.import { Module } from "github.com/StevenBuglione/spice/annotation/modulith"
//	// @Module(allowedDependencies=["example.com/app/inventory"])
//	package orders
func Module() sdk.Definition {
	return sdk.Definition{
		Name:    "modulith.Module",
		Summary: "Declares a compile-time application-module root.",
		Targets: []sdk.Target{sdk.TargetPackage},
		Arguments: []sdk.Argument{{
			Name:             "allowedDependencies",
			Kinds:            []sdk.Kind{sdk.KindList},
			ListElementKinds: []sdk.Kind{sdk.KindString},
			Description:      "Explicit module import paths or named APIs this module may use.",
		}},
		Examples: []sdk.Example{{
			Title: "Module root",
			Code:  "// @Module(allowedDependencies=[\"example.com/app/inventory\"])\npackage orders",
		}},
		Compatibility: sdk.Compatibility{
			Since:        "0.1.0",
			MinimumSpice: "0.1.0",
		},
		Implementation: sdk.Implementation{
			Tool:     "github.com/StevenBuglione/spice/cmd/spice-annotation-core",
			Handler:  "modulith/module",
			Protocol: sdk.ProtocolV1Alpha1,
			Source: sdk.Symbol{
				Package: "github.com/StevenBuglione/spice/internal/annotationcore",
				Name:    "ModuleHandler",
			},
		},
	}
}
