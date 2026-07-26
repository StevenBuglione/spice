# PostgreSQL starter

`starter/postgres` integrates pgx v5.10.0 through its `database/sql` adapter, so
the same pool works with Spice transactions and typed repositories.

```go
database, err := postgres.Open(postgres.Options{
    URL:                   configuration.DatabaseURL,
    ApplicationName:       "orders-service",
    MaxOpenConnections:    40,
    MaxIdleConnections:    20,
    ConnectionMaxLifetime: 30 * time.Minute,
    ConnectionMaxIdleTime: 5 * time.Minute,
})
if err != nil {
    return err
}
cleanup := func(context.Context) error { return database.Close() }
```

`Open` parses configuration and creates a caller-owned pool without connecting.
Use `postgres.Ping` with a startup-owned timeout when readiness requires a live
database. The application owns pool shutdown.

Connection configuration must be a complete `postgres://` or
`postgresql://` URL containing user, non-empty password, host, explicit port,
and database. This prevents pgx from silently completing connection identity
from process environment variables.
The starter adds `sslmode=verify-full` when it is absent and rejects
`allow`/`prefer`. `require`, `verify-ca`, and `verify-full` are explicit secure
choices. `sslmode=disable` requires `AllowInsecure` and is intended only for
isolated local tests.

Defaults are 20 open connections, 10 idle connections, a 30-minute connection
lifetime, a five-minute idle lifetime, and application name `spice`. Invalid
configuration errors never include the connection URL or password.

The tagged integration test targets the pinned official
`postgres:18.4-alpine3.24` image:

```bash
docker run --rm --name spice-postgres-test \
  -e POSTGRES_PASSWORD=spice-test \
  -e POSTGRES_DB=spice \
  -p 55432:5432 \
  -d postgres:18.4-alpine3.24
SPICE_POSTGRES_TEST_URL='postgres://postgres:spice-test@127.0.0.1:55432/spice?sslmode=disable' \
  go test -tags=integration ./starter/postgres
docker rm -f spice-postgres-test
```

In PowerShell, set the URL before the same `go test` command with:

```powershell
$env:SPICE_POSTGRES_TEST_URL = 'postgres://postgres:spice-test@127.0.0.1:55432/spice?sslmode=disable'
go test -tags=integration ./starter/postgres
Remove-Item Env:SPICE_POSTGRES_TEST_URL
```

The tagged test fails, rather than silently skipping, when the URL is absent.
The test performs a real ping, creates a table, commits through `data.Manager`,
and reads through `data/repository`.
