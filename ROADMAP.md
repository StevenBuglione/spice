# Spice Roadmap

## Current developer-proof priority

Before broader database, messaging, or starter expansion, Spice must prove one
complete development workflow: a minimal annotated `main.go`, compile-time
local-module discovery, stable generate/verify/build/run behavior, a
last-known-good `spice dev` supervisor, one shared overlay-aware compiler
service, editor-neutral LSP support, a polished Zed integration, and one secure
external mail integration exercised by the modular commerce application.

The preferred package-main discovery and same-package command bridge are now
exercised by the commerce application; the compatible legacy layout remains
available pre-1.0. `spice run` now applies guarded generation, builds a unique
trimpath candidate, and relays the selected package-main process on Windows and
Unix. Coverage-map labels are not substitutes for executable acceptance. The
shared `spice.diagnostics/v1` contract now backs text and machine-readable
verification while retaining physical/source-mapped positions for editor use.
The stable-core executable matrix is recorded in
`docs/stable-core-acceptance.md`; remaining security/data/mail work is explicit
reference integration debt rather than an untested core contract. The
reusable dev supervisor and portable recursive watcher now provide
deterministic debounce, unique candidate builds, last-known-good recovery, and
bounded Windows/Unix process-group restart through `spice dev`. A native watch
accelerator remains follow-up; content polling is the correctness fallback.
The isolated overlay compiler service now returns the shared diagnostics,
resolved annotations, exact provider/application/module/configuration models,
safe fixes, and pure generation readiness with cancellation, stale-sequence
rejection, and bounded content-keyed caching. Generate, build, run, and dev
now consume that service, including selected-starter metadata and reviewed
module-version enforcement. `spice lsp` now publishes the same versioned
diagnostics and provides compiler-derived annotation, module, configuration,
hover, safe-edit, and semantic-token metadata over bounded stdio JSON-RPC.
The repository-owned Zed extension launches that server beside `gopls`, uses
the standard settings/PATH contracts on Windows and Linux, and preserves valid
Go while providing native completion, diagnostics, exact modifier-click
annotation navigation, quick fixes, and structured semantic annotation
presentation. The bounded test mail transport now supplies immutable
decoded delivery snapshots, deterministic failure injection, explicit
capacity behavior, and payload-free observations. The next slice is secure
SMTP, followed by the final reference workflow. Broad MongoDB,
OpenSearch, Kafka, RabbitMQ, GraphQL, and WebSocket work remains paused until
that workflow is repeatable on Windows and Linux.

## M0 — Product and compiler proof

- Annotation syntax and parser.
- Source scanning and declaration association.
- Typed annotation definition model.
- Source-positioned diagnostics.
- Deterministic generation contract.
- Competitive and Spring capability inventory.
- Developer interviews and ergonomics benchmarks.

## M1 — Application and module foundation

- Constructor provider graph.
- Lifecycle and shutdown.
- Typed bootstrap-feature metadata and generated command composition.
- Application module discovery.
- API/internal boundary rules.
- Allowed dependency declarations.
- Module cycle detection.
- Module documentation generation.
- Module-focused test support.

## M2 — Web application MVP

- Controller and route annotations.
- Parameter and body binding.
- Validation.
- Standard error model.
- OpenAPI generation.
- HTTP middleware and filters.
- Health, liveness, readiness, and graceful shutdown.
- Reference modular-monolith application.

## M3 — Enterprise foundations

- Externalized configuration and profiles.
- Structured logging and OpenTelemetry.
- Security policies, OAuth2/OIDC resource server support.
- SQL, generated typed HTTP transaction boundaries, migrations, and repository
  support.
- Cache abstraction.
- Annotation-driven fixed-delay scheduling and asynchronous execution.
- Restartable ordered batch execution with process-local and lease-aware
  PostgreSQL persistence.
- Focused module execution plus generated HTTP and transaction-rollback data
  test slices.
- Typed application events and durable publication.

## M4 — Broad Spring Boot coverage

- Kafka, RabbitMQ, and additional messaging starters.
- Redis, MongoDB, Elasticsearch/OpenSearch, and selected data starters.
- gRPC, GraphQL, WebSocket, and outbound HTTP clients.
- Immutable mail composition, test delivery, and a secure SMTP starter.
- Last-known-good development supervision and cross-platform restart.
- Starter SDK and third-party annotation SDK.
- Shared overlay analysis, language server, and Zed integration.

## Release gates

A milestone does not complete because APIs exist. It completes when:

- Reference applications run.
- Integration tests exercise real behavior.
- Error messages are actionable.
- Generated code is inspectable.
- Documentation supports a new developer without tribal knowledge.
- Benchmarks show acceptable startup, build, and runtime overhead.
