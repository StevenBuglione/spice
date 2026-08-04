# ADR 0011: Package-oriented incremental builds

Status: Superseded in part by ADR 0012

## Context

Spice publishes many independently useful runtime capabilities and also owns a
large compiler, generators, editor adapters, examples, and release tooling.
One complete verification run intentionally exercises all of them, but making
that complete graph the edit-time unit makes framework iteration needlessly
slow. Splitting every capability into a separate Go module would add version,
replacement, vendor, and annotation-tool coordination without improving Go's
package-level build cache.

## Decision

Spice remains one synchronized product module. Every independently usable
runtime capability, annotation family, compiler analysis feature, renderer,
and optional integration instead owns a cohesive Go package.

The recurring dependency direction is:

```text
public capability
    <- optional starter

annotation descriptor
    -> public annotation SDK

compiler feature
    -> public contracts

generator orchestrator
    -> internal feature renderers
```

Public runtime packages never import compiler, command, generated, or starter
packages. Starters never become an aggregate ambient registry. Compiler
facades retain stable imports while broad implementations are decomposed into
internal packages.

Edit-time verification derives changed packages and their reverse dependency
closure from `go list`, including test imports and nested consumer modules.
Uncertain ownership widens the selection. The complete repository verification
gate remains authoritative.

## Consequences

Ordinary Go imports select only the libraries an application uses, and Go's
native cache can retain every unaffected package. Spice keeps one version and
one module graph while gaining independently testable invalidation boundaries.
A new Go module is justified only by an independent compatibility and release
lifecycle, not by build speed alone.

The package-oriented invalidation decision remains valid inside each module.
The single-module decision is superseded by
[ADR 0012](0012-multi-repository-product-boundaries.md) because editor
artifacts, external-service starters, reference applications, and the
compiler toolchain now have demonstrably independent dependency,
compatibility, release, and verification lifecycles.
