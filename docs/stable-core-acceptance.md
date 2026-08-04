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
| Resources and conversion | Explicit immutable `fs.FS` mounts, canonical location/path rejection, bounded/canceled reads, typed codec composition, safe scalar failures, custom formatting, and shared configuration/HTTP scalar conversion | accepted |
| Expressions, interception, and bean composition | Restricted typed schema/parser/evaluator boundaries, compiler-rejected unsafe authorization expressions, generated executable request/response interceptor chains around direct/transaction/cache calls, nil/cancellation/order tests, and generated named override layers with deterministic child precedence and invalid/duplicate-name rejection | accepted |
| Validation | Layer-neutral typed validators, immutable violation ordering, cancellation, operational-failure short circuiting, bounded aggregation, and rejected-value-free errors | accepted |
| Modules | Compile-time root discovery, longest-root ownership, default/named API rules, allowed dependencies, internal-access rejection, cycles, unassigned packages, JSON/Mermaid/PlantUML rendering, and focused module tests | accepted |
| Configuration | Generated binders/metadata, defaults, profiles, JSON/environment precedence, provenance, validation, cancellation, secret redaction, management reporting, and raw-value leak regressions | accepted |
| Web | Generated strict body/path binding, validation, content negotiation, JSON/no-content responses, RFC 9457 mapping, middleware/observer ordering, panic handling, OpenAPI ownership, and graceful server drain | accepted |
| Security | Typed policy validation, generated deny-by-default route guards, exact role/scope decisions, safe 401/403 problems, bounded observations, and generated authorization execution | accepted |
| Data | Transaction commit/rollback/panic/cancellation, generated transactional routes, bounded repository cardinality and query secrecy, migration prefix/drift/failure behavior, SQL test slices, and opt-in PostgreSQL integration fixtures | accepted |

The mandatory repository path additionally runs formatting, module tidiness,
vendor reproduction, vet, allowlisted lint and NilAway, gosec, govulncheck,
shuffled and race suites, fuzz smoke, the 85% coverage floor, offline vendor
tests, module rendering/focus, generated freshness, and executable Commerce.

## Developer-proof integration

The final reference workflow now composes the accepted contracts:

- Commerce currently proves package-main discovery, exact generated DI,
  modules, typed configuration/redaction, lifecycle, generated HTTP/RFC 9457,
  management, metrics/logging, public cache-safe metadata, asynchronous work,
  scheduling, typed events, generated serializable transaction ownership,
  explicit repository interface injection, module-owned migration startup,
  and persisted retrieval.
- Order routes now exercise generated exact-scope security policies. The
  executable slice proves allowed decisions, safe unauthenticated 401
  responses, insufficient-scope 403 responses, and bounded authorization
  observations. Principal-specific order data is not placed in the shared
  cache; the compiler rejects that unsafe combination.
- Commerce uses a typed `storage.Orders` interface and generated direct
  binding to a reflection-free SQL repository. Its zero-dependency developer
  backend is an instance-owned transaction-aware `database/sql` connector, and
  its `integration`-tagged PostgreSQL path proves migration reconciliation,
  commit, pool close/reopen, and durable retrieval against the reviewed pgx
  starter.
- `notifications.Delivery` is an explicit concrete `mail.Sender` binding and
  `SystemClock` is an explicit clock binding. Commerce composes a deterministic
  typed MIME receipt with an attachment after transaction commit, delivers it
  through the bounded test transport in the default workflow, and can select
  the isolated secure SMTP starter through typed secret-redacted
  configuration. Tests inspect the decoded delivery and prove cancellation
  plus error sanitization.
- `spice dev`, the overlay compiler service, the editor-neutral LSP, the
  primary GoLand integration, and the supported Zed integration are available.
  GoLand's exact prefix concealment, native token colors, PSI navigation, and
  light/Darcula visual reports are repository-gated, including proof that
  concealment preserves saved and copied valid Go. Raw annotation lines receive
  exact LSP diagnostics and a versioned comment-prefix repair instead of an
  opaque temporary-loader failure. Together with the commerce executable slice,
  these gates cover the decisive invalid-edit/fix/regenerate/restart/
  authorize/persist/deliver workflow without weakening valid Go source.
- The documented process workflow enables an explicit secret-redacted
  developer bearer token only on a loopback listener. It exercises the real
  `spice dev` process and generated guards without creating a default
  production backdoor; production authentication remains an OAuth2/OIDC
  integration concern.

## Reproduction

From the repository root with Go 1.26.5:

```text
make verify
go run ./cmd/spice verify --format=json ./examples/commerce/...
go run ./cmd/spice generate --check --target Commerce ./examples/commerce
go run ./cmd/spice build --target Commerce ./examples/commerce
go run ./cmd/spice run --target Commerce ./examples/commerce -- -check
go run ./cmd/spice modules --format=json ./examples/commerce/...
go run ./cmd/spice test --module github.com/StevenBuglione/spice/examples/commerce/orders --count=1 ./examples/commerce/...
```
