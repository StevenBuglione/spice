# Outbound HTTP client

The public `httpclient` package provides an explicit standard-library client
for generated or handwritten beans. It does not perform discovery or hidden
network access.

`httpclient.New` requires an absolute HTTP(S) base URL without credentials,
query, or fragment. Relative requests remain below that base path; absolute
references, authority changes, dot-segment traversal, and fragments are
rejected. Redirects receive the same origin and base-path enforcement, so a
remote redirect cannot silently turn a configured integration into an
arbitrary outbound request.

A nil `http.Client` selects a 30-second timeout. Caller-supplied clients are
copied, not mutated, and retain their transport, cookie jar, timeout, and
redirect callback behind Spice's scope check. Request contexts remain
caller-owned.

```go
client, err := httpclient.New(httpclient.Options{
    BaseURL:   "https://inventory.example/v1",
    UserAgent: "commerce/1.2.0",
})
if err != nil {
    return err
}

response, err := httpclient.DoJSON[InventoryResponse](
    ctx,
    client,
    http.MethodPost,
    "reservations",
    CreateReservation{SKU: sku, Quantity: quantity},
)
```

Typed JSON helpers:

- set JSON request/response media types;
- bound response and error bodies to 4 MiB by default, with a 64 MiB maximum
  for typed decoding (larger payloads use the raw streaming API);
- accept JSON and `application/*+json`;
- return zero values for empty and 204 successes;
- optionally reject unknown response fields;
- close response bodies and report close/read failures;
- return status and defensive response headers.

Non-2xx responses return `*httpclient.ResponseError`. A valid RFC 9457 document
is available through `RemoteProblem`, but `Error()` includes only status and
never renders the remote detail or raw body. Raw `NewRequest`/`Do` remains
available when streaming or custom codecs are required; callers own the
returned response body, while Spice still enforces the configured base URL.
