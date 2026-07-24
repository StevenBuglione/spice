# Typed Annotation Resolution and CLI Migration

Date: 2026-07-24

## Question

After Spice has a single type-aware `compiler/load.Program`, how should it associate `// @...` metadata with the exact Go declarations selected by the active build, preserve reliable diagnostics, and migrate the CLI without reparsing the source tree or introducing runtime reflection?

## Current delivery state

- Issue #6 / PR #9 is the active implementation lane and is in `CHANGES_REQUESTED`. This research does not change that branch or its acceptance criteria.
- Issue #8 is the only current `[agent-ready]` issue. It intentionally adds typed package loading and stable symbols while leaving `compiler/scan.Tree` and the existing CLI path unchanged.
- The next coherent compiler slice is therefore the bridge from parsed annotations to the typed symbols produced by issue #8.

## Why this bridge is foundational

The current scanner walks the filesystem and reparses every `.go` file. It records a declaration kind, a textual name, a filename, and parsed annotation metadata. That was appropriate for bootstrapping, but it cannot prove that an annotation belongs to the same declaration the Go compiler selected for the current build.

A type-resolved occurrence is required before Spice can safely implement:

- constructor and factory dependency injection;
- controller signature validation and HTTP adapter generation;
- module ownership and cross-module dependency rules;
- configuration, event, scheduling, cache, transaction, and security annotations;
- deterministic generated references to real Go symbols.

Spring obtains annotation metadata from runtime-visible Java declarations and container post-processors. Spice should provide the same developer outcome through the already-loaded Go syntax and type model, with failures before ordinary Go compilation or application startup.

Spring sources, accessed 2026-07-24:

- https://docs.spring.io/spring-framework/reference/core/beans/annotation-config.html
- https://docs.spring.io/spring-framework/reference/core/beans/annotation-config/autowired.html

## Primary-source findings

### 1. Reuse the exact AST and type universe returned by `go/packages`

`go/packages.Package.Syntax` contains the parsed syntax trees used for the package's type checking, and `TypesInfo` describes those same trees. `Syntax` is ordered with `CompiledGoFiles`, subject to documented removal of nil parse results. `Fset` is the shared position space for syntax and type information.

Spice must not call `parser.ParseFile` again after issue #8 loads the program. A second parse would create unrelated AST nodes and token positions, making pointer- or position-based joining fragile and wasting build latency.

Sources, accessed 2026-07-24:

- https://pkg.go.dev/golang.org/x/tools/go/packages#Package
- https://pkg.go.dev/golang.org/x/tools/go/packages#Load

`golang.org/x/tools` is BSD-3-Clause. Issue #8 already owns the version pin and vendoring policy; this bridge needs no additional external dependency.

### 2. Resolve declaring identifiers through `types.Info.Defs`

The standard `go/types` documentation states that `Info.Defs` maps declaring identifiers to the `types.Object` values they define and preserves the invariant that the object position equals the identifier position. `ObjectOf` reads from `Defs` and `Uses`.

Resolution rules should be direct and declaration-specific:

- `*ast.FuncDecl`: use `TypesInfo.Defs[decl.Name]`;
- `*ast.TypeSpec`: use `TypesInfo.Defs[spec.Name]`;
- each `*ast.ValueSpec` name: use `TypesInfo.Defs[name]`;
- package annotations: resolve to the package record rather than inventing a `types.Object` for the package clause.

If an expected definition is absent in an otherwise accepted root package, return an internal consistency diagnostic and do not generate code.

Sources, accessed 2026-07-24:

- https://pkg.go.dev/go/types#Info
- https://pkg.go.dev/go/types#Info.ObjectOf

The Go standard library is BSD-3-Clause.

### 3. Parse declaration documentation, not arbitrary associated comments

Spice annotation syntax is declaration metadata, so only documentation comment groups should be interpreted:

- `ast.File.Doc` for package metadata;
- `ast.FuncDecl.Doc` for functions and methods;
- `ast.GenDecl.Doc` only under an unambiguous declaration rule;
- `ast.TypeSpec.Doc` and `ast.ValueSpec.Doc` for individual grouped specs.

`ast.NewCommentMap` intentionally associates trailing and nearby comments with the largest suitable node. That behavior is useful when rewriting ASTs, but it is too permissive for compiler directives: an end-of-line comment or a visually nearby note must not silently become framework metadata.

Source, accessed 2026-07-24:

- https://pkg.go.dev/go/ast#CommentMap
- https://pkg.go.dev/go/ast#NewCommentMap

### 4. Fail closed for ambiguous grouped declarations

The bootstrap scanner currently attaches a `GenDecl.Doc` annotation to the first spec and a `ValueSpec.Doc` annotation to the first declared name. That is convenient but unsafe once annotations cause generated behavior.

Recommended rules:

1. A `GenDecl.Doc` annotation is valid only when the declaration contains exactly one spec.
2. A `ValueSpec` annotation is valid only when the spec declares exactly one non-blank name.
3. A grouped declaration can annotate each spec independently using a spec documentation comment.
4. Blank identifiers cannot be Spice annotation targets.

Invalid example:

```go
// @Configuration
type (
    HTTPConfig struct{}
    DBConfig   struct{}
)
```

Diagnostic direction:

```text
config.go:3:1: annotation @Configuration is ambiguous on a grouped declaration with 2 targets; place the annotation on one declaration or spec
```

Invalid multi-name example:

```go
// @Something
var primary, fallback Client
```

This should fail rather than select `primary` silently or duplicate one annotation across both variables.

### 5. Preserve both physical and display source positions

`token.FileSet.PositionFor` can return positions adjusted by `//line` directives. Go developers expect user-facing diagnostics to agree with the standard toolchain, but adjusted paths are unsuitable as the sole stable identity because generated or unusual source can remap them.

Each resolved occurrence should retain:

- physical filename and byte offset from the loaded file;
- display position using `PositionFor(pos, true)` for diagnostics;
- stable package and symbol ID from `compiler/load`.

Sorting should use physical package/file/offset identity, then annotation name. Rendering should use the adjusted display position. This keeps output deterministic while respecting Go's diagnostic conventions.

Source, accessed 2026-07-24:

- https://pkg.go.dev/go/token#FileSet.PositionFor

### 6. Model one immutable resolved occurrence

A useful internal boundary is:

```go
package resolve

type Occurrence struct {
    Annotation   annotation.Annotation
    Target       annotation.Target
    SymbolID     load.SymbolID
    PackagePath  string
    PhysicalFile string
    PhysicalOff  int
    Display      token.Position
}

type Result struct {
    Files       int
    Occurrences []Occurrence
    Diagnostics []diagnostic.Diagnostic
}

func Annotations(program *load.Program) Result
```

Exact exported names can be refined during implementation, but the following properties are required:

- the result references stable Spice symbol IDs rather than textual declaration names;
- it does not own or mutate the program's live `types.Object` values;
- all slices are deterministically sorted;
- syntax, annotation parsing, ambiguity, and resolution failures use the same diagnostic model;
- no process-global mutable registry is introduced.

Downstream validation can resolve the symbol through the immutable per-run program when it needs the live `types.Object` or `types.Signature`.

### 7. Load once per CLI command

After issue #8, both `spice annotations` and `spice verify` should perform exactly one typed package load per invocation:

```text
package patterns
  -> compiler/load.Program
  -> compiler/resolve annotations
  -> annotation target/argument validation
  -> output
```

The CLI should pass package patterns through unchanged instead of translating `./...` into a filesystem root. This aligns Spice with `go test`, workspaces, build tags, vendoring, `GOOS`/`GOARCH`, and future `GOPACKAGESDRIVER` integrations.

The current `compiler/scan.Tree` package may remain temporarily for regression comparison, but it should no longer be the authoritative `verify` or `annotations` path after this migration. Removing it can be a later cleanup once no code depends on it.

### 8. Do not silently scan files excluded by the active build

Only syntax returned for the loaded root packages should produce occurrences. An annotation in a file excluded by build constraints must not affect verification or generation for the current build.

Required fixture:

```go
//go:build impossible_spice_test_tag

package fixture

// @UnknownAnnotation
func Ignored() {}
```

With the tag disabled, the package must verify without seeing that annotation. This is a direct correctness improvement over filesystem walking.

### 9. Treat generated-file policy as a separate explicit decision

`CompiledGoFiles` may contain files produced through cgo or other build processing. Whether Spice should accept annotations in generated files affects security, starter generation, and debugging. This bridge should not infer a policy accidentally.

Bootstrap behavior should analyze whatever root syntax the Go package loader type-checked and document that behavior. A later RFC may add a default rejection or explicit opt-in based on `ast.IsGenerated`, but that should not be mixed into the symbol-resolution migration.

### 10. Keep diagnostics actionable and deterministic

Required failures include:

- malformed annotation syntax;
- ambiguous grouped declaration annotation;
- multi-name value annotation;
- blank-identifier target;
- missing symbol for a declaration in an otherwise accepted package;
- package load, syntax, or type errors inherited from issue #8;
- target and argument errors from existing validation.

Diagnostics should identify the annotation, declaration form, and concrete fix. Independent errors should accumulate and sort deterministically; an ill-typed package should still block semantic validation and generation.

## Spring and Go capability comparison

| Concern | Spring behavior | Spice direction |
|---|---|---|
| Metadata source | Runtime annotations on Java declarations | Go documentation directives on the exact loaded AST |
| Symbol identity | JVM class/method/field metadata | Stable package-qualified Spice symbol ID plus live per-run `types.Object` |
| Build selection | Classpath/application context | Active Go module/workspace/build constraints through `go/packages` |
| Ambiguous declaration metadata | Java annotation target is one declaration | Grouped Go declarations fail closed unless one target is unambiguous |
| Failure timing | Often container startup or runtime metadata processing | `spice verify`/generation before ordinary application startup |
| Reflection | Central to container metadata | Not required for declaration resolution |

The result preserves the familiar annotation-driven developer experience while strengthening compile-time determinism and standard Go toolchain compatibility.

## Recommended implementation sequence

After issue #8 merges:

1. add a resolver that walks each loaded root package's existing `Syntax` trees;
2. parse only declaration documentation groups;
3. resolve each target through the package's stable symbol table and `TypesInfo.Defs`;
4. add grouped-declaration ambiguity diagnostics;
5. migrate `spice annotations` and `spice verify` to one load-and-resolve pipeline;
6. preserve existing target and argument validation behavior;
7. test package patterns, build constraints, package annotations, methods, grouped declarations, multi-name values, line directives, deterministic ordering, and ill-typed packages;
8. leave provider graphs, generation, caching, test variants, and generated-file policy out of scope.

## Recommended next issue

Create one bounded implementation issue dependent on #8:

> Resolve Spice annotations against typed Go symbols and migrate CLI verification to the loaded program model.

This issue is the final compiler-front-end bridge before a constructor provider graph, route signature processing, or module dependency model can safely begin.

## Deferred decisions

- annotations in generated files;
- test package variants and test-specific application graphs;
- language-server overlays and incomplete typed programs;
- incremental package-load caching;
- third-party annotation definition discovery;
- provider selection, qualifiers, scopes, groups, and lifecycle;
- deletion of the old filesystem scanner;
- public stability of compiler-internal result types.

## Recommendation

Use the exact `go/packages` AST and `go/types` universe as Spice's sole semantic source, parse declaration documentation groups only, fail closed for ambiguous grouped declarations, retain both physical and display positions, and migrate both CLI annotation commands through one deterministic load-resolve-validate pipeline. This creates a trustworthy typed annotation model without changing Go syntax, using runtime reflection, or diverging from the active Go build.