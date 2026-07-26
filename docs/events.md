# Typed application events

Spice event topics are explicit generic values, not a process-global registry:

```go
topic, err := event.NewTopic(
    event.Definition{
        ID:     "orders.OrderPlaced",
        Module: "example.com/shop/orders",
    },
    []event.Subscriber[OrderPlaced]{
        {
            ID:     "inventory.Reserve",
            Module: "example.com/shop/inventory",
            Order:  10,
            Handle: inventory.Reserve,
        },
    },
    interactionObserver,
)
```

The compiler can now derive this topic contract from valid Go declarations:

```go
// @event.Listener(order=10)
func (*Inventory) Reserve(context.Context, OrderPlaced) error {
    return nil
}

// @event.Topic
func OrderEvents(*Inventory) event.Publisher[OrderPlaced] {
    panic("compile-time marker; Spice never executes this body")
}
```

The marker becomes a synthetic exact `event.Publisher[OrderPlaced]` provider.
Its parameters are ordinary exact dependencies on listener-owner providers.
The compiler validates method signatures, provider ownership, exported payload
identity, unique topic selection, module ownership, deterministic order, and
the resulting provider graph. The immutable IR is available now; direct
generated `event.NewTopic` construction is the next bounded generation slice.

Producers depend on `event.Publisher[OrderPlaced]`, preserving the exact event
payload type at compile time. Explicitly assembled topics can be injected
directly today, and generated construction consumes the compiled metadata in
the following slice. Subscriber order is stable:
lower explicit order first, then module import path and stable subscriber ID.
The constructor copies its inputs, starts no goroutine, scans no package, and
installs no global state.

`Publish` is synchronous and fail-fast. It uses the caller's context and
goroutine, stops before the next subscriber when cancellation or an error is
observed, and wraps failures with event/subscriber identity. A handler panic is
reported to observers and re-raised with its original value. Applications
therefore retain ordinary Go control flow and can decide where an event belongs
relative to a transaction commit.

Observers receive publisher module, subscriber module, stable identities,
order, duration, error, and panic state without the event payload. They can
enrich each handler context and finish in reverse nesting order. Raw payloads
are deliberately excluded from the observation contract.

Asynchronous delivery, retries, and durable publication are separate layers.
The transactional protocol and at-least-once dispatcher are documented in
[`outbox.md`](outbox.md). They do not change this in-process contract or claim
durability before an event is committed to application-owned storage.
