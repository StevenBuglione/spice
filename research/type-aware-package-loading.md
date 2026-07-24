# Type-Aware Go Package Loading and Symbol Resolution

Date: 2026-07-24

## Question

What is the smallest package-loading and symbol-resolution foundation Spice should build after annotation target and argument validation so later dependency injection, controller binding, module verification, configuration, events, scheduling, and generated code operate on the same Go program the standard toolchain sees?

## Current Spice limitation

`compiler/scan.Tree` currently walks directories with `filepath.WalkDir` and parses every `.go` file it encounters. That is sufficient for the lexical bootstrap, but it is not a compiler-equivalent program model:

- it does not apply the active build tags, `GOOS`, `GOARCH`, cgo selection, or workspace rules;
- it does not understand module import paths, `go.work`, `replace`, vendoring, or package patterns;
- it cannot distinguish a declaration name from the `go/types.Object` it denotes;
- it cannot inspect function and method signatures, receiver types, interfaces, generic type parameters, or imported types;
- it cannot derive package dependency edges reliably;
- it scans files that the Go compiler may ignore and may miss package variants selected by the build system.

Those gaps block the compile-time equivalents of Spring constructor injection, Spring MVC handler signature processing, conditional starter activation, and Spring Modulith dependency analysis.

## Primary-source findings

### 1. Use `golang.org/x/tools/go/packages` as the package-loading boundary

The official `go/packages` package is designed to load Go packages for inspection and analysis. It delegates package patterns to the active build tool, normally the `go` command, and can return source syntax, module metadata, imports, `go/types.Package`, and complete `go/types.Info` for matched packages.

Source:

- https://pkg.go.dev/golang.org/x/tools/go/packages

The package documentation recommends passing command-line package patterns through to `packages.Load` rather than translating them into filesystem roots. This means `spice verify ./...`, import paths, absolute/relative package directories, and future editor `file=` queries can use the same pattern semantics as the Go toolchain.

The Go command's package-pattern rules explicitly account for module/workspace context, vendor behavior, ignored directories, and `...` expansion. Spice should not recreate those rules.

Source:

- https://pkg.go.dev/cmd/go#hdr-Package_lists_and_patterns
- https://go.dev/ref/mod

### 2. Load typed syntax once per compiler command

`packages.LoadSyntax` includes package names, files, imports, exported types, syntax trees, `TypesInfo`, and effective type sizes for the matched packages. Spice also needs `packages.NeedModule` so package records can retain module ownership.

Recommended bootstrap mode:

```go
packages.LoadSyntax | packages.NeedModule
```

Do not request `NeedDeps` initially. It would cause the same syntax-heavy mode to apply to dependencies and can substantially increase memory and latency. Root package type checking already resolves imported symbols from export data. A later module/import-graph issue can add a narrower dependency traversal based on demonstrated needs.

One `packages.Load` call must feed all compiler phases in a command. The official documentation warns that type objects from separate load calls must not be mixed because each call creates a distinct type universe/importer. Spice must therefore create one immutable per-run `Program` and pass it through annotation resolution, typed IR construction, validation, and generation.

Source:

- https://pkg.go.dev/golang.org/x/tools/go/packages#LoadMode
- https://pkg.go.dev/golang.org/x/tools/go/packages#Package

### 3. Resolve declarations through `go/types.Info`, not names alone

The standard `go/types` package performs name resolution and type checking. `Info.Defs`, `Info.Uses`, `Info.Selections`, and `Info.ObjectOf` connect syntax identifiers to `types.Object` values. Function signatures are available through `types.Signature`; named and generic receiver types can be normalized through `types.Named` and its defining object.

Source:

- https://pkg.go.dev/go/types

Each scanned annotation occurrence should ultimately reference a resolved declaration record containing:

- source position;
- package import path;
- declaration kind;
- declaration name;
- stable Spice symbol ID;
- the live `types.Object` for the current compiler run;
- function/method signature when applicable;
- normalized receiver identity for methods;
- originating syntax node.

The live `types.Object` must never be serialized or reused across `packages.Load` calls. The stable Spice symbol ID is the durable identity.

### 4. Define stable symbol IDs independently of `packages.Package.ID`

`packages.Package.ID` is unique within a load result, but test variants can produce IDs such as `pkg [pkg.test]`. It is not an appropriate public or serialized Spice identity.

Recommended IDs:

```text
package:   <package-path>
type:      <package-path>.<type-name>
function:  <package-path>.<function-name>
method:    <package-path>.<receiver-origin-name>.<method-name>
variable:  <package-path>.<variable-name>
constant:  <package-path>.<constant-name>
```

For method IDs, erase pointer indirection and normalize an instantiated generic receiver to the origin named receiver type. Go does not overload package functions or methods, so this identity is sufficient for declaration targets in the bootstrap compiler.

Keep the signature separate from the stable ID. A signature change should be detectable without changing the declaration's logical identity.

### 5. Production analysis should exclude test package variants by default

`packages.Config.Tests=true` returns normal packages, in-package test variants, external test packages, and generated test binaries. That is useful for a future Spice test compiler, but it introduces duplicate declarations and package identities into normal application verification.

The default application load should use:

```go
Tests: false
```

A later test-support API can explicitly request test variants and model them separately.

Source:

- https://pkg.go.dev/golang.org/x/tools/go/packages#Config

### 6. Fail closed on package, syntax, and type errors

A typed compiler model is unsafe when the loaded package is ill-typed. `go/packages` reports driver, list, parse, and type errors with source positions and marks packages as `IllTyped`.

For command-line `spice verify` and generation:

1. collect package-loading diagnostics;
2. normalize paths relative to the requested working directory where possible;
3. sort by file, line, column, error kind, and message;
4. return all deterministic diagnostics;
5. do not run semantic validation or generation for an ill-typed root package.

A future language-server path may provide partial analysis with overlays, but it must be a separate mode with explicit incomplete-state semantics.

### 7. Preserve Go build-system semantics and editor compatibility

`packages.Config` supports:

- `Dir` for the workspace/query directory;
- `Env` and `BuildFlags` for active build settings;
- `Overlay` for unsaved editor buffers;
- external package drivers through `GOPACKAGESDRIVER`.

Spice's loader should expose these capabilities internally without inventing a separate build-tag or workspace configuration language. This preserves a path toward Bazel drivers and a future `spice-ls` without coupling the initial issue to either.

Source:

- https://pkg.go.dev/golang.org/x/tools/go/packages#Config

### 8. Pin the latest `x/tools` release compatible with Spice's current Go floor

Spice currently declares `go 1.23.0` and verifies Go 1.23.x.

Official module files show:

- `golang.org/x/tools v0.36.0` declares `go 1.23.0`;
- `v0.37.0` raises the requirement to `go 1.24.0`;
- the current `v0.48.0` release declares `go 1.25.0`.

Sources:

- https://github.com/golang/tools/blob/v0.36.0/go.mod
- https://github.com/golang/tools/blob/v0.37.0/go.mod
- https://github.com/golang/tools/blob/v0.48.0/go.mod

Recommendation: pin `golang.org/x/tools v0.36.0` while Go 1.23 remains a supported Spice build target. Record the dependency rationale in the implementation PR. Revisit the pin when Spice deliberately raises its minimum Go version.

`x/tools` uses the BSD-3-Clause license.

Source:

- https://pkg.go.dev/golang.org/x/tools@v0.36.0

### 9. Offline execution needs an explicit dependency policy

The current scheduled sandbox can execute Go 1.23 code but may not resolve public module hosts. Adding `x/tools` without a cached module can make `make verify` fail even though GitHub Actions has network access.

The implementation issue should not silently weaken local verification. It must choose and document one reproducible approach:

1. commit a `vendor/` tree and run with normal Go vendor semantics; or
2. prove the execution environment can obtain the pinned module before claiming completion.

For Spice's GitHub-first autonomous workflow, vendoring is the safer default once the dependency is introduced. It keeps scheduled implementation and verification runs independent of public proxy availability. The vendor tree must be generated by the standard `go mod vendor` command and verified in CI; agents must not hand-copy dependency files.

The Go module reference documents that `-mod=vendor` avoids network and module-cache access, and modules with a sufficiently new `go` directive automatically prefer a consistent vendor directory when present.

Source:

- https://go.dev/ref/mod#build-commands

### 10. Type resolution directly enables the next coherent vertical slices

Spring Framework constructor injection derives dependencies from constructor/factory method arguments. Spice needs resolved function signatures before it can build a constructor provider graph.

Source:

- https://docs.spring.io/spring-framework/reference/core/beans/dependencies/factory-collaborators.html

Spring MVC selects behavior from controller method parameter and return types. Spice needs resolved method signatures before it can generate safe `net/http` adapters and binding diagnostics.

Source:

- https://docs.spring.io/spring-framework/reference/web/webmvc/mvc-controller/ann-methods/arguments.html

Spring Modulith derives module dependencies from actual references between application modules and verifies API/internal boundaries and allowed dependencies. Spice needs resolved package imports and type references before it can implement equivalent Go-native rules.

Sources:

- https://docs.spring.io/spring-modulith/reference/fundamentals.html
- https://docs.spring.io/spring-modulith/reference/verification.html

## Proposed bootstrap API direction

The exact exported names may be refined during implementation, but the compiler needs a boundary comparable to:

```go
package load

type Options struct {
    Dir        string
    Patterns   []string
    Env        []string
    BuildFlags []string
    Overlay    map[string][]byte
    Tests      bool
}

type Program struct {
    Packages   []Package
    Diagnostics []Diagnostic
}

type Package struct {
    Path       string
    Name       string
    Dir        string
    ModulePath string
    Files      []string
    Symbols    []Symbol
}

type Symbol struct {
    ID          ID
    Kind        Kind
    PackagePath string
    Name        string
    Receiver    string
    Position    token.Position
    Object      types.Object
    Node        ast.Node
}
```

Important constraints:

- returned package and symbol slices are deterministically sorted;
- the loader does not expose mutable process-global state;
- callers provide a `context.Context`;
- package patterns pass through unchanged;
- the same program object is reused by all compiler phases;
- `types.Object` remains process-local while `ID` is stable and serializable;
- package load diagnostics are represented in Spice's deterministic diagnostic model rather than printed inside the loader.

## Minimal implementation sequence

The next issue should be intentionally smaller than complete annotation resolution:

1. add `compiler/load` using `go/packages`;
2. load root packages with typed syntax and module metadata;
3. expose deterministic package records and package-level diagnostics;
4. resolve top-level types, functions, methods, variables, and constants to stable symbol records;
5. test package patterns, build constraints, imported types, pointer/generic receivers, deterministic ordering, and type failures;
6. do not yet replace `compiler/scan.Tree` or integrate with annotation validation.

A following issue can join scanner occurrences to resolved symbols by source node/position and migrate `spice verify` to the typed program model. Separating these steps keeps the first implementation bounded and makes loader behavior independently testable.

## Deferred decisions

- test package and test-binary modeling;
- full dependency-package syntax loading;
- incremental caching across command invocations;
- language-server overlays and partial results;
- build-driver-specific compatibility guarantees;
- generated-file annotation policy;
- annotation occurrence-to-symbol joining;
- public third-party compiler APIs;
- raising the minimum Go version beyond 1.23.

## Recommendation

Adopt `go/packages` plus `go/types` as Spice's single per-command program model, pin `x/tools v0.36.0` while Go 1.23 is supported, make stable Spice symbol IDs independent of loader-internal IDs, fail closed on ill-typed root packages, and add a narrowly scoped typed-loader implementation before provider graphs, route generation, or module verification.
