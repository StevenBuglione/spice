# Dependency review: pgx

- Decision: approved for the isolated `starter/postgres` package.
- Version: `github.com/jackc/pgx/v5` v5.10.0.
- Upstream: <https://github.com/jackc/pgx>.
- License: MIT; retained in the vendored module license.
- Maintenance: active stable v5 line, released June 3, 2026; upstream supports
  current Go releases and PostgreSQL versions from the last five years.
- Security: complete URLs prevent environment fallback, TLS hostname
  verification is the default, insecure mode requires explicit opt-in, and
  configuration failures do not include URLs or passwords. gosec and
  govulncheck cover the reachable dependency graph. pgx's minimum
  `golang.org/x/text` v0.29.0 is overridden to v0.39.0 because the former is
  reachable and affected by `GO-2026-5970`.
- Cancellation: connection establishment, ping, queries, and transactions use
  caller-owned contexts through `database/sql`.
- Observability: application names are bounded and safe for server logs; pgx
  tracing and OpenTelemetry transports can be supplied by later opt-in
  adapters without global state.
- Configuration: Spice validates pool bounds and TLS policy before creating a
  pool. `Open` performs no network I/O and the application owns `Close`.
- Integration: a tagged test runs against the pinned official PostgreSQL 18.4
  Alpine image and exercises Spice transactions and repositories.

Primary references:

- <https://pkg.go.dev/github.com/jackc/pgx/v5>
- <https://pkg.go.dev/github.com/jackc/pgx/v5/stdlib>
- <https://hub.docker.com/_/postgres>
