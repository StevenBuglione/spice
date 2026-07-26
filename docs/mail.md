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
recipient data. The bounded test sender and planned SMTP starter implement this
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
