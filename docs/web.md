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

When the provider graph contains an exact `*http.ServeMux`, Spice registers all
generated routes on that instance; HTTP server beans can depend on the same mux.
Otherwise the generated application creates an internal mux. In both cases
`Application.Handler()` exposes the final handler. Registration uses
`web.Register`, which converts ServeMux pattern/conflict panics into
construction errors so lifecycle cleanup rollback remains available.

`ApplicationOptions.Middleware` applies one ordered list to every generated
typed and raw route, including routes registered on an application-provided
`*http.ServeMux`. The first middleware observes the request first and the
response last. Nil middleware and middleware that returns a nil handler fail
application construction with the route pattern and list index.

`ApplicationOptions.HTTPObservers` is the dependency-free metrics/tracing
adapter seam. Every generated route supplies its stable symbol ID, module
import path, HTTP method, and route pattern. Observers begin in list order,
share a derived request context, and finish in reverse order with response
status, bytes, duration, and panic state. Observation wraps caller middleware,
so authentication rejections and other short circuits are still measured.
Typed-nil observers fail application construction.

The response wrapper preserves flushing, hijacking, HTTP/2 push, streaming
`io.ReaderFrom`, and `http.ResponseController` unwrapping. This lets
OpenTelemetry or a metrics package adapt the seam without changing generated
controller signatures or requiring a telemetry dependency in core.

## Controller contract

`compiler/controller` validates controller metadata from the same typed program
used for dependency injection. An exported, non-generic named struct marked
`@Controller` does not create an instance: an exact `@Bean` provider must
produce the receiver type used by every route.

Typed route methods use:

```go
// @Controller(prefix="/users")
type Users struct{}

type GetUserRequest struct {
    ID      UserID `path:"id"`
    Verbose bool   `query:"verbose"`
    TraceID string `header:"X-Trace-ID"`
}

// @Get("/{id}")
func (*Users) Get(
    context.Context,
    GetUserRequest,
) (UserResponse, error)
```

The exact typed signature is
`func(context.Context, RequestDTO) (Response, error)`. Request DTOs are exported
named struct values. Every exported field declares exactly one `path`, `query`,
`header`, or `body` tag, or opts out with `web:"-"`. Query and header tags may
add `,required`; path values and the single JSON body are always required.
Supported scalar bindings are strings, Booleans, signed integers, and
`time.Duration`, including exported named forms.

After every field is bound, generated adapters invoke an optional exact
value-receiver method:

```go
func (GetUserRequest) Validate(context.Context) error
```

The compiler rejects pointer receivers and lookalike signatures. Explicit
`ProblemCarrier` errors retain their response policy; ordinary validator errors
produce a safe 400 response without exposing their text. Validation always
runs before the controller method.

Only simple full-segment `{name}` path wildcards are supported. The compiler
rejects missing/extra wildcard fields, duplicate bindings, duplicate routes,
GET bodies, unsupported scalar types, invalid headers and paths, and mismatched
receiver providers.

For endpoints that deliberately own `net/http` details, this exact raw escape
hatch is supported:

```go
// @Get("/stream")
func (*Users) Stream(http.ResponseWriter, *http.Request)
```

Raw methods own their complete response and error policy. Typed methods use the
generated Spice binding and RFC 9457 policy.

## OpenAPI

Every target with controllers emits a deterministic OpenAPI 3.1 document at
`internal/spicegen/<target>/openapi.json`. Typed operations include path, query,
and header parameters; JSON request bodies; JSON or 204 success responses; and
the shared RFC 9457 problem schema. Component schemas preserve JSON field
names, omission rules, arrays, maps, pointers, recursive references,
`time.Time`, and `time.Duration`.

Raw `net/http` routes remain visible with an explicitly unconstrained response
because Spice cannot safely infer a handler-owned wire contract. Module import
paths become operation tags and stable Spice symbol/module extensions preserve
compiler ownership. The ownership manifest makes `spice generate --check` and
`--diff` detect stale or manually changed API documents.
