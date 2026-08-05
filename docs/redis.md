# Redis starter

[`github.com/spice-framework/starter-redis`](https://github.com/spice-framework/starter-redis)
is an independently versioned, opt-in integration built on the official
`go-redis` client. Importing it or adding its dependency never activates it.
Its explicit-constructor manifest contributes `cache.redis` and `data.redis`
capabilities and pins the reviewed dependency version.

## Client ownership and security

`Open` validates configuration and creates a pool without network I/O:

```go
client, cleanup, err := redis.Open(redis.Options{
    URL:        "rediss://default:secret@cache.example.test:6380/0",
    ClientName: "orders-service",
})
```

`rediss` with TLS 1.2 or newer and authenticated access are the defaults.
Plaintext `redis://` and missing credentials require the independent
`AllowInsecure` and `AllowUnauthenticated` opt-ins intended for controlled
local environments. URLs must identify one standalone server and database;
query options, fragments, Unix sockets, and ambiguous hosts are rejected.
Errors never contain the URL or credentials.

Spice fixes protocol, retry, timeout, FIFO pool, connection-limit, idle, and
lifetime defaults. This avoids CPU-dependent pool sizing and unbounded
connections. Each network operation uses its caller's context.
`Open` returns an exact `lifecycle.Cleanup`; generated applications can
register it immediately for rollback and reverse shutdown. `Ping` is explicit,
so construction never performs hidden network access.

## Typed distributed cache

`NewJSONStore[V]` implements `cache.Store[string,V]`:

```go
orders, err := redis.NewJSONStore[Order](
    client,
    redis.StoreOptions{
        Definition: cache.Definition{
            ID:     "orders.by-id",
            Module: "example.com/shop/orders",
        },
        Prefix:        "orders-by-id",
        MaxValueBytes: 1 << 20,
    },
    cacheObserver,
)
```

The prefix is required and prevents accidental cross-cache key collisions.
Keys and prefixes are bounded printable identities. `Get` uses a bounded
`GETRANGE`, so an oversized remote value is rejected without requesting its
full payload. `Put` rejects negative TTLs, unsupported JSON values, and encoded
values over the configured limit. Zero TTL means no expiration. `Delete` is
idempotent.

Local snapshots count successful hits, misses, puts, and effective deletes.
Redis owns distributed size, eviction, and expiration cardinality, so those
snapshot fields remain zero. Observations contain only cache and module
identity plus bounded operation metadata—never keys, values, URLs, or
credentials.

## Integration verification

The tagged integration test requires `SPICE_REDIS_TEST_URL`. It exercises a
real server, expiration, typed encoding, cancellation, observations, and
cleanup. The reviewed test image is
`redis:8.4.0-alpine@sha256:4eec4565e45aa0b3966554c866bc73211e281b0b3d89fe9a33c982e6faca809d`.

```text
go test -tags=integration -race -shuffle=on -count=1 ./...
```

Production credentials and endpoints belong in typed, redacted application
configuration. They must never be annotation arguments or committed starter
manifests.
