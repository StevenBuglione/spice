# External messaging

The public `messaging` package defines the transport-neutral boundary used by
opt-in Kafka, RabbitMQ, or other broker starters. It does not connect to a
broker, start a goroutine, or choose retry/dead-letter policy.

`messaging.Message` is immutable and bounded. It contains a caller-owned
idempotency ID, logical topic, optional ordering key, normalized content type,
defensively copied payload, deterministic portable headers, and UTC occurrence
time. Payloads are limited to 1 MiB and headers to 8 KiB so an application
cannot accidentally depend on an unbounded broker-specific envelope.

Producers depend on the narrow interface:

```go
type Publisher interface {
	Publish(context.Context, messaging.Message) error
}
```

The transaction-owned outbox can adapt its immutable messages to this
publisher. A transport starter must preserve the ID as its downstream
idempotency key when supported and must honor caller cancellation.

Consumers receive a `messaging.Delivery` with payload-free attempt/consumer
metadata and a transport-owned settlement. Copies share race-safe settlement
state. `Handle` invokes one handler and permits exactly one settlement:

| Outcome | Disposition |
| --- | --- |
| handler succeeds | acknowledge |
| handler returns an error or panics | retry |
| caller context is already canceled | reject |

Settlement uses `context.WithoutCancel` so a broker can finish a bounded
acknowledgement after handler cancellation. A starter remains responsible for
adding its own finite settlement deadline. Handler and settlement failures are
joined so neither is hidden.

This is deliberately not JMS. Ordering, consumer groups, partitions,
exchanges, routing keys, dead letters, exactly-once claims, and broker
transactions are starter/application policy and must be explicit. Package
presence never creates a connection or listener.

## Kafka

The independently versioned
[`github.com/spice-framework/starter-kafka`](https://github.com/spice-framework/starter-kafka)
module is the first reviewed transport. Install it through ordinary Go modules:

```text
go get github.com/spice-framework/starter-kafka@latest
```

`Open` constructs a caller-owned franz-go v1.21 client without network I/O and
returns lifecycle cleanup. It requires verified TLS 1.2+ and authentication by
default; plaintext and unauthenticated local development require separate
explicit flags. The producer retains franz-go's idempotent behavior, requires
all in-sync replica acknowledgements, publishes synchronously with caller
cancellation, and emits payload-free observations.

The starter maps `Message.Topic`, `Key`, and payload directly and adds reserved
`content-type`, `spice-message-id`, and `spice-occurred-at` headers. Application
headers may not shadow those names. Shutdown first flushes with the supplied
lifecycle context and then closes the client exactly once.

`OpenConsumer` constructs an instance-owned consumer group with the same secure
transport defaults. `Run` uses bounded polling and sequential handling: an
acknowledged delivery commits synchronously, a retryable handler failure remains
uncommitted and returns control for caller-owned backoff/restart, and an explicit
rejection commits so poison messages do not loop silently. Kafka records must
carry the reserved metadata written by the producer. Malformed envelopes fail
closed without committing. Consumer observers receive only group, topic,
partition, duration, and error facts.

The owning repository proves authenticated Redpanda delivery, manual commit,
restart/no-redelivery, cancellation, and cleanup. That acceptance deliberately
uses the explicit local plaintext transport switch; it does not claim a live
TLS broker handshake. Production teams own acceptance against their exact
broker distribution, version, TLS certificates, authentication mechanism,
replication/min-ISR policy, and failure topology. Retry/dead-letter routing,
transactions, generated listeners, and those deployment-specific checks remain
explicit integration work rather than a universal Kafka-platform claim.

The starter repository owns the canonical [dependency
review](https://github.com/spice-framework/starter-kafka/blob/main/docs/dependency-review.md),
[support policy](https://github.com/spice-framework/starter-kafka/blob/main/docs/support.md),
compatibility manifest, broker-acceptance contract, and hosted evidence. This
core document remains the transport-neutral ecosystem composition guide.
