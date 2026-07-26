# Spice Roadmap

## Current bootstrap priority

The annotation-driven application bootstrap is the foundation for subsequent
starter work. `@Application` now composes generated command construction,
conventional environment loading, structured command logging, signal handling,
stable exit codes, and typed bounded shutdown. Explicit
`@management.Enable(expose=[...])` and `@observability.Logging` companions are
resolved through typed definitions, preserved in the immutable application IR,
and rendered without runtime discovery.

The next starter slices must reuse this deterministic feature pipeline. The
public third-party annotation/manifest SDK now defines the portable metadata
boundary. A committed `.spice/starters.json` selection now drives fail-closed
CLI/compiler feature composition without dependency-presence activation. The
compiler now validates explicitly supplied starter entrypoint functions in the
same typed program, merges them into the exact provider graph, and renders
ordinary direct calls with cleanup/rollback behavior and provenance-sensitive
ownership hashes. The CLI now loads selected entrypoint packages as isolated
auxiliary roots and maps explicit-constructor or annotation-selected subsets
into that graph without import-based activation. The CLI now snapshots the
selected Go module graph in read-only offline mode and requires every active
starter dependency to match its exact reviewed module version and replacement
identity before provider analysis. Automated module-cache discovery and richer
compile-time conditions remain future work.

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
- Mail and development tooling.
- Starter SDK and third-party annotation SDK.
- Language server and editor extensions.

## Release gates

A milestone does not complete because APIs exist. It completes when:

- Reference applications run.
- Integration tests exercise real behavior.
- Error messages are actionable.
- Generated code is inspectable.
- Documentation supports a new developer without tribal knowledge.
- Benchmarks show acceptable startup, build, and runtime overhead.
