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

The in-process store is deliberately not presented as durable storage.

## SQL store

`SQLStore` adapts the same contract to a caller-owned `database/sql` executor.
Construction performs no database operation. The database dialect supplies
four fixed trusted statements; request values are always bound arguments.

The begin statement atomically inserts, resumes, or observes an instance and
returns one of the exported `SQLBeginOutcome...` values, a positive signed-SQL
attempt number, and the completed step IDs as JSON. The transition statements
must affect exactly one active attempt. Zero rows become `ErrStaleAttempt`;
multiple rows and unsupported affected-row counts fail closed.

`SQLStoreOptions.AttemptLease` is explicit and bounded to 24 hours. Begin
claims the lease and every checkpoint renews it. A crashed runner can therefore
be superseded after expiry. Steps that can exceed the lease must be idempotent:
another runner may resume them, giving the durable execution contract
at-least-once step semantics. The injected clock is a deterministic test seam;
nil selects `time.Now`.

The SQL store validates every reconstructed completed prefix before returning
it to the runner. Database errors are wrapped without adding the instance
identity. Dialect statements still own transactionality, locking, schema, and
lease comparison details; the PostgreSQL schema and reviewed statements remain
the next integration slice.

The complete restart flow is executable as the `ExampleMemoryStore_restart`
example in the `batch` package.
