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
`@data.Transactional` typed HTTP adapters supply these from immutable
application/module IR. Observers receive the same bounded metadata, elapsed
duration, error, and panic state. They may enrich the callback context and
finish in reverse nesting order; they must not panic or block indefinitely.

Drivers, migrations, generated repositories, retry policy, and OpenTelemetry
transaction adapters are separate opt-in slices. The core transaction package
uses only the standard library and performs no implicit retry: repeating a
transaction is safe only when application semantics make the entire callback
retryable.

## Generated HTTP transaction boundaries

Annotate an exported typed `@Get` or `@Post` method and make the transaction
dependency explicit as parameter 1:

```go
// @Post("/orders")
// @data.Transactional(isolation="serializable")
func (*OrdersController) Create(
    ctx context.Context,
    queries data.Executor,
    request CreateOrderRequest,
) (CreateOrderResponse, error) {
    // Every repository call receives queries explicitly.
    return createOrder(ctx, queries, request)
}
```

The application graph must contain exactly one `@Bean` whose exact output type
is `*data.Manager`. Spice rejects an executor parameter without the annotation,
an annotated route without the executor parameter, raw `net/http` handlers,
and missing or ambiguous manager providers. Generation emits a direct
`Manager.Within` call; it does not use reflection, a service locator, or a
transaction stored in `context.Context`.

`isolation` is optional and defaults to `default`. Accepted values are
`default`, `read-uncommitted`, `read-committed`, `write-committed`,
`repeatable-read`, `snapshot`, `serializable`, and `linearizable`. Optional
`readOnly` is Boolean. These values are passed to `database/sql`; whether a
driver supports a particular isolation mode or enforces read-only semantics
remains driver-specific.

Request binding and validation run before the transaction starts. The generated
adapter commits only after the route returns a nil error. Route errors, panics,
and request-context cancellation follow `data.Manager` rollback semantics.
Response encoding occurs after commit, so an HTTP write failure never causes a
committed transaction to be retried or misreported as rolled back.

## Typed repository queries

`data/repository` provides immutable, reflection-free query definitions for
application-owned and generated repositories:

```go
findOrder, err := repository.NewQuery(repository.QuerySpec[Order]{
    ID:        "orders.find",
    Module:    "example.com/shop/orders",
    Statement: "SELECT id, quantity FROM orders WHERE id = $1",
    MaxRows:   1,
    Decode: func(row repository.Scanner) (Order, error) {
        var order Order
        err := row.Scan(&order.ID, &order.Quantity)
        return order, err
    },
})
if err != nil {
    return err
}

order, err := findOrder.One(ctx, queries, orderID)
```

`One` requires exactly one result, `Optional` accepts zero or one, and `List`
preserves driver order while enforcing an explicit in-memory bound. Their
sentinel errors work with `errors.Is`. Every path closes rows and reports
iteration and close failures. Failures identify the stable query ID without
including SQL text or argument values.

SQL, placeholders, indexes, ordering, and database-side result limits remain
dialect/application concerns. `MaxRows` is a final memory-safety boundary, not
a replacement for a `LIMIT` clause. The same query can run against a pool or
the transaction-owned executor supplied to `Manager.Within`.
