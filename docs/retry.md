# Bounded retries

Spice retries are explicit around a complete operation:

```go
err := retry.Run(ctx, retry.Policy{
    ID:             "payments.Charge",
    Module:         "example.com/shop/payments",
    MaxAttempts:    3,
    InitialBackoff: 100 * time.Millisecond,
    MaxBackoff:     time.Second,
    Retryable: func(err error) bool {
        return errors.Is(err, ErrTemporarilyUnavailable)
    },
}, charge)
```

Policies are finite. More than one attempt requires an explicit classifier, so
Spice never assumes that arbitrary errors or non-idempotent operations are safe
to replay. The default multiplier is two and backoff is capped without integer
overflow. Jitter is caller-supplied and range-checked; deterministic execution
is the default.

`Run` handles error-only operations and generic `Do` returns a typed value.
Both stop on a non-retryable error, context cancellation, panic, invalid jitter,
or exhaustion. Exhaustion returns `*retry.ExhaustedError` and preserves the
last cause through `errors.Is`/`errors.As`. Cancellation during waiting
preserves both the last operation error and the context cause.

Waiting uses a context-aware timer unless the policy provides an explicit
waiter for virtual time or an application scheduler. Each completed attempt can
be observed with stable operation/module identity, duration, error, next
backoff, and panic state. The runtime starts no goroutine and uses no global
random source, timer registry, or retry policy.

Retries do not create idempotency. Database transactions, message publication,
and outbound requests must opt in only after their entire operation has a
reviewed replay contract.
