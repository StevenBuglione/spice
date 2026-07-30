# ADR 0010: Two-stage recoverable self-hosting

Status: Accepted

## Context

Spice should use its own application model for Spice-owned applications and
libraries. Making the only compiler executable depend on generated output,
however, would create a bootstrap cycle: a missing or damaged generated target
could require that same target in order to regenerate it.

The production command also needs to prove real framework value rather than
wrapping its existing dispatch behind a cosmetic annotation. It must exercise
typed construction, imported library defaults, module validation, lifecycle,
generated source mapping, and typed test replacement.

## Decision

Spice uses two executable stages:

- `cmd/spice-bootstrap` is an ordinary-Go recovery compiler. It imports
  `internal/cli` and compiler packages but no generated application package.
- `cmd/spice` is the production application. It imports the committed
  `internal/spicegen/spice` target, constructs and starts it, invokes its typed
  `Command` component, and stops it with a fresh bounded context.

`internal/spiceapp` owns the application marker and module declaration.
`internal/autoconfigure` contributes the reviewed fallback runtime, 13
independent CLI handler factories, and the command factory through an explicit
blank import. The generated target constructs each handler as a distinct
interface bean, injects the ordered `[]cli.Handler` collection into the
command, and exposes every node through typed `Components` and
`BeanOverrides`.

The compiler, CLI, development loop, guarded generator filesystem, LSP, and
application marker declare an executable Modulith canvas. Compiler packages
consumed across a module boundary are explicit named interfaces. Canonical
auto-configuration descriptors stay auxiliary and cannot contribute module
metadata.

Compiler, generator, CLI implementation, and guarded-filesystem packages may
not import the production generated target. The repository bootstrap gate
audits that stage zero has no generated dependency and stage one has exactly
the `spice` generated target tree. It verifies target freshness using both
executables and retains an isolated zero-output deterministic recovery proof.

## Boundaries

- Stage zero is recovery infrastructure, not a second product implementation.
- Both commands delegate to the same `internal/cli` behavior.
- Stage zero assembles the same exported runtime and handler factories
  manually; stage one obtains them through generated dependency injection.
- Generated construction performs no reflection, runtime package scan, global
  registration, or string-based lookup.
- The production manifest and generated Go are committed and guarded by normal
  ownership checks.
- Missing production output must always be recoverable with stage zero and the
  repository vendor graph while network access is disabled.

## Consequences

- Spice's shipped command exercises the same generated application contract
  offered to users.
- Self-hosting cannot strand the compiler behind its own generated output.
- CLI application tests use exact typed overrides and normal lifecycle cleanup.
- A typed handler override must flow into the generated interface collection,
  and production LSP/configuration behavior must execute through that graph.
- Any production graph change must regenerate the `Spice` target and pass both
  dependency audits.
- Parser, type checker, and renderer packages remain ordinary Go rather than
  being forced into runtime bean semantics.
