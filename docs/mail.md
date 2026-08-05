# Mail Messages

The `mail` package provides the immutable message boundary shared by test and
production delivery transports. It performs no network access and owns no
global client.

## Construct a message

Callers provide identity and time explicitly:

```go
message, err := mail.NewMessage(mail.MessageSpec{
	ID:       "order-41@example.com",
	Date:     time.Now(),
	From:     "Orders <orders@example.com>",
	To:       []string{"customer@example.com"},
	Subject:  "Order 41 is ready",
	TextBody: "Your order is ready.",
	HTMLBody: "<p>Your order is ready.</p>",
	Attachments: []mail.AttachmentSpec{{
		Filename:    "receipt.txt",
		ContentType: "text/plain; charset=utf-8",
		Data:        receipt,
	}},
})
```

Production code should normally obtain the date and message ID from explicit,
application-owned dependencies. The constructor does not read a clock,
hostname, environment variable, or random source.

`Message.EnvelopeFrom`, `Message.Recipients`, and `Message.Bytes` expose the
normalized SMTP envelope and complete serialized MIME. Slice and byte accessors
return defensive copies. Constructor inputs are copied as well.

## MIME and envelope behavior

- To, Cc, and Bcc recipients retain stable input order. Duplicate envelope
  addresses are delivered once at the first occurrence.
- Bcc recipients are part of the delivery envelope but never appear in the
  serialized headers.
- A text and HTML body becomes `multipart/alternative`. Attachments add a
  `multipart/mixed` envelope around the body.
- MIME boundaries derive deterministically from the caller-owned message ID and
  avoid all body and attachment content.
- Bodies use quoted-printable transfer encoding, attachments use 76-column
  base64, and all serialized lines use CRLF.
- Header injection, path-like filenames, malformed addresses, malformed media
  types, and multipart attachments are rejected.

HTML bodies are serialized as supplied. The mail package does not sanitize
application HTML; render untrusted values through an escaping template before
constructing a message.

## Bounds

Construction fails before returning a message when a contract exceeds:

| Input | Limit |
|---|---:|
| Envelope recipients | 100 |
| Subject | 256 bytes |
| Text body | 1 MiB |
| HTML body | 1 MiB |
| Attachments | 16 |
| One attachment | 10 MiB |
| All attachments | 16 MiB |
| Serialized message | 25 MiB |

Addresses, message IDs, and filenames have additional bounded validation.
Envelope addresses are ASCII-only until an explicit SMTPUTF8 transport contract
exists.

## Delivery boundary

Transports implement:

```go
type Sender interface {
	Send(context.Context, mail.Message) error
}
```

The context belongs to the caller. Transport implementations own connection
timeouts, cancellation, retry classification, TLS policy, and bounded
observations. They must not log message bodies, attachments, credentials, or
recipient data. The bounded test sender and secure SMTP starter implement this
same instance-owned contract.

## Test transport

`mail/mailtest` provides an instance-owned `mail.Sender` for unit tests and
local reference applications:

```go
sender, err := mailtest.New(mailtest.Config{
	Capacity: 100,
	Failures: []error{temporaryFailure, nil},
	Observer: func(ctx context.Context, observation mailtest.Observation) {
		// Payload-free attempt metadata only.
	},
})
if err != nil {
	return err
}
```

`Failures` is a defensive one-based attempt plan: a non-nil entry fails that
accepted attempt, and an omitted or nil entry succeeds. This makes retry tests
deterministic without a callback running inside sender locks. A context already
canceled on entry returns immediately without consuming capacity.

The sender retains at most `Capacity` accepted attempts. Once full, `Send`
returns a typed `mailtest.CapacityError` that matches
`mailtest.ErrCapacityExceeded`; it never drops a message silently. The
monotonic attempt count includes an explicit capacity rejection, while retained
history remains bounded.

`Attempts` exposes delivered, configured-failure, and late-cancellation
outcomes. `Messages` returns successful deliveries only. Each message snapshot
provides defensive access to:

- envelope sender and stable recipients;
- decoded subject;
- CRLF-normalized text and HTML bodies;
- ordered MIME attachment filenames, content types, and bytes;
- the exact serialized MIME.

Returned slices and bytes are deep copies. Concurrent sends and inspection are
race-safe. Built-in observations contain only attempt number, message ID, and
outcome; they exclude recipient data, subject, bodies, attachments, and error
text. The optional observer runs synchronously after the sender unlocks, so it
may safely inspect the same sender.

## Secure SMTP transport

[`github.com/spice-framework/starter-smtp`](https://github.com/spice-framework/starter-smtp)
is the independently versioned production transport. Add it through the normal
Go module graph, import it as `smtp`, and construct an instance explicitly. It
performs no network work during construction and requires verified TLS:

```go
sender, err := smtp.New(smtp.Config{
	Address:        "smtp.example.com:587",
	ServerName:     "smtp.example.com",
	Mode:           smtp.TLSModeStartTLS,
	Username:       username,
	Password:       password,
	Timeout:        10 * time.Second,
	MaxAttempts:    3,
	InitialBackoff: 250 * time.Millisecond,
	MaxBackoff:     2 * time.Second,
	Observer: func(ctx context.Context, observation smtp.Observation) {
		// Message ID, attempt, outcome, stage, status, duration, and backoff only.
	},
})
if err != nil {
	return err
}
if err := sender.Send(ctx, message); err != nil {
	return err
}
```

The zero TLS mode means required STARTTLS. `TLSModeImplicitTLS` is available
for services that negotiate TLS before the SMTP greeting. STARTTLS absence,
certificate verification failure, TLS below 1.2, renegotiation, cleartext
authentication, conflicting server identity, and partially specified
credentials all fail closed. A caller-supplied `tls.Config` is cloned;
disabling certificate verification is always rejected.

Each attempt owns one connection and uses the earlier of the caller's deadline
and the configured timeout. Cancellation closes an in-flight connection, so it
interrupts greeting, TLS, authentication, envelope, and body operations rather
than merely preventing the next step.

Retries are deliberately conservative. Only transient connection or SMTP 4xx
failures before `DATA` begins are eligible. Once the server accepts `DATA`, a
write or final-acceptance failure may represent a delivered message and is
never replayed automatically. `DeliveryError` exposes the stage, numeric SMTP
status, temporary classification, and replay-safety bit while keeping raw
server text, addresses, credentials, subject, bodies, and attachments out of
its public error string.

The optional observer is synchronous and receives bounded, payload-free
attempt metadata. It does not retain observations internally and it never owns
a logging or telemetry global. Applications can translate it into their chosen
structured logging, metrics, or tracing implementation.

The transport intentionally supports the stable SMTP subset in Go's
standard-library `net/smtp`: STARTTLS or implicit TLS, AUTH PLAIN after TLS, and
ordinary ASCII envelope addresses. SMTPUTF8, custom SASL mechanisms, DSN, and
automatic ambiguous-delivery replay are not claimed. The dependency and
security decision is owned by the
[`starter-smtp` dependency review](https://github.com/spice-framework/starter-smtp/blob/main/docs/dependency-review.md).
