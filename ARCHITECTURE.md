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

### Package invalidation boundaries

Spice is one synchronized product module composed of independently cacheable
Go packages. Each public capability, annotation family, compiler feature,
renderer, and opt-in integration owns its smallest cohesive package boundary;
there is no aggregate runtime or starter registry. Broad facade packages keep
stable imports while implementation phases live in compiler-internal packages.

Edit-time verification obtains the actual package and test-import graph from
Go and executes the reverse dependency closure of changed source. Unknown
ownership, module inputs, and vendor inputs widen rather than under-select.
This follows [ADR 0011](adrs/0011-package-oriented-incremental-builds.md);
complete verification remains the commit contract.

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

### Core application contracts

Spring Core outcome parity is owned by cohesive public Go packages rather than
one aggregate runtime. `bean` and `lifecycle` own typed construction and
cleanup; `config` owns environments and profiles; `resource` owns explicit
instance-scoped `fs.FS` locations; `conversion` owns reflection-free typed
codecs; `validation` owns layer-neutral typed validators; `event` and `i18n`
own application-context interactions. Configuration and HTTP binding share the
same safe scalar conversion contract.

Spice deliberately does not add a runtime expression language, mutable
BeanFactory, universal pointcut proxy, or load-time weaver. Typed Go functions,
explicit constructors, generated feature decorators, static imports, and
ordinary debugger frames provide those useful outcomes without runtime symbol
lookup. The complete classification is recorded in
[`docs/spring-core-parity.md`](docs/spring-core-parity.md).

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

The development loop records a structural SHA-256 fingerprint for each
handwritten Go source after successful generation. Function and method bodies
are canonicalized out of that fingerprint; annotation comments, imports,
declarations, signatures, types, fields, and top-level values are retained.
Consequently, `spice dev` can send a body-only edit directly through Go's
incremental builder while every compiler-relevant change still rebuilds the
single immutable model and passes through guarded generation.

The preferred `@Application` is the ordinary parameterless `func main()` in
package `main`. The selected Go package scope is already bounded by the
standard package driver; Spice discovers package-documentation modules and
annotated application features within that scope, resolves exact types, and
emits their real imports and calls in an importable
`internal/spicegen/<target>` package. Multi-application repositories use
explicit target/package scope. This is compile-time analysis, not runtime
scanning, `init` registration, or implicit dependency resolution.

Generated execution is intentionally decomposed rather than collected in one
target monolith. Named files own contracts, configuration, dependency
construction, bounded assembly, optional features, HTTP coordination,
lifecycle, and process commands. Each HTTP route owns a readable, stable
symbol-and-hash-derived target file. Separately, every contributing handwritten
file owns exactly one
mirrored unit below `sources/<source-directory>/<source>_spice_gen.go`; that is
where its direct factory adapter, configuration binder, application metadata,
and interface assertions live. This gives Go debuggers truthful physical
frames and gives developers predictable source-to-generated navigation without
pretending generated execution occurred in handwritten code. Repository
reference applications enforce a 400-line ceiling on every target-level
generated unit; growth beyond that budget requires another semantic shard,
not a larger catch-all file.

The mirror preserves the source tree except for Go-reserved import-boundary
segments: `internal` and `vendor` become `internal_` and `vendor_`. Without
that encoding, a target importing its own nested adapter would violate Go's
`internal` or vendoring rules. Exact physical source paths remain in generated
headers, source mappings, and the ownership manifest.

The handwritten renderer follows the same boundary rule. Target orchestration
remains in `compiler/generate/generate.go`; source mirrors, source provenance,
components and overrides, provider construction, HTTP, configuration,
runtime/lifecycle, dependency imports, validation, model hashing, and semantic
naming each live in a focused renderer file. These files share the immutable
IR and pure buffer-writing helpers, not mutable phase state. The CLI similarly
keeps dispatch separate from module testing, module reporting, and generation
commands. A compiler failure therefore has a bounded implementation owner that
matches the generated artifact or command being debugged.

Generated identifiers are semantic first. Dependency fields and locals use
stable bean/type names, source adapters identify the constructed bean, source
package aliases identify their owning package, and route functions identify
the package, controller, and method. A short deterministic digest is retained
only where an exported cross-file identifier needs collision resistance.
Stable compiler symbol identities remain in ownership metadata and error
provenance; they are not the only human-visible debugger name.

`@Application` supplies safe command conventions. Explicitly imported
companion annotations opt into behavior with exposure or operational
consequences. The compiler resolves and validates those annotations once,
launches only the exact handler tools authorized by the target `go.mod`,
strictly incorporates their normalized typed contributions into the immutable
application IR, and renders direct construction. Rendering does not rescan
comments. Feature activation never depends on classpath-style scanning,
package `init`, a runtime annotation lookup, or a mutable registry.

### Self-hosting boundary

Spice uses a two-stage bootstrap. `cmd/spice-bootstrap` is the stage-zero
ordinary-Go compiler and imports no generated application package.
`cmd/spice` is the production stage-one application and imports only
`internal/spicegen/spice`.

The handwritten `internal/spiceapp` marker declares the production application
root and module boundary. Its explicit blank import of
`internal/autoconfigure` selects a reviewed graph containing the CLI runtime,
13 ordered `internal/cli.Handler` interface beans, and the
`*internal/cli.Command` that consumes their generated `[]Handler` collection.
Every handler has its own ordinary Go factory and mirrored source adapter.
Generation exposes every bean through typed `Components` and `BeanOverrides`;
replacing one interface bean changes the downstream command collection through
ordinary direct calls. The production entrypoint constructs, starts, invokes,
and stops that graph with caller-owned streams and a fresh bounded shutdown
context.

Spice also validates its own production module canvas. `compiler` is one module
whose supported CLI/LSP packages are explicit named interfaces; `cli`,
`devloop`, `genfs`, `lsp`, and `spiceapp` are separate modules with exact
allowed dependencies. The self-hosting package set has no cycles or unassigned
packages. Canonical `autoconfigure` packages remain compiler auxiliaries, so
their package annotations cannot silently change application module ownership.

The mandatory bootstrap proof builds stage zero offline, audits the absence of
generated dependencies, checks the committed production target, builds and
executes stage one, audits that it uses no other generated target, and then
proves zero-output deterministic recovery against an isolated application.
Compiler, CLI implementation, and filesystem-generation packages never import
stage one, so a missing or damaged production graph is recoverable.

`make dogfood` is the bounded inner-loop proof. It runs focused compiler,
generated-application, test-context, dev-loop, and LSP tests, then uses stage
zero and stage one to check the production target, inspect bean and module
models, navigate a source-to-generated mapping, and execute a focused module
test. `make verify` retains the broader release-grade gate.

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

### Release construction

Release construction is also Go-owned and offline. `cmd/spice-release`
cross-builds the ordinary CLI from the committed vendor graph with
`CGO_ENABLED=0`, `-trimpath`, and no VCS path embedding; writes deterministic
ZIP/tar metadata from the source commit epoch; derives an SPDX 2.3 dependency
document from `vendor/modules.txt`; and signs the exact archive/SBOM checksum
set with an operator-supplied Ed25519 key. It refuses an existing output path,
a dirty or incorrectly tagged release checkout, and an unsigned non-rehearsal
build. Release packaging never becomes a second dependency resolver.

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
explicitly supplies a trusted-content template type. An optional exact error
template maps only validated RFC 9457 problem fields into browser responses;
failed or canceled rendering falls back to the safe JSON problem writer.
Immutable UTF-8 properties catalogs provide bounded locale/message counts,
quality-aware `Accept-Language` matching, locale-to-default key fallback, and
explicit constructor injection without a global locale or message registry.

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

The MySQL starter constructs go-sql-driver connectors without global driver or
TLS registration. Complete URLs, verified TLS, bounded pool lifetimes, parsed
dates, context cancellation, and caller-owned cleanup are explicit. MySQL DDL
is atomic per supported InnoDB statement but implicitly commits, so consumers
must not present it as the transactional migration backend contract.
Petclinic's MySQL target instead pins one connection, holds a database-scoped
advisory lock, verifies immutable checksums, and replays only idempotent steps
before recording completion. An interrupted migration is therefore observable
and safely resumable without overstating cross-statement atomicity.

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

Built-in bootstrap features use the same qualified annotation SDK model
available to third parties. Library-owned default beans use explicit Go imports
instead of a Spice selection file: a blank-imported `.../autoconfigure` package
exposes one statically decoded `SpiceAutoConfiguration` descriptor containing
typed factory references. Candidate packages join the compiler's one typed
load as auxiliary roots, while typed primary-source inspection decides which
imports actually activate. Application exact-output beans replace defaults
before construction when an output has one default. For repeated collection
outputs, matching bean identities replace one default while distinct beans
extend the collection. Defaults whose required inputs are unavailable back
off. Generation emits ordinary direct calls with the same cleanup and rollback
contract as application beans. Descriptor bodies and factories never execute
during analysis; no `init`, runtime scan, module-presence activation, or hidden
download occurs. `spice beans --explain` exposes every selection decision and
its resolved Go module/replacement and review provenance.

The reserved `observability.http-server` feature role composes selected typed
entrypoint outputs into generated route observers. The compiler requires each
mapped output to implement the exact `web.HTTPObserver` contract, and the
renderer appends the already constructed provider before route middleware is
created. `starter/otel` uses this role for `@otel.Enable`; neither importing
OpenTelemetry nor retaining its compatibility manifest activates telemetry without the
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
the exact pointer/value method set, and plans a source-owned generated Go
compile-time assertion. Each contributing handwritten source file owns exactly
one mirrored generated source unit below
`internal/spicegen/<target>/sources/<source-directory>`. That unit contains
the direct constructor/allocation adapter, configuration binder, and explicit
interface assertions derived from that source. Target-wide lifecycle and graph
coordination remains separately identified as orchestrator code, so debugger
navigation and ownership never imply that coordination belongs to one
handwritten file. Namespace `@import` declarations can name any package
in the loaded Go module graph for these type expressions; GoLand's own index is
never a DI input. A factory returning an interface already supplies that exact
interface and needs no binding. Graph selection uses live `go/types` identity
for exact outputs and explicit bindings, reports missing or ambiguous
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
