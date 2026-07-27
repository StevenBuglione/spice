package core

import "github.com/StevenBuglione/spice/annotation/sdk"

// Configuration marks a struct as generated typed configuration.
//
// Field metadata remains on ordinary Go struct tags. Spice derives stable
// keys, defaults, required values, secret redaction, validation, and generated
// metadata without reflecting over the application at runtime. Prefix is
// optional and must be dot-separated identifiers.
//
//	// @spice.import { Configuration } from "github.com/StevenBuglione/spice/annotation/core"
//	// @Configuration(prefix="orders")
//	type Settings struct {
//		Limit int `spice:"limit,default=100"`
//	}
func Configuration() sdk.Definition {
	return sdk.Definition{
		Name:    "core.Configuration",
		Summary: "Declares a generated typed configuration struct.",
		Targets: []sdk.Target{sdk.TargetType},
		Arguments: []sdk.Argument{{
			Name:        "prefix",
			Kinds:       []sdk.Kind{sdk.KindString},
			Description: "Optional dot-separated property-key prefix.",
		}},
		Examples: []sdk.Example{{
			Title: "Configuration",
			Code:  "// @Configuration(prefix=\"orders\")\ntype Settings struct{}",
		}},
		Compatibility: sdk.Compatibility{
			Since:        "0.1.0",
			MinimumSpice: "0.1.0",
		},
		Implementation: sdk.Implementation{
			Tool:     "github.com/StevenBuglione/spice/cmd/spice-annotation-core",
			Handler:  "core/configuration",
			Protocol: sdk.ProtocolV1Alpha1,
			Source: sdk.Symbol{
				Package: "github.com/StevenBuglione/spice/internal/annotationcore",
				Name:    "ConfigurationHandler",
			},
		},
	}
}
