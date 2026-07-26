# Dependency review: go-redis

- Decision: approved for the isolated `starter/redis` package.
- Version: `github.com/redis/go-redis/v9` v9.21.0.
- Upstream: <https://github.com/redis/go-redis>.
- License: BSD-2-Clause; retained in the vendored module license.
- Maintenance: the official Redis Go client has an active stable v9 line and
  supports the current Redis releases.
- Security: complete URLs prevent environment fallback, `rediss` with TLS 1.2
  or newer is the default, authentication is required by default, local
  plaintext and unauthenticated use require separate explicit opt-ins, query
  options are rejected, and failures never include the URL or password.
  `gosec` and `govulncheck` cover the reachable dependency graph.
- Cancellation: every network operation uses a caller-owned context with
  context deadlines enabled in the client.
- Observability: the client name is bounded and validated for safe server
  attribution. The typed cache adapter emits module-aware Spice observations
  without keys or values; optional OpenTelemetry hooks remain
  application-owned.
- Configuration: Spice fixes protocol, retry, timeout, and bounded-pool
  defaults instead of inheriting CPU- or process-environment-dependent
  behavior. `Open` performs no network I/O and returns exact lifecycle cleanup.
- Integration: a tagged test runs against the official Redis 8.4.0 Alpine
  image (`sha256:4eec4565e45aa0b3966554c866bc73211e281b0b3d89fe9a33c982e6faca809d`)
  and exercises connection, ping, bounded typed JSON cache operations,
  expiration, cancellation, and cleanup.

Primary references:

- <https://github.com/redis/go-redis/releases/tag/v9.21.0>
- <https://redis.io/docs/latest/develop/clients/go/connect/>
- <https://redis.io/docs/latest/develop/clients/go/error-handling/>
