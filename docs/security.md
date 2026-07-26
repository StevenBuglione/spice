# Authentication and authorization

Spice separates authentication from authorization. An OAuth2/OIDC starter
verifies a token and constructs an immutable `security.Principal`; core policy
code never parses or trusts an unverified bearer token.

Generated guards construct policies at application startup:

```go
policy, err := security.NewPolicy(security.PolicySpec{
    Definition: security.Definition{
        ID:     "orders.Place",
        Module: "example.com/shop/orders",
    },
    AllRoles:  []string{"customer"},
    AllScopes: []string{"orders:write"},
})
```

An empty policy is invalid and cannot grant access. Missing principals receive
a safe RFC 9457 HTTP 401 response; authenticated principals missing an exact
case-sensitive role or scope receive 403. The guard includes a standard Bearer
challenge on 401 responses. Policies can require authentication, all roles, any
one role, and all scopes.

Verified principals are attached to request contexts with `WithPrincipal`.
Their role/scope inputs are copied, sorted, and deduplicated. Generated HTTP
routes apply `Guard` after authentication middleware. Direct service calls use
the same `Authorizer.Authorize` contract.

Errors and observations never contain the subject, issuer, roles, or scopes.
Observers receive only compiler-owned policy/module identity, allowed state,
reason class, and duration. There is no global principal, policy registry,
authorization cache, token parser, or network discovery in the core package.
