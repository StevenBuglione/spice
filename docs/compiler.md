# Spice compiler program model

## Type-aware loading boundary

`compiler/load` is the only package that directly depends on `golang.org/x/tools/go/packages`. One `load.Load` call performs one `packages.Load` operation and returns the package, syntax, type, symbol, and diagnostic records that later Spice compiler phases reuse.

The loader deliberately accepts standard Go package patterns such as `./...` rather than translating them into a filesystem walk. It also passes through caller-provided working directories, environments, build flags, overlays, and cancellation.

Package directories come from `go/packages.Package.Dir`, with selected source files as a deterministic fallback for drivers that omit it. Each package exposes a deterministic `Files` view that pairs a physical compiled-file path with its AST. The compatibility `CompiledGoFiles` and `Syntax` slices are derived from that same view and remain index-aligned; they are never sorted independently. Cgo-transformed build-cache inputs remain visible without redefining source package ownership.

Normal application compilation keeps `Tests` disabled. Requests with `Options.Tests` set to true fail immediately with a deterministic configuration diagnostic. Test-package and generated test-binary variants remain unsupported until Spice defines separate identities for production packages, in-package test variants, external test packages, and generated test binaries. This prevents duplicate stable package and symbol IDs from entering later compiler phases.

## Program lifetime

A `Program` owns one Go type universe. Package records retain live `go/types`, AST, token-file-set, and `go/packages` references for that load only. Objects or types from independent `Program` values must never be compared by pointer identity or mixed in a later compiler phase.

Ordered record methods return copies of the package, symbol, and diagnostic slices. The underlying Go semantic objects remain read-only by convention.

## Stable symbol IDs

Stable IDs describe logical source declarations and do not use absolute paths or `packages.Package.ID` values. Spice preserves identity as the structured tuple `(version, kind, package path, receiver origin, declaration name)` and serializes it canonically as:

```text
spice:symbol:v1|<kind>|<package-field>|<receiver-field>|<name-field>
field = <decimal-UTF-8-byte-length>:<exact-bytes>
```

All three fields are always present. Package symbols use empty receiver and name fields; non-method declarations use an empty receiver; methods use the normalized defining receiver origin. Examples:

```text
spice:symbol:v1|package|19:example.com/foo.bar|0:|0:
spice:symbol:v1|type|19:example.com/foo.bar|0:|1:T
spice:symbol:v1|method|19:example.com/foo.bar|1:T|1:M
```

Length-prefixing makes the encoder injective even when package paths contain dots, slashes, colons, pipes, digits, or text resembling another encoded field. Components are copied exactly: the encoder does not path-clean, case-fold, Unicode-normalize, percent-decode, or otherwise rewrite identity data. `Package.ID` and the matching package `Symbol.ID` use the same canonical package key, while `Package.Path` remains the ordinary Go import path.

`Symbol.DisplayLabel` is a concise dot-style human label such as `example.com/app.Service.Start`. It is useful in diagnostics and diagrams but is not a key and may legitimately be shared by a declaration and a package whose path contains the same suffix. Compiler logic should use the canonical ID or the structured `Kind`, `PackagePath`, `Receiver`, and `Name` fields.

Symbols are ordered by package path, the fixed kind rank `package`, `type`, `function`, `method`, `variable`, `constant`, then receiver origin, declaration name, and physical source position. Ordering never depends on the lexical details of the serialized ID grammar.

Pointer receivers normalize to the defining named type. Generic receiver declarations normalize through the `go/types.Named` origin, so receiver spelling and instantiation syntax do not change the method ID. Function and method signatures remain separate live type data because signatures can evolve independently of logical identity.

The catalog omits package-level `init` functions and every blank-identifier declaration (`_`), including types, package functions, methods, variables, and constants. Go permits multiple declarations with those names, and later Spice phases cannot address them by logical name. Excluding them preserves the one-to-one stable-ID contract without introducing filesystem- or source-order-based suffixes.

Every symbol retains two positions from the same token file set: `PhysicalPosition` uses the unadjusted loaded Go file, while `Position` is the developer-facing `//line`-adjusted location. Source ownership accepts a declaration when either its physical loaded file or its adjusted origin belongs to a selected `GoFiles` source. This preserves ordinary source-mapped generated Go and user declarations from an `import "C"` file while excluding cgo cache helpers whose physical and adjusted provenance are both generated. Adjusted paths are display metadata only and never filesystem authority.

## Diagnostics

The library does not print, exit, or mutate module files. Package-list, parse, and type errors are collected into deterministic diagnostics and returned through `LoadError`. Rendered Go positions remain available in `Diagnostic.Position`, while filename, line, and column are retained as structured fields so line 2 sorts before line 10. An ill-typed root package remains visible in the result for diagnostics, but it is marked unsafe for semantic generation.

CLI layers decide how diagnostics are rendered. The caller context is forwarded to `go/packages` and external `GOPACKAGESDRIVER` processes; cancelling an already-running load terminates the driver and returns `context.Canceled`.

## Dependency and offline policy

Spice requires Go 1.26.5 and pins `golang.org/x/tools v0.48.0`. The dependency is BSD-3-Clause licensed and remains isolated behind `compiler/load` so upgrades stay controlled.

The repository commits the standard output of `go mod vendor`. The local quality gate proves offline product execution with:

```text
make offline
```

Development tools are pinned separately in `tools/go.mod`; they do not enter the product module or runtime dependency graph. The loader never enables network access itself. The Go command continues to honor the caller's `GOPROXY`, `GOPRIVATE`, `GOSUMDB`, and `GOVCS` policies.

## Typed annotation resolution

`compiler/resolve` consumes one existing `load.Program`; it never walks the filesystem, reparses files, or creates another Go type universe. Only documentation comments on packages and declarations contribute annotations, and only files selected by the active Go build are examined.

Each occurrence carries its canonical symbol ID, package path, target, physical file/offset, and developer-facing `//line`-adjusted position. Physical identity controls deterministic ordering; adjusted paths are display metadata only.

Grouped declaration metadata fails closed when it could describe multiple specs or names, and blank identifiers cannot be annotation targets. Place metadata on one individual spec or split a multi-name declaration.

The `spice annotations` and `spice verify` commands accept ordinary Go package patterns, default to `.`, and perform one load-resolve-validate pipeline. `compiler/scan.Tree` remains for compatibility tests but is no longer the authoritative CLI source.

## Typed provider catalog

`compiler/provider` consumes the same `load.Program` and `resolve.Result` already produced for one CLI command. It never reloads packages, reparses files, walks function bodies, reflects on runtime values, or executes provider functions.

After ordinary annotation target and argument validation, each valid package-level `@Bean` function contributes one deterministic provider record. Validated `@Configuration` structs contribute explicit generated-binder provider records before graph construction. Accepted bean signatures are `func(dependencies...) T`, `func(dependencies...) (T, error)`, `func(dependencies...) (T, lifecycle.Cleanup)`, and `func(dependencies...) (T, lifecycle.Cleanup, error)`. `lifecycle.Cleanup` is the canonical named `func(context.Context) error` type. Recognition uses the result's live `go/types` named identity from the owning program: aliases to the canonical type are accepted, while unnamed or distinct defined callback types are rejected. No second package load or assignability-based callback inference occurs.

The first result remains the sole output. Provider records retain `ReturnsCleanup` and `ReturnsError` flags but no runtime callback value. Inputs preserve parameter order and positions, and cleanup metadata creates no dependency edge or injectable implicit value. Records retain live `go/types.Type` values only for the owning program, plus import-path-qualified stable type strings for diagnostics and later serialization.

Catalog output is sorted by stable provider symbol ID. Exact output conflicts use `types.Identical` and fail closed with one deterministic diagnostic naming every conflicting declaration. Distinct named types remain distinct even when their underlying representations match. The catalog does not perform assignability-based interface selection, provider or cleanup invocation, scopes, startup/shutdown orchestration, or code generation.

`spice verify` runs this catalog stage only after loading, typed annotation resolution, target validation, and argument validation have succeeded. Library code remains quiet; the CLI owns rendering and exit status.


## Deterministic provider dependency graph

`compiler/graph` consumes one already validated `provider.Catalog` from the owning typed compiler run. Every bootstrap provider is currently active. Each parameter resolves only to the one provider whose live output type is semantically identical under `go/types.Identical`; readable type IDs remain diagnostics and serialization data, not semantic lookup authority. Spice does not implicitly project concrete values to interfaces, equate distinct named types, convert pointers and values, or supply framework-specific defaults.

Graph construction returns stable provider nodes, parameter edges, and a dependency-first order with stable provider IDs breaking ties. Missing inputs accumulate as source-positioned diagnostics. Tarjan strongly connected component analysis reports every self-cycle and multi-provider cycle with a deterministic closed path. Any missing input or cycle suppresses construction order. The library is quiet and never executes provider bodies.

`spice verify` runs graph validation after provider-catalog validation. This stage validates bootstrap-wide singleton metadata only. Provider cleanup flags are preserved on graph nodes but do not change nodes, edges, missing-dependency analysis, cycles, or construction order. Reachable-provider pruning, generated constructor calls, cleanup invocation, lifecycle execution, scopes, conditions, interface bindings, qualifiers, overrides, and module rules remain explicit later phases.

## Typed lifecycle-hook catalog

`compiler/lifecycle` consumes the same `load.Program`, resolved annotations, and validated provider catalog. Argument-free `@OnStart` and `@OnStop` annotations must target ordinary non-generic, non-variadic methods with the exact signature `func(receiver)(context.Context) error`. Canonical `context.Context`, predeclared `error`, receiver types, and provider outputs are compared in the existing live `go/types` universe; aliases are accepted, while assignability, convertibility, structural equality, method-set promotion, and pointer/value call convenience are not ownership rules.

Each participating provider contributes one deterministic component with an optional start hook and optional stop hook. A stop hook requires a start hook, and duplicate roles fail with source-positioned diagnostics. Components are sorted by stable provider symbol ID, diagnostics by physical source identity, and accessors return defensive copies. Provider cleanup metadata remains separate, and hooks do not become providers, outputs, dependencies, graph nodes, or edges.

`spice verify` runs lifecycle validation after provider-graph validation. The compiler stage is quiet and never executes methods, providers, or cleanup callbacks. It records metadata only. The public `lifecycle.Coordinator` implements caller-context state transitions, dependency-order start, reverse successful-start stop, reverse construction cleanup, startup rollback, deterministic joined errors, idempotent stop, and run/wait/shutdown composition. Generated code supplies direct hook method values. Reusable application APIs retain caller-owned signals, shutdown contexts, logging, and command policy; only the explicitly invoked generated command helper applies Spice's process conventions.

## Immutable application model

`compiler/application` is the authoritative generation input assembled from the
same loaded program and resolved annotations. It runs the provider, graph, and
controller, and lifecycle stages once, retains dependency-first provider order
and cleanup flags, reorders lifecycle components by that construction order,
and validates typed `@Application` roots.

An `@Application` marker is an argument-free package-level function with no type
parameters, variadic parameter, or results. Its ordinary parameter types are
roots and must be exactly identical to one bean or generated configuration
provider output. The marker function is never invoked. Zero markers are valid
for library verification; multiple markers become stable application targets.

The model returns defensive metadata copies and stops at the first invalid
compiler stage. Generation must reject any model with diagnostics and must not
reload packages, rebuild the graph, or inspect declaration bodies.

Qualified bootstrap annotations are resolved by the same typed annotation
pipeline. `@management.Enable` accepts one required endpoint list and
`@observability.Logging` accepts no arguments. The bootstrap compiler validates
their exact application-marker target, duplicate/conflicting declarations,
known endpoints, and graph requirements, then stores normalized immutable
feature metadata on each application target. List order cannot affect the
model or generated bytes. The renderer consumes only this metadata; it never
re-reads raw comments.

The feature compiler is an explicit typed definition seam for qualified
annotations. The public `starter` manifest SDK now provides portable syntax
and application-feature specifications plus compatibility metadata.
`compiler/starter` validates an explicitly supplied manifest catalog against
the Spice API and Go version, composes its annotation registry, and maps
application features into `application.BuildWithOptions`. Source identity,
version, normalized options, requirements, and exported entrypoints survive in
immutable IR and the generation input hash. Definitions and manifests must be
deliberately supplied. The CLI strictly composes manifests embedded in the
application-owned `.spice/starters.json` selection for verification, module
checks, generation, and build.

`provider.BuildEntrypoints` validates an explicitly selected exported
package-level starter function from the same typed program used for application
analysis. It applies the ordinary provider signature contract without invoking
the function and retains the starter ID and version as provenance.
`application.BuildOptions.ProviderCatalogs` merges these validated nodes into
the exact-type graph. The renderer then emits the same direct dependency-first
call, immediate cleanup registration, rollback, and error handling used for an
`@Bean`; provenance participates in the ownership hash. The current public seam
is programmatic. Resolving the repository selection to loaded entrypoint
packages in the CLI and dependency alignment remain separate slices. Imports
or `go.mod` entries have no activation effect.

## Deterministic generation plan

`compiler/generate` consumes the loaded program and immutable application model
without another load or graph pass. `DefaultTarget` maps one application marker
to `internal/spicegen/<target>/zz_spice_gen.go` and
`.spice/<target>.manifest.json` inside its owning module. The module root remains
an in-memory execution detail and never appears in generated bytes.

The renderer emits the standard generated-code marker, sorted explicit import
aliases, generated configuration schema/binders, generated typed/raw
`net/http` adapters, direct exported provider calls in dependency-first order,
existing graph-edge arguments, immediate cleanup registration, wrapped stable
errors, and direct lifecycle method values.
Targets with controllers also emit
`internal/spicegen/<target>/openapi.json`; it is deterministic, manifest-owned,
and protected by the same safe apply/check/diff protocol as generated Go.
Configured targets add `ApplicationOptions`, `ConfigurationSchema`, and
`NewApplicationWithOptions`; sources and profiles remain caller-owned.
Generated `NewApplication`, `State`, `Start`, `Stop`, and `Run` methods delegate
only generic state and rollback mechanics to the small public lifecycle
coordinator. `Run` accepts the caller's run context and shutdown-context
factory, so the reusable application never registers signals or creates hidden
deadline/background contexts. Generated code imports no compiler package and
uses no reflection, package scan, service locator, or global registry.

Every application target also receives `Main(arguments) int` and the injectable
`RunCommand(CommandOptions) int` seam. The baseline command resolves the
generated schema from a conventional `SPICE_` OS-environment source, emits
structured construction/start/failure logs, supports `-check`, owns process
signals only in `Main`, and creates a fresh bounded shutdown context after
termination. `spice.shutdown-timeout` is typed configuration with a `10s`
default and `SPICE_SHUTDOWN_TIMEOUT` override. Exit codes are stable: zero for
success, one for construction/run/shutdown failure, and two for invalid usage.
Errors flow through safe configuration errors and never include raw values.

Resolved companions add direct generated composition. Logging installs ordered
`log/slog` lifecycle and HTTP observers. Management constructs lifecycle
checks, optional bounded route metrics, and exactly the normalized exposed
routes, then registers its handler directly on the generator-owned
`*http.ServeMux`. Construction failures pass through coordinator abort so
registered cleanup runs in reverse order.

Each plan includes canonical JSON ownership metadata with schema, target,
generator/formatter compatibility, a canonical model-input SHA-256, and exact
generated-file SHA-256 values. Repeated rendering is byte-identical and contains
no timestamps, absolute paths, raw environment, random values, or host data.

Target-scoped output cannot access unexported functions or import `package main`.
Those source shapes fail with actionable render diagnostics, as do an
application package importing its own generated output and a generated package
that would import itself. Guarded filesystem application, check/diff mode, safe
stale removal, and collision handling consume this plan in the next layer.

## Guarded filesystem application and commands

`internal/genfs` validates every plan through Go's rooted filesystem boundary.
It rejects traversal, portable Windows device names, case collisions, output
symlinks, foreign manifest targets/schemas, unowned path collisions, manual
edits, and unexpected unowned Spice generated markers before writing.

`spice generate` exclusively locks one target, rechecks ownership, writes
same-directory temporary files, syncs and parses generated Go, verifies exact
hashes, replaces recoverably, removes only unchanged manifest-owned stale files,
and replaces the manifest last. Byte-identical source and manifest files are
not rewritten, preserving mtimes and Go build cache inputs. This is a guarded
multi-file protocol, not a claim of global filesystem atomicity.

`spice generate --check` and `--diff` are read-only. Check mode reports every
deterministically sorted difference and returns nonzero; diff mode additionally
prints bounded unified-style expected/current content. `spice build` performs
the guarded generation operation and then runs `go build -trimpath ./...` in the
selected module.

Generated files include `//go:build !spice_generate`. Spice reserves and adds
the `spice_generate` tag only to generation analysis, merging existing explicit
and `GOFLAGS` tags. Verification and annotation listing load the ordinary
committed program so commands importing generated applications remain valid.
Targeted regeneration can therefore exclude stale output, while ordinary Go
commands omit the tag and compile committed output.

The committed commerce target is the executable reference for this contract.
`examples/commerce` declares four modules, typed configuration, generated
controllers, explicit providers, lifecycle hooks, and qualified management and
logging bootstrap annotations.
`internal/spicegen/commerce` contains the generated direct-call application and
OpenAPI document; its generated command owns conventional environment loading,
process signals, management composition, metrics, and the shutdown deadline.
The handwritten `main` is only the `os.Exit(Main(args))` process boundary.
Repository
verification runs generation freshness, generated construction, live typed
HTTP, failure mapping, metrics, and graceful-drain checks.

## Module discovery

`compiler/modulith` consumes the same typed program and resolved annotation
result as the application compiler. Package-documentation `@Module` markers
create full-import-path module identities. Each selected package is assigned to
the longest matching root in its Go module; nested roots therefore take
deterministic ownership, while packages outside every root are retained as
sorted unassigned metadata.

The root package is the default API. Repeatable package-level
`@NamedInterface` markers expose explicitly named descendant packages.
`allowedDependencies` entries identify an exact root API or
`module::interface`; discovery rejects malformed, duplicate, self, unknown
module, and unknown-interface references. Model accessors return defensive
copies.

Spice projects the selected program's real Go imports into distinct
cross-module package edges. Imports of a root API or named interface must match
an exact `allowedDependencies` entry. Any other descendant import is rejected
as internal at the import position. The projected module graph is decomposed
into strongly connected components; every multi-module component produces a
stable member set and representative closed cycle path. This metadata and its
diagnostics are part of the immutable application IR, so `spice verify`
enforces the same boundaries before generation.

`spice modules` loads `./...` by default and is read-only. `--format=json`
emits schema `spice.modules/v1` with a canvas for every module: owned packages,
the root default API, named interfaces, declared dependencies, observed
dependencies, exact package edges, cycle metadata, and unassigned packages.
`--format=mermaid` and `--format=plantuml` render the same sorted graph with
stable synthetic node IDs and aggregated API labels. Invalid module
architecture blocks output.

`--focus=<full-module-import-path>` produces a module test graph containing the
selected module plus only transitively observed dependencies. It excludes
dependents, unrelated modules, unassigned packages, and declared-but-unused
dependencies. JSON includes the focus identity and dependency-first composition
order; Mermaid and PlantUML highlight the selected module.

`spice test --module=<full-module-import-path>` validates the same model before
starting a subprocess. It passes the dependency-first owned-package list
directly to `go test -trimpath`, with optional race, count, run-expression, and
timeout controls. No unassigned, dependent, or unrelated package can enter the
test invocation, and the command performs no generation. Generated application
contexts and specialized web/data harnesses are separate future slices.

Generation maps each provider package back to its discovered owning module.
Generated cleanup registration and lifecycle hooks carry that full import-path
identity, and module ownership participates in the canonical manifest input
hash. `NewApplication` accepts optional lifecycle observers before provider
construction; generated applications also expose registration while still in
the constructed state. The coordinator emits synchronous begin/end events for
start, stop, and cleanup with stable component ID, module ID, operation, phase,
and callback outcome. Core selects no global observer, tracer, meter, or
exporter.
