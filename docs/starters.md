# Starter manifests and annotation SDK

Spice starters are opt-in Go integrations. Importing a package or adding a
module to `go.mod` never enables one. The public `annotation/sdk/starter`
package defines the portable `spice.starter/v1` manifest used to review and
compose them without `init` hooks, global registries, reflection, or runtime
package scanning. The root `starter` package temporarily re-exports compatible
aliases so existing integrations remain source compatible during repository
extraction; new integrations should import the SDK package directly.

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
Every entrypoint package must belong to the declared starter module; a manifest
cannot redirect construction to unrelated application or dependency code.

```go
import starter "github.com/spice-framework/spice/annotation/sdk/starter"

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
        EntryPoints: []starter.EntryPoint{
            {
                Package: "example.com/acme/starter/search",
                Symbol:  "New",
            },
        },
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
requirement identities before those definitions can reach compilation. Every
explicit-annotation feature names the exact subset of activation entrypoints it
selects; missing, duplicated, undeclared, and never-selected entrypoints fail
manifest validation.

`compiler/starter.New` is the explicit compiler adapter. It accepts SDK
manifests selected by the application, verifies their exact Spice API and
minimum Go contracts, sorts them by import-path identity, and rejects duplicate
manifest, annotation, or capability identities. The compiler does not import
the runtime `starter` package or a compiled integration registry.
`Catalog.Registry` composes contributed syntax with a caller-owned base
registry; `Catalog.BootstrapDefinitions` supplies immutable feature definitions
to `application.BuildWithOptions`.
Compiled features retain manifest identity, version, normalized options,
runtime requirements, and exported entrypoints. Those inputs participate in
the generated ownership hash, so changing selected integration metadata invalidates
`spice generate --check`.

`provider.BuildEntrypoints` is the explicit construction adapter. Given
entrypoints selected from a validated manifest, it resolves only exported
package-level functions already present in the application compiler's typed
program, applies the ordinary exact provider signature contract, and records
the starter ID and version without executing function bodies.
`application.BuildOptions.ProviderCatalogs` composes that catalog into the
application graph. Generation emits ordinary direct Go calls with
dependency-first ordering, immediate cleanup ownership, construction rollback,
and wrapped deterministic errors. Starter provenance participates in the
generated ownership hash even though it adds no runtime registry or reflection.

These adapters perform no repository scan and never treat an imported or
downloaded module as active.

`Catalog.Dependencies` exposes the exact reviewed module version and SPDX
license contracts retained from every selected manifest.
`Catalog.ActiveDependencies` narrows that set to explicit-constructor
manifests and the explicit-annotation features present on an `@Application`
marker. The matching validation APIs compare those contracts with a supplied
Go module graph. Missing modules, MVS upgrades or downgrades, ambiguous graph
records, and replacements that do not resolve to the same reviewed module
identity and version fail deterministically. Selected manifests cannot publish
conflicting version or license reviews for the same dependency.

## Go-native auto-configuration

A library may publish a dedicated package whose final import-path element is
`autoconfigure`. Applications select its defaults with an ordinary explicit Go
blank import:

```go
import _ "example.com/acme/search/autoconfigure"
```

The package exposes one canonical, navigable descriptor:

```go
package autoconfigure

import (
    "example.com/acme/search"
    "github.com/spice-framework/spice/starter"
)

func DefaultClient(options search.Options) (*search.Client, error) {
    return search.New(options)
}

func SpiceAutoConfiguration() starter.AutoConfiguration {
    return starter.AutoConfiguration{
        Review: "docs/dependency-review.md",
        Beans: []starter.AutoBean{{
            Factory:  DefaultClient,
            Fallback: true,
        }},
    }
}
```

The compiler statically decodes the returned composite literal from its one
typed program. It never calls the descriptor or its factories during analysis,
never executes `init`, and never scans packages at runtime. Factory references
must be exported package-level Go functions in the descriptor package and
must satisfy the same typed provider signature contract as `@Bean`.

Auto-configuration is conditional before construction:

- a sole library default backs off when an application bean has its exact
  output type;
- when multiple defaults intentionally provide one collection element type, a
  matching application bean name or alias replaces only that default, while a
  distinct application bean extends the collection;
- defaults whose required inputs are unavailable back off;
- optional and collection inputs do not force activation;
- selected defaults enter the ordinary exact-type graph and direct generated
  Go with normal cleanup, rollback, scopes, and deterministic errors.

`spice beans --explain` reports every selected, replaced, or inactive default,
its reason, output type, resolved module/version or replacement, and dependency
review reference. This is a Go-native condition-evaluation report without
classpath scanning.

The retired `.spice/starters.json` file is rejected with a migration diagnostic.
`go.mod`, `go.sum`, `vendor`, and normal Go imports are the only dependency and
activation mechanism. Merely requiring a module is not activation; the
dedicated blank import is.

Selected factory source identity and module provenance participate in generated
ownership and freshness. Package loading remains read-only and offline; if the
normal Go module cache or vendor tree cannot satisfy the explicit import, Spice
reports the missing Go dependency and never downloads it.

## Shipped starter metadata

Every current integration retains a package-level `Manifest()` compatibility
and dependency-review record. These records do not activate behavior:

| Package | Capabilities | Reviewed dependency |
|---|---|---|
| `starter/grpc` | `rpc.grpc.client`, `rpc.grpc.server` | `google.golang.org/grpc` v1.82.1 |
| `starter/kafka` | `messaging.kafka.consumer-group`, `messaging.kafka.producer` | `github.com/twmb/franz-go` v1.21.0 |
| `starter/websocket` | `web.websocket.client`, `web.websocket.server` | `github.com/coder/websocket` v1.8.15 |
| `starter/postgres` | `batch.postgresql`, `data.postgresql`, `data.sql`, `event.outbox.postgresql`, `migration.postgresql` | `github.com/jackc/pgx/v5` v5.10.0 |
| `starter/redis` | `cache.redis`, `data.redis` | `github.com/redis/go-redis/v9` v9.21.0 |
| `starter/smtp` | `mail.smtp` | Go standard library |
| `starter/oidc` | `security.oidc-resource-server` | `github.com/coreos/go-oidc/v3` v3.20.0 |
| `starter/oauth2client` | `security.oauth2-client-credentials` | `golang.org/x/oauth2` v0.36.0 |
| `starter/otel` | `observability.http-server`, `observability.metrics`, `observability.module-events`, `observability.tracing` | OpenTelemetry API modules v1.44.0 |

Applications call explicit constructors directly or select separately published
auto-configuration packages. OpenTelemetry contributes `@otel.Enable` through
the annotation SDK and maps it to the reviewed `NewHTTPObserver` entrypoint plus
the reserved `observability.http-server` generator role. Tests compile those
symbols, validate every canonical compatibility record, and require each review
document to exist.

An HTTP-observation feature is not an unchecked interface plug-in. Its selected
entrypoint must produce the exact structural `web.HTTPObserver` contract and
the selected application graph must provide the declared `http.serve-mux`
capability. The compiler reports either defect at the application annotation
before rendering. Generated composition uses the provider already constructed
from the typed application model; it performs no runtime lookup or registration.

## Review policy

A manifest is not a substitute for its review document. Adoption still
requires maintenance, license, security, cancellation, observability,
configuration, and network-behavior analysis plus executable integration
tests. Manifests contain identities and decisions only—never credentials,
tokens, connection strings, or environment-specific configuration.
