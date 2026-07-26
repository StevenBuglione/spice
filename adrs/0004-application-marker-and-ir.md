# ADR 0004: Typed Application Markers and One Immutable IR

Status: Accepted

## Decision

The preferred `@Application` marks the ordinary process entrypoint:

```go
package main

import "os"

// @Application
func main() {
    os.Exit(spiceMain(os.Args[1:]))
}
```

The preferred marker is package `main`'s parameterless, result-free,
non-generic `func main()`. Its selected local Go package scope supplies
compile-time discovery of package-documentation modules and supported annotated
application features. The generated `spiceMain` bridge is emitted into that
same package. There is no runtime scan, registration hook, reflection, or dummy
module import.

During the pre-1.0 compatibility period, a non-main package-level marker may
retain exact parameter roots:

```go
// @Application
func Commerce(server *httpapi.Server, worker jobs.Worker) {}
```

Each legacy parameter must be semantically identical under
`go/types.Identical` to one validated `@Bean` output. Aliases are accepted
because they preserve Go type identity; assignability, implicit interface
implementation, pointer/value conversion, and underlying-type equality do not
select roots.

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
4. validated package-main discovery targets or compatible exact
   provider-backed legacy roots.

The model never reloads or reparses packages, executes declaration bodies,
reflects on runtime values, or writes files. Any stage diagnostic makes the
model invalid for generation. Accessors return defensive slice and nested
metadata copies.

## Consequences

- Application source remains ordinary valid Go.
- The ordinary `main.go` expresses process ownership without framework
  assembly or manual module enumeration.
- Legacy marker functions remain compile-time root metadata only.
- Provider and application bodies cannot cause analysis side effects.
- Generator packages consume one authoritative model rather than rebuilding
  provider, graph, or lifecycle metadata.
- Missing or implicitly assignable roots fail before generation.
