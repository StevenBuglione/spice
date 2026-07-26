# Transactional outbox

`event/outbox` defines the storage and delivery protocol for durable,
at-least-once event publication. It does not claim durability without a
database-backed `Store`.

Create the immutable serialized message before application work commits, then
enqueue it through the same transaction executor:

```go
err := transactions.Within(ctx, boundary, func(ctx context.Context, tx data.Executor) error {
    if err := orders.Save(ctx, tx, order); err != nil {
        return err
    }
    message, err := outbox.NewMessage(outbox.MessageSpec{
        ID:          command.ID,
        Topic:       "orders.OrderPlaced",
        Module:      "example.com/shop/orders",
        ContentType: "application/json",
        Payload:     payload,
        OccurredAt:  clock(),
    })
    if err != nil {
        return err
    }
    return store.Enqueue(ctx, tx, message)
})
```

Message IDs are caller-owned idempotency keys. Payloads are copied and limited
to 1 MiB; metadata is validated and bounded. A store atomically claims the
oldest available messages ordered by occurrence time and ID, returning opaque
lease receipts and one-based attempts.

`Dispatcher.RunOnce` publishes a bounded batch sequentially, completes
successful leases, and releases failed publishes with an explicit delay. It
starts no goroutine; applications can invoke it through Spice scheduling or
their own lifecycle loop. Cancellation stops before the next message.

Delivery is deliberately at least once. If publishing succeeds and completion
fails, the lease eventually expires and the message can be published again.
Transport publishers must therefore use the message ID as their downstream
idempotency key.

Publisher panics are observed and re-raised; the lease is left to expire.
Observations contain topic/module/attempt and outcome metadata, never payloads
or lease receipts. The forthcoming SQL store supplies atomic statement
execution and affected-row checks while leaving locking syntax dialect-owned.
