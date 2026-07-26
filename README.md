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
- Exact-type `@Application` roots assembled with provider and lifecycle data in one immutable application IR.
- A pure deterministic renderer for direct provider/lifecycle calls and SHA-256 ownership manifests.
- Guarded generated-file ownership with manual-edit refusal, freshness checks, bounded diffs, and unchanged-file preservation.
- Import-path application modules with root APIs, named interfaces, explicit dependencies, internal-boundary checks, unassigned-package reporting, and deterministic cycle detection.
- Module-aware synchronous lifecycle observations that generated applications expose without a global tracer or telemetry dependency.
- Reflection-free typed configuration declarations, exact provider injection, generated schema/binders, and a runtime with rooted JSON/profile files, explicit precedence, provenance, environment mapping, defaults, validation, and secret redaction.
- Standard-library SQL transaction management with repository-friendly executors, module-owned boundary metadata, rollback-safe error/panic behavior, and synchronous observations.
- Immutable generic event topics with exact payload types, deterministic subscriber order, cancellation/failure semantics, and module-interaction observations.
- Explicit bounded retries with opt-in error classification, capped deterministic backoff, cancellation, typed exhaustion, and attempt observations.
- Generic cache contracts and a bounded in-memory LRU/TTL cache with explicit expiration, caller-owned time, safe metrics, and no cleanup goroutine.
- A strict HTTP runtime with RFC 9457 problems, secure error mapping, bounded JSON decoding, content negotiation, safe scalar binding, and explicit no-content responses.
- Typed controller/route compilation and deterministic generated `net/http` adapters with exact receiver/mux providers, request DTO binding, RFC 9457 errors, ServeMux wildcard checks, and raw escape hatches.
- A runnable `spice` CLI with `version`, `annotations`, `verify`, `modules`, `generate`, and `build` commands.
- A committed generated HTTP application with real provider, lifecycle, route, and graceful-drain tests.
- A cross-platform Go-owned quality gate with pinned format, lint, nil-safety, security, vulnerability, race, fuzz, coverage, offline-vendor, and executable checks.
- Product, architecture, annotation, and Spring-coverage documents.

## Annotation syntax

Both of these are accepted:

```go
//@Controller(prefix="/users")
// @Controller(prefix="/users")
type UserController struct{}
```

`gofmt` canonicalizes the second form, so official Spice documentation uses:

```go
// @Controller(prefix="/users")
type UserController struct{}
```

Qualified annotations are available for collisions:

```go
// @security.Authorize(roles=["admin"])
```

## Run it

Install Go 1.26.5 and GNU Make, then run:

```bash
make verify
go run ./cmd/spice version
go run ./cmd/spice annotations ./examples/commerce/...
go run ./cmd/spice verify ./...
go run ./cmd/spice generate --check --target Commerce ./examples/commerce/bootstrap ./examples/commerce/inventory ./examples/commerce/orders ./examples/commerce/payments ./examples/commerce/platform
go run ./examples/commerce -check
```

In an application module containing one typed `@Application` marker:

```bash
spice generate ./...
spice generate --check ./...
spice generate --diff ./...
spice build ./...
```

Controller targets also own
`internal/spicegen/<target>/openapi.json`; generation check/diff verifies it
alongside the generated application.

Production services can mount the opt-in `management` handler for deterministic
health, liveness, readiness, and caller-owned info endpoints; see
[`docs/management.md`](docs/management.md).

Outbound integrations can use the base-scoped, bounded typed JSON client in
[`docs/http-client.md`](docs/http-client.md).

SQL repositories and generated transaction decorators can use the explicit
contracts in [`docs/data.md`](docs/data.md).

Typed in-process event contracts are documented in
[`docs/events.md`](docs/events.md).

Context-aware resilience policies are documented in
[`docs/retry.md`](docs/retry.md).

Typed caching and the built-in bounded store are documented in
[`docs/cache.md`](docs/cache.md).

For a repository containing package-level `@Module` roots:

```bash
spice modules --format=json ./...
spice modules --format=mermaid ./...
spice modules --format=plantuml ./...
spice modules --focus=example.com/shop/orders --format=json ./...
```

JSON contains complete portable module canvases. Mermaid and PlantUML aggregate
the same verified package-import edges into deterministic module diagrams.
`--focus` retains one module and only its transitively observed dependencies,
with dependency-first composition order for module test slices.

Use `--target Name` when the selected packages contain multiple application
markers. Generation writes only manifest-owned files under
`internal/spicegen/<target>` and `.spice/<target>.manifest.json`.

To start the example HTTP server:

```bash
go run ./examples/commerce
curl -H "Content-Type: application/json" -d "{\"quantity\":2}" http://localhost:8081/orders
curl http://localhost:8081/actuator/health/readiness
curl http://localhost:8081/actuator/metrics
```

The modular commerce command owns `SIGINT`/`SIGTERM` handling and supplies a
fresh ten-second shutdown context to the generated application's `Run` method.
It explicitly opts into environment configuration and mounts isolated
management health, readiness, info, and generated-route metrics endpoints.
Generated source and OpenAPI are committed under
`internal/spicegen/commerce`; the matching ownership manifest is
`.spice/commerce.manifest.json`.

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
- `config/`: public configuration schema, source, snapshot, decode, validation, and redaction runtime.
- `data/`: public `database/sql` executor, transaction manager, and observation contracts.
- `event/`: public generic application-event topics, subscribers, and interaction observations.
- `retry/`: public bounded retry policies and typed execution helpers.
- `cache/`: public generic cache contracts and bounded in-memory implementation.
- `observability/`: instance-owned structured lifecycle and HTTP logging adapters.
- `starter/`: reviewed opt-in third-party integrations, including OpenTelemetry.
- `tools/`: isolated, pinned development tools module.
- `examples/`: executable reference applications.
- `docs/`: user and product documentation.
- `docs/quality.md`: exact verification, tool, linter, and suppression policy.
- `rfcs/`: proposed designs.
- `adrs/`: accepted architectural decisions.

## Status

Spice is pre-alpha. The active program is completing deterministic application generation and lifecycle, Modulith-style architecture enforcement, the production web/configuration platform, and reviewed opt-in enterprise starters before freezing a v1.0 compatibility policy.
