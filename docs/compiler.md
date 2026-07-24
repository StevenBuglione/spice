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

Spice pins `golang.org/x/tools v0.36.0`, the adjacent release whose module still declares Go 1.23. The dependency is BSD-3-Clause licensed and is isolated behind `compiler/load` so upgrades remain controlled.

The repository commits the standard output of `go mod vendor`. CI and scheduled sandboxes must be able to run:

```text
GOPROXY=off go test -mod=vendor ./...
```

The loader never enables network access itself. The Go command continues to honor the caller's `GOPROXY`, `GOPRIVATE`, `GOSUMDB`, and `GOVCS` policies.

## Typed annotation resolution

`compiler/resolve` consumes one existing `load.Program`; it never walks the filesystem, reparses files, or creates another Go type universe. Only documentation comments on packages and declarations contribute annotations, and only files selected by the active Go build are examined.

Each occurrence carries its canonical symbol ID, package path, target, physical file/offset, and developer-facing `//line`-adjusted position. Physical identity controls deterministic ordering; adjusted paths are display metadata only.

Grouped declaration metadata fails closed when it could describe multiple specs or names, and blank identifiers cannot be annotation targets. Place metadata on one individual spec or split a multi-name declaration.

The `spice annotations` and `spice verify` commands accept ordinary Go package patterns, default to `.`, and perform one load-resolve-validate pipeline. `compiler/scan.Tree` remains for compatibility tests but is no longer the authoritative CLI source.

## Typed provider catalog

`compiler/provider` consumes the same `load.Program` and `resolve.Result` already produced for one CLI command. It never reloads packages, reparses files, walks function bodies, reflects on runtime values, or executes provider functions.

After ordinary annotation target and argument validation, each valid package-level `@Bean` function contributes one deterministic provider record. Accepted signatures are `func(dependencies...) T` and `func(dependencies...) (T, error)`. Records retain live `go/types.Type` values only for the owning program, plus import-path-qualified stable type strings for diagnostics and later serialization. Inputs preserve parameter order and positions.

Catalog output is sorted by stable provider symbol ID. Exact output conflicts use `types.Identical` and fail closed with one deterministic diagnostic naming every conflicting declaration. Distinct named types remain distinct even when their underlying representations match. The catalog does not perform assignability-based interface selection, dependency graph construction, provider invocation, scopes, lifecycle, or code generation.

`spice verify` runs this catalog stage only after loading, typed annotation resolution, target validation, and argument validation have succeeded. Library code remains quiet; the CLI owns rendering and exit status.
