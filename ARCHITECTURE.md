# Spice Architecture

## Product thesis

Spice is a Go-native enterprise application platform inspired by the useful outcomes of Spring Boot and Spring Modulith. It seeks capability parity where that improves developer productivity, but it does not attempt implementation parity with JVM mechanisms.

## Core pipeline

```text
Valid Go source
  -> Go AST and type information
  -> Spice annotation parser
  -> typed Spice intermediate representation
  -> typed application-bootstrap feature metadata
  -> module and dependency graphs
  -> static validation
  -> deterministic generated Go
  -> standard Go compiler
```

## Non-negotiable rules

1. Application source remains valid Go.
2. Generated behavior is ordinary, inspectable Go.
3. Constructor injection is preferred over field injection.
4. Compile-time generation is preferred over reflection-heavy runtime discovery.
5. Interfaces and explicit wrappers replace subclass proxies.
6. Architecture enforcement is a first-class product feature.
7. Each feature must include executable tests and an observable runnable path.
8. The standard library remains the default dependency choice.
9. External dependencies require an explicit rationale.
10. No implementation may claim success without running the repository verification command.

## Major subsystems

### Compiler front end

- Annotation lexical and syntactic parser.
- Go package loading and AST inspection.
- Symbol and type resolution.
- Source-positioned diagnostics.
- Annotation target and argument validation.

### Application model

- Components and providers.
- Dependency graph and lifecycle.
- Application roots and explicit qualified bootstrap features.
- Application modules and named interfaces.
- Configuration ownership.
- Routes, event contracts, and transaction boundaries.

### Code generation

- Dependency wiring.
- Command bootstrap and explicit feature composition.
- HTTP adapters.
- Cross-cutting decorators.
- Configuration binders.
- Metadata and documentation.

Generation is split into a pure in-memory plan and a guarded filesystem
application. The pure renderer consumes the immutable application IR, emits
canonical target-scoped Go plus SHA-256 ownership metadata, and performs no
filesystem or network mutation.

`@Application` supplies safe command conventions. Qualified companion
annotations opt into behavior with exposure or operational consequences. The
compiler resolves and validates those annotations once, carries normalized
typed metadata in the immutable application IR, and renders direct
construction. Rendering does not rescan comments. Feature activation never
depends on classpath-style scanning, `go.mod`, `init`, or a mutable registry.

### Runtime

The runtime should stay small. Its responsibilities are application lifecycle,
generated registry execution, request scopes, shutdown, and integrations that
cannot be resolved at compile time. A generated reusable `Application` never
captures process signals. Only its explicitly invoked command helper owns
`SIGINT`/`SIGTERM`, while lower-level APIs preserve caller-owned signal and
context policy.

SQL access remains based on `database/sql`. Repositories accept the common
executor contract implemented by both pools and transactions. Instance-owned
transaction managers retain commit/rollback ownership and consume
compiler-generated boundary and module identities; there is no ambient
transaction, global pool, or retry hidden in a context.

Repository queries are immutable, typed definitions with stable module and
operation identities, explicit dialect SQL, caller-supplied row decoders, and
mandatory list bounds. Single-result cardinality and row lifecycle errors fail
closed without logging SQL or argument values.

Database migrations form one immutable application-global version sequence
while retaining module ownership. Core normalizes and checksums SQL, reconciles
the durable registry as an exact plan prefix, and delegates advisory locking,
transactional DDL policy, and atomic registry writes to explicit dialect
backends.

The PostgreSQL starter adapts pgx to the standard SQL contracts. Applications
provide complete URLs, own pool lifetimes, and explicitly ping during startup;
TLS hostname verification is the default and construction never connects.

Application events use immutable generic topics assembled by generated code.
Payloads retain exact Go types, subscriber order is stable, and delivery is
synchronous unless an explicit asynchronous or durable adapter is injected.
Publisher/subscriber module identities feed architecture-aware observations
without a global event bus.

### Starters

Third-party integrations live under `starter/` and remain opt-in at the package
boundary. Each dependency requires a recorded maintenance, license, security,
cancellation, observability, and configuration review. Starters accept
caller-owned clients/providers, install no global state, and must not make
network calls during construction unless their documented contract explicitly
requires it.

Built-in bootstrap features use the same qualified, typed definition model
available to the public starter manifest SDK. Manifests provide strict,
deterministic compatibility, dependency, entrypoint, annotation, feature, and
review metadata without registering global behavior. An application-owned
`.spice/starters.json` document explicitly selects embedded manifests for CLI
and compiler composition. Spice does not scan dependencies or execute manifest
functions; importing a starter alone has no activation semantics, and generated
entrypoint selection remains separate.

Outbound OAuth2 service clients receive separate caller-owned token and
resource clients plus an application-lifetime context. Token endpoints are
HTTPS-only, bounded, and non-redirecting; provider failures cross the starter
boundary only as safe cancellation-aware classes.

### Verification

`spice verify` will eventually enforce:

- Valid annotation syntax and targets.
- No unresolved dependency providers.
- No dependency cycles.
- No module-level cycles.
- No access to another module's internals.
- Only declared module dependencies.
- Valid route and event contracts.
- Valid transaction and configuration ownership.
- Generated-code freshness.

Focused module testing consumes the same validated Modulith model.
`spice test --module=<full-import-path>` selects the focused module plus only
its transitively observed dependencies and invokes ordinary `go test` for
their owned packages in dependency-first order. It creates no alternate
container or runtime discovery path.

## Capability parity policy

Spice tracks Spring Boot and Spring Modulith capabilities in `docs/spring-coverage.md`. Each capability must be classified as one of:

- `planned`: a Spice-native implementation is intended.
- `in-progress`: active implementation exists.
- `available`: supported and tested.
- `integration`: delegated to a mature Go library through a Spice starter.
- `not-planned`: intentionally excluded with rationale.

The target is broad practical coverage, not class-by-class porting.
