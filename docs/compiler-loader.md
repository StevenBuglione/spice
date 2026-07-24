# Type-Aware Go Package Loader

`compiler/load` is Spice's single boundary around `golang.org/x/tools/go/packages`.
It loads the root package patterns selected by the standard Go toolchain and
returns one per-run type universe for later annotation resolution, dependency
injection, controller processing, configuration metadata, and module analysis.

## Usage

```go
program, err := load.Load(ctx, load.Config{
    Dir:        workspace,
    Env:        os.Environ(),
    BuildFlags: []string{"-tags=integration"},
    Overlay:    unsavedFiles,
}, "./...")
```

Package patterns are passed to the Go package driver unchanged. Spice does not
turn `./...` into a filesystem walk. Test variants are disabled by default.

A successful `Program` exposes deterministic package and symbol records while
retaining live `go/types` objects and AST nodes from the same load operation.
Those live values belong to that `Program` only and must not be mixed with
objects from another call.

## Stable symbol IDs

Stable IDs are logical, readable identities independent of temporary checkout
paths and opaque `packages.Package.ID` values:

```text
package                       example.com/app/users
type/function/variable/const  example.com/app/users.User
method                        example.com/app/users.User.Find
```

Pointer receivers and instantiated generic receivers normalize to the defining
named receiver origin. Function signatures are retained separately so changing
a signature does not silently change declaration identity.

## Diagnostics

The loader is a quiet library. It does not print or exit. Package-list, parse,
and type errors are normalized into deterministic source-positioned
`load.Diagnostic` values. Any diagnostic or ill-typed root package returns a
`load.LoadError`, and semantic generation must stop.

## Dependency and offline policy

Spice pins `golang.org/x/tools v0.36.0` because it is the newest adjacent
`go/packages` release compatible with the project's Go 1.23 floor. Its
transitive `golang.org/x/mod v0.27.0` and `golang.org/x/sync v0.16.0`
dependencies are pinned by `go.sum`.

`golang.org/x/tools` is BSD-3-Clause licensed. The dependency is isolated behind
`compiler/load`, and the standard `go mod vendor` output is committed so builds
and scheduled agents can run with public module access disabled:

```text
GOPROXY=off go test -mod=vendor ./...
```

The loader never changes proxy, checksum, private-module, or VCS policies. It
passes the caller's environment to the standard Go package driver.

## Deliberate boundaries

This compiler slice does not:

- join Spice annotations to typed symbols;
- replace the existing filesystem annotation scanner or CLI flow;
- load syntax for dependencies with `NeedDeps`;
- include test package or generated test-binary variants by default;
- create dependency-injection graphs or generated code;
- maintain a mutable process-global cache or type universe.
