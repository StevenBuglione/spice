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
An exact `@data.Transactional` typed HTTP route makes the dependency visible in
ordinary Go as `data.Executor`. Generated code obtains the exact
application-owned `*data.Manager`, opens the boundary around the direct method
call, and supplies only its transaction-owned executor. Calls made outside the
generated adapter remain ordinary explicit calls and cannot accidentally join
a hidden transaction.

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
Its migration backend pins the underlying pgx connection while holding a
session advisory lock, runs each parameter-free migration script and
parameterized registry insert in one transaction, and closes the physical
connection whenever unlock ownership cannot be confirmed.

Application events use immutable generic topics. `@event.Topic` marker
functions declare exact `event.Publisher[T]` provider nodes; their parameters
select exact provider-owned `@event.Listener` methods without executing either
marker or listener bodies. The compiler carries payload, module, order,
subscriber, and graph metadata in immutable IR and fails on unassigned
listeners, duplicate publishers, missing exact owners, and cycles. Direct
generation constructs one `event.Topic[T]`, binds listener method values to
already-constructed provider receivers, injects application-owned observers,
and assigns the result to the synthetic exact publisher dependency. Topic
construction failure participates in ordinary reverse provider rollback.
Delivery remains synchronous unless an explicit asynchronous or durable
adapter is injected, and there is no global event bus.

HTTP response caching is an explicit generated boundary. A qualified
`@cache.Cacheable(name="…")` declaration is valid only on a typed `GET` route
whose request DTO is an exported comparable struct value. The compiler carries
stable cache, route, module, key, and value identities in immutable IR. It
rejects write/raw/no-content routes, transaction ownership, and authorization
until those semantics can be represented in an explicit cache key. Generation
adds typed capacity/TTL properties, constructs one bounded instance-owned store,
and emits direct get/call/put control flow around the route. Caller-owned clocks
and observers remain explicit application options; handler errors are never
cached.

HTTP authorization is explicit route metadata. A qualified
`@security.Authorize` declaration is validated against a real controller
method, normalized into immutable IR with module-or-package ownership, and
rendered as direct policy/guard construction. Caller middleware owns
authentication and runs outside the generated guard; route observation remains
outermost. There is no global security context, runtime annotation lookup, or
claim-bearing diagnostic.

Fixed-delay scheduling is explicit method metadata. A qualified
`@schedule.FixedDelay` declaration must belong to exactly one provider output
and use the exact context-aware error signature. Normalized durations and
module ownership enter immutable application IR and deterministic ownership
hashes. Generation supplies direct method values to one instance-owned
scheduler, ordered after provider startup and before provider shutdown.
Application options retain caller-owned lifetime, virtual-time, and observation
seams; there is no runtime scan, global scheduler, or hidden clock.

Asynchronous entrypoints use the same explicit ownership model.
`@async.Execute` must target one exported method on exactly one provider output
with canonical context, error, and generated-nameable argument types. The
compiler derives collision-free typed submit names and carries copied task
metadata into application IR without invoking the method. Generation creates
one configured executor after provider construction, registers shutdown before
provider teardown, and emits readiness-gated typed application methods that
submit direct provider calls. Bounded execution remains instance-owned; there
is no proxy, service locator, hidden queue, or global worker pool.

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
functions; importing a starter alone has no activation semantics. Selected
constructor packages join the compiler's one typed load as auxiliary roots, so
their exported symbols can become exact provider nodes without their own
comments or package structure entering application annotation or Modulith
discovery. Explicit-constructor manifests select all declared entrypoints;
explicit-annotation manifests select only feature-mapped subsets whose
qualified annotation is present on an application marker. Generation emits
ordinary direct calls with the same cleanup and rollback contract as
application beans. Before provider analysis, dependencies declared by active
starters are checked against a bounded, read-only Go module graph snapshot with
proxy and checksum-database access disabled. Version and replacement identity
must match the manifest review exactly; inactive annotation features do not
impose dependencies and Spice never downloads them implicitly.

The reserved `observability.http-server` feature role composes selected starter
entrypoint outputs into generated route observers. The compiler requires each
mapped output to implement the exact `web.HTTPObserver` contract, and the
renderer appends the already constructed provider before route middleware is
created. `starter/otel` uses this role for `@otel.Enable`; neither importing
OpenTelemetry nor selecting its manifest activates telemetry without the
application annotation.

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

Applications may explicitly expose the same validated graph through the
generated `modules` management endpoint. Rendering emits ordinary module,
named-interface, dependency-edge, and unassigned-package literals using the
portable `spice.modules/v1` schema. The runtime validates and owns that report;
it never imports the compiler or scans packages.

## Capability parity policy

Spice tracks Spring Boot and Spring Modulith capabilities in `docs/spring-coverage.md`. Each capability must be classified as one of:

- `planned`: a Spice-native implementation is intended.
- `in-progress`: active implementation exists.
- `available`: supported and tested.
- `integration`: delegated to a mature Go library through a Spice starter.
- `not-planned`: intentionally excluded with rationale.

The target is broad practical coverage, not class-by-class porting.
