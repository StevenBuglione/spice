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

Producers depend on `event.Publisher[OrderPlaced]`, preserving the exact event
payload type at compile time. Generated applications construct one immutable
topic per event contract and inject it directly. Subscriber order is stable:
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

Asynchronous delivery, retries, and the durable publication registry are
separate layers. They must not silently change this in-process contract or
claim durability before an event is committed to application-owned storage.
