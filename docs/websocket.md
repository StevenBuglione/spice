# WebSocket integration

`starter/websocket` provides an explicit RFC 6455 boundary backed by
`github.com/coder/websocket`. It does not add an annotation language, global
connection hub, hidden goroutine pool, authentication scheme, or application
message protocol.

## Server

`NewHandler` returns an ordinary `http.Handler` for a generated or manually
registered Spice route:

```go
handler, err := websocketstarter.NewHandler(
    websocketstarter.ServerConfig{
        Subprotocols:    []string{"petclinic.events.v1"},
        MaxMessageBytes: 1 << 20,
        MaxConnections:  256,
    },
    func(ctx context.Context, connection *websocketstarter.Connection) error {
        messageType, payload, err := connection.Read(ctx)
        if err != nil {
            return err
        }
        return connection.Write(ctx, messageType, payload)
    },
    observer,
)
```

TLS is required by default and remains owned by the surrounding `http.Server`.
Plain HTTP development or trusted-proxy hops require an explicit
`AllowInsecure` decision. Browser origins are same-host only by default.
Cross-origin host patterns are explicit, bounded, deduplicated, and validated;
`*` is rejected. Disabling origin verification requires the separately visible
`AllowAnyOrigin` flag.

Message size defaults to 1 MiB and cannot exceed 16 MiB. Active sessions
default to 256 and cannot exceed 4096. Capacity is acquired before the upgrade,
so excess handshakes receive HTTP 503 rather than consuming an unbounded socket.
Compression is disabled by default; the opt-in mode uses no context takeover
and a bounded threshold. The caller owns the session handler and all
authentication, authorization, heartbeat, fan-out, persistence, and
application-level backpressure.

When a session finishes, the handler performs a bounded close handshake.
Lifecycle cancellation uses `going away`; application failures use a generic
internal-error close without leaking error text. If the handshake cannot finish
within the configured timeout, the socket is force-closed.

## Client

`Dial` requires a `wss://host:port/path` URL and verified TLS 1.2 or newer by
default:

```go
connection, response, cleanup, err := websocketstarter.Dial(
    ctx,
    websocketstarter.ClientConfig{
        URL:          "wss://events.example.internal:8443/owners",
        Subprotocols: []string{"petclinic.events.v1"},
        Header: http.Header{
            "Authorization": []string{"Bearer " + token},
        },
    },
    observer,
)
```

Credentials in URL user-info are rejected. Fragments, missing ports, reserved
WebSocket headers, noncanonical header names, CR/LF injection, oversized
headers, old TLS, and disabled certificate verification fail before dialing.
Caller headers and TLS configuration are defensively cloned. Environmental HTTP
proxies are not consulted. `ws://` is accepted only with
`AllowInsecure: true`, which cannot be combined with TLS configuration.

The returned `Connection` provides bounded `Read`, `Write`, `Ping`, negotiated
subprotocol inspection, and idempotent context-aware `Close`. Canceling an
active read follows the native library contract and closes the connection.
Ping requires a concurrent reader so pong control frames can be processed.

## Observability

Observers receive only direction, negotiated subprotocol, terminal
normal/canceled/failed outcome, and duration. URLs, headers, origins, peer
addresses, close reasons, credentials, and message payloads are deliberately
absent.
