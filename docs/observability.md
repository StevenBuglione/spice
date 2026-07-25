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
