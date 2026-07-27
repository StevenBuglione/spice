package web

import "github.com/StevenBuglione/spice/annotation/sdk"

// Post maps a controller method to an HTTP POST route.
//
// Path is required, may be positional, and must be absolute. Request bodies
// use explicit DTO tags and bounded decoding; generated adapters return
// RFC 9457 problem responses for validation and application failures.
//
//	// @import { Post } from "github.com/StevenBuglione/spice/annotation/web"
//	// @Post("/orders")
//	func (*Controller) Create(context.Context, CreateRequest) (Order, error)
func Post() sdk.Definition {
	return sdk.Definition{
		Name:    "web.Post",
		Summary: "Maps a controller method to an HTTP POST route.",
		Targets: []sdk.Target{sdk.TargetMethod},
		Arguments: []sdk.Argument{{
			Name:        "path",
			Kinds:       []sdk.Kind{sdk.KindString},
			Description: "Absolute route path, including optional path variables.",
			Required:    true,
			Positional:  true,
		}},
		Examples: []sdk.Example{{
			Title: "POST route",
			Code:  "// @Post(\"/orders\")",
		}},
		Compatibility: sdk.Compatibility{
			Since:        "0.1.0",
			MinimumSpice: "0.1.0",
		},
		Implementation: sdk.Implementation{
			Tool:     "github.com/StevenBuglione/spice/cmd/spice-annotation-core",
			Handler:  "web/post",
			Protocol: sdk.ProtocolV1Alpha1,
			Source: sdk.Symbol{
				Package: "github.com/StevenBuglione/spice/internal/annotationcore",
				Name:    "PostHandler",
			},
		},
	}
}
