# Go-Native Application Module Discovery and Boundary Model

Date: 2026-07-24

## Question

After Spice has one typed Go program and annotations resolved to stable symbols, what application-module discovery and boundary contract should it adopt so Spring Modulith-style module verification is useful in real Go projects without relying on JVM package rules, reflection, filesystem conventions that break in workspaces, or a second source-loading pipeline?

## Current delivery state

At the time of this research:

- issue #8 / PR #15 is the single active delivery lane for typed package loading and stable symbols and is being recovered from verifier-requested stable-symbol identity gaps;
- this research does not pin or modify that mutable implementation branch; actual GitHub branch, lease, state-comment, and CI state remain authoritative;
- issue #11 is the next ready slice for resolving annotations against typed Go symbols and migrating CLI loading;
- issue #13 follows with `@Bean` provider signature analysis and a deterministic provider catalog;
- issue #17 follows with deterministic provider dependency graph validation;
- the ready backlog is already at the cap of three, so this research creates no additional implementation issue and does not change any active acceptance criteria.

The module model described here depends on issues #8 and #11. It can be implemented independently of provider execution, but it must consume the same loaded package and resolved annotation universe rather than reloading or rescanning source.

## Why this is the next architecture question

Spice's roadmap treats architecture enforcement as a first-class product feature, but a module cycle detector alone is not enough. Before dependency rules can be checked, Spice needs deterministic answers for:

- what declares an application module;
- which loaded packages belong to it;
- what its stable identity is across directories, workspaces, and checkout locations;
- which package is its default public API;
- how additional API surfaces are exposed;
- how incoming access to internal packages is rejected;
- how outgoing allowed-dependency rules are represented;
- how blank imports, dot imports, build tags, generated files, and workspace modules affect the graph;
- how package-level Go cycles differ from logical module cycles;
- which packages are intentionally outside the module model;
- what belongs in the first bounded implementation versus later named-interface, allow-list, documentation, and module-test slices.

Spring Modulith provides the desired outcomes: logical module discovery, a default API package, internal-package protection, optional named interfaces, optional allowed dependencies, module-cycle verification, documentation, and focused tests. Spice should preserve those outcomes while using Go import paths, package documentation annotations, the Go build-selected package graph, and deterministic compiler data.

## Primary sources and status

Sources were accessed on 2026-07-24.

### Spring Modulith 2.1.0

- Project overview and current stable version:
  - https://docs.spring.io/spring-modulith/reference/index.html
- Application-module fundamentals, default API packages, internals, open modules, named interfaces, allowed dependencies, and detection strategies:
  - https://docs.spring.io/spring-modulith/reference/fundamentals.html
- Structural verification rules:
  - https://docs.spring.io/spring-modulith/reference/verification.html
- Current module detection strategy API:
  - https://docs.spring.io/spring-modulith/docs/current/api/org/springframework/modulith/core/ApplicationModuleDetectionStrategy.html
- Current `@ApplicationModule` API:
  - https://docs.spring.io/spring-modulith/docs/current/api/org/springframework/modulith/ApplicationModule.html
- Current `@NamedInterface` API:
  - https://docs.spring.io/spring-modulith/docs/current/api/org/springframework/modulith/NamedInterface.html
- Current module type semantics:
  - https://docs.spring.io/spring-modulith/docs/current/api/org/springframework/modulith/ApplicationModule.Type.html
- Module-focused test bootstrap modes:
  - https://docs.spring.io/spring-modulith/reference/testing.html

Spring Modulith is Apache License 2.0. Spice uses its public documentation as capability evidence, not as source code and not as a requirement to reproduce ArchUnit or JVM runtime mechanics.

Relevant Spring outcomes:

- direct-subpackage discovery is the default, with explicitly annotated discovery available;
- a module root package is its default API;
- nested packages are internal unless exposed through named interfaces or the module is open;
- outgoing module dependencies may be explicitly restricted;
- module cycles are invalid for closed modules;
- module tests can bootstrap one module with none, direct, or all dependencies.

### Go language and toolchain

- Go language specification, package and import semantics:
  - https://go.dev/ref/spec
- Official Go module layout guidance:
  - https://go.dev/doc/modules/layout
- Go internal-package enforcement:
  - https://go.dev/doc/go1.4#internalpackages
- `go` package-list, wildcard, build-context, and import-graph semantics:
  - https://pkg.go.dev/cmd/go
- `go/packages` typed package and import graph API:
  - https://pkg.go.dev/golang.org/x/tools/go/packages@v0.36.0

Go and `golang.org/x/tools` use BSD-style licenses. Spice already pins `golang.org/x/tools` for issue #8, so module analysis should not add another source loader or architecture dependency.

Relevant Go constraints:

- the package, not the type, is the import and visibility unit;
- exported declarations are visible to any package that can import their package;
- the `internal` directory rule restricts imports outside a filesystem/import-path subtree, but does not express logical sibling-module boundaries inside that subtree;
- Go rejects package-level import cycles, yet a cycle can still exist after multiple packages are aggregated into larger logical modules;
- `go/packages` returns the build-selected roots and an import graph for one build context;
- test variants produce synthetic and duplicate package identities and remain explicitly outside issue #8.

### Go architecture tools

- `go-arch-lint`:
  - https://github.com/fe3dback/go-arch-lint
- Depguard:
  - https://github.com/OpenPeeDeeP/depguard

`go-arch-lint` is MIT licensed and demonstrates demand for named components, allowed dependency directions, graph output, and migration-friendly rule adoption. Depguard v2 is GPL-3.0 and demonstrates import allow/deny policy, but its license and file-glob configuration make it unsuitable as a Spice dependency.

Both tools are comparison evidence only. Spice needs a model integrated with its typed program, annotations, diagnostics, generation, module tests, runtime metadata, and Spring capability map rather than a parallel YAML-only linter.

## Findings and decisions

### 1. Use explicit package annotations for module roots

The first Spice module model should use an explicit package documentation annotation:

```go
// @Module
package order
```

The annotation belongs in the package documentation group selected by the Go build and resolved by issue #11 to the stable package symbol.

Spice should not initially infer modules from every direct subpackage of a repository, Go module, `internal` directory, or CLI working directory. Those conventions are attractive for demos but ambiguous in real projects containing:

- `cmd`, `internal`, `pkg`, generated, migration, and integration packages;
- multiple binaries;
- multiple Go modules in one workspace;
- additional application roots;
- nested technical packages that are not business modules;
- package paths whose first directory is organizational rather than functional.

Spring's explicitly annotated detection mode proves that explicit module roots remain compatible with the intended capability. For Go, explicit package metadata is also more refactor-visible and does not require a separate YAML path matcher.

A future optional convention mode may infer direct child modules, but it must produce the same catalog and be explicitly selected. It must not silently become a second semantic model.

### 2. Stable module identity is the root package import path

The canonical module ID should initially be the module root package's full import path:

```text
example.com/shop/internal/order
```

Do not derive identity from:

- absolute directories;
- package names;
- the final path segment alone;
- source order;
- `packages.Package.ID`;
- repository checkout location.

Package names and final path segments can collide across workspace modules. Full import paths are already the stable names used by the Go type system and loader.

A later optional short alias may improve display and allow-list ergonomics, but it must be explicitly declared, globally unique within the application model, and never replace the canonical import-path identity in serialized compiler records.

### 3. A module owns its root package and descendants

A module contains its root package and every loaded descendant package whose import path has the root followed by a path separator.

For example:

```text
example.com/shop/internal/order
example.com/shop/internal/order/api
example.com/shop/internal/order/domain
example.com/shop/internal/order/storage/postgres
```

all belong to the `order` module rooted at `example.com/shop/internal/order`.

Membership must be computed from import paths, not filesystem prefix strings. Path-segment boundaries matter: `.../orderhistory` is not a descendant of `.../order`.

The initial implementation should reject overlapping or nested `@Module` roots with an actionable diagnostic. Spring supports nested modules, but nested ownership, parent APIs, test bootstrap, documentation, and dependency qualification deserve a separate bounded design. Silently choosing nearest-ancestor ownership now would commit Spice to parent/child semantics before those questions are answered.

### 4. The root package is the default module API

A closed module exposes its root package to other modules. Descendant packages are internal by default.

This maps Spring Modulith's default API-package outcome directly onto Go's package import unit:

```text
example.com/shop/internal/order           public module API
example.com/shop/internal/order/domain    internal to order
example.com/shop/internal/order/storage   internal to order
```

The rule applies to package imports, including imports of exported declarations. Exported Go names inside an internal module package remain useful to sibling packages inside the same module but do not become cross-module API merely because they begin with an uppercase letter.

This is a core value Spice adds beyond the Go compiler.

### 5. Additional named interfaces should be package-level only at first

Additional API surfaces should be declared on package documentation:

```go
// @NamedInterface("events")
package events
```

The package must already belong to exactly one module and must be a descendant of that module root. Its stable named-interface identity is:

```text
<module-import-path>::<interface-name>
```

The first model should not support exposing individual types from an otherwise internal package. Spring can model type-level named interfaces because its verifier reasons over Java classes. In Go, importing a package exposes all its exported declarations, so a type-level interface would either:

- claim stronger encapsulation than the language provides; or
- require symbol-by-symbol usage analysis and confusing diagnostics after the package has already been imported.

Package-level named interfaces are predictable, refactor-friendly, and aligned with Go's visibility unit. Type-level named interfaces can be reconsidered only with a concrete developer need.

### 6. Go `internal` and Spice module internals are complementary

Spice should encourage normal Go `internal` layout for preventing imports from outside the repository or parent subtree. It must not treat that mechanism as sufficient module enforcement.

Example:

```text
example.com/shop/internal/order/storage
example.com/shop/internal/inventory
```

Both packages are within the parent allowed to import `internal`, so the Go toolchain may permit `inventory` to import `order/storage`. Spice must still reject that cross-module internal access.

Spice should never weaken or emulate the Go toolchain's `internal` check. Go compilation remains authoritative for language-level legality; Spice adds logical application architecture rules.

### 7. Import declarations are the first dependency evidence

For the initial module graph, a direct dependency exists when a loaded package in module A imports a loaded package in module B.

This includes:

- normal imports;
- aliased imports;
- dot imports;
- blank side-effect imports.

Blank imports must count because they create initialization and runtime behavior. Dot imports must count because they still create a package dependency even though selectors disappear from source.

The graph should not initially inspect function bodies, call graphs, SSA, reflection strings, configuration files, SQL, network calls, or generated runtime registration. Cross-package Go references require imports, making the import graph the correct low-cost first boundary.

Later event-contract and configuration-ownership analysis may add typed logical edges that are not ordinary imports, but they should be represented as additional edge kinds rather than replacing import evidence.

### 8. Use the existing loaded program and source import positions

Module analysis must consume the exact `compiler/load.Program` and resolved annotations from issues #8 and #11.

It must not:

- call `packages.Load` again;
- run `go list` separately;
- walk directories;
- parse files again;
- create another token file set;
- inspect files excluded by build constraints.

Each dependency edge should retain:

- source module ID;
- source package path;
- target module ID;
- target package path;
- import path spelling;
- import kind: normal, alias, dot, or blank;
- physical source identity for deterministic sorting;
- user-facing display position.

Boundary violations should point to the offending `ast.ImportSpec`, not merely the source package or target package declaration.

### 9. Only loaded first-party packages participate

The module catalog should operate over the loaded root package set selected by the caller's Go package patterns.

Packages not present in that application set are external dependencies and are outside the application-module graph. This includes standard-library packages and third-party module dependencies.

This gives a simple, deterministic boundary:

- Spice module rules govern the application packages the developer asked Spice to analyze;
- ordinary dependency and vulnerability tools govern external modules;
- a narrow package pattern intentionally produces a partial application model.

Commands that promise whole-application module verification should therefore document and use a broad package pattern such as `./...`. They must pass the pattern unchanged to issue #8's loader.

### 10. Unassigned first-party packages must be visible, not silently ignored

A loaded first-party package that belongs to no explicit module should be represented as unassigned metadata rather than quietly discarded.

The first module catalog should report unassigned packages deterministically but should not automatically fail solely because they exist. Common legitimate examples include:

- `cmd/...` entrypoints;
- generated application packages;
- migration commands;
- build tooling;
- a thin application shell.

Boundary verification should apply these initial rules:

- unassigned packages may import module APIs to compose or run the application;
- unassigned packages may not import module-internal descendant packages;
- module packages importing unassigned first-party packages should produce a warning-quality or separately configurable diagnostic until shared/infrastructure module policy is designed;
- no unassigned package should implicitly become a shared module.

A later strict policy may require all non-command first-party packages to belong to a module. The catalog must preserve enough metadata to add that check without changing identity rules.

### 11. Closed modules only in the first implementation

The initial module type should be closed:

- root package and named-interface packages are exposed;
- all other descendant packages are internal;
- module cycles are invalid.

Do not add an `open` flag in the first slice. Spring describes open modules primarily as a migration aid and warns that they often indicate suboptimal modularization. Spice can later add an explicit migration mode that exposes every descendant package, but it should be visibly weaker and excluded from strict architecture guarantees.

### 12. Allowed outgoing dependencies are optional and tri-state

Spice should eventually match Spring's useful migration behavior:

- metadata absent: any other module's exposed API may be imported;
- explicit empty list: no outgoing application-module dependencies are allowed;
- explicit non-empty list: only the named modules or named interfaces are allowed.

The representation must preserve the difference between absent and empty.

Do not encode multiple dependencies in a comma-separated string. That is hard to validate, escape, refactor, and format. The annotation model should first gain either:

- a first-class string-list argument kind; or
- explicitly repeatable dependency annotations with deterministic duplicate handling.

A string-list argument on the module annotation is the cleaner long-term shape:

```go
// @Module(allowed=[
//   "example.com/shop/internal/inventory",
//   "example.com/shop/internal/payment::api",
// ])
package order
```

The exact annotation syntax must be designed in a bounded parser/definition issue. This research defines semantics, not a silent expansion of issue #11 or the current parser.

Unknown module IDs, unknown named interfaces, self-dependencies, duplicate entries, and references to internal packages must fail with source-positioned diagnostics.

### 13. Module-level cycles require their own graph analysis

The Go compiler prevents package import cycles, but logical module cycles can still exist.

Example:

```text
order/api       -> inventory/query
inventory/event -> order/model
```

No individual package cycle is required, yet aggregating packages produces `order <-> inventory`.

Spice should build the directed module graph from cross-module import edges and detect all strongly connected components with more than one module, plus any self-cycle created by incorrectly modeled nested roots.

Diagnostics should:

- report every cyclic component, not stop at the first cycle;
- use stable module IDs;
- list a deterministic canonical path through the component;
- retain the concrete package import edges that prove the cycle;
- sort independent components and edges deterministically.

### 14. Named-interface enforcement is target-package enforcement

When module A imports a package in module B:

- importing B's root package is access to B's unnamed default interface;
- importing a package explicitly assigned to named interface `x` is access to `B::x`;
- importing any other descendant package is an internal-access violation.

If A declares an outgoing allow-list:

- allowing B without a named interface allows only B's unnamed default interface;
- allowing `B::x` allows only packages assigned to that named interface;
- allowing B should not automatically allow all of B's named interfaces;
- a dependency may list both B and one or more named interfaces when required.

This keeps named interfaces meaningful and prevents broad module permission from silently exposing every secondary API.

### 15. Build context and workspaces are inherited from the loader

The module model is build-context-specific. Files and packages excluded by the active GOOS, GOARCH, tags, overlay, workspace, vendor mode, or package patterns do not participate.

The same source tree may therefore produce different valid module graphs for different build contexts. Spice should record the model inputs needed for reproducibility but should not merge multiple build contexts into one graph implicitly.

Workspace packages can participate when selected by the package patterns. Canonical IDs remain full package import paths, avoiding collisions between workspace modules with the same package name or final path segment.

### 16. Generated packages require explicit policy

A generated package inside a module root is part of that module unless it declares a separate module root, which the initial nested-root rule rejects.

A generated application bootstrap package outside all module roots remains unassigned application-shell metadata and may import module APIs.

Spice must not infer "generated" from a directory name alone. A later generation manifest may identify Spice-owned generated packages for freshness and documentation, but module membership remains import-path based.

### 17. Test variants remain out of scope

Issue #8 intentionally excludes Go test variants because they can duplicate logical package and symbol identities and introduce generated test binaries.

The initial module model is therefore the production application graph only. Module test slices require a separate design that can distinguish:

- production packages;
- in-package test variants;
- external test packages;
- generated test binaries;
- test-only dependency edges.

Spring Modulith's standalone, direct-dependencies, and all-dependencies test modes are valuable future outcomes, but they must be generated from a stable test package model rather than enabled by flipping `packages.Config.Tests` in the production loader.

### 18. Security and correctness defaults

The module analyzer should:

- fail closed on duplicate module roots, duplicate IDs, invalid named-interface placement, and unresolved allow-list entries;
- never execute application code, generators, providers, or annotation callbacks;
- use only the Go build-selected typed syntax already loaded;
- avoid following arbitrary filesystem symlinks or reading excluded files;
- preserve all independent violations rather than hiding later errors after the first one;
- cap or aggregate pathological diagnostic volume without making results order-dependent;
- never mutate `go.mod`, `go.sum`, workspace files, or source files during verification;
- avoid network access beyond whatever the caller's already-configured package load requires.

### 19. Performance target

Module discovery and import-boundary analysis should be linear in the loaded application model apart from deterministic sorting:

```text
O(packages + import specs + annotations)
```

with sorting bounded by:

```text
O(modules log modules + dependencies log dependencies + diagnostics log diagnostics)
```

It should not require SSA, call-graph construction, method-body traversal, or another typed load. The primary cost remains issue #8's single `go/packages` operation.

## Proposed compiler model

The following is conceptual and not a public API commitment:

```go
type Catalog struct {
    Modules     []Module
    Packages    []PackageMembership
    Interfaces  []NamedInterface
    Imports     []ImportDependency
    Diagnostics []Diagnostic
}

type Module struct {
    ID          string // canonical root package import path
    RootPackage string
    Position    token.Position
}

type PackageMembership struct {
    PackagePath string
    ModuleID    string // empty when unassigned
    Role        PackageRole
}

type PackageRole string

const (
    PackageUnassigned PackageRole = "unassigned"
    PackageModuleAPI  PackageRole = "module-api"
    PackageNamedAPI   PackageRole = "named-interface"
    PackageInternal   PackageRole = "internal"
)

type NamedInterface struct {
    ID          string // <module-id>::<name>
    ModuleID    string
    Name        string
    PackagePath string
    Position    token.Position
}

type ImportDependency struct {
    SourceModule  string
    SourcePackage string
    TargetModule  string
    TargetPackage string
    InterfaceID   string
    ImportKind    string
    Position      token.Position
}
```

Required invariants:

- module IDs are unique;
- module root package paths are unique;
- initial module roots do not overlap;
- every loaded package has at most one module owner;
- named-interface IDs are unique;
- every named-interface package belongs to its declaring module;
- every dependency record has one stable source import position;
- exported slices are deterministic and immutable-by-construction;
- diagnostics are deterministic and source-positioned.

## Recommended bounded delivery sequence

### Slice A — Explicit module catalog

Depends on issues #8 and #11.

- add package-target `@Module` metadata;
- discover explicit non-overlapping roots;
- derive canonical IDs from import paths;
- assign loaded packages as root API, internal descendant, or unassigned;
- expose deterministic catalog records and diagnostics;
- do not yet enforce imports, named interfaces, allow-lists, or cycles.

### Slice B — Named interfaces and incoming boundary checks

- add package-target `@NamedInterface("name")`;
- classify exposed packages;
- materialize cross-module import edges;
- reject imports into internal packages;
- include normal, alias, dot, and blank imports;
- point diagnostics at import specs.

### Slice C — Allowed dependencies and module cycles

- add first-class list or repeatable annotation support;
- preserve absent versus explicit-empty allowed dependencies;
- validate module and named-interface references;
- reject disallowed outgoing edges;
- detect all module strongly connected components;
- generate deterministic machine-readable graph output.

### Slice D — Documentation and focused module tests

- Mermaid, PlantUML, and JSON module views;
- exposed API and dependency tables;
- generated module test graphs for standalone, direct-dependency, and transitive-dependency modes;
- test-package model designed separately from production loading.

Each slice should be independently runnable and should not expand an earlier issue while it is active.

## Required test matrix for future implementation

### Module discovery

- one explicit root with root and nested packages;
- two independent modules;
- packages from two workspace modules with colliding package names;
- path-segment collision such as `order` and `orderhistory`;
- build-tag-excluded module annotation;
- unassigned command and generated-shell packages;
- duplicate annotation on one package;
- overlapping or nested module roots;
- duplicate short aliases if aliases are later added;
- deterministic catalog ordering across repeated runs.

### Boundaries

- module root API import allowed;
- same-module internal import allowed;
- cross-module internal import rejected;
- named-interface package import allowed;
- normal, alias, dot, and blank imports all materialized;
- external and standard-library imports ignored by the module graph;
- unassigned shell importing module API allowed;
- unassigned shell importing module internal rejected;
- source position identifies the exact import spec;
- independent violations accumulate deterministically.

### Allowed dependencies and cycles

- absent allow-list permits exposed APIs;
- explicit empty allow-list rejects every cross-module edge;
- exact module default-interface permission;
- exact named-interface permission;
- unknown module and named-interface references;
- duplicate and self references;
- two-module logical cycle without a Go package cycle;
- multiple independent cyclic components;
- stable cycle and edge rendering across repeated runs.

### Executable proof

A future module slice should record at least:

```text
make verify
GOPROXY=off go test -mod=vendor ./...
go tool github.com/spice-framework/toolchain/cmd/spice verify ./...
```

and dedicated fixtures proving valid modular structure, cross-module internal access failure, build-tag behavior, blank-import handling, and deterministic graph output.

## Alternatives considered and rejected

### Direct-subpackage convention as the only model

Rejected for the first implementation because Go server repositories commonly place business packages below `internal`, use multiple commands, and contain technical packages that are not modules. A convention may be optional later but should not be implicit.

### YAML-only architecture configuration

Rejected as the primary model because it duplicates package identity, drifts during refactors, does not naturally share typed annotation and generation metadata, and creates a second source of truth. Exporting or importing policy files may be useful later.

### Treat every Go package as a module

Rejected because a logical module often needs several packages to keep API, domain, infrastructure, and adapters separated. This would make the module graph too fine-grained and fail to provide Spring Modulith's functional-module outcome.

### Use only Go `internal` directories

Rejected because `internal` controls imports outside a parent subtree, not logical sibling-module boundaries inside an application. It also cannot express named interfaces, allowed dependencies, module graphs, documentation, or focused test bootstraps.

### Type-level named interfaces

Deferred because Go visibility and imports operate at package granularity. Package-level named interfaces are honest about what another package can import.

### Reflection or runtime registration

Rejected because module boundaries are compile-time source architecture. Runtime verification would delay feedback, increase startup cost, and make generated/test tooling depend on application execution.

### SSA or call-graph dependency analysis

Rejected for the initial boundary because imports already capture every direct cross-package Go dependency. SSA is more expensive and does not improve the core API/internal rule.

## Conclusion

Spice should model application modules as explicitly annotated Go package subtrees with canonical import-path identities. A module's root package is its default API, package-level named interfaces may expose additional descendant packages, and every other descendant package is internal. The dependency graph should be derived from build-selected source import declarations in the existing typed program, with module-level SCC analysis layered above Go's package-cycle checks.

This design preserves Spring Modulith's practical outcomes while remaining honest about Go's package visibility, `internal` rules, workspaces, build contexts, and test variants. It also creates one reusable module catalog for verification, documentation, generated test graphs, observability metadata, and future starter policies without introducing a reflection container or a parallel YAML architecture system.
