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
