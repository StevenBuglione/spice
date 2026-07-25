// Package builtin defines the annotation metadata shipped with Spice.
package builtin

import "github.com/StevenBuglione/spice/annotation"

// Registry returns a fresh immutable-by-construction registry of built-in annotations.
func Registry() annotation.Registry {
	return annotation.MustRegistry(
		annotation.Definition{Name: "Application", Targets: annotation.Targets(annotation.TargetFunction)},
		annotation.Definition{Name: "Bean", Targets: annotation.Targets(annotation.TargetFunction)},
		annotation.Definition{Name: "Configuration", Targets: annotation.Targets(annotation.TargetType)},
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
				{Name: "path", Kinds: []annotation.Kind{annotation.KindString}, Required: true, Positional: true},
			},
		},
		annotation.Definition{
			Name:    "Module",
			Targets: annotation.Targets(annotation.TargetPackage),
			Arguments: []annotation.ArgumentDefinition{
				{Name: "allowedDependencies", Kinds: []annotation.Kind{annotation.KindList}},
			},
		},
		annotation.Definition{
			Name:       "NamedInterface",
			Targets:    annotation.Targets(annotation.TargetPackage),
			Repeatable: true,
			Arguments: []annotation.ArgumentDefinition{
				{Name: "name", Kinds: []annotation.Kind{annotation.KindString}, Required: true, Positional: true},
			},
		},
		annotation.Definition{Name: "OnStart", Targets: annotation.Targets(annotation.TargetMethod)},
		annotation.Definition{Name: "OnStop", Targets: annotation.Targets(annotation.TargetMethod)},
		annotation.Definition{
			Name:    "Post",
			Targets: annotation.Targets(annotation.TargetMethod),
			Arguments: []annotation.ArgumentDefinition{
				{Name: "path", Kinds: []annotation.Kind{annotation.KindString}, Required: true, Positional: true},
			},
		},
		annotation.Definition{Name: "Service", Targets: annotation.Targets(annotation.TargetType)},
	)
}
