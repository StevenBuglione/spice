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
and bounded operation results; keys and values are never included. The
reviewed opt-in
[`starter-redis`](https://github.com/spice-framework/starter-redis) module
supplies a bounded typed JSON implementation of the same contract. Other
distributed stores remain separate integrations.

The compiler recognizes explicit cacheable typed HTTP reads:

```go
// @Get("/products/{id}")
// @cache.Cacheable(name="products.by-id")
func (*Products) Product(
    context.Context,
    ProductRequest,
) (ProductResponse, error)
```

The request DTO must be an exported comparable named struct value. Cacheable
routes must be typed `GET` reads with response values and cannot currently own
a transaction or authorization policy. This is a fail-closed boundary:
principal-aware caching requires an explicit principal-bearing key contract,
and mutating requests are never cached implicitly. The immutable compiler IR
contains the stable cache/route/module identity and exact key/value types.
Direct bounded-memory construction, typed configuration for capacity/TTL, and
generated route wrapping use ordinary inspectable Go. Each cache contributes:

- `spice.cache.<name>.capacity`, default `256`;
- `spice.cache.<name>.ttl`, default `5m`, where `0s` disables expiration.

The generated environment names use
`SPICE_CACHE_<NORMALIZED_NAME>_CAPACITY` and
`SPICE_CACHE_<NORMALIZED_NAME>_TTL`. Names that would normalize to the same
environment prefix fail at compile time, as do user configuration keys or
environment mappings that collide with generated properties. Capacity must fit
a positive platform `int`; TTL cannot be negative.

Generated route logic validates and binds the request before lookup, returns a
hit without invoking the controller, and stores only a successful controller
response. Lookup and put failures use the normal generated RFC 9457 error path.
`ApplicationOptions.CacheClock` and `CacheObservers` are explicit test and
observability seams. A cache construction failure participates in reverse
provider rollback.
