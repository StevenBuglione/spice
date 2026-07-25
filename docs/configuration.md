# Configuration

Spice configuration is generated and reflection-free. The runtime resolves raw
values and provenance; generated Go performs typed struct construction.

## Resolution contract

`config.Resolve` applies:

1. generated schema defaults;
2. each explicit source in argument order;
3. required-key and scalar validation.

Later sources override earlier sources. Source names must be unique. Unknown
keys fail closed unless `Options.AllowUnknown` is deliberately enabled.

```go
schema := config.MustSchema(
    config.Property{
        Key:        "server.port",
        Kind:       config.KindInteger,
        Default:    "8080",
        HasDefault: true,
    },
    config.Property{
        Key:       "database.password",
        Kind:      config.KindString,
        Required:  true,
        Secret:    true,
        Module:    "example.com/shop/orders",
    },
)

environment, err := config.OSEnvironment("SHOP_")
if err != nil {
    return err
}
snapshot, err := config.Resolve(
    ctx,
    schema,
    config.Options{Profiles: []string{"production"}},
    environment,
)
```

Profiles match `^[a-z0-9][a-z0-9-]*$`, retain caller order, and are passed to
every source. File/profile expansion is implemented by the file source rather
than hidden in the resolver.

## Generated decoding

Generated binders call `RequiredString`, `Boolean`, `Integer`, `Duration`, or
`Lookup` and return an ordinary typed value. `config.Decode` then invokes typed
validators in declaration order.

```go
type Server struct {
    Port int64
}

server, err := config.Decode(ctx, snapshot, func(snapshot config.Snapshot) (Server, error) {
    port, err := snapshot.Integer("server.port")
    if err != nil {
        return Server{}, err
    }
    return Server{Port: port}, nil
})
```

The hand-written decoder above demonstrates the generated contract; application
code will consume generated binders.

## Environment and secrets

The environment source checks only keys present in the generated schema. With
prefix `SHOP_`, `server.port` maps to `SHOP_SERVER_PORT`. A property's explicit
`Environment` name overrides that mapping. Collisions fail.

`Snapshot.Lookup` and typed accessors intentionally expose values to generated
application code. `Snapshot.Redacted` and `Snapshot.String` are the safe
logging surfaces and replace every secret with `<redacted>`. Resolver and
scalar errors never include raw values.
