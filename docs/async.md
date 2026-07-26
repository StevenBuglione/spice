# Bounded asynchronous execution

`async.Executor` is an instance-owned, lifecycle-scoped alternative to
unbounded `go` statements:

```go
executor, err := async.NewExecutor(applicationContext, 16, taskObserver)
err = executor.Submit(admissionContext, async.Definition{
    ID:     "orders.SendConfirmation",
    Module: "example.com/shop/orders",
}, sendConfirmation)
```

The execution context belongs to the application and is supplied to every
accepted task. The admission context bounds how long a producer waits for a
slot. At most the configured number of task goroutines exist; `Submit` applies
backpressure instead of creating an implicit queue.

`Shutdown` atomically closes admission and waits for accepted tasks. Concurrent
shutdown calls share one terminal result. Task failures are joined in
submission order, regardless of completion order. If the shutdown context
ends, the executor cancels task contexts and returns; tasks that ignore
cancellation may continue, which remains visible through `Done`.

A panic cannot cross an asynchronous call boundary back to the submitter. The
executor therefore contains it, reports `*async.PanicError`, and deliberately
omits the recovered value from errors and observations. Snapshots and
observers expose only stable task/module identity and bounded result metadata.

Generated applications can provide `Shutdown` as a lifecycle cleanup. The
executor starts no worker, scheduler, or maintenance goroutine during
construction; its only non-task goroutine is created when explicit shutdown
needs a context-selectable wait.
