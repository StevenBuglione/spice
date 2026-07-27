// Package policy demonstrates namespace-qualified third-party annotations.
package policy

import "github.com/StevenBuglione/spice/annotation/sdk"

// Policy classifies a type under a fixture-owned architecture policy.
//
// The strict mode contributes a stereotype. The deny mode intentionally emits
// a source-positioned plugin diagnostic, giving the integration fixture a
// deterministic failure path without hiding validation in the compiler.
//
// Use a namespace import to keep the annotation's owner visible:
//
//	// @import * as fixture from "example.com/spice-annotation-fixture/annotation/policy"
//	// @fixture.Policy(mode="strict")
//	type Settings struct{}
func Policy() sdk.Definition {
	return sdk.Definition{
		Name:    "fixture.Policy",
		Summary: "Applies the fixture's documented architecture policy.",
		Targets: []sdk.Target{sdk.TargetType},
		Arguments: []sdk.Argument{{
			Name:          "mode",
			Kinds:         []sdk.Kind{sdk.KindString},
			AllowedValues: []string{"strict", "deny"},
			Description:   "Selects strict classification or a deliberate diagnostic.",
			Default:       "strict",
		}},
		Examples: []sdk.Example{{
			Title: "Namespaced policy",
			Code:  "// @fixture.Policy(mode=\"strict\")\ntype Settings struct{}",
		}},
		Compatibility: sdk.Compatibility{
			Since:        "0.1.0",
			MinimumSpice: "0.1.0",
		},
		Implementation: sdk.Implementation{
			Tool:     "example.com/spice-annotation-fixture/cmd/spice-annotations",
			Handler:  "fixture/policy",
			Protocol: sdk.ProtocolV1Alpha1,
			Source: sdk.Symbol{
				Package: "example.com/spice-annotation-fixture/internal/handler",
				Name:    "PolicyHandler",
			},
		},
	}
}
