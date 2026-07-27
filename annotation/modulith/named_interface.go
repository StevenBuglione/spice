package modulith

import "github.com/StevenBuglione/spice/annotation/sdk"

// NamedInterface exposes one descendant package as a named module API.
//
// The descriptor is repeatable on package documentation. Consumers reference
// the owning module and API explicitly; a named interface never exposes other
// descendant packages by implication.
//
//	// @import { NamedInterface } from "github.com/StevenBuglione/spice/annotation/modulith"
//	// @NamedInterface("events")
//	package api
func NamedInterface() sdk.Definition {
	return sdk.Definition{
		Name:       "modulith.NamedInterface",
		Summary:    "Exposes one package as a named module API.",
		Targets:    []sdk.Target{sdk.TargetPackage},
		Repeatable: true,
		Arguments: []sdk.Argument{{
			Name:        "name",
			Kinds:       []sdk.Kind{sdk.KindString},
			Description: "Stable API name exported by the owning module.",
			Required:    true,
			Positional:  true,
		}},
		Examples: []sdk.Example{{
			Title: "Named API",
			Code:  "// @NamedInterface(\"events\")\npackage api",
		}},
		Compatibility: sdk.Compatibility{
			Since:        "0.1.0",
			MinimumSpice: "0.1.0",
		},
		Implementation: sdk.Implementation{
			Tool:     "github.com/StevenBuglione/spice/cmd/spice-annotation-core",
			Handler:  "modulith/named-interface",
			Protocol: sdk.ProtocolV1Alpha1,
			Source: sdk.Symbol{
				Package: "github.com/StevenBuglione/spice/internal/annotationcore",
				Name:    "NamedInterfaceHandler",
			},
		},
	}
}
