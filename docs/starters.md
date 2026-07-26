# Starter manifests and annotation SDK

Spice starters are opt-in Go integrations. Importing a package or adding a
module to `go.mod` never enables one. The public `starter` package defines the
portable `spice.starter/v1` manifest used to review and compose them without
`init` hooks, global registries, reflection, or runtime package scanning.

## Compatibility record

Every manifest declares:

- a full import-path identity and owning Go module;
- the starter version, Spice API line, and minimum Go version;
- the starter and dependency SPDX license identities;
- a dependency-review reference;
- stable capability identities;
- reviewed direct third-party dependencies and exact versions;
- explicit exported Go entrypoints;
- either explicit-constructor or explicit-annotation activation.

`starter.New` validates and normalizes package-owned specifications.
`starter.Parse` strictly decodes portable JSON, rejects unknown fields and
trailing values, and applies the same validation. Accessors return defensive
copies; `Manifest.JSON` sorts unordered metadata and emits stable bytes without
timestamps, absolute paths, environment data, or host information.

```go
manifest, err := starter.New(starter.Spec{
    Schema:    starter.Schema,
    ID:        "example.com/acme/starter/search",
    Version:   "1.2.0",
    Module:    "example.com/acme",
    SpiceAPI:  starter.APIVersion,
    MinimumGo: "1.26",
    License:   "Apache-2.0",
    Review:    "docs/dependency-review.md",
    Activation: starter.Activation{
        Mode: starter.ActivationExplicitConstructor,
        EntryPoints: []starter.EntryPoint{
            {
                Package: "example.com/acme/starter/search",
                Symbol:  "New",
            },
        },
    },
    Capabilities: []string{"search.client"},
})
```

`Manifest.Compatible` fails closed when the requested Spice API differs or the
current Go version is older than the declared minimum. Spice API matching is
exact while this pre-1.0 compiler contract evolves.

## Qualified annotation definitions

An explicit-annotation manifest carries portable `AnnotationSpec` and
`FeatureSpec` values. Annotation targets and argument kinds use the public
`annotation` model. Feature metadata adds the capability, deterministic option
rules, and runtime requirements:

```go
Annotations: []starter.AnnotationSpec{
    {
        Name:    "search.Enable",
        Targets: []annotation.Target{annotation.TargetFunction},
        Arguments: []starter.ArgumentSpec{
            {
                Name:     "indexes",
                Kinds:    []annotation.Kind{annotation.KindList},
                Required: true,
            },
        },
    },
},
ApplicationFeatures: []starter.FeatureSpec{
    {
        Annotation: "search.Enable",
        Capability: "search.client",
        Options: []starter.OptionSpec{
            {
                Name:         "indexes",
                Kind:         annotation.KindList,
                UniqueItems:  true,
                MinimumItems: 1,
                SortItems:    true,
            },
        },
    },
},
```

`Manifest.Definitions` returns fresh `annotation.Definition` values for an
explicitly composed compiler registry. The SDK validates qualified names,
targets, arguments, option relationships, unique capabilities, and runtime
requirement identities before those definitions can reach compilation.

The manifest SDK is the public metadata boundary. Repository discovery,
compiler composition, generated entrypoint selection, and starter dependency
alignment are subsequent slices. Until those adapters are present, annotation
manifests do not change generated behavior and must not be represented as
active merely because their module is installed.

## Review policy

A manifest is not a substitute for its review document. Adoption still
requires maintenance, license, security, cancellation, observability,
configuration, and network-behavior analysis plus executable integration
tests. Manifests contain identities and decisions only—never credentials,
tokens, connection strings, or environment-specific configuration.
