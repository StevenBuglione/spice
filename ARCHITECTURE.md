# Spice Architecture

## Product thesis

Spice is a Go-native enterprise application platform inspired by the useful outcomes of Spring Boot and Spring Modulith. It seeks capability parity where that improves developer productivity, but it does not attempt implementation parity with JVM mechanisms.

## Core pipeline

```text
Valid Go source
  -> Go AST and type information
  -> file-scoped annotation imports and static SDK descriptors
  -> authorized go.mod tool handlers
  -> validated typed contributions
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
- Fail-closed named, aliased, and namespace annotation imports.
- Statically decoded one-file public SDK descriptors that are never executed.
- Exact target-module `tool` authorization and offline Go module provenance.
- One persistent, bounded, version-negotiated handler process per workspace
  and tool, launched only through `go tool <full-package-path>`.
- A strictly validated typed contribution union; compiler subsystems select
  capability kinds and payloads rather than annotation package/name switches.
- Go package loading and AST inspection.
- Symbol and type resolution.
- One immutable diagnostic contract with stable codes, physical URI/ranges,
  source-mapped display positions, related information, and safe versioned
  edits.
- One instance-owned overlay compiler service with bounded content-keyed
  caching and per-workspace stale-request rejection.
- One editor-neutral stdio language server that projects the service's
  versioned diagnostics, annotation/module/configuration metadata, and safe
  edits into LSP without parsing a second application model.
- One thin repository-owned Zed adapter that resolves the caller-installed
  `spice` executable and launches that language server beside `gopls`; it owns
  no compiler metadata, source rewrite, download, or global workspace state.
- Annotation target and argument validation.

### Application model

- Components and providers.
- Dependency graph and lifecycle.
- Preferred package-main compile-time discovery, compatible typed legacy roots,
  and explicit qualified bootstrap features.
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

The preferred `@Application` is the ordinary parameterless `func main()` in
package `main`. The selected Go package scope is already bounded by the
standard package driver; Spice discovers package-documentation modules and
annotated application features within that scope, resolves exact types, and
emits their real imports and calls beside `main.go`. Multi-application
repositories use explicit target/package scope. This is compile-time analysis,
not runtime scanning, `init` registration, or implicit dependency resolution.

`@Application` supplies safe command conventions. Explicitly imported
companion annotations opt into behavior with exposure or operational
consequences. The compiler resolves and validates those annotations once,
launches only the exact handler tools authorized by the target `go.mod`,
strictly incorporates their normalized typed contributions into the immutable
application IR, and renders direct construction. Rendering does not rescan
comments. Feature activation never depends on classpath-style scanning,
package `init`, a runtime annotation lookup, or a mutable registry.

### Runtime

The runtime should stay small. Its responsibilities are application lifecycle,
generated registry execution, request scopes, shutdown, and integrations that
cannot be resolved at compile time. A generated reusable `Application` never
captures process signals. Only its explicitly invoked command helper owns
`SIGINT`/`SIGTERM`, while lower-level APIs preserve caller-owned signal and
context policy.

The `spice run` CLI is a development process boundary above those reusable
APIs. It applies guarded generation, builds only the selected package-main
target with `-trimpath` into a unique temporary artifact, and executes that
candidate without a shell. The runner isolates the child process group,
forwards interrupt/termination, waits for the generated graceful shutdown, and
preserves nonzero application exit codes. It never makes a legacy metadata
package look executable.

`spice dev` builds on a reusable supervisor with injected watcher, clock/timer,
preparation, launcher, process-stop, and event-sink boundaries. Relevant
recursive file changes are content-hashed and deterministically coalesced by a
quiet period with a maximum-delay starvation bound. Each accepted revision
runs analysis and guarded generation before compiling a unique candidate. A
failed analysis or build never stops the active process; only a complete
candidate initiates bounded process-group shutdown and replacement. Candidate
artifacts are instance-owned and idempotently released. The portable polling
watcher is the current Windows/Linux implementation and the correctness
fallback for a future native notification accelerator.

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

HTTP sessions are typed, stateless, and instance-owned. The runtime seals
bounded JSON with AES-256-GCM, authenticates the cookie identity and key ID,
and applies explicit expiry and bounded key rotation. It installs no global
session registry and does not imply server-side revocation, mutable clustering,
or CSRF protection. Applications that require those properties compose an
explicit store or request defense.

Server-side views compile from caller-owned filesystems through
`html/template`. Source paths, definitions, function maps, and output are
bounded and validated; parsing and name order are deterministic. Execution
uses a private buffer so template or cancellation failures do not partially
commit HTTP state. Contextual escaping remains mandatory unless an application
explicitly supplies a trusted-content template type.

Mail composition is transport-neutral and instance-owned. A caller supplies
the message ID, date, envelope, bodies, and attachment bytes; construction
validates strict size and injection boundaries, defensively copies the inputs,
and emits deterministic CRLF MIME without clocks, hostnames, randomness, or
network access. Bcc recipients remain in the immutable SMTP envelope and never
appear in serialized headers. Delivery is an explicit `mail.Sender` dependency,
so test and SMTP transports can own cancellation, retry, security, and
observation policy without a global client. The test adapter owns a bounded
attempt history, deterministic failure plan, defensive decoded MIME snapshots,
and payload-free observations; overflow is an explicit failure rather than
silent eviction.

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

### Constructor and interface dependency injection

`@Service`, `@Controller`, and `@Repository` are constructible bean
stereotypes. The compiler selects an explicit typed constructor, the
same-package `New<Type>` function, an unambiguous package `New`, or generated
`new(T)`, in that order. Constructor signatures use the ordinary provider
result/error/cleanup contract. The immutable provider IR records the
declaration, selected constructor, exact result type, dependencies, and
construction form; generation emits only direct calls or allocation.

Concrete providers are never projected to interfaces by scanning method sets.
`@Implements(pkg.Interface)` explicitly adds a named interface candidate after
the compiler resolves the type expression in the annotation file, verifies
the exact pointer/value method set, and finds the corresponding ordinary Go
compile-time assertion. A factory returning an interface already supplies that
exact interface and needs no binding. Graph selection uses live `go/types`
identity for exact outputs and explicit bindings, reports missing or ambiguous
candidates deterministically, and passes concrete variables directly to
interface constructor parameters in generated Go. There is no reflection,
runtime service locator, string-based type lookup, or implicit package scan.

Provider identity and selection are compile-time data. Stereotypes and
`@Bean` may declare a stable name and aliases. Repeatable `@Qualifier` metadata
on a bean forms its selection set; the same annotation on one exact
constructor parameter is a request. After exact concrete/interface candidate
collection, selection applies qualifiers, prefers non-fallback beans, accepts
a unique candidate, then a sole `@Primary`, then an exact parameter-name match
against bean names or aliases. Every ambiguity remains a source-positioned
error. Slice and string-keyed map dependencies contain every candidate ordered
by `@Order`, bean name, and source identity.

`bean.Optional[T]`, `bean.Lazy[T]`, and `bean.Provider[T]` are recognized by
exact generic package/type identity, never by spelling. Optional dependencies
allow absence but not ambiguity. Lazy dependencies resolve once in the owning
scope. A provider handle returns `(T, lifecycle.Cleanup, error)` from
`Acquire(ctx)`. Singleton cleanup belongs to the application lifecycle;
prototype cleanup belongs to the acquiring caller; request and session
cleanup belongs to an explicit typed context scope. Narrower-scoped beans can
only cross a constructor boundary through `bean.Provider[T]`. Framework-owned
controllers, lifecycle components, scheduled jobs, asynchronous tasks, and
event listeners remain singleton because those execution sites have no caller
lease to own narrower-scope cleanup.

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
