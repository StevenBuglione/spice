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

## Kafka producer

`starter/kafka` is the first reviewed transport. `Open` constructs a
caller-owned franz-go v1.21 client without network I/O and returns lifecycle
cleanup. It requires verified TLS 1.2+ and authentication by default; plaintext
and unauthenticated local development require separate explicit flags. The
producer retains franz-go's idempotent behavior, requires all in-sync replica
acknowledgements, publishes synchronously with caller cancellation, and emits
payload-free observations.

The starter maps `Message.Topic`, `Key`, and payload directly and adds reserved
`content-type`, `spice-message-id`, and `spice-occurred-at` headers. Application
headers may not shadow those names. Shutdown first flushes with the supplied
lifecycle context and then closes the client exactly once.

Consumer groups, retry/dead-letter policy, transactions, and real-broker
acceptance remain explicit follow-up work; the producer manifest is therefore
classified as an integration rather than a complete Kafka platform.
