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

## OpenTelemetry starter

`starter/otel` adapts the same generated route seam to the stable OpenTelemetry
Go trace and metric APIs. Applications must pass their own tracer and meter
providers:

```go
telemetry, err := spiceotel.NewHTTPObserver(spiceotel.Options{
    TracerProvider: tracerProvider,
    MeterProvider:  meterProvider,
})
if err != nil {
    return err
}

application, err := generated.NewApplicationWithOptions(ctx, generated.ApplicationOptions{
    HTTPObservers: []web.HTTPObserver{telemetry},
})
```

Every request creates a server span named from the method and route template.
Spans include stable route ID, module, method, template, response status, and
panic state. The starter records request count, active requests, duration, and
response body size with the same bounded generated labels.

The starter does not install global OpenTelemetry providers, select an
exporter, read environment variables, or contact a collector. Applications own
provider/exporter construction and shutdown deadlines. See the
[dependency review](dependency-reviews/opentelemetry-go.md).
