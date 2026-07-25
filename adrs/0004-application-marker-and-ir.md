# ADR 0004: Typed Application Markers and One Immutable IR

Status: Accepted

## Decision

`@Application` marks an argument-free package-level function declaration. The
function's exact parameter types identify application roots:

```go
// @Application
func Commerce(server *httpapi.Server, worker jobs.Worker) {
    panic("compile-time marker; never executed by Spice")
}
```

The marker must be non-generic and non-variadic and return no results. Each
parameter must be semantically identical under `go/types.Identical` to one
validated `@Bean` output. Aliases are accepted because they preserve Go type
identity; assignability, implicit interface implementation, pointer/value
conversion, and underlying-type equality do not select roots.

Verification may analyze a package set with zero or multiple application
markers. Each marker has a stable symbol identity and becomes one generation
target. A later generation command must select an unambiguous target before
writing files.

## Application model

`compiler/application` is the single immutable-by-convention compiler boundary
for generation. It consumes the existing loaded program and resolved
annotations, then assembles:

1. the validated provider catalog and cleanup flags;
2. the exact provider graph and dependency-first construction order;
3. lifecycle components ordered by provider construction order;
4. validated application targets and exact provider-backed roots.

The model never reloads or reparses packages, executes declaration bodies,
reflects on runtime values, or writes files. Any stage diagnostic makes the
model invalid for generation. Accessors return defensive slice and nested
metadata copies.

## Consequences

- Application source remains ordinary valid Go.
- Marker functions exist only to express typed compile-time roots.
- Provider and application bodies cannot cause analysis side effects.
- Generator packages consume one authoritative model rather than rebuilding
  provider, graph, or lifecycle metadata.
- Missing or implicitly assignable roots fail before generation.
