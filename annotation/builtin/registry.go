// Package builtin defines the annotation metadata shipped with Spice.
package builtin

import "github.com/StevenBuglione/spice/annotation"

// Registry returns a fresh immutable-by-construction registry of built-in
// annotations. Returning a value instead of exposing mutable global state keeps
// tests and compiler runs independent.
func Registry() annotation.Registry {
	return annotation.MustRegistry(
		annotation.Definition{
			Name:    "Application",
			Targets: annotation.Targets(annotation.TargetFunction),
		},
		annotation.Definition{
			Name:    "Configuration",
			Targets: annotation.Targets(annotation.TargetType),
		},
		annotation.Definition{
			Name:    "Controller",
			Targets: annotation.Targets(annotation.TargetType),
			Arguments: []annotation.ArgumentDefinition{
				{Name: "prefix", Kinds: []annotation.Kind{annotation.KindString}},
			},
		},
		annotation.Definition{
			Name:    "Get",
			Targets: annotation.Targets(annotation.TargetMethod),
			Arguments: []annotation.ArgumentDefinition{
				{Name: "path", Kinds: []annotation.Kind{annotation.KindString}, Required: true},
			},
		},
		annotation.Definition{
			Name:    "Post",
			Targets: annotation.Targets(annotation.TargetMethod),
			Arguments: []annotation.ArgumentDefinition{
				{Name: "path", Kinds: []annotation.Kind{annotation.KindString}, Required: true},
			},
		},
		annotation.Definition{
			Name:    "Service",
			Targets: annotation.Targets(annotation.TargetType),
		},
	)
}
