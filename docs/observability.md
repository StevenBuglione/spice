# Observability

Spice keeps its core observation contracts dependency-free and instance-owned.
Generated applications accept ordered lifecycle observers and HTTP observers;
route metadata always uses compiler-owned symbol IDs, module import paths,
methods, and templates rather than raw request paths.

The `observability` package adapts both seams to `log/slog`:

```go
logger := slog.New(slog.NewJSONHandler(os.Stderr, nil))

httpLogs, err := observability.NewSlogHTTPObserver(logger)
if err != nil {
    return err
}
lifecycleLogs, err := observability.NewSlogLifecycleObserver(logger)
if err != nil {
    return err
}

application, err := generated.NewApplicationWithOptions(ctx, generated.ApplicationOptions{
    HTTPObservers: []web.HTTPObserver{httpLogs},
    Observers:     []lifecycle.Observer{lifecycleLogs},
})
```

HTTP completion records contain only stable route metadata, status, byte count,
duration, and panic state. They never contain raw URLs, headers, bodies, query
values, or path values. Successful requests log at info, client failures at
warn, and server failures or panics at error.

Lifecycle records contain compiler-generated component/module ownership,
operation, phase, and an internal error on failure. No global logger or
registry is installed. Applications choose handlers, levels, redaction, and
export destinations explicitly.

Generated applications can request these adapters on their application marker:

```go
// @Application
// @observability.Logging
func Application(*Server) {}
```

The compiler records the opt-in as typed bootstrap metadata and generated
construction installs both lifecycle and HTTP logging observers in stable
order. Generated command construction/start/failure messages remain baseline
command behavior; the companion controls component lifecycle and request
records. Importing the package alone never activates logging. Callers that need
another handler or observer set can omit the annotation and use
`NewApplicationWithOptions` or `RunCommand`.

## OpenTelemetry starter

`starter/otel` adapts generated route and typed module-event seams to the
stable OpenTelemetry Go trace and metric APIs. Its manifest contributes the qualified
`@otel.Enable` application annotation. After explicitly embedding that manifest
in `.spice/starters.json`, provide the application-owned OpenTelemetry inputs
as an exact bean and enable the feature:

```go
// @Bean
func OpenTelemetryOptions(
    providers *TelemetryProviders,
) spiceotel.Options {
    return spiceotel.Options{
        TracerProvider: providers.Tracer,
        MeterProvider:  providers.Meter,
    }
}

// @Application
// @otel.Enable
func Application(*Server) {}
```

The compiler activates `spiceotel.NewHTTPObserver` only for that annotation,
validates its exact `web.HTTPObserver` output contract and required reachable
`*http.ServeMux`, and carries the feature and starter provenance into the
generation hash. Generated code constructs the observer through the ordinary
provider graph and installs it before route middleware is created. Invalid
provider inputs, a missing mux capability, or an incompatible observer output
fail before generated files are written. Importing the package, adding its
module dependency, or selecting its manifest without the annotation does
nothing.

Every request creates a server span named from the method and route template.
Spans include stable route ID, module, method, template, response status, and
panic state. The starter records request count, active requests, duration, and
response body size with the same bounded generated labels.

Typed events expose compiler-owned publisher and subscriber module identities.
`NewEventObserver` turns each synchronous delivery into one internal span plus
delivery count, active-delivery, and duration metrics. Attributes contain only
the event ID, module IDs, subscriber ID/order, and a bounded
`success`/`error`/`panic` outcome; event values and error text are never
recorded.

`NewObserver` composes the HTTP and event adapters when one caller-owned value
should observe both seams:

```go
telemetry, err := spiceotel.NewObserver(options)
if err != nil {
    return err
}
application, err := generated.NewApplicationWithOptions(ctx, generated.ApplicationOptions{
    HTTPObservers:  []web.HTTPObserver{telemetry},
    EventObservers: []event.Observer{telemetry},
})
```

Generated event topics already carry publishing and subscribing module
identity and accept `ApplicationOptions.EventObservers`; no runtime module
registry or payload reflection is involved. `@otel.Enable` continues to
compose the HTTP adapter automatically. Event observation remains an explicit
application option so applications can independently choose its sampling and
export lifecycle.

The starter does not install global OpenTelemetry providers, select an
exporter, read environment variables, or contact a collector. Applications own
provider/exporter construction and shutdown deadlines. See the
[dependency review](dependency-reviews/opentelemetry-go.md).

Applications that need custom ordering or conditional observation can omit
`@otel.Enable`, call `spiceotel.NewHTTPObserver` themselves, and pass it through
`NewApplicationWithOptions.HTTPObservers`. This remains the explicit
lower-level escape hatch.
