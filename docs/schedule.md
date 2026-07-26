# Fixed-delay scheduling

Annotate an exported method on an exact `@Bean` output:

```go
type Inventory struct{}

// @Bean
func NewInventory() *Inventory {
    return &Inventory{}
}

// @schedule.FixedDelay(delay="1m", initialDelay="5s", continueOnError=true)
func (*Inventory) Refresh(ctx context.Context) error {
    return refreshInventory(ctx)
}
```

`delay` is a required positive Go duration string. `initialDelay` is optional
and non-negative. `continueOnError` defaults to false and must be selected
explicitly when retrying a failed invocation is safe.

The compiler requires the exact non-generic, non-variadic
`func(receiver)(context.Context) error` signature and exactly one provider for
the receiver type. Aliases preserve identity. Pointer/value convenience,
interface assignability, structural context lookalikes, promoted methods,
missing providers, duplicate annotations, invalid durations, and unexported
methods all fail with source-positioned diagnostics.

Spice carries normalized durations and provider ownership through the immutable
application IR. Generated code constructs one `schedule.Scheduler` using direct
method values; it performs no reflection, package scan, annotation lookup, or
method execution during analysis. The scheduler starts after provider lifecycle
hooks and stops before them, so no scheduled work can outlive its dependencies.
Construction failures still run registered provider cleanup in reverse order.

Each job runs serially on one lifecycle-owned goroutine. The delay begins after
the previous run finishes, so a slow job never overlaps itself. A failed run
stops that job unless `ContinueOnError` is explicitly selected. Panics are
contained at the scheduling boundary, reported without the recovered value, and
always stop the job.

Generated `ApplicationOptions` adds `ScheduleContext`, `ScheduleWaiter`, and
`ScheduleObservers` only when jobs exist. By default, the scheduler preserves
construction-context values but owns cancellation through application
shutdown. Supplying `ScheduleContext` selects an explicit job lifetime.
`ScheduleWaiter` supports deterministic virtual-time tests, and observers
receive stable method and module identities without arguments or secrets.

`Shutdown` first stops timers and future runs while allowing an in-flight run
to finish. Only when the caller's shutdown context ends are current job
contexts canceled. Jobs that ignore cancellation remain visible through
`Done`. Terminal errors are aggregated in deterministic module/job order.

The generated command's cancellation path uses the same fresh bounded shutdown
context as other application components. The reusable `Application` never owns
operating-system signals. Construction starts no goroutine, reads no
environment, and installs no global scheduler or clock.

The lower-level runtime remains available for dynamic job sets:

```go
scheduler, err := schedule.New(applicationContext, jobs, waiter, observers...)
```
