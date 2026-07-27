# Spice Framework for Go

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
- Exact-type bean/configuration provider catalog and deterministic dependency graph validation.
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
- Deterministic module-owned migration plans with global version ordering, normalized SHA-256 checksums, registry drift detection, and a concrete PostgreSQL advisory-lock/transaction backend.
- Immutable generic event topics with exact payload types, deterministic
  subscriber order, cancellation/failure semantics, module-interaction
  observations, and compile-time `@event.Topic`/`@event.Listener` graph
  metadata rendered as direct, rollback-safe topic construction.
- A transactional outbox with immutable bounded messages, a driver-neutral SQL store, atomic enqueue/lease contracts, at-least-once dispatch, explicit failure delay, and payload-free observations.
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
- An explicitly selected `@otel.Enable` OpenTelemetry v1.43 HTTP trace/metric
  starter with exact generated observer-role validation and
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
  responses, cancellation, and concurrent rendering.
- Immutable transport-neutral mail messages with caller-owned identity and
  time, stable envelope recipients, Bcc-safe deterministic MIME, text/HTML
  alternatives, bounded attachments, defensive copies, and no hidden network
  client.
- An instance-owned `mail/mailtest` sender with bounded immutable attempts,
  deterministic injected failures, explicit overflow, payload-free
  observations, concurrent inspection, and typed MIME snapshots.
- A strict HTTP runtime with RFC 9457 problems, secure error mapping, bounded JSON decoding, content negotiation, safe scalar binding, and explicit no-content responses.
- Typed controller/route compilation and deterministic generated `net/http` adapters with exact receiver/mux providers, request DTO binding, RFC 9457 errors, ServeMux wildcard checks, and raw escape hatches.
- A runnable `spice` CLI with `version`, `annotations`, `verify`, `modules`,
  `generate`, `build`, `run`, last-known-good `dev`, and editor-neutral `lsp`
  commands.
- A repository-owned GoLand plugin that renders canonical declaration comments
  as zero-width-prefix annotations, applies configurable native syntax colors,
  resolves highlighted PSI references to their real Go SDK descriptor
  declarations, shows descriptor provenance in the gutter, checks light/dark
  rendering against committed visual goldens, and launches the same LSP for
  descriptor documentation, handler implementation navigation, completion,
  diagnostics, safe edits, and confirmed hash-guarded `go get -tool`
  preview/apply.
- A supported secondary Zed extension that launches the same LSP beside
  `gopls` for completion, diagnostics, hover, modifier-click annotation
  navigation, safe quick fixes, module/configuration metadata, and structured
  valid-Go annotation highlighting on Windows and Linux.
- A generated-application HTTP test slice with loopback-only serving, bounded
  detached responses, strict JSON/problem decoding, construction rollback,
  and idempotent lifecycle cleanup, plus transaction-scoped generic SQL
  subjects that always roll back.
- A committed generated HTTP application with real provider, lifecycle, route, and graceful-drain tests.
- A cross-platform Go-owned quality gate with pinned format, lint, nil-safety, security, vulnerability, race, fuzz, coverage, offline-vendor, and executable checks.
- Product, architecture, annotation, and Spring-coverage documents.

## Annotation syntax

Annotations are valid declaration comments with explicit file-scoped imports:

```go
// @spice.import { Controller } from "github.com/StevenBuglione/spice/annotation/web"

// @Controller(prefix="/users")
type UserController struct{}
```

Named imports keep common annotations clean, aliases resolve local collisions,
and namespace imports keep provenance visible:

```go
// @spice.import { Get as GET } from "github.com/StevenBuglione/spice/annotation/web"
// @spice.import * as security from "github.com/StevenBuglione/spice/annotation/security"

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

The independent modules under
[`testdata/annotationfixture`](testdata/annotationfixture) and
[`testdata/annotationapp`](testdata/annotationapp) prove that a third party can
use only the public SDK/protocol to supply aliases, namespaces, diagnostics,
real editor navigation, provider semantics, and inspectable generated Go.

## Run it

Install Go 1.26.5 and GNU Make, then run:

```bash
make verify
go run ./cmd/spice version
go run ./cmd/spice annotations list ./examples/commerce/...
go run ./cmd/spice annotations doctor ./examples/commerce/...
go run ./cmd/spice verify ./...
go run ./cmd/spice verify --format=json ./examples/commerce/...
go run ./cmd/spice test --module github.com/StevenBuglione/spice/examples/commerce/orders --count=1 ./examples/commerce/...
go run ./cmd/spice generate --check --target Commerce ./examples/commerce/...
go run ./cmd/spice run --target Commerce ./examples/commerce/... -- -check
go run ./cmd/spice dev --target Commerce ./examples/commerce/...
```

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
compile into direct-call Go beside it:

```go
package main

import "os"

// @spice.import { Application } from "github.com/StevenBuglione/spice/annotation/core"
// @spice.import { Enable } from "github.com/StevenBuglione/spice/annotation/management"
// @spice.import { Logging } from "github.com/StevenBuglione/spice/annotation/observability"

// @Application
// @Enable(expose=["health", "liveness", "readiness", "info", "metrics", "configprops", "modules"])
// @Logging
func main() {
    os.Exit(spiceMain(os.Args[1:]))
}
```

`spiceMain` is generated into the same package and returns an exit code; it
does not call `os.Exit`. It resolves
the generated schema from the `SPICE_` environment convention, logs command
startup and failures, owns `SIGINT`/`SIGTERM`, and creates a fresh bounded
shutdown context. `spice.shutdown-timeout` defaults to `10s` and can be set
with `SPICE_SHUTDOWN_TIMEOUT`.

Controller targets also own `openapi.json` beside the annotated command;
generation check/diff verifies it alongside the generated application.

Production services opt into only the management routes they intend to expose
with `@management.Enable(expose=[...])`. The endpoint allowlist is exact and
validated at compile time; package presence or a `go.mod` dependency never
activates it. See
[`docs/management.md`](docs/management.md).

The preferred annotated `main.go`, compile-time discovery scope, generated
bridge, and legacy migration contract are documented in
[`docs/application.md`](docs/application.md).

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
module list. Preferred package-main generation writes manifest-owned files
beside `main.go` and `.spice/<target>.manifest.json`. Legacy marker targets
retain `internal/spicegen/<target>` during the pre-1.0 compatibility period.

To start the example HTTP server:

```bash
go run ./cmd/spice run --target Commerce ./examples/commerce/...
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
OpenAPI are committed as `examples/commerce/zz_spice_gen.go` and
`examples/commerce/openapi.json`; the matching ownership manifest is
`.spice/commerce.manifest.json`.

For embedding and specialized policies, the generated application retains
`NewApplication`, `NewApplicationWithOptions`, `Application.Start`,
`Application.Stop`, `Application.Run`, and `RunCommand`. These seams support
caller-owned contexts, signals, configuration sources, middleware, error
mapping, lifecycle/HTTP observers, writers, loggers, and shutdown timing.

## Repository map

- `annotation/`: public annotation model.
- `compiler/parser/`: annotation parser.
- `compiler/load/` and `compiler/resolve/`: the authoritative typed-program front end.
- `compiler/provider/`, `compiler/graph/`, `compiler/lifecycle/`, and `compiler/application/`: application dependency, lifecycle, root, and immutable IR metadata.
- `compiler/generate/`: pure generated Go and ownership-manifest planning.
- `compiler/scan/`: compatibility source-tree scanner.
- `cmd/spice/`: CLI entry point.
- `internal/cli/`: CLI implementation.
- `internal/genfs/`: rooted, ownership-checked generated-file application.
- `internal/qualitygate/`: cross-platform repository verification.
- `editors/goland/`: primary native IntelliJ Platform presentation and LSP adapter.
- `editors/zed/`: supported secondary Rust/WASM adapter that launches `spice lsp` for Go.
- `config/`: public configuration schema, source, snapshot, decode, validation, and redaction runtime.
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
- `rfcs/`: proposed designs.
- `adrs/`: accepted architectural decisions.

## Status

Spice is pre-alpha. The active program is completing deterministic application generation and lifecycle, Modulith-style architecture enforcement, the production web/configuration platform, and reviewed opt-in enterprise starters before freezing a v1.0 compatibility policy.
