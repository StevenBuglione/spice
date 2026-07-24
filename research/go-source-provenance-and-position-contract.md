# Go Source Provenance, Positions, and File Identity

Date: 2026-07-24

## Question

What source-identity and position contract should Spice use across package loading, typed annotation resolution, diagnostics, module enforcement, and generated-code freshness so valid Go source remains discoverable under `//line` directives and cgo rewriting, while toolchain-generated internals do not leak into Spice's logical program model?

## Decision summary

Spice should represent source provenance with two distinct position concepts and one explicit file model:

1. **Physical identity** comes from `token.FileSet.PositionFor(pos, false)`. It identifies the file and byte offset that supplied the AST node to the current load. Spice uses it for ownership, joining AST nodes to typed objects, deterministic semantic ordering, and stable source fingerprints.
2. **Display position** comes from `token.FileSet.PositionFor(pos, true)`. It honors valid `//line` directives and is used for human-facing diagnostics, generated source maps, and external tool output.
3. A **loaded file record** pairs one syntax tree with its physical compiled-file identity and any source/display mapping. Spice must never sort a filename slice independently from the AST slice and continue implying positional correspondence.
4. Package source ownership is decided from physical and display provenance together, not from display filenames alone.
5. Diagnostic positions are structured fields—filename, numeric line, numeric column, and optional offset—not opaque strings.
6. Adjusted filenames are untrusted metadata. Spice may display them, but must never use them as filesystem paths, module ownership evidence, or authorization boundaries.
7. Context cancellation is part of the loader contract and requires a committed integration test with an already-running blocking package driver.

This preserves the Go toolchain's own source mapping while giving later Spice phases one deterministic, security-conscious program model. It also resolves the apparent conflict between retaining committed source-mapped/generated Go and excluding cgo cache helpers.

## Why this needs an explicit contract

A typed Go package is not always a one-to-one list of repository `.go` files:

- `//line` directives can change the filename, line, and column reported for a token without changing the physical file that supplied the AST.
- cgo transforms source and adds generated Go files in the build cache.
- `go/packages.Package.GoFiles`, `CompiledGoFiles`, and `Syntax` describe related but different layers of the load.
- external `GOPACKAGESDRIVER` implementations may return package metadata through another build system.
- overlays replace file contents while retaining a logical target path.
- parse or type errors may arrive as rendered position strings even though deterministic ordering requires numeric components.

If Spice collapses those concepts into one filename string, it will either drop valid source declarations or admit generated implementation details. Later annotation, dependency-injection, route, configuration, and module phases would then operate on a different logical program from the developer's source.

## Primary sources and status

Sources were accessed on 2026-07-24.

### Go token positions and line directives

- `go/token.FileSet.PositionFor`:
  - https://pkg.go.dev/go/token#FileSet.PositionFor
- Go compiler line-directive syntax:
  - https://pkg.go.dev/cmd/compile#hdr-Compiler_Directives
- Go parser example showing adjusted and unadjusted positions:
  - https://pkg.go.dev/go/parser#example-ParseFile
- Go scanner error sorting:
  - https://cs.opensource.google/go/go/+/refs/tags/go1.26.0:src/go/scanner/errors.go

The Go standard library is BSD-3-Clause licensed.

Relevant facts:

- `PositionFor(pos, true)` applies line-directive adjustments.
- `PositionFor(pos, false)` returns the unadjusted physical position.
- A legal line directive can name a non-Go file such as a schema, template, or generated-source origin.
- The Go scanner sorts errors by filename, numeric line, numeric column, and message; it does not rely on lexical ordering of rendered `file:line:column` strings.
- The scanner implementation notes that offsets alone are insufficient when line directives are involved.

### `go/packages` file layers

- Package documentation:
  - https://pkg.go.dev/golang.org/x/tools/go/packages
- Spice's pinned release source:
  - https://github.com/golang/tools/tree/v0.36.0/go/packages
- Current package model source for comparison:
  - https://cs.opensource.google/go/x/tools/+/master:go/packages/packages.go

`golang.org/x/tools` is BSD-3-Clause licensed. Spice currently pins v0.36.0 because that is the latest line compatible with the project's Go 1.23 floor.

Relevant facts:

- `GoFiles` identifies package Go source before transformations such as cgo preprocessing.
- `CompiledGoFiles` identifies files suitable for type checking and can contain generated build-cache paths.
- `Syntax` is syntax for compiled files. The package documentation describes file-order correspondence, while also noting that nil ASTs may be omitted. Consumers therefore need an explicit mapping rather than silently assuming equal slice lengths forever.
- `Package.Dir` is the build driver's package directory and is a better package-root signal than taking the directory of the first alphabetically sorted compiled file.
- `Config.Context` is forwarded to package loading and external drivers.

### Go command and cgo metadata

- `go list` package fields:
  - https://pkg.go.dev/cmd/go#hdr-List_packages_or_modules
- cgo command behavior:
  - https://pkg.go.dev/cmd/cgo
- Go package build model:
  - https://pkg.go.dev/go/build

Relevant facts:

- The Go command distinguishes `GoFiles`, `CgoFiles`, and `CompiledGoFiles`.
- With compiled-file reporting, cgo-transformed files may be absolute paths in the Go build cache.
- cgo generates compiler-facing declarations and helper functions that are implementation details rather than logical package declarations authored by the developer.

### Cancellation and subprocess behavior

- `os/exec.CommandContext`:
  - https://pkg.go.dev/os/exec#CommandContext
- `exec.Cmd.WaitDelay`:
  - https://pkg.go.dev/os/exec#Cmd.WaitDelay

Relevant facts:

- A context-aware command can interrupt an already-running process.
- Process termination and pipe cleanup require bounded waits; tests should prove that cancellation returns and the helper process exits rather than merely checking an already-cancelled fast path.

### Spring capability relationship

- Spring Boot ahead-of-time processing:
  - https://docs.spring.io/spring-boot/reference/packaging/aot.html
- Spring Modulith application-module verification:
  - https://docs.spring.io/spring-modulith/reference/verification.html

Spring projects are Apache License 2.0. These sources are capability evidence only; Spice should not copy JVM container or classpath-scanning implementation.

The useful Spring outcome is trustworthy build-time metadata: framework decisions, generated initialization, and architecture verification must refer to the same logical declarations the developer sees. Spice achieves that through Go's package, token, AST, and type systems rather than reflection.

## Findings and decisions

### 1. Physical identity and display position are different data

A single `token.Pos` should be expanded into both forms at the compiler boundary:

```go
type SourcePosition struct {
    Physical token.Position
    Display  token.Position
}
```

Exact exported names may differ. The semantic rule matters:

- `Physical = fset.PositionFor(pos, false)`
- `Display = fset.PositionFor(pos, true)`

Physical identity answers:

- Which loaded file supplied this AST node?
- Is this node owned by a selected source or compiled file?
- How do two annotations sort deterministically when their display locations are remapped?
- Which overlay target or file content should be fingerprinted?

Display position answers:

- What source location should the developer see?
- What origin did a generator intentionally encode with `//line`?
- What location should an editor or generated diagnostic hyperlink show?

Neither field replaces the other.

### 2. Adjusted filenames are not filesystem authority

A line directive can legally contain a relative path, an absolute-looking path, or a non-Go filename. Spice must treat the adjusted filename as untrusted display metadata.

Spice must not:

- open the adjusted path;
- use it to decide module membership;
- use it to authorize generated output;
- use it to decide whether a source file is inside the repository;
- normalize it against the process working directory and treat the result as physical truth;
- include host-specific adjusted absolute paths in stable generated output without an explicit sanitization policy.

This is both correctness and security policy. A source comment must not redirect framework filesystem access outside the selected package or module.

### 3. Loaded files need an explicit paired record

The package model should expose or internally preserve a record conceptually like:

```go
type File struct {
    PhysicalPath string
    Syntax       *ast.File
    SourcePath   string
    DisplayPath  string
    Generated    bool
}
```

The first implementation does not need every field publicly exported. It does need one authoritative pairing between an AST and the physical file from which the current token file set parsed it.

Recommended construction:

1. For each non-nil syntax tree, obtain its physical filename through `PositionFor(file.Pos(), false)`.
2. Associate that physical path with the corresponding compiled-file metadata when available.
3. Retain the original order returned by `go/packages` for raw compatibility.
4. Build a separately sorted deterministic view by physical path and stable tie-breakers when Spice needs stable iteration.
5. Never sort `CompiledGoFiles` in place or independently while exposing `Syntax` as if the indexes still correspond.

If the documented `Syntax`/`CompiledGoFiles` cardinality differs because nil ASTs were removed, physical filenames provide the join key. A mismatch or duplicate physical key should become a deterministic internal diagnostic rather than an unchecked index assumption.

### 4. Package directory comes from package metadata, not sorted files

Use `packages.Package.Dir` when available from the pinned version and driver. Validate and clean it, but do not infer the package directory from the first element of an independently sorted file list.

Fallback order when a custom driver omits `Dir` should be explicit:

1. common directory of selected physical source files when all agree;
2. module metadata plus import path only when that mapping is unambiguous;
3. a deterministic missing-directory diagnostic rather than a guessed build-cache directory.

A package record may still expose deterministically sorted file paths, but those are derived views and must not redefine package ownership.

### 5. Source ownership uses both physical and display provenance

The central ownership problem has three important cases:

| Case | Physical file | Adjusted/display file | Keep as logical source? |
|---|---|---|---|
| Ordinary source with no directive | selected source | selected source | yes |
| Committed generated/source-mapped Go | selected source | schema/template/origin | yes |
| cgo user declaration after transformation | generated compiled file | selected original cgo source | yes |
| cgo synthetic helper | generated compiled file | generated/cache helper | no |

A robust initial predicate is:

```text
owned when:
- the syntax file's physical identity is a selected source file; OR
- the declaration's adjusted origin is a selected source file;

and never owned solely because a generated helper name resembles a user name.
```

This predicate solves both active edge cases:

- A declaration in a committed `.go` file remains owned even when `//line` changes its display filename to `generated/schema.proto`.
- A cgo-transformed user declaration remains owned when its physical AST is cache-backed but its adjusted position maps back to the selected original cgo source.
- A cgo helper is excluded when neither physical nor adjusted provenance points to selected source.

Implementation should compare normalized physical paths against `GoFiles`, overlay target paths, and driver metadata using platform-aware path handling. It should not compare only `PositionFor(pos, true).Filename`.

### 6. Names remain a secondary defense, not the ownership mechanism

Filters for non-addressable declarations such as package `init` and `_` remain valid because later compiler phases cannot refer to them by stable logical name.

Cgo helper prefixes such as `_C`, `_cgo`, and `__cgo` can support assertions and diagnostics, but prefix filtering must not be the primary ownership rule. Toolchain helper names can change, and users can legally declare unusual names. Provenance should decide whether a declaration is source-owned; name policy decides whether an owned declaration is addressable by Spice.

### 7. Stable identity uses logical declarations; source identity supports diagnostics and joins

Stable symbol IDs should remain package-and-declaration based:

```text
<package-path>.<name>
<package-path>.<receiver-origin>.<method>
```

Physical position is not appended to ordinary IDs merely to make collisions disappear. That would make IDs sensitive to file movement and source formatting and would weaken the logical identity contract.

Instead:

- omit non-addressable declaration kinds that cannot have unique logical names;
- reject impossible duplicate addressable identities as a compiler-model invariant;
- keep physical source identity as separate metadata;
- use source identity for AST/type joins, diagnostics, and deterministic ordering.

### 8. Diagnostics must use structured numeric positions

Rendered strings such as `file.go:10:16` must not be the primary sort key. Lexical sorting places line 10 before line 2.

Normalize package errors into fields conceptually like:

```go
type Diagnostic struct {
    PackagePath string
    Physical    SourcePoint
    Display     SourcePoint
    Kind        string
    Message     string
}

type SourcePoint struct {
    Filename string
    Offset   int
    Line     int
    Column   int
}
```

When `go/packages.Error.Pos` is the only available representation, parse it from the right so filenames containing colons remain representable where possible. Prefer structured token positions whenever the AST or `types.Error` supplies them.

Recommended diagnostic order:

1. display filename;
2. numeric display line;
3. numeric display column;
4. kind;
5. message;
6. package path;
7. physical filename and offset as stable tie-breakers.

For compiler-internal semantic occurrence order, use physical filename and offset first. This follows the user-visible behavior of Go's scanner while preserving deterministic identity under source mapping.

### 9. `//line` needs positive and adversarial tests

The committed test matrix should include:

- a normal `.go` file with `//line generated/schema.proto:40` before a type, variable, or function;
- exact assertion that the symbol remains in the catalog;
- physical filename/offset assertion pointing to the committed `.go` file;
- display filename/line assertion pointing to the directive origin;
- stable ordering when display line numbers run opposite to physical order;
- a directive naming an absolute-looking or parent-traversing path, proving Spice displays but never opens or owns that path;
- repeated runs with byte-identical symbol and diagnostic summaries;
- mutation proof that using only adjusted positions loses the declaration.

This contract is also required by issue #11's typed annotation resolver: physical ordering and adjusted display positions must be preserved simultaneously.

### 10. Cgo requires an executable provenance test

A Linux integration fixture with cgo enabled should assert:

- `Package.Dir` is the source package directory;
- selected source files and compiled files are represented separately;
- each exposed syntax tree has a recoverable physical compiled-file identity;
- user declarations in the `import "C"` file remain present;
- ordinary committed generated `.go` declarations remain present;
- cgo helper declarations remain absent from Spice symbols;
- package and symbol display positions point to meaningful source origins;
- all stable symbol IDs remain unique;
- changing only C implementation text does not change user-declaration stable IDs.

Skip only when cgo or a C compiler is genuinely unavailable. CI should execute rather than skip on the supported Linux jobs.

### 11. Active cancellation is a public loader contract

The existing already-cancelled fast path proves only that Spice checks `ctx.Err()` before loading. It does not prove that an in-flight `packages.Load` or external driver is cancelled.

A committed integration test should use a helper process as `GOPACKAGESDRIVER`:

1. Start the helper and have it signal that it is blocked.
2. Call `Load` with a cancellable context and the helper selected through `Options.Env`.
3. Wait for the started signal before cancelling.
4. Assert that `Load` returns within a bounded timeout with `context.Canceled` or the documented wrapped equivalent.
5. Assert that the helper process terminates and no child remains waiting on pipes.
6. Retain the separate already-cancelled test for the zero-work fast path.

Use synchronization signals rather than relying only on sleeps. Platform-specific helper process details may be isolated behind test helpers, but the supported Linux CI path must execute.

### 12. Overlays retain physical target identity

For an overlay, the physical source identity is the overlay target filename supplied to the package loader, not a temporary file chosen by an editor or test harness. Display positions may still be remapped by line directives inside overlay content.

Spice should:

- clone overlay bytes at the load boundary;
- avoid serializing contents or absolute overlay paths into stable IDs;
- use the overlay target for physical source joins;
- apply the same untrusted-display policy to line directives in overlay content;
- include overlay-sensitive content fingerprints only in ephemeral or explicitly scoped generation freshness data.

### 13. External package drivers require invariant validation

`GOPACKAGESDRIVER` is a supported Go ecosystem extension point and should remain available for Bazel or other build systems. Spice must not assume every driver returns identical path spelling or complete optional metadata.

After loading, validate deterministic invariants:

- every root package has a stable package path;
- every non-nil AST has a physical token-file identity;
- file-to-AST joins are unambiguous;
- package directory is either known or diagnosed;
- addressable stable symbol IDs are unique;
- selected source ownership does not depend on adjusted display paths alone;
- cancellation is honored.

The driver remains user-selected trusted build tooling, but its output is still input to Spice's compiler model and should fail closed when internally inconsistent.

## Recommended internal model

A future-compatible model can remain small:

```go
type SourceFile struct {
    PhysicalPath string
    SourcePath   string
    Syntax       *ast.File
}

type Position struct {
    Physical token.Position
    Display  token.Position
}

type Package struct {
    Path       string
    Name       string
    Dir        string
    ModulePath string
    Files      []SourceFile
    Raw        *packages.Package
}

type Symbol struct {
    ID       ID
    Kind     Kind
    Position Position
    Object   types.Object
    Node     ast.Node
}
```

This is a design boundary, not a demand for those exact exported names. It can be implemented incrementally while preserving current compatibility accessors.

## Deterministic ordering policy

Use different ordering for different products:

### Package and file model

- packages by package path, then loader ID as a final in-run tie-breaker;
- files by physical path;
- raw loader order retained only in a clearly named compatibility field;
- AST association never inferred from a separately sorted sibling slice.

### Symbols and resolved annotations

- package path;
- stable symbol ID;
- physical filename;
- physical byte offset;
- kind.

### User diagnostics

- display filename;
- numeric display line;
- numeric display column;
- diagnostic kind;
- message;
- package path and physical identity as tie-breakers.

### Generated output and manifests

- stable semantic IDs and module-relative physical paths;
- never display paths from `//line` as output ownership paths;
- never absolute sandbox/build-cache paths.

## Security analysis

### Path injection through line directives

A source file can name arbitrary display paths. Treating those paths as physical files could cause path traversal, accidental disclosure, or incorrect module-boundary decisions. Display them only after normal output escaping; never read or write them.

### Build-cache leakage

Cgo and custom drivers can expose absolute cache paths. These should remain ephemeral compiler details and must not enter stable IDs, committed manifests, generated Go, or portable diagnostics when a source-relative display is available.

### Environment and secret handling

`Options.Env` may contain secrets. Source provenance and generation fingerprints must not serialize raw environment values. Use explicit allowlisted build-context fields or hashes where a future generation contract needs freshness.

### Denial of service and cancellation

Package drivers can block or produce unexpectedly large metadata. Context cancellation, bounded process cleanup, and no duplicate package loads are required resource controls.

### Fail-closed behavior

Ambiguous file/AST association, duplicate stable IDs, malformed structured positions, or missing package ownership should produce deterministic diagnostics and prevent later semantic generation.

## Compatibility and performance

### Go versions

- Keep `golang.org/x/tools v0.36.0` while Go 1.23 remains supported.
- Verify assumptions against that exact release, not only current documentation.
- Isolate `go/packages` behind `compiler/load` so future field or ordering changes have one migration boundary.

### Workspaces and custom build systems

- Pass package patterns, environment, build flags, overlays, and context through unchanged.
- Use package-driver metadata instead of repository walking.
- Avoid requiring that physical paths live under one module when analysis legitimately spans a workspace.

### Performance

- Compute both physical and display positions from the existing file set; no second parse or package load is needed.
- Build path sets/maps once per package.
- Pair files once, then retain deterministic derived views.
- Structured diagnostics avoid repeated reparsing during sorting and rendering.

### Developer ergonomics

- Display adjusted positions when a generator intentionally maps back to a schema or template.
- Include the physical Go location as secondary detail only when it helps correction or when the display origin is not locally resolvable.
- Explain provenance failures in terms of files and build selection rather than internal `token.Pos` or cgo helper terminology.

## Spring capability alignment

This contract supports several practical Spring outcomes:

- **Annotation metadata discovery:** declarations remain attached to the correct logical symbol even when generated or source-mapped.
- **AOT/generated initialization:** source fingerprints and diagnostics refer to stable physical inputs without leaking build-cache details.
- **Dependency injection and web adapters:** provider and handler metadata do not include cgo implementation helpers.
- **Modulith verification:** package/module ownership is based on actual loaded source and import paths, not arbitrary display filenames.
- **Developer diagnostics:** generated origins can be shown without weakening filesystem or architecture boundaries.

Spice remains Go-native: the implementation uses the standard token/AST/type model and one `go/packages` program rather than reproducing classpath scanning or a runtime annotation container.

## Bounded implementation implications

This research does not create a fourth ready issue and does not change an active issue's acceptance criteria. The current backlog already sequences the relevant work:

- the typed loader establishes provenance and structured positions;
- typed annotation resolution consumes physical identity and display positions;
- provider and graph phases consume stable logical symbols rather than raw loader internals.

When backlog capacity exists, a separate public-model cleanup issue is justified only if the loader cannot adopt the file-pair and structured-position contract within its existing bounded work. Do not create a duplicate issue merely to restate these findings.

## Required regression matrix for the compiler foundation

A complete contract test set should cover:

1. ordinary source with identical physical/display position;
2. committed generated Go with `//line` to a non-Go origin;
3. source-mapped declarations retained with exact physical and display positions;
4. cgo user declarations retained;
5. cgo helpers excluded;
6. compiled-file/AST association under cgo;
7. diagnostics at lines 2 and 10 sorted numerically;
8. two diagnostics with identical display location but distinct physical offsets;
9. overlay source with a line directive;
10. adjusted path attempting parent traversal, displayed but never opened;
11. already-cancelled context fast path;
12. active cancellation of a blocking external package driver;
13. custom driver metadata missing an optional field, producing a deterministic diagnostic;
14. repeated load summaries with identical output;
15. vendor-only execution with network access disabled.

## Recommendation

Adopt dual physical/display positions and an explicit file-to-AST provenance record as compiler invariants. Determine source ownership from both physical loaded-file identity and adjusted source origin, never from display filenames alone. Keep raw loader compatibility data distinct from deterministic Spice views, sort diagnostics numerically from structured positions, and protect in-flight cancellation with an executable driver test.

This is the smallest durable contract that retains valid Go source mapping, excludes cgo implementation details, supports future typed annotation and module joins, and keeps Spice's build-time behavior deterministic, secure, and understandable.
