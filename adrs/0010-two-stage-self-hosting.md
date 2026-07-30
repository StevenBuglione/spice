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
`internal/autoconfigure` contributes the reviewed fallback command factory
through an explicit blank import. The generated target calls that factory
directly and exposes typed `Components` and `BeanOverrides`.

Compiler, generator, CLI implementation, and guarded-filesystem packages may
not import the production generated target. The repository bootstrap gate
audits that stage zero has no generated dependency and stage one has exactly
the `spice` generated target tree. It verifies target freshness using both
executables and retains an isolated zero-output deterministic recovery proof.

## Boundaries

- Stage zero is recovery infrastructure, not a second product implementation.
- Both commands delegate to the same `internal/cli` behavior.
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
- Any production graph change must regenerate the `Spice` target and pass both
  dependency audits.
- Parser, type checker, and renderer packages remain ordinary Go rather than
  being forced into runtime bean semantics.
