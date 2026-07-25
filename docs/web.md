# Web runtime

Spice generates `net/http` adapters and keeps reusable HTTP policy in the small
public `web` package. Core does not require a third-party router: generated
patterns target Go's method-aware `http.ServeMux`.

The runtime provides:

- RFC 9457 `Problem` responses and a secure default error mapper;
- strict, bounded JSON request decoding with unknown-field rejection;
- JSON `Accept` negotiation;
- safe path/query/header scalar parsing;
- explicit `NoContent`;
- JSON, problem, and 204 response writers.

Binding errors retain an internal parser/decoder cause for server logs but
never retain or render the raw client value. Unknown application errors map to
an empty-detail 500 response. Explicit errors can implement `ProblemCarrier`;
invalid problem metadata is replaced with the same secure 500 response.

JSON request bodies must use `application/json` or a `+json` media type, contain
exactly one value, fit the configured positive byte limit, match the generated
request shape, and contain no unknown object fields. The default body limit is
1 MiB.

Generated adapters own routing, DTO construction, controller invocation,
response status selection, and error-handler calls. Applications remain free to
provide raw `http.Handler` beans when generated controller semantics are not
appropriate.
