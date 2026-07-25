# ADR 0008: Import-path application modules

Status: Accepted

## Context

Go already gives every package a stable identity, import rules, and an
inspectable dependency graph. Spice needs Modulith-style boundaries without
inventing a second naming system or requiring runtime package scanning.

## Decision

A package-documentation `@Module` annotation makes that package the root of an
application module. The module ID is the root package's full Go import path.
The root package is the default public API. Descendant packages belong to the
longest matching module root and are internal unless their package
documentation declares a named interface.

`@NamedInterface("name")` is repeatable. Names use portable lowercase
identities matching `^[a-z][a-z0-9-]*$`. A named interface exposes its entire
package; declaration-level export sets are outside this contract.

`@Module` accepts an optional list:

```go
// @Module(allowedDependencies=["example.com/shop/inventory", "example.com/shop/payments::spi"])
```

The source syntax places one annotation invocation on one comment line. A plain
import path selects the target module's root-package API. `module::name`
selects a named interface.
References are exact: short names and suffix matching are not accepted.

Packages in a Go module that contains selected `@Module` roots but are not
descendants of any root are reported as unassigned. Nested module roots are
allowed and take ownership of their own descendants.

## Boundaries

Discovery consumes the existing loaded program and resolved annotations. It
does not reload source, execute code, inspect the filesystem, or infer modules
from directory names alone.

Every selected Go import between different discovered modules becomes one
distinct architecture edge. Imports of another module's root API or named
interface require an exact allowed-dependency entry. Other descendant imports
are rejected as internal. Strongly connected module components produce stable
member sets and representative closed paths.

`spice modules --format=json|mermaid|plantuml` renders this model without
writing files. JSON contains complete portable canvases; diagram formats use
stable synthetic node IDs and aggregate repeated package imports by module API.
`--focus=<module-import-path>` retains the selected module and its transitively
observed dependencies, excluding dependents, unrelated modules, unassigned
packages, and allowed-but-unused dependencies. JSON records dependency-first
composition order, and diagrams highlight the selected module.

## Consequences

- Module identities remain unique wherever Go import paths are unique.
- Refactors across module roots are explicit dependency-contract changes.
- Root packages are intentionally small public facades.
- Descendant packages are fail-closed by default.
- Unassigned packages remain visible rather than silently becoming modules.
