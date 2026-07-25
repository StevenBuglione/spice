# Management endpoints

The public `management` package provides an opt-in, standard-library-first
health subsystem. It has no global registry and mounts no hidden routes.

Each `management.Check` has a stable name, optional module import path, one or
more explicit groups, and a caller-owned `func(context.Context) error` probe.
`management.New` rejects invalid names, nil probes, missing or unknown groups,
and duplicate names within a group. It copies and sorts checks so report order
does not depend on registration order.

Reports expose only `UP` or `DOWN`, check name, and module ownership. The
underlying error is never included in JSON. Canceled request contexts mark
unexecuted probes down. Empty groups are up, which makes optional liveness or
readiness groups explicit rather than synthesized from unrelated dependencies.

`management.LifecycleChecks` adapts generated `Application.State()`:

- constructed, starting, ready, and stopping are live;
- only ready accepts traffic;
- stopped, failed, and invalid are not live.

An isolated handler serves:

| Method and path | Contract |
|---|---|
| `GET /actuator/health` | broad health report |
| `GET /actuator/health/liveness` | process liveness report |
| `GET /actuator/health/readiness` | traffic readiness report |
| `GET /actuator/info` | caller-owned copied string metadata |
| `GET /actuator/metrics` | generated-route HTTP metrics when a collector is supplied |

Down reports use HTTP 503; up reports and info use HTTP 200. Responses use the
same secure JSON writer as generated controllers. The default base path is
`/actuator`; a custom path must be a clean absolute path below `/`.

`management.HTTPMetrics` implements `web.HTTPObserver`. It records requests,
in-flight work, status counts, bytes, total/max duration, and panics per stable
generated route. Route labels contain only compiler-generated symbol, module,
method, and pattern values—never raw paths or other client-controlled
cardinality. The zero value is usable, snapshots are immutable and sorted, and
completion is idempotent. A hard 4,096-route cap bounds memory; excess
observations are counted in `dropped_observations`.

Mount the handler explicitly on the application's mux:

```go
checks, err := management.LifecycleChecks(
    "application",
    "example.com/commerce",
    application.State,
)
if err != nil {
    return err
}
manager, err := management.New(checks...)
if err != nil {
    return err
}
handler, err := management.NewHandler(management.HandlerOptions{
    Manager: manager,
    Metrics: metrics,
    Info: map[string]string{
        "name":    "commerce",
        "version": buildVersion,
    },
})
if err != nil {
    return err
}
mux.Handle(handler.Pattern(), handler)
```

Pass the same collector to the generated application:

```go
application, err := spicegen.NewApplicationWithOptions(ctx, spicegen.ApplicationOptions{
    HTTPObservers: []web.HTTPObserver{metrics},
})
```

External dependency checks normally belong to both `health` and `readiness`,
not `liveness`; a database outage should stop traffic without asking an
orchestrator to restart an otherwise healthy process.
