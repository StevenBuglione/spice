# Typed cache contracts

Generated cache decorators depend on `cache.Store[K, V]`. The built-in memory
implementation is bounded, typed, and instance-owned:

```go
products, err := cache.NewMemory[string, Product](
    cache.Definition{
        ID:     "products.by-id",
        Module: "example.com/shop/products",
    },
    10_000,
    nil,
    cacheObserver,
)
```

`Put` accepts an explicit TTL; zero means no expiration and negative values are
rejected. `Get` removes expired values and updates least-recently-used order.
Inserting beyond capacity evicts one least-recently-used value. `Delete` is
idempotent, and `PurgeExpired` performs explicit bulk maintenance.

No cleanup goroutine, global registry, serializer, or wall-clock override is
hidden in the package. Applications may inject a clock for deterministic tests.
All operations honor cancellation before mutating state and are safe for
concurrent use.

Snapshots expose aggregate hit, miss, put, delete, eviction, expiration, and
size counters. Observations contain only compiler-owned cache/module identity
and bounded operation results; keys and values are never included. Redis and
other distributed stores remain opt-in implementations of the same contract.
