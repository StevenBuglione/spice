# SQL data access and transactions

Spice keeps `database/sql` visible. Repositories depend on `data.Executor`,
which is implemented by both `*sql.DB` and `*sql.Tx`, while an instance-owned
`data.Manager` controls each transaction:

```go
manager, err := data.NewManager(db, transactionObserver)
if err != nil {
    return err
}

definition := data.Definition{
    ID:        "orders.PlaceOrder",
    Module:    "example.com/shop/orders",
    Isolation: sql.LevelSerializable,
}
err = manager.Within(ctx, definition, func(ctx context.Context, queries data.Executor) error {
    _, err := queries.ExecContext(ctx, "INSERT INTO orders (id) VALUES (?)", orderID)
    return err
})
```

The database and its pool settings remain application-owned. Constructing a
manager does not open a connection, change pool settings, start a goroutine, or
install global state.

The manager owns commit and rollback; its callback receives no transaction
control methods. A callback error causes rollback and any rollback failure is
joined with the original error. A panic also causes rollback and synchronous
observation before the original panic value is re-raised. Commit occurs only
after a nil callback result. Context cancellation is handled by
`database/sql` and the selected driver.

Definitions require stable boundary and module identities. Generated
transaction decorators will supply these from typed application/module IR.
Observers receive the same bounded metadata, elapsed duration, error, and panic
state. They may enrich the callback context and finish in reverse nesting
order; they must not panic or block indefinitely.

Drivers, migrations, generated repositories, retry policy, and OpenTelemetry
transaction adapters are separate opt-in slices. The core transaction package
uses only the standard library and performs no implicit retry: repeating a
transaction is safe only when application semantics make the entire callback
retryable.
