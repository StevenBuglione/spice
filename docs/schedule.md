# Fixed-delay scheduling

Generated applications construct an immutable `schedule.Scheduler` from typed
job declarations:

```go
scheduler, err := schedule.New(applicationContext, []schedule.Job{{
    Definition: schedule.Definition{
        ID:     "inventory.Refresh",
        Module: "example.com/shop/inventory",
    },
    InitialDelay: time.Second,
    Delay:        time.Minute,
    Run:          refreshInventory,
}}, nil, jobObserver)
```

Each job runs serially on one lifecycle-owned goroutine. The delay begins after
the previous run finishes, so a slow job never overlaps itself. A failed run
stops that job unless `ContinueOnError` is explicitly selected. Panics are
contained at the scheduling boundary, reported without the recovered value, and
always stop the job.

`Start(context.Context) error` matches the generated lifecycle hook contract.
The start context bounds only that transition; job contexts come from the
caller-owned application lifetime supplied to `New`.

`Shutdown` first stops timers and future runs while allowing an in-flight run
to finish. Only when the caller's shutdown context ends are current job
contexts canceled. Jobs that ignore cancellation remain visible through
`Done`. Terminal errors are aggregated in deterministic module/job order.

The default waiter uses context-aware timers. Tests and virtual-time
environments may inject a waiter. Construction starts no goroutine, reads no
environment, and installs no global scheduler or clock.
