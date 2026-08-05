# Spice Framework for Go

> **Ecosystem migration:** Spice is moving to independently versioned core,
> toolchain, editor, starter, and reference-application repositories under
> [`github.com/spice-framework`](https://github.com/spice-framework). The
> history-preserving topology and acceptance ledger are documented in
> [ADR 0012](adrs/0012-multi-repository-product-boundaries.md) and the
> [repository migration ledger](docs/repository-migration.md). Until the first
> signed preview is published, the project remains pre-alpha.

Spice is an opinionated, compile-time application platform for Go. Its goal is to bring the breadth, productivity, and operational completeness associated with Spring Boot together with Spring Modulith-style architectural enforcement—without importing JVM runtime magic into Go.

## Product direction

Spice is designed around five commitments:

1. **Broad application-platform coverage.** The roadmap intentionally covers web APIs, configuration, dependency injection, validation, security, data access, transactions, messaging, scheduling, observability, testing, and modular architecture.
2. **Excellent developer ergonomics.** Common application behavior should be easy to express, errors should point to source, generated behavior should be inspectable, and the happy path should be obvious.
3. **Valid Go source.** Spice annotations are ordinary Go comments such as `// @Controller(prefix="/users")`, so standard Go tools continue to parse the project.
4. **Compile-time enforcement.** Wiring, annotation validation, and module rules should fail before deployment whenever possible.
5. **Runnable software, not paper architecture.** Every implementation change must compile, execute its relevant smoke path, and pass tests before it is considered complete.

## Current foundation

The repository currently provides:

- A typed Go package-loading pipeline with stable declaration identities.
- Annotation parsing, resolution, and source-positioned validation.
- Exact concrete bean/configuration providers, constructible service/controller/
  repository stereotypes, and explicit `@Implements` interface bindings with
  deterministic qualifiers, primary/fallback selection, ordered collection
  injection, typed optional/lazy/provider handles, and explicit
  singleton/prototype/request/session ownership.
- Typed provider cleanup and `@OnStart`/`@OnStop` lifecycle metadata with a race-safe rollback and shutdown coordinator.
- A preferred annotated package-main `func main()` that discovers the selected
  local module scope at compile time, plus a pre-1.0 compatible exact-type
  parameter-root marker form, both assembled with provider, lifecycle, and
  typed bootstrap-feature data in one immutable application IR.
- Annotation-driven generated commands with conventional environment
  configuration, structured command logging, explicit management/logging
  companions, redacted configuration reporting, stable exit codes, signal
  ownership, and bounded graceful shutdown.
- A pure deterministic renderer for direct provider/lifecycle calls and SHA-256 ownership manifests.
- Guarded generated-file ownership with manual-edit refusal, freshness checks, bounded diffs, and unchanged-file preservation.
- Import-path application modules with root APIs, named interfaces, explicit dependencies, internal-boundary checks, unassigned-package reporting, and deterministic cycle detection.
- Module-aware synchronous lifecycle observations that generated applications expose without a global tracer or telemetry dependency.
- Reflection-free typed configuration declarations, exact provider injection, generated schema/binders, and a runtime with rooted JSON/profile files, explicit precedence, provenance, environment mapping, defaults, validation, and secret redaction.
- Standard-library SQL transaction management with repository-friendly executors, module-owned boundary metadata, rollback-safe error/panic behavior, synchronous observations, and generated `@data.Transactional` typed HTTP boundaries.
- Immutable reflection-free repository queries with explicit SQL, typed row decoders, exact single-result cardinality, bounded lists, and safe failures.
- An opt-in pgx PostgreSQL starter with secure URL/TLS policy, explicit pool ownership, and real-container transaction, repository, and migration verification.
- An opt-in go-redis starter with secure URL/TLS/authentication policy,
  deterministic bounded pool ownership, exact cleanup, and a namespaced typed
  JSON cache store verified against a real Redis server.
- An instance-owned secure SMTP starter with required verified TLS, bounded
  cancellation/retry, duplicate-safe failure handling, and payload-free
  observations.
- Deterministic module-owned migration plans with global version ordering, normalized SHA-256 checksums, registry drift detection, and a concrete PostgreSQL advisory-lock/transaction backend.
- Immutable generic event topics with exact payload types, deterministic
  subscriber order, cancellation/failure semantics, module-interaction
  observations, and compile-time `@event.Topic`/`@event.Listener` graph
  metadata rendered as direct, rollback-safe topic construction.
- A transactional outbox with immutable bounded messages, a driver-neutral SQL store, atomic enqueue/lease contracts, at-least-once dispatch, explicit failure delay, and payload-free observations.
- Immutable bounded external-message envelopes plus explicit publisher,
  handler, and acknowledgement/retry/reject settlement contracts for isolated
  broker starters.
- An opt-in franz-go Kafka producer and sequential consumer group with verified
  TLS/authentication defaults, explicit settlement, bounded polling, lifecycle
  ownership, and payload-free observations.
- An opt-in grpc-go server/client starter with TLS-by-default transport,
  bounded messages and streams, ordinary generated-protobuf registration,
  standard health, graceful lifecycle drain, and payload-free observations.
- An opt-in RFC 6455 WebSocket server/client starter with same-origin and
  TLS-by-default policy, bounded messages/connections/close, explicit
  subprotocols and compression, and payload-free session observations.
- Explicit bounded retries with opt-in error classification, capped deterministic backoff, cancellation, typed exhaustion, and attempt observations.
- Generic cache contracts, a bounded in-memory LRU/TTL cache, and compile-time
  `@cache.Cacheable` typed-read generation with configured capacity/TTL,
  deterministic keys, safe route validation, and bounded observations.
- Compile-time `@async.Execute` tasks with readiness-gated typed generated
  submit methods, direct provider calls, configured bounded admission,
  caller-owned lifetime contexts and observers, deterministic failure
  aggregation, panic containment, snapshots, and lifecycle-owned shutdown.
- Compile-time `@schedule.FixedDelay` jobs with exact provider ownership, direct generated method calls, non-overlap, explicit failure continuation, graceful drain, panic containment, observations, and virtual-time test seams.
- Restartable ordered batch jobs with atomic attempt/checkpoint contracts, exact
  completed-prefix validation, fresh failure contexts, panic containment,
  bounded observations, a concurrency-safe capacity-bounded in-process store,
  a driver-neutral lease-aware SQL persistence protocol, and a real-container
  verified PostgreSQL backend.
- An explicitly selected `@otel.Enable` OpenTelemetry v1.44 HTTP trace/metric
  starter with exact generated observer-role validation, plus explicit
  payload-free module-event interaction spans and metrics, with
  application-owned providers/exporters.
- Immutable authenticated principals plus compile-time `@security.Authorize`
  route policies that generate deny-by-default RFC 9457 guards, stable
  module/policy identities, and bounded authorization observations.
- An opt-in OIDC JWT resource server with strict bearer parsing, signature/issuer/audience/expiry verification, exact claim mapping, required or route-guard-compatible optional authentication, bounded discovery/JWK transport, and safe authentication failures.
- An opt-in OAuth2 client-credentials integration with separate timed transports, HTTPS-only bounded token acquisition, safe failures, and cached Bearer authorization.
- Typed stateless HTTP sessions with AES-256-GCM confidentiality/integrity,
  bounded key rotation, embedded expiry, strict decoding, secure host-only
  cookie defaults, and concurrent-use verification.
- Deterministic server-side HTML templates with contextual escaping, strict
  missing-key execution, duplicate-definition rejection, bounded atomic
  responses, cancellation, concurrent rendering, and generated form/view
  adapters with immutable binding results and safe local 303 redirects.
- Immutable transport-neutral mail messages with caller-owned identity and
  time, stable envelope recipients, Bcc-safe deterministic MIME, text/HTML
  alternatives, bounded attachments, defensive copies, and no hidden network
  client.
- An instance-owned `mail/mailtest` sender with bounded immutable attempts,
  deterministic injected failures, explicit overflow, payload-free
  observations, concurrent inspection, and typed MIME snapshots.
- A strict HTTP runtime with RFC 9457 problems, secure error mapping, bounded JSON and URL-encoded decoding, JSON/HTML negotiation, safe scalar binding, and explicit no-content responses.
- Typed controller/route compilation and deterministic generated `net/http` adapters with exact receiver/mux/renderer providers, request DTO and form binding, closed HTML/redirect outcomes, RFC 9457 errors, ServeMux wildcard checks, and raw escape hatches.
- A runnable `spice` CLI with `version`, `annotations`, `verify`, `modules`,
  `generate`, `build`, `run`, last-known-good `dev`, and editor-neutral `lsp`
  commands.
- A repository-owned GoLand plugin that renders canonical declaration comments
  as zero-width-prefix annotations, applies configurable native syntax colors,
  resolves highlighted PSI references to their real Go SDK descriptor
  declarations, shows descriptor provenance in the gutter, runs applications
  through `spice run`, generates before native GoLand package debugging,
  checks light/dark rendering against committed visual goldens, and launches
  the same LSP for descriptor documentation, handler implementation
  navigation, completion, diagnostics, safe edits, and cancellable confirmed
  hash-guarded `go get -tool` preview/apply. Interface completion and bean
  selection come from Spice's typed compiler catalog; the plugin uses GoLand's
  index only to navigate and generate methods for symbols already in source.
- A supported secondary Zed extension that launches the same LSP beside
  `gopls` for completion, diagnostics, hover, modifier-click annotation
  navigation, safe quick fixes, module/configuration metadata, and structured
  valid-Go annotation highlighting on Windows and Linux.
- A generated-application HTTP test slice with loopback-only serving, bounded
  detached responses, strict JSON/problem decoding, construction rollback,
  and idempotent lifecycle cleanup, plus transaction-scoped generic SQL
  subjects that always roll back.
- A committed generated HTTP application with real provider, lifecycle, route, and graceful-drain tests.
- An independent, consumer-owned Spring Petclinic port whose `go.mod`
  authorizes the annotation tool and whose generated source, manifest, vendor
  tree, tests, and executable workflow are verified outside the framework
  module.
- A cross-platform Go-owned quality gate with pinned format, lint, nil-safety, security, vulnerability, race, fuzz, coverage, offline-vendor, and executable checks.
- Product, architecture, annotation, and Spring-coverage documents.

## Annotation syntax

Annotations are valid declaration comments with explicit file-scoped imports:

```go
// @import { Controller } from "github.com/StevenBuglione/spice/annotation/web"

// @Controller(prefix="/users")
type UserController struct{}
```

Named imports keep common annotations clean, aliases resolve local collisions,
and namespace imports keep provenance visible:

```go
// @import { Get as GET } from "github.com/StevenBuglione/spice/annotation/web"
// @import * as security from "github.com/StevenBuglione/spice/annotation/security"

// @GET("/orders/{id}")
// @security.Authorize(anyRoles=["admin"], allScopes=["orders:write"])
func (*Controller) Get(context.Context, Request) (Response, error)
```

The application root `go.mod` authorizes annotation handlers through standard
Go tool dependencies:

```go
tool github.com/StevenBuglione/spice/cmd/spice-annotation-core
```

Spice statically decodes each one-file Go descriptor and launches only its
authorized full package path through `go tool`; there is no plugin manifest or
custom dependency resolver. Editor installation assistance uses the standard Go
command against a temporary modfile, shows the exact `go.mod`/`go.sum` diff,
and requires a separate confirmed action before applying the still-current
preview.

`spice new` creates a minimal valid-Go application and exact Go module without
downloads or destructive overwrite. `spice add package@version` and
`spice add --tool package@version` expose the same temporary-modfile planner to
developers: preview is the default, and `--apply` writes only a freshly
validated hash-guarded plan. The clean-room smoke gate generates from zero
output, vendors offline, compiles, tests, builds, and executes that scaffold.

The independent modules under
[`testdata/annotationfixture`](testdata/annotationfixture) and
[`testdata/annotationapp`](testdata/annotationapp) prove that a third party can
use only the public SDK/protocol to supply aliases, namespaces, diagnostics,
real editor navigation, provider semantics, and inspectable generated Go.

### Explicit bean selection and scopes

Spice never treats general Go assignability as dependency discovery. A
concrete result becomes an interface candidate only through `@Implements` and
the compiler's exact `go/types` verification, while a factory returning the
exact interface type is already an interface bean. Spice writes the matching
Go compile-time assertion into a manifest-owned source shard beside the
implementation; handwritten source stays focused on behavior.

```go
// @import { Implements, Primary, Qualifier, Service } from "github.com/StevenBuglione/spice/annotation/core"
// @import * as payments from "example.com/commerce/payments"

// @Service(name="stripeProcessor", aliases=["payments"])
// @Implements(payments.Processor)
// @Qualifier("stripe")
// @Primary
type StripeProcessor struct{}

func NewCheckout(
	// @Qualifier("stripe")
	processor payments.Processor,
) *Checkout
```

Single values first apply requested qualifiers, ignore fallback beans when a
regular candidate exists, select a unique candidate or primary, and finally
use an exact parameter-name match against bean names and aliases. Ambiguity is
a compile error with candidate locations. `[]T` and `map[string]T` inject all
matching beans in `@Order`, bean-name, and source order.

The generic `bean.Optional[T]`, `bean.Lazy[T]`, and `bean.Provider[T]` contracts
remain ordinary typed Go. `Optional` permits absence but never ambiguity.
`Lazy` resolves a singleton once. `Provider.Acquire(ctx)` returns the exact
value, an idempotent `lifecycle.Cleanup`, and an error. Prototype cleanup is
caller-owned; request and session beans are cached in an explicit typed
`bean.Scope` attached to context. The generated application installs request
scope middleware when required. Session scopes are deliberately created and
closed by application code—there is no global session registry.

## Run it

Install Go 1.26.5 and GNU Make, then run:

New users should begin with the executable
[`getting-started.md`](docs/getting-started.md) walkthrough. Spring developers
can use [`spring-to-spice.md`](docs/spring-to-spice.md) as a concept and
migration map.

```bash
make fast
make check
go run ./cmd/spice version
go run ./cmd/spice verify ./...
cd examples/commerce
spice annotations list ./...
spice annotations doctor ./...
spice verify --format=json ./...
spice test --module github.com/StevenBuglione/spice/examples/commerce/orders --count=1 ./...
spice generate --check --target Commerce .
spice run --target Commerce . -- -check
spice dev --target Commerce .
```

Use `make fast` for affected-package feedback and `make check` for the broader
edit loop. Run the complete `make verify` on the
exact tree before committing; release automation additionally runs
`make verify-release` and the benchmark budgets.

In an application module containing one typed `@Application` marker:

```bash
spice generate ./...
spice generate --check ./...
spice generate --diff ./...
spice build ./...
spice run ./... -- -check
spice dev ./...
```

Application-platform conventions live on the ordinary process entrypoint and
compile into a directly imported generated target package:

```go
package main

import (
    "os"

    spiceapp "github.com/StevenBuglione/spice/examples/commerce/internal/spicegen/commerce"
)

// @import { Application } from "github.com/StevenBuglione/spice/annotation/core"
// @import { Enable } from "github.com/StevenBuglione/spice/annotation/management"
// @import { Logging } from "github.com/StevenBuglione/spice/annotation/observability"

// @Application
// @Enable(expose=["health", "liveness", "readiness", "info", "metrics", "configprops", "modules"])
// @Logging
func main() {
    os.Exit(spiceapp.Main(os.Args[1:]))
}
```

The generated `Main` returns an exit code and does not call `os.Exit`. It resolves
the generated schema from the `SPICE_` environment convention, logs command
startup and failures, owns `SIGINT`/`SIGTERM`, and creates a fresh bounded
shutdown context. `spice.shutdown-timeout` defaults to `10s` and can be set
with `SPICE_SHUTDOWN_TIMEOUT`.

Controller targets also own `artifacts/openapi.json` in the internal generated
package; generation check/diff verifies it alongside the orchestrator and
source-owned units.

Production services opt into only the management routes they intend to expose
with `@management.Enable(expose=[...])`. The endpoint allowlist is exact and
validated at compile time; package presence or a `go.mod` dependency never
activates it. See
[`docs/management.md`](docs/management.md).

The preferred annotated `main.go`, compile-time discovery scope, explicit
generated-package boundary, and legacy migration contract are documented in
[`docs/application.md`](docs/application.md).

The two-stage self-hosting boundary, production generated CLI graph, and
zero-output regeneration/recovery proof are documented in
[`docs/dogfooding-readiness.md`](docs/dogfooding-readiness.md).

Pre-1.0 module, generated-source, tool, and editor upgrade procedures are
documented in [`docs/upgrading.md`](docs/upgrading.md).

Explicit imports, static descriptors, `go.mod` tool authorization, typed
contributions, offline behavior, and extension security are documented in
[`docs/annotation-sdk.md`](docs/annotation-sdk.md).

Stable text/JSON diagnostic codes, physical and source-mapped ranges, related
information, and version-aware safe edit contracts are documented in
[`docs/diagnostics.md`](docs/diagnostics.md).

The isolated overlay-aware analysis API shared by generation, run, dev, and
editor clients is documented in
[`docs/compiler-service.md`](docs/compiler-service.md).

The stdio language server, versioned diagnostics, annotation/configuration
completion, hover, safe code actions, and workspace settings are documented in
[`docs/lsp.md`](docs/lsp.md).

The primary GoLand plugin, exact prefix concealment, native color settings,
PSI navigation, language-server setup, installable archive, and repeatable
light/Darcula visual acceptance path are documented in
[`docs/goland.md`](docs/goland.md).

The supported secondary Zed extension, PATH/settings setup, modifier-click
definition navigation, semantic annotation presentation, supported-API
limitation, and diagnostic fixture are documented in
[`docs/zed.md`](docs/zed.md).

The recursive watcher, deterministic debounce policy, unique candidate builds,
last-known-good recovery, and graceful restart controls used by `spice dev` are
documented in [`docs/development-loop.md`](docs/development-loop.md).

The executable foundation and the remaining reference-application integration
debt are tracked explicitly in
[`docs/stable-core-acceptance.md`](docs/stable-core-acceptance.md).

Outbound integrations can use the base-scoped, bounded typed JSON client in
[`docs/http-client.md`](docs/http-client.md).

SQL repositories and generated `@data.Transactional` HTTP boundaries use the
explicit contracts in [`docs/data.md`](docs/data.md).

Typed in-process event contracts are documented in
[`docs/events.md`](docs/events.md).

Context-aware resilience policies are documented in
[`docs/retry.md`](docs/retry.md).

Typed caching and the built-in bounded store are documented in
[`docs/cache.md`](docs/cache.md).

Secure Redis client ownership and distributed typed caching are documented in
[`docs/redis.md`](docs/redis.md).

Bounded asynchronous task execution is documented in
[`docs/async.md`](docs/async.md).

Fixed-delay job registration and lifecycle are documented in
[`docs/schedule.md`](docs/schedule.md).

Restartable batch jobs and persistence contracts are documented in
[`docs/batch.md`](docs/batch.md).

Authentication boundaries and generated authorization policies are documented
in [`docs/security.md`](docs/security.md).

OIDC JWT resource-server integration is documented in
[`docs/oidc-resource-server.md`](docs/oidc-resource-server.md).

OAuth2 service-client integration is documented in
[`docs/oauth2-client.md`](docs/oauth2-client.md).

Transactional outbox storage and dispatch semantics are documented in
[`docs/outbox.md`](docs/outbox.md).

Transport-neutral external messaging contracts are documented in
[`docs/messaging.md`](docs/messaging.md).

Module-owned database migration planning is documented in
[`docs/migrations.md`](docs/migrations.md).

PostgreSQL pool configuration and integration testing are documented in
[`docs/postgres.md`](docs/postgres.md).

Portable starter compatibility metadata and qualified annotation definitions
are documented in [`docs/starters.md`](docs/starters.md).

Focused module execution and generated HTTP test slices are documented in
[`docs/testing.md`](docs/testing.md).

For a repository containing package-level `@Module` roots:

```bash
spice modules --format=json ./...
spice modules --format=mermaid ./...
spice modules --format=plantuml ./...
spice modules --focus=example.com/shop/orders --format=json ./...
spice test --module=example.com/shop/orders --race --count=1 ./...
```

JSON contains complete portable module canvases. Mermaid and PlantUML aggregate
the same verified package-import edges into deterministic module diagrams.
`--focus` retains one module and only its transitively observed dependencies,
with dependency-first composition order for module test slices.
`spice test` validates that same graph and invokes ordinary `go test -trimpath`
for exactly its owned packages, excluding unrelated and unassigned packages.
See [`docs/testing.md`](docs/testing.md).

Use `--target Name`, the command import path, or the stable marker symbol ID
when the selected packages contain multiple application markers. Positional Go
package patterns provide explicit compile-time scope in a multi-application
monorepo; an ordinary single-application module needs no dummy imports or
module list. Package-main generation writes an importable
`internal/spicegen/<target>` package split into contracts, configuration,
providers, bounded assembly, optional features, HTTP coordination, readable
stable per-route files, lifecycle, and command behavior. It mirrors each
contributing handwritten file to one nested
`sources/.../<source>_spice_gen.go` unit. No generated Go is written beside
handwritten code, and there is no catch-all target file. Every concern role,
source origin, and generated range is recorded in the schema-5
`.spice/<target>.manifest.json`; `spice generated` queries that mapping in
either direction.

To start the example HTTP server:

```bash
cd examples/commerce
spice run --target Commerce .
curl -H "Content-Type: application/json" -d "{\"quantity\":2}" http://localhost:8081/orders
curl http://localhost:8081/actuator/health/readiness
curl http://localhost:8081/actuator/metrics
curl http://localhost:8081/actuator/configprops
curl http://localhost:8081/actuator/modules
```

The modular commerce `main.go` enables structured request/lifecycle logging
and exactly seven management endpoints. Its generated command owns
`SIGINT`/`SIGTERM`, conventional environment loading, check mode, stable exit
codes, and fresh bounded shutdown. Its generated application also owns the
fixed-delay audit and exposes a typed, bounded asynchronous inventory
verification method that drains before provider cleanup. The generated
`Application` itself never captures process signals. Generated source and
OpenAPI are committed under `examples/commerce/internal/spicegen/commerce`;
source-owned
application metadata, configuration binders, constructors, and interface
checks use mirrored files below
`examples/commerce/internal/spicegen/commerce/sources`. The matching ownership
manifest is `examples/commerce/.spice/commerce.manifest.json`. Commerce is a
separate consuming Go module with its own module graph and vendor tree.

For embedding and specialized policies, the generated application retains
`NewApplication`, `NewApplicationWithOptions`, `Application.Start`,
`Application.Stop`, `Application.Run`, `Application.Components`, and
`RunCommand`. `Components` is a typed singleton snapshot for tests and
embedding—not a runtime lookup container. These seams support caller-owned
contexts, signals, configuration sources, middleware, error mapping,
lifecycle/HTTP observers, writers, loggers, and shutdown timing.

## Repository map

- `annotation/`: public annotation model.
- `compiler/parser/`: annotation parser.
- `compiler/load/` and `compiler/resolve/`: the authoritative typed-program front end.
- `compiler/provider/`, `compiler/graph/`, `compiler/lifecycle/`, and `compiler/application/`: application dependency, lifecycle, root, and immutable IR metadata.
- `compiler/generate/`: pure generated Go and ownership-manifest planning,
  partitioned by source mirrors, providers, HTTP, configuration, runtime,
  validation, hashing, and naming.
- `compiler/scan/`: compatibility source-tree scanner.
- `cmd/spice/`: production CLI entrypoint backed by the generated `Spice`
  application.
- `cmd/spice-bootstrap/`: ordinary-Go stage-zero compiler used for recovery.
- `cmd/spice-release/` and [`docs/releasing.md`](docs/releasing.md):
  reproducible signed cross-platform release construction and ceremony.
- `internal/cli/`: CLI dispatch plus bounded command implementations for
  generation, module reporting/testing, development, and execution. Production
  dispatch is an injected ordered collection of explicit `Handler` interface
  beans; stage zero manually assembles the same exported factories.
- `internal/spiceapp/` and `internal/autoconfigure/`: handwritten production
  application marker and reviewed runtime/handler/command contributions.
- `internal/spicegen/spice/`: committed typed production CLI application graph.
- `internal/genfs/`: rooted, ownership-checked generated-file application.
- `internal/qualitygate/`: cross-platform repository verification.
- `editors/goland/`: primary native IntelliJ Platform presentation and LSP adapter.
- `editors/zed/`: supported secondary Rust/WASM adapter that launches `spice lsp` for Go.
- `config/`: public configuration schema, source, snapshot, decode, validation, and redaction runtime.
- `resource/`: instance-owned canonical locations over explicit caller-owned
  Go filesystems with bounded context-aware reads.
- `conversion/`: typed reflection-free converters and canonical scalar/time/URL
  codecs shared by configuration and HTTP binding.
- `expression/`: bounded typed Boolean/string expressions over explicit
  positional schemas, with no reflection or ambient symbol access.
- `intercept/`: generic typed invocation chains used by generated route
  decorators and ordinary Go libraries.
- `validation/`: immutable violations plus layer-neutral typed validator
  composition.
- `data/`: public `database/sql` executor, transaction manager, typed repository query, and observation contracts.
- `migration/`: deterministic module-owned plans, registry reconciliation, and execution contracts.
- `event/`: public generic application-event topics, subscribers, and interaction observations.
- `event/outbox/`: transactional durable-message and at-least-once dispatch contracts.
- `retry/`: public bounded retry policies and typed execution helpers.
- `cache/`: public generic cache contracts and bounded in-memory implementation.
- `async/`: public bounded asynchronous execution and lifecycle contracts.
- `schedule/`: public fixed-delay job registration and lifecycle runtime.
- `security/`: public principals, deny-by-default policies, authorizer, and HTTP guards.
- `observability/`: instance-owned structured lifecycle and HTTP logging adapters.
- `starter/`: reviewed opt-in integrations, including OpenTelemetry telemetry and OIDC JWT resource-server authentication.
- `tools/`: isolated, pinned development tools module.
- `examples/`: executable reference applications.
- `docs/`: user and product documentation.
- `docs/quality.md`: exact verification, tool, linter, and suppression policy.
- `docs/dogfooding-readiness.md`: bootstrap boundary and self-hosting
  acceptance.
- `rfcs/`: proposed designs.
- `adrs/`: accepted architectural decisions.

## Status

Spice is pre-1.0. Its developer-proof core, generated application lifecycle,
Modulith enforcement, web/configuration/security/data foundations, Petclinic
reference workflow, and primary GoLand integration are executable and locally
gated. Public contracts may still evolve until every coverage row is resolved
and the v1.0 compatibility policy is frozen.
