# Module-owned database migrations

The `migration` package builds one immutable application plan from
module-owned SQL declarations:

```go
plan, err := migration.NewPlan([]migration.Spec{
    {
        Version: 202607250001,
        Module:  "example.com/shop/orders",
        Name:    "create orders",
        SQL:     ordersSchema,
    },
})
result, err := runner.Run(ctx, plan)
```

Versions are positive, application-global, and monotonically increasing. This
lets every module add later migrations without inserting work ahead of a
version that another module already applied. Duplicate versions fail during
plan construction.

SQL line endings are normalized to LF and checksummed with SHA-256. The durable
registry must be an exact prefix of the current plan: version, module, name,
checksum, and applied time are all checked before new SQL executes. Removed,
reordered, renamed, or edited migrations fail closed with no SQL text in the
error.

`Backend.RunLocked` owns the database-specific advisory lock and must invoke its
callback exactly once. The supplied `Session` reads the durable registry and
atomically executes each migration with its registry insert. Spice enforces
sequential version order, cancellation between entries, stop-on-first-failure,
and bounded SQL-free observations.

Core does not assume transactional DDL, invent a portable lock, start a
goroutine, or choose a driver. Dialect starters provide lock, transaction,
registry schema, and SQL execution policies appropriate to their database.

## PostgreSQL

The PostgreSQL starter supplies a concrete backend over a caller-owned pgx
`database/sql` pool:

```go
backend, err := postgres.NewMigrationBackend(database, postgres.MigrationOptions{
    Schema: "public",
})
if err != nil {
    return err
}
runner, err := migration.NewRunner(backend)
if err != nil {
    return err
}
result, err := runner.Run(ctx, plan)
```

The configured schema must already exist. The backend owns the fixed
`spice_schema_history` table within that schema and validates the schema as a
PostgreSQL identifier; table names and registry SQL are never derived from
migration content. A zero lock ID selects Spice's stable default. Applications
sharing a database but intentionally maintaining independent registries should
select distinct nonzero lock IDs and schemas.

Each run pins one physical pgx connection and holds a PostgreSQL session-level
advisory lock across reconciliation and application. Each migration script and
its parameterized registry insert commit in one transaction. Scripts can
contain multiple PostgreSQL statements. A failed script or registry write rolls
back both. Lock waits honor cancellation. If unlock cannot be confirmed, Spice
closes the physical connection so a session lock is never returned to the
pool.

The registry stores the complete Go `uint64` version domain as constrained
`numeric(20,0)`, orders versions numerically, and returns timestamps in UTC.
Errors contain migration identity but never SQL text, connection URLs, or
credentials.
