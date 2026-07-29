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
Every entrypoint package must belong to the declared starter module; a manifest
cannot redirect construction to unrelated application or dependency code.

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

`compiler/starter.New` is the explicit compiler adapter. It accepts
application-selected manifests, verifies their exact Spice API and minimum Go contracts,
sorts them by import-path identity, and rejects duplicate manifest,
annotation, or capability identities. `Catalog.Registry` composes contributed
syntax with a caller-owned base registry; `Catalog.BootstrapDefinitions`
supplies immutable feature definitions to `application.BuildWithOptions`.
Compiled features retain manifest identity, version, normalized options,
runtime requirements, and exported entrypoints. Those inputs participate in
the generated ownership hash, so changing a selected starter invalidates
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

## Repository selection

An application opts into third-party compiler metadata by committing
`.spice/starters.json` at the CLI invocation root. The strict
`spice.starters/v1` document embeds one or more complete `spice.starter/v1`
manifests:

```json
{
  "schema": "spice.starters/v1",
  "manifests": [
    {
      "schema": "spice.starter/v1",
      "id": "example.com/acme/starter/search",
      "version": "1.2.0",
      "module": "example.com/acme",
      "spice_api": "v1alpha1",
      "minimum_go": "1.26.0",
      "license": "Apache-2.0",
      "review": "docs/dependency-review.md",
      "activation": {
        "mode": "explicit-annotation",
        "entry_points": [
          {
            "package": "example.com/acme/starter/search",
            "symbol": "New"
          }
        ]
      },
      "capabilities": ["search.client"],
      "annotations": [
        {
          "name": "search.Enable",
          "targets": ["function"],
          "arguments": [
            {
              "name": "indexes",
              "kinds": ["list"],
              "list_element_kinds": ["string"],
              "required": true
            }
          ]
        }
      ],
      "application_features": [
        {
          "annotation": "search.Enable",
          "capability": "search.client",
          "entry_points": [
            {
              "package": "example.com/acme/starter/search",
              "symbol": "New"
            }
          ],
          "options": [
            {
              "name": "indexes",
              "kind": "list",
              "list_item_kinds": ["string"],
              "required": true,
              "unique_items": true,
              "minimum_items": 1,
              "sort_items": true
            }
          ]
        }
      ]
    }
  ]
}
```

`spice verify`, `spice modules`, `spice test --module`, `spice generate`, and
`spice build` strictly parse the selection, compose its annotation registry,
and carry application features into generation freshness. The file is bounded
to 4 MiB, must be a regular file, rejects unknown fields and trailing values,
and fails before filesystem generation on any invalid or incompatible manifest.

Selection is explicit repository configuration, not dependency discovery.
Spice does not search module caches, call `Manifest()` functions, load plugins,
or infer activation from imports. Before its one typed package load, the CLI
adds every declared constructor package as an auxiliary root. Auxiliary symbols
remain available for exact provider analysis while their comments and package
structure cannot become application annotations or modules.

An `explicit-constructor` manifest activates every declared entrypoint when the
manifest is selected. An `explicit-annotation` manifest activates only the
entrypoint subset mapped to each qualified annotation actually present in
source on an `@Application` marker; absent or misplaced annotations emit no
constructor. Active functions must exist as exported package-level symbols and
satisfy the ordinary provider signature contract. The resulting exact-type
nodes participate in graph diagnostics, direct generated calls, immediate
cleanup registration, reverse rollback, executable builds, and
provenance-sensitive freshness checks.

Before provider analysis, the CLI checks dependencies declared by those active
starters against `go list -mod=readonly -m -json all`. This snapshot is
shell-free, bounded, and forced offline with `GOPROXY=off` and `GOSUMDB=off`.
Exact reviewed versions are required, and replacements fail closed unless both
their module identity and version match the review. An inactive
explicit-annotation starter imposes no dependency requirement. If the local Go
module cache cannot answer the read-only query, Spice reports the inspection
failure; users retain control over any separate dependency download.

## Shipped starter metadata

Every current integration exposes a package-level `Manifest()`:

| Package | Capabilities | Reviewed dependency |
|---|---|---|
| `starter/kafka` | `messaging.kafka.consumer-group`, `messaging.kafka.producer` | `github.com/twmb/franz-go` v1.21.0 |
| `starter/postgres` | `batch.postgresql`, `data.postgresql`, `data.sql`, `event.outbox.postgresql`, `migration.postgresql` | `github.com/jackc/pgx/v5` v5.10.0 |
| `starter/redis` | `cache.redis`, `data.redis` | `github.com/redis/go-redis/v9` v9.21.0 |
| `starter/smtp` | `mail.smtp` | Go standard library |
| `starter/oidc` | `security.oidc-resource-server` | `github.com/coreos/go-oidc/v3` v3.20.0 |
| `starter/oauth2client` | `security.oauth2-client-credentials` | `golang.org/x/oauth2` v0.36.0 |
| `starter/otel` | `observability.http-server`, `observability.metrics`, `observability.tracing` | OpenTelemetry API modules v1.43.0 |

Kafka, PostgreSQL, Redis, SMTP, OIDC, and OAuth2-client use
`explicit-constructor` activation.
OpenTelemetry contributes `@otel.Enable` through `explicit-annotation`
activation and maps it to the reviewed `NewHTTPObserver` entrypoint plus the
reserved `observability.http-server` generator role. Tests compile those
symbols, parse every canonical manifest, verify the Go 1.26.5 compatibility
decision, and require each review document to exist.

An HTTP-observation feature is not an unchecked interface plug-in. Its selected
entrypoint must produce the exact structural `web.HTTPObserver` contract and
the selected application graph must provide the declared `http.serve-mux`
capability. The compiler reports either defect at the application annotation
before rendering. Generated composition uses the provider already constructed
from the explicit manifest; it performs no runtime lookup or registration.

## Review policy

A manifest is not a substitute for its review document. Adoption still
requires maintenance, license, security, cancellation, observability,
configuration, and network-behavior analysis plus executable integration
tests. Manifests contain identities and decisions only—never credentials,
tokens, connection strings, or environment-specific configuration.
