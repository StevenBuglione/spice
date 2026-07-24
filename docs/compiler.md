# Spice compiler program model

## Type-aware loading boundary

`compiler/load` is the only package that directly depends on `golang.org/x/tools/go/packages`. One `load.Load` call performs one `packages.Load` operation and returns the package, syntax, type, symbol, and diagnostic records that later Spice compiler phases reuse.

The loader deliberately accepts standard Go package patterns such as `./...` rather than translating them into a filesystem walk. It also passes through caller-provided working directories, environments, build flags, overlays, and cancellation.

Package directories come from the selected source files reported by `go/packages.Package.GoFiles`. The deterministically sorted `CompiledGoFiles` list remains available separately because cgo can replace a source file with generated build-cache inputs during type checking. This keeps module ownership and future architecture checks anchored to the developer's source tree without hiding the actual compiled-file set.

Normal application compilation keeps `Tests` disabled. Requests with `Options.Tests` set to true fail immediately with a deterministic configuration diagnostic. Test-package and generated test-binary variants remain unsupported until Spice defines separate identities for production packages, in-package test variants, external test packages, and generated test binaries. This prevents duplicate stable package and symbol IDs from entering later compiler phases.

## Program lifetime

A `Program` owns one Go type universe. Package records retain live `go/types`, AST, token-file-set, and `go/packages` references for that load only. Objects or types from independent `Program` values must never be compared by pointer identity or mixed in a later compiler phase.

Ordered record methods return copies of the package, symbol, and diagnostic slices. The underlying Go semantic objects remain read-only by convention.

## Stable symbol IDs

Stable IDs describe logical source declarations and do not use absolute paths or `packages.Package.ID` values:

```text
package                         <package-path>
type/function/variable/constant <package-path>.<name>
method                          <package-path>.<receiver-origin>.<name>
```

Pointer receivers normalize to the defining named type. Generic receiver declarations normalize through the `go/types.Named` origin, so receiver spelling and instantiation syntax do not change the method ID. Function and method signatures remain separate live type data because signatures can evolve independently of logical identity.

The catalog omits package-level `init` functions and every blank-identifier declaration (`_`), including types, package functions, methods, variables, and constants. Go permits multiple declarations with those names, and later Spice phases cannot address them by logical name. Excluding them preserves the one-to-one stable-ID contract without introducing filesystem- or source-order-based suffixes.

Logical symbols and the package symbol must resolve to source files selected in `GoFiles`, including caller overlays and ordinary committed generated `.go` files. cgo's cache-backed helper files remain part of the live type universe and compiled-file metadata, but their `_C*`, `_cgo*`, and hash-bearing declarations are not source-addressable Spice symbols. User declarations from an `import "C"` file retain their original source positions through cgo line directives.

## Diagnostics

The library does not print, exit, or mutate module files. Package-list, parse, and type errors are collected into deterministic diagnostics and returned through `LoadError`. An ill-typed root package remains visible in the result for diagnostics, but it is marked unsafe for semantic generation.

CLI layers decide how diagnostics are rendered.

## Dependency and offline policy

Spice pins `golang.org/x/tools v0.36.0`, the adjacent release whose module still declares Go 1.23. The dependency is BSD-3-Clause licensed and is isolated behind `compiler/load` so upgrades remain controlled.

The repository commits the standard output of `go mod vendor`. CI and scheduled sandboxes must be able to run:

```text
GOPROXY=off go test -mod=vendor ./...
```

The loader never enables network access itself. The Go command continues to honor the caller's `GOPROXY`, `GOPRIVATE`, `GOSUMDB`, and `GOVCS` policies.
