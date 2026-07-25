# Configuration

Spice configuration is generated and reflection-free. The runtime resolves raw
values and provenance; generated Go performs typed struct construction.

## Declaration contract

`@Configuration` targets a defined, non-generic named struct and accepts an
optional named `prefix` string. Every exported field must declare an explicit
`spice` tag or opt out with `spice:"-"`. Untagged private fields are ignored;
embedded fields and tagged private fields are rejected because generated code
cannot initialize them safely.

```go
// @Configuration(prefix="server")
type Server struct {
    Port     int           `spice:"port,default=8080,env=SERVER_PORT"`
    Timeout time.Duration `spice:"timeout,default=5s"`
    Token   string        `spice:"token,required,secret,env=SERVER_TOKEN"`
}
```

Tag options are `default=<value>`, `env=<NAME>`, `required`, and `secret`.
Strings, Booleans, signed integers (including named forms and aliases), and
`time.Duration` are supported. The compiler validates keys, scalar defaults,
integer widths, environment names, duplicate properties, duplicate environment
variables, and module ownership before provider-graph construction.

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

## JSON files and profiles

`config.NewJSONSource` uses `os.Root` to constrain all reads to one explicit
directory. With base name `application` and active profiles `production`,
`us-east`, it applies:

1. `application.json`;
2. `application-production.json`;
3. `application-us-east.json`.

The base file can be required or optional; profile files are always optional.
Each file is limited to 1 MiB by default and can set a different positive
`MaxBytes`.

```go
files, err := config.NewJSONSource(
    "files",
    configurationDirectory,
    "application",
    config.JSONOptions{Required: true},
)
if err != nil {
    return err
}
snapshot, err := config.Resolve(
    ctx,
    schema,
    config.Options{Profiles: []string{"production"}},
    files,
    environment, // later source: environment wins over files
)
```

Nested objects flatten to dotted keys. Strings, Booleans, and JSON numbers are
supported scalar inputs. Spice rejects duplicate object keys, collisions
between nested and dotted keys, arrays, nulls, invalid configuration-key
identities, non-object roots, trailing JSON values, oversized files, and
rooted-path escapes.

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

The hand-written decoder above demonstrates the generated contract. The
compiler already retains the exact field types and property metadata in the
immutable application IR; emitting and injecting these binders is the next
configuration slice.

## Environment and secrets

The environment source checks only keys present in the generated schema. With
prefix `SHOP_`, `server.port` maps to `SHOP_SERVER_PORT`. A property's explicit
`Environment` name overrides that mapping. Collisions fail.

`Snapshot.Lookup` and typed accessors intentionally expose values to generated
application code. `Snapshot.Redacted` and `Snapshot.String` are the safe
logging surfaces and replace every secret with `<redacted>`. Resolver and
scalar errors never include raw values.
