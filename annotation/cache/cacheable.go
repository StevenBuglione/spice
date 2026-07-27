// Package cache defines the canonical descriptor for generated cache
// boundaries.
package cache

import "github.com/StevenBuglione/spice/annotation/sdk"

// Cacheable marks a GET controller method whose successful response may be
// cached under a stable cache name.
//
// Spice generates deterministic key material from the validated request and
// keeps the cache implementation explicit through dependency injection.
// Errors and unauthorized responses are never cached.
//
//	// @import { Cacheable } from "github.com/StevenBuglione/spice/annotation/cache"
//	// @Cacheable(name="orders.by-id")
func Cacheable() sdk.Definition {
	return sdk.Definition{
		Name:    "cache.Cacheable",
		Summary: "Declares a named generated HTTP cache boundary.",
		Targets: []sdk.Target{sdk.TargetMethod},
		Arguments: []sdk.Argument{{
			Name:        "name",
			Kinds:       []sdk.Kind{sdk.KindString},
			Description: "Stable cache identity used for configuration and observations.",
			Required:    true,
		}},
		Examples: []sdk.Example{{
			Title: "Cached route",
			Code:  "// @Cacheable(name=\"orders.by-id\")",
		}},
		Compatibility: sdk.Compatibility{
			Since:        "0.1.0",
			MinimumSpice: "0.1.0",
		},
		Implementation: sdk.Implementation{
			Tool:     "github.com/StevenBuglione/spice/cmd/spice-annotation-core",
			Handler:  "cache/cacheable",
			Protocol: sdk.ProtocolV1Alpha1,
			Source: sdk.Symbol{
				Package: "github.com/StevenBuglione/spice/internal/annotationcore",
				Name:    "CacheableHandler",
			},
		},
	}
}
