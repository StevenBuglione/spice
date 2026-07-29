# Dependency review: coder/websocket

- Decision: approved for the isolated `starter/websocket` package.
- Version: `github.com/coder/websocket` v1.8.15.
- Upstream: <https://github.com/coder/websocket>.
- License: ISC; retained with the mechanically vendored source.
- Maintenance: Coder actively maintains the v1 line. The selected release was
  published June 15, 2026. The library implements RFC 6455, passes the Autobahn
  test suite, supports context-aware I/O, close handshakes, concurrent writes,
  ping/pong, subprotocols, same-origin checks, and RFC 7692 compression.
- Dependency scope: the module is small and adds no transitive modules. Spice
  does not adopt a separate HTTP server, router, codec, or message protocol.
- Security: inbound TLS and same-origin checks are required by default.
  Cross-origin patterns and any-origin behavior are separate explicit
  decisions. Outbound URLs reject credentials, fragments, missing ports, and
  plaintext without opt-in. TLS verification cannot be disabled. Headers,
  message sizes, connections, subprotocols, origins, compression thresholds,
  and close timeouts are bounded.
- Cancellation: reads, writes, ping, dial, and close use caller contexts.
  Native read cancellation closes the connection and is documented. Close
  performs a bounded handshake and force-closes on cancellation.
- Observability: the Spice seam exposes direction, subprotocol, outcome, and
  duration only. It cannot receive headers, URLs, peer addresses, close reasons,
  or payload bytes.
- Configuration: `NewHandler` performs no network work and returns an ordinary
  `http.Handler`. `Dial` is the only outbound connection operation. No package
  import starts a listener, installs a global registry, or downloads modules.
- Verification: race-enabled loopback tests exercise text exchange,
  subprotocol negotiation, ping/read coordination, secure-server rejection,
  cross-origin rejection, capacity exhaustion, size limits, close cancellation,
  observation, and defensive configuration.

Primary references:

- <https://github.com/coder/websocket/releases/tag/v1.8.15>
- <https://pkg.go.dev/github.com/coder/websocket@v1.8.15>
- <https://github.com/coder/websocket/blob/v1.8.15/LICENSE.txt>
