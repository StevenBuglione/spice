# Dependency review: go-sql-driver/mysql

- Decision: approved for the isolated `starter/mysql` package.
- Version: `github.com/go-sql-driver/mysql` v1.10.0.
- Upstream: <https://github.com/go-sql-driver/mysql>.
- License: MPL-2.0; the unchanged vendored dependency retains its license and
  Spice does not modify its files.
- Maintenance: active stable driver for Go's `database/sql` API with context
  cancellation and current MySQL support.
- Security: Spice constructs the driver configuration without parsing an
  arbitrary DSN, verifies TLS certificates and hostnames by default, rejects
  unsafe driver options, disables multi-statements and file loading, and never
  includes the URL or password in errors. Insecure local connections require
  an option and an explicit URL marker.
- Cancellation: connection establishment, ping, queries, and transactions use
  caller-owned contexts. Driver cancellation closes an in-flight connection,
  matching `database/sql` cancellation semantics.
- Observability: a bounded application name is sent as a connection attribute;
  the application owns tracing wrappers and the pool lifecycle.
- Configuration: Spice validates identities, pool bounds, timeouts, and TLS
  policy before constructing a pool. `Open` performs no network I/O and
  registers no global TLS configuration.
- Migration limitation: MySQL DDL is atomic per supported InnoDB statement but
  is not transactional and implicitly commits. Petclinic therefore uses a
  locked, idempotent, resumable migration profile instead of claiming the core
  transactional migration backend contract.

Primary references:

- <https://pkg.go.dev/github.com/go-sql-driver/mysql>
- <https://github.com/go-sql-driver/mysql>
- <https://dev.mysql.com/doc/refman/8.0/en/atomic-ddl.html>
- <https://dev.mysql.com/doc/refman/9.6/en/locking-functions.html>
