# Web runtime

Spice generates `net/http` adapters and keeps reusable HTTP policy in the small
public `web` package. Core does not require a third-party router: generated
patterns target Go's method-aware `http.ServeMux`.

The runtime provides:

- RFC 9457 `Problem` responses and a secure default error mapper;
- strict, bounded JSON request decoding with unknown-field rejection;
- strict, bounded URL-encoded form decoding with unknown-field rejection;
- JSON and HTML `Accept` negotiation;
- safe path/query/header/form scalar parsing;
- immutable binding and validation results that never retain rejected values;
- explicit `NoContent`;
- JSON, HTML view, redirect, problem, and 204 response writers.

Binding errors retain an internal parser/decoder cause for server logs but
never retain or render the raw client value. Unknown application errors map to
an empty-detail 500 response. Explicit errors can implement `ProblemCarrier`;
invalid problem metadata is replaced with the same secure 500 response.

JSON request bodies must use `application/json` or a `+json` media type, contain
exactly one value, fit the configured positive byte limit, match the generated
request shape, and contain no unknown object fields. The default body limit is
1 MiB.

URL-encoded forms must use
`application/x-www-form-urlencoded`, optionally with the UTF-8 charset. They
share the configured request-body bound. Repeated scalar values and unknown
fields fail closed. Generated form adapters collect those safe failures in
`web.BindingResult` and still invoke the controller so it can re-render the
form. Parser causes and submitted values are not retained in that result.

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

An explicit `@security.Authorize` on a route adds a generated authorization
guard inside caller middleware and outside the controller adapter. This order
lets caller-owned authentication middleware attach a verified principal before
the deny-by-default policy runs. Unannotated routes remain unchanged. See
`docs/security.md` for the exact policy and observer contracts.

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

The ordinary exact typed signature is
`func(context.Context, RequestDTO) (Response, error)`. A route annotated with
`@data.Transactional` instead uses
`func(context.Context, data.Executor, RequestDTO) (Response, error)` so its
transaction-owned executor remains visible; see [`data.md`](data.md). Request
DTOs are exported named struct values. Every exported field declares exactly
one `path`, `query`, `header`, `body`, or `form` tag, or opts out with
`web:"-"`.
Query and header tags may add `,required`; path values and the single JSON body
are always required. Supported scalar bindings are strings, Booleans, signed
integers, and `time.Duration`, including exported named forms.

After every field is bound, generated adapters invoke an optional exact
value-receiver method:

```go
func (GetUserRequest) Validate(context.Context) error
```

The compiler rejects pointer receivers and lookalike signatures. Explicit
`ProblemCarrier` errors retain their response policy; ordinary validator errors
produce a safe 400 response without exposing their text. Validation always
runs before the controller method.

Server-rendered form routes make their error flow explicit:

```go
type SaveOwnerRequest struct {
    ID        int    `path:"id"`
    FirstName string `form:"firstName,required"`
    Age       int    `form:"age,required"`
}

// @Post("/{id}")
func (*Owners) Save(
    ctx context.Context,
    request SaveOwnerRequest,
    binding web.BindingResult,
) (view.Result, error) {
    if !binding.Valid() {
        return view.Render("owner-form", OwnerPage{
            Owner: request,
            Errors: binding.Errors(),
        })
    }
    return view.SeeOther("/owners/" + strconv.Itoa(request.ID))
}
```

`web.BindingResult` is legal only immediately after a request DTO containing
at least one `form` field. Transactional form routes place
`data.Executor` before the DTO. A form route must return exact `view.Result`,
and every target containing a view route must provide exactly one
`*view.Renderer` bean. The generated adapter calls that bean directly; there is
no global renderer or runtime lookup.

`view.Result` is a closed validated value: `view.Render` and
`view.RenderStatus` select a known template outcome, while `view.SeeOther`
selects a bodyless 303 to a safe local absolute path. HTML negotiation happens
before binding. Rendering remains atomic within the renderer's configured
bound.

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
`internal/spicegen/<target>/openapi.json` for both package-main and compatible
legacy targets. Typed operations
include path, query, and header parameters; JSON request bodies; JSON or 204
success responses; URL-encoded form request bodies; HTML and 303 view
outcomes; and the shared RFC 9457 problem schema. Component schemas
preserve JSON field names, omission rules, arrays, maps, pointers, recursive
references, `time.Time`, and `time.Duration`.

Protected operations additionally declare the generated Bearer security scheme,
401/403 problem responses, and stable `x-spice-authorization` requirements.

Raw `net/http` routes remain visible with an explicitly unconstrained response
because Spice cannot safely infer a handler-owned wire contract. Module import
paths become operation tags and stable Spice symbol/module extensions preserve
compiler ownership. The ownership manifest makes `spice generate --check` and
`--diff` detect stale or manually changed API documents.
