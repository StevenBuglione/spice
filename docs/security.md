# Authentication and authorization

Spice separates authentication from authorization. An OAuth2/OIDC starter
verifies a token and constructs an immutable `security.Principal`; core policy
code never parses or trusts an unverified bearer token.

Generated guards are declared on HTTP routes:

```go
// @Post("/")
// @security.Authorize(
//     authenticated=true,
//     anyRoles=["support", "customer"],
//     allScopes=["orders:write"],
// )
func (*Orders) Place(
    context.Context,
    PlaceOrderRequest,
) (OrderResponse, error)
```

The compiler accepts this annotation only on exactly one valid `@Get` or
`@Post` method. It rejects empty policies, blank or duplicate names, wrong
argument kinds, repeated annotations, and non-route targets with source
positions. Role and scope lists are sorted before entering the immutable route
IR, generated source, ownership hash, and OpenAPI extensions. Policy ownership
uses the exact Modulith module when assigned and the declaring package identity
otherwise.

Generated Go calls `security.NewPolicy` and `security.Guard` directly during
application construction. An empty policy remains invalid and cannot grant
access. Missing principals receive a safe RFC 9457 HTTP 401 response;
authenticated principals missing an exact case-sensitive role or scope receive
403. The guard includes a standard Bearer challenge on 401 responses. Policies
can require authentication, all roles, any one role, and all scopes.

Verified principals are attached to request contexts with `WithPrincipal`.
Their role/scope inputs are copied, sorted, and deduplicated. Caller-supplied
`ApplicationOptions.Middleware` runs before the generated guard, so an explicit
OIDC or other authentication middleware can attach the principal. The route
observation middleware remains outside both and records 401/403 outcomes.
Unannotated routes receive no authorization guard.

For applications with both public and protected routes, the OIDC starter's
`OptionalMiddleware` is the safe global adapter: no credentials means no
principal, while any presented credentials must verify successfully. Its
required `Middleware` variant is appropriate when every route must authenticate.

`ApplicationOptions.AuthorizationObservers` observes generated policy
decisions. `AuthorizationWriteFailure` optionally receives a response-write
failure that cannot flow through `http.Handler`. Typed-nil observers fail
application construction before providers run. Direct service calls use the
same `Authorizer.Authorize` contract explicitly; Spice does not pretend an HTTP
guard secures an independent service call.

Generated OpenAPI declares a Bearer security scheme, 401/403 problem
responses, and an `x-spice-authorization` policy summary only for protected
operations.

Errors and observations never contain the subject, issuer, roles, or scopes.
Observers receive only compiler-owned policy/module identity, allowed state,
reason class, and duration. There is no global principal, policy registry,
authorization cache, token parser, or network discovery in the core package.
