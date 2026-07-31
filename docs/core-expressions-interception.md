# Core expressions, interception, and bean composition

Spice provides the useful outcomes of Spring expressions, AOP, and hierarchical
bean configuration without introducing a reflective runtime container.

## Restricted expressions

`expression.Compile` accepts a caller-owned schema containing only explicitly
declared Boolean/string variables and functions. The language supports `!`,
`&&`, `||`, `==`, `!=`, literals, calls, and parentheses. Programs are typed,
immutable, bounded, context-aware, and evaluated against positional bindings.
They cannot traverse properties, resolve bean names, invoke arbitrary methods,
assign, allocate, perform I/O, or access ambient process state.

`@security.Authorize` is the first generated integration:

```go
// @Authorize(
//     allScopes=["orders:read"],
//     expression="authenticated && issuer == \"https://issuer.example\" && hasRole(\"reader\")",
// )
```

The compiler validates the expression at its annotation. Generated Go carries
the exact validated literal into an immutable `security.Policy`; policy
construction reuses the same parser and schema. The available symbols are:

| Symbol | Type |
| --- | --- |
| `authenticated` | Boolean |
| `subject` | string |
| `issuer` | string |
| `hasRole(string)` | Boolean |
| `hasScope(string)` | Boolean |

An expression policy always requires an authenticated principal. Expression
requirements combine with role/scope requirements using AND semantics.

## Typed interception

`intercept.Invocation[Request, Response]` and
`intercept.Interceptor[Request, Response]` are ordinary Go functions. The
first interceptor in `intercept.Chain` is outermost. Nil functions, nil
contexts, cancellation, and wrong construction fail closed.

For every typed request/response route, generated contracts expose a typed
field:

```go
options := spicegen.ApplicationOptions{
    Interceptors: spicegen.RouteInterceptors{
        ControllerGet: []intercept.Interceptor[
            orders.GetOrderRequest,
            orders.OrderResponse,
        ]{
            observeOrderRead,
        },
    },
}
```

Generated terminal invocations contain the direct controller call and any
generated transaction or cache boundary. The configured chain is frozen while
the application is constructed. Debuggers therefore show the interceptor,
generated boundary, and handwritten method as ordinary frames. Raw HTTP and
form/binding-result routes retain their explicit `web.Middleware` boundary.

Spice annotations for transactions, caching, authorization, observations, and
HTTP middleware remain purpose-built generated decorators. Spice does not
select methods through runtime pointcut strings, synthesize subclasses, or
intercept concrete self-invocation.

## Immutable bean override layers

Every generated application with overridable singleton beans exposes
`BeanOverrideLayer` and `ComposeBeanOverrides`:

```go
overrides, err := spicegen.ComposeBeanOverrides(
    spicegen.BeanOverrideLayer{
        Name: "library",
        Overrides: spicegen.BeanOverrides{
            Sender: bean.Replace(testSender),
        },
    },
    spicegen.BeanOverrideLayer{
        Name: "application",
        Overrides: spicegen.BeanOverrides{
            Sender: bean.Replace(applicationSender),
        },
    },
)
```

Layer names must be unique and canonical. Layers are applied parent-to-child;
a later enabled typed field deliberately replaces the earlier field. The
result is passed through `ApplicationOptions.Overrides` before construction.
The running graph is never mutated, and construction, cleanup, rollback,
module ownership, and exact Go type checks remain unchanged.

Compile-time auto-configuration supplies library defaults separately through
reviewed fallback factories. Override layers are for explicit embedding and
tests, not dependency-by-presence activation.

## Boundary with Java mechanisms

Go has no classloader or subclass-proxy model. Spice therefore treats literal
load-time weaving as not applicable. Its equivalent is deterministic
build-time generation with source maps and inspectable Go. Universal runtime
pointcuts and unrestricted expression evaluation remain deliberate non-goals;
their useful application outcomes are the typed contracts above.
