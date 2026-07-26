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

Generated platform features also contribute typed properties. A
`@cache.Cacheable(name="products.by-id")` route adds:

- `spice.cache.products.by-id.capacity` /
  `SPICE_CACHE_PRODUCTS_BY_ID_CAPACITY`, default `256`;
- `spice.cache.products.by-id.ttl` /
  `SPICE_CACHE_PRODUCTS_BY_ID_TTL`, default `5m`.

Generated property keys and environment variables are framework-owned.
Collisions with `@Configuration` fields fail before rendering. Cache capacity
must fit a positive platform `int`; a zero TTL disables expiration and a
negative TTL fails application construction.

Applications with `@async.Execute` methods also receive the framework-owned
integer property `spice.async.max-concurrency`, mapped conventionally to
`SPICE_ASYNC_MAX_CONCURRENCY`. It defaults to 16 and must fit a positive
platform `int`; invalid values fail construction before any task is accepted.

Applications may explicitly expose
`@management.Enable(expose=["configprops"])`. The generated report uses this
same schema and resolved snapshot to show deterministic key, kind, module,
source, default, and resolution metadata. Secret values are always
`<redacted>`; they are never copied into the management report.
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

The compiler retains exact field types and property metadata in the immutable
application IR. Each configuration struct becomes an exact-type provider node,
so ordinary `@Bean` parameters and `@Application` roots can consume it.
Generated binders emit direct scalar access, named-type conversion, and
integer-width checks before constructing the struct.

Configured generated applications expose:

```go
type ApplicationOptions struct {
    Profiles                  []string
    Sources                   []config.Source
    AllowUnknownConfiguration bool
    Observers                 []lifecycle.Observer
}
```

`NewApplicationWithOptions` resolves all configuration once, then constructs
providers in dependency order. `NewApplication(ctx, observers...)` remains the
concise compatibility entrypoint and uses schema defaults only. Neither
constructor reads files, environment variables, signals, or the network unless
the caller explicitly supplies a source that does so. `ConfigurationSchema`
returns fresh, validated generated metadata without a mutable global registry.

## Generated command convention

An `@Application` target always includes the typed
`spice.shutdown-timeout` property. Its default is `10s`, and its explicit
environment name is `SPICE_SHUTDOWN_TIMEOUT`.

The generated `Main` helper opts into `config.OSEnvironment("SPICE_")` and
passes that source to construction. Reusable `NewApplication` and
`NewApplicationWithOptions` do not read the process environment: callers can
provide their own ordered sources, profiles, and unknown-key policy.
`RunCommand` also exposes the `ApplicationOptions` seam for tests and embedded
commands. Ports, credentials, database URLs, secrets, and environment-specific
timeouts remain configuration properties rather than annotation arguments.

## Environment and secrets

The environment source checks only keys present in the generated schema. With
prefix `SHOP_`, `server.port` maps to `SHOP_SERVER_PORT`. A property's explicit
`Environment` name overrides that mapping. Collisions fail.

`Snapshot.Lookup` and typed accessors intentionally expose values to generated
application code. `Snapshot.Redacted` and `Snapshot.String` are the safe
logging surfaces and replace every secret with `<redacted>`. Resolver and
scalar errors never include raw values.
