// Package data defines canonical descriptors for generated data boundaries.
package data

import "github.com/StevenBuglione/spice/annotation/sdk"

// Transactional marks a provider-owned method that must run inside an explicit
// database/sql transaction.
//
// The generated wrapper propagates context, commits only on success, rolls
// back on error or panic, and records the owning application module.
// Isolation is optional and readOnly defaults to false.
//
//	// @spice.import { Transactional } from "github.com/StevenBuglione/spice/annotation/data"
//	// @Transactional(isolation="serializable")
func Transactional() sdk.Definition {
	return sdk.Definition{
		Name:    "data.Transactional",
		Summary: "Declares a generated database transaction boundary.",
		Targets: []sdk.Target{sdk.TargetMethod},
		Arguments: []sdk.Argument{
			{
				Name:          "isolation",
				Kinds:         []sdk.Kind{sdk.KindString},
				AllowedValues: []string{"", "default", "read-uncommitted", "read-committed", "write-committed", "repeatable-read", "snapshot", "serializable", "linearizable"},
				Description:   "database/sql isolation level name.",
			},
			{
				Name:        "readOnly",
				Kinds:       []sdk.Kind{sdk.KindBoolean},
				Description: "Whether the transaction is declared read-only.",
				Default:     "false",
			},
		},
		Examples: []sdk.Example{{
			Title: "Serializable transaction",
			Code:  "// @Transactional(isolation=\"serializable\")",
		}},
		Compatibility: sdk.Compatibility{
			Since:        "0.1.0",
			MinimumSpice: "0.1.0",
		},
		Implementation: sdk.Implementation{
			Tool:     "github.com/StevenBuglione/spice/cmd/spice-annotation-core",
			Handler:  "data/transactional",
			Protocol: sdk.ProtocolV1Alpha1,
			Source: sdk.Symbol{
				Package: "github.com/StevenBuglione/spice/internal/annotationcore",
				Name:    "TransactionalHandler",
			},
		},
	}
}
