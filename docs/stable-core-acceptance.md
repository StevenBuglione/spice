# Stable Core Acceptance

This matrix records executable evidence for the developer-proof foundation. A
coverage-map label is not evidence by itself. Every core row below is exercised
by ordinary tests or a repository-owned CLI smoke path in `make verify`.

## Core matrix

| Area | Acceptance evidence | Result |
| --- | --- | --- |
| Generate | `compiler/generate` deterministic render and executable fixtures; `internal/genfs` ownership, no-op, stale removal, collision, symlink, manual-edit, check, and bounded diff tests; Commerce `generate --check` smoke | accepted |
| Verify and diagnostics | Loader, resolution, validation, exact provider, lifecycle, module, configuration, controller, bootstrap, and application-model failures; physical-order/source-map regression; shared `spice.diagnostics/v1` text/JSON tests and fuzz smoke | accepted |
| Build | Package-main generation/build fixture, `go build -trimpath`, generated-code compilation, offline vendor suite, and Windows plus Linux CLI compilation | accepted |
| Run | Real package-main generation/build/launch, application argument and exit-code preservation, temporary-candidate cleanup, legacy rejection, cancellation, Windows/Unix process-group adapters, and Commerce `spice run -- -check` smoke | accepted |
| Lifecycle | Dependency-order start, reverse stop/cleanup, construction and startup rollback, joined failures, cancellation, concurrent transitions, idempotent stop, fresh shutdown contexts, command timeout, and HTTP drain tests | accepted |
| Modules | Compile-time root discovery, longest-root ownership, default/named API rules, allowed dependencies, internal-access rejection, cycles, unassigned packages, JSON/Mermaid/PlantUML rendering, and focused module tests | accepted |
| Configuration | Generated binders/metadata, defaults, profiles, JSON/environment precedence, provenance, validation, cancellation, secret redaction, management reporting, and raw-value leak regressions | accepted |
| Web | Generated strict body/path binding, validation, content negotiation, JSON/no-content responses, RFC 9457 mapping, middleware/observer ordering, panic handling, OpenAPI ownership, and graceful server drain | accepted |
| Security | Typed policy validation, generated deny-by-default route guards, exact role/scope decisions, safe 401/403 problems, bounded observations, and generated authorization execution | accepted |
| Data | Transaction commit/rollback/panic/cancellation, generated transactional routes, bounded repository cardinality and query secrecy, migration prefix/drift/failure behavior, SQL test slices, and opt-in PostgreSQL integration fixtures | accepted |

The mandatory repository path additionally runs formatting, module tidiness,
vendor reproduction, vet, allowlisted lint and NilAway, gosec, govulncheck,
shuffled and race suites, fuzz smoke, the 85% coverage floor, offline vendor
tests, module rendering/focus, generated freshness, and executable Commerce.

## Developer-proof integration still open

The core contracts are sufficient for the dev supervisor and editor service,
but the final reference workflow intentionally remains incomplete:

- Commerce currently proves package-main discovery, exact generated DI,
  modules, typed configuration/redaction, lifecycle, generated HTTP/RFC 9457,
  management, metrics/logging, cache, asynchronous work, scheduling, and typed
  events.
- Commerce does not yet exercise a generated security policy at its public
  endpoint.
- Commerce uses instance-owned in-memory order state; transactional SQL
  persistence, migrations, and repository retrieval must be added to the final
  Docker-backed reference path.
- The bounded test mail transport, deterministic failure injection, decoded
  inspection, and payload-free observations are available. The isolated SMTP
  starter now proves verified STARTTLS and implicit TLS, authentication after
  TLS, cancellation/timeouts, conservative transient retry, ambiguous-delivery
  protection, and payload-free observations. Commerce still needs to compose
  and deliver its notification through this boundary.
- `spice dev`, the overlay compiler service, the editor-neutral LSP, the
  primary GoLand integration, and the supported Zed integration are available.
  GoLand's exact prefix concealment, native token colors, PSI navigation, and
  light/Darcula visual reports are repository-gated, including proof that
  concealment preserves saved and copied valid Go. Raw annotation lines receive
  exact LSP diagnostics and a versioned comment-prefix repair instead of an
  opaque temporary-loader failure. The final editor/dev reference walkthrough
  remains coupled to the pending security/data/mail commerce flow.

Those are integration deliverables, not missing foundational runtime behavior.
They stay visible here and in `docs/spring-coverage.md`; they must be closed
before the decisive developer-proof workflow can be called complete.

## Reproduction

From the repository root with Go 1.26.5:

```text
make verify
go run ./cmd/spice verify --format=json ./examples/commerce/...
go run ./cmd/spice generate --check --target Commerce ./examples/commerce/...
go run ./cmd/spice build --target Commerce ./examples/commerce/...
go run ./cmd/spice run --target Commerce ./examples/commerce/... -- -check
go run ./cmd/spice modules --format=json ./examples/commerce/...
go run ./cmd/spice test --module github.com/StevenBuglione/spice/examples/commerce/orders --count=1 ./examples/commerce/...
```
