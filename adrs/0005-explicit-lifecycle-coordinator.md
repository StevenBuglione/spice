# ADR 0005: Explicit Generated Lifecycle Coordination

Status: Accepted

## Decision

Generated applications use the small public `lifecycle.Coordinator` for generic
state transitions and ordered callback execution. Generated code remains
responsible for concrete provider calls, concrete receiver method calls, stable
provider IDs, and dependency order.

The coordinator accepts only explicit typed callbacks:

- construction cleanups are registered immediately after provider success;
- start hooks run serially in dependency-first generated order;
- a start failure stops only previously successful hooks in reverse order;
- construction cleanups always run in reverse construction order;
- normal stop reverses successful starts, then reverses cleanups;
- every rollback/stop callback is attempted and failures are deterministically
  joined while preserving `errors.Is` and `errors.As`;
- stop is idempotent and concurrent stop callers wait or honor their own
  cancellation;
- concurrent start/stop and other invalid state transitions return a typed
  error;
- caller-owned contexts are passed unchanged to callbacks.

The observable states are constructed, starting, ready, stopping, stopped, and
failed. Construction abort and startup rollback end in failed; normal stop ends
in stopped.

## Boundaries

The coordinator does not:

- discover providers or hooks;
- resolve types or dependency order;
- scan packages or use reflection;
- own process signals, timeouts, logging, or exit behavior;
- create background contexts;
- recover panics;
- store a global registry or application singleton.

These boundaries keep generated application behavior ordinary, inspectable Go
while centralizing the concurrency-sensitive state machine.
