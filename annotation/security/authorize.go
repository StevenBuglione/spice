// Package security defines canonical descriptors for generated authorization
// policies.
package security

import "github.com/StevenBuglione/spice/annotation/sdk"

// Authorize attaches a secure-deny authorization policy to one HTTP route.
//
// At least one authentication, role, or scope requirement is required.
// Multiple categories are combined with AND semantics; anyRoles requires one
// listed role while allRoles and allScopes require every listed value.
//
//	// @import { Authorize } from "github.com/StevenBuglione/spice/annotation/security"
//	// @Authorize(authenticated=true, anyRoles=["operator", "admin"])
func Authorize() sdk.Definition {
	return sdk.Definition{
		Name:    "security.Authorize",
		Summary: "Declares a generated secure-deny route authorization policy.",
		Targets: []sdk.Target{sdk.TargetMethod},
		Arguments: []sdk.Argument{
			{
				Name:        "authenticated",
				Kinds:       []sdk.Kind{sdk.KindBoolean},
				Description: "Require an authenticated principal.",
				Default:     "false",
			},
			{
				Name:             "anyRoles",
				Kinds:            []sdk.Kind{sdk.KindList},
				ListElementKinds: []sdk.Kind{sdk.KindString},
				Description:      "Require at least one listed role.",
			},
			{
				Name:             "allRoles",
				Kinds:            []sdk.Kind{sdk.KindList},
				ListElementKinds: []sdk.Kind{sdk.KindString},
				Description:      "Require every listed role.",
			},
			{
				Name:             "allScopes",
				Kinds:            []sdk.Kind{sdk.KindList},
				ListElementKinds: []sdk.Kind{sdk.KindString},
				Description:      "Require every listed OAuth2 scope.",
			},
		},
		Examples: []sdk.Example{{
			Title: "Authenticated operators",
			Code:  "// @Authorize(authenticated=true, anyRoles=[\"operator\"])",
		}},
		Compatibility: sdk.Compatibility{
			Since:        "0.1.0",
			MinimumSpice: "0.1.0",
		},
		Implementation: sdk.Implementation{
			Tool:     "github.com/StevenBuglione/spice/cmd/spice-annotation-core",
			Handler:  "security/authorize",
			Protocol: sdk.ProtocolV1Alpha1,
			Source: sdk.Symbol{
				Package: "github.com/StevenBuglione/spice/internal/annotationcore",
				Name:    "AuthorizeHandler",
			},
		},
	}
}
