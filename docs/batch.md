# Restartable batch jobs

The `batch` package executes a finite, ordered set of named steps while an
explicit store owns attempt and checkpoint transitions. It is intended for
restartable imports, reconciliation, maintenance, and other application-owned
jobs—not for hiding arbitrary work behind a global scheduler or container.

Define a stable module-owned job:

```go
job, err := batch.NewJob(
    batch.Definition{
        ID:     "orders.import",
        Module: "example.com/shop/orders",
    },
    []batch.StepSpec{
        {ID: "extract", Run: extractOrders},
        {ID: "load", Run: loadOrders},
    },
)
```

Job, module, instance, and step identities are bounded printable metadata.
Definitions are immutable, step IDs are unique, and callbacks have the exact
`func(context.Context) error` contract. The caller supplies an idempotent
instance identity such as a source revision or business date.

```go
store, err := batch.NewMemoryStore(1_000)
runner, err := batch.NewRunner(store, failureContext, observers...)
result, err := runner.Run(ctx, job, "2026-07-26")
```

`Run` atomically begins an attempt, validates that stored checkpoints are an
exact prefix of the current definition, skips that prefix, and executes the
remaining steps serially. Each successful step is checkpointed before the next
one runs. Completion is accepted only after every step is durable in the
store.

On a returned error, cancellation, checkpoint failure, completion failure, or
contained panic, the runner asks the caller's `ContextFactory` for a fresh
bounded context and records a failure transition. This avoids trying to persist
restart state with the canceled step context. Transition failure is joined
with the original error. Panic values, application values, and instance
identities are not included in observations or failure metadata.

Only one attempt for a definition and instance may be active. `ErrAlreadyRunning`
identifies concurrent ownership, `ErrStaleAttempt` protects transitions from
old attempts, and `ErrDefinitionChanged` prevents a revised ordered definition
from silently reusing incompatible checkpoints.

## In-process store

`MemoryStore` is concurrency-safe and capacity-bounded. It preserves exact
restart state for the lifetime of one process and is useful for tests,
development, and jobs whose checkpoints do not need to survive a restart.
Capacity counts retained instances, not active attempts. `Delete` removes only
inactive state and releases capacity.

`Snapshot` provides a defensive diagnostic view without returning the caller's
instance identity. It exposes the one-based attempt number, completed prefix,
active/completed state, and bounded last-failure classification.

The in-process store is deliberately not presented as durable storage. A
production store must implement the four atomic `Store` transitions in its
database and enforce the same exact-definition and stale-attempt invariants.
The PostgreSQL implementation and generated/scaffolded batch registration
remain follow-up slices.

The complete restart flow is executable as the `ExampleMemoryStore_restart`
example in the `batch` package.
