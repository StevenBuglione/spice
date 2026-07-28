# Server-side HTML templates

`view` wraps Go's contextual `html/template` engine with deterministic parsing,
strict execution, response bounds, and atomic HTTP writes:

```go
//go:embed templates/*.html
var templates embed.FS

renderer, err := view.Parse(
    templates,
    []string{"templates/*.html"},
    template.FuncMap{
        "formatMoney": formatMoney,
    },
    view.Options{MaxOutputBytes: 512 << 10},
)
if err != nil {
    return err
}
```

Patterns are bounded relative slash paths expanded only against the supplied
`fs.FS`. Matched regular files are de-duplicated and parsed in lexical path
order within a 4 MiB aggregate source limit. Duplicate file or `define` names
fail construction instead of silently overriding one another. Function names
and values are validated before reaching `html/template`, so invalid maps
return errors rather than panicking. The caller owns and trusts the filesystem
and functions; an `os.DirFS` with attacker-controlled symlinks is not a
sandbox.

```go
err := renderer.Render(
    request.Context(),
    writer,
    "orders",
    http.StatusOK,
    page,
)
```

Generated controllers normally return the declarative closed result instead
of writing through the renderer themselves:

```go
func (*Owners) Show(
    context.Context,
    ShowOwnerRequest,
) (view.Result, error) {
    return view.Render("owner-details", OwnerPage{Owner: owner})
}
```

An exact `*view.Renderer` bean makes ownership visible in the ordinary Spice
dependency graph. The compiler records that provider on each view route, and
generated Go invokes `Renderer.Respond` directly. `view.SeeOther("/owners/7")`
provides a safe bodyless 303 form redirect. Relative, cross-origin, scheme,
network-path, fragment, control-character, and untrimmed destinations are
rejected.

Missing map keys are errors. Rendering occurs in a private bounded buffer, so
parse, execution, cancellation, and output-limit failures do not modify the
response. A successful render sets:

- `Content-Type: text/html; charset=utf-8`;
- `X-Content-Type-Options: nosniff`;
- an exact `Content-Length`;
- the caller-selected body-bearing status.

The default output limit is 1 MiB and the maximum is 8 MiB. Informational,
204, 304, and invalid statuses are rejected because they cannot carry the
rendered body. Network write failures can still produce a partial response;
HTTP transports cannot roll back bytes already sent.

`html/template` provides contextual HTML, JavaScript, CSS, and URL escaping.
Caller functions that return `template.HTML` or related trusted-content types
explicitly bypass that protection and require their own security review.
Content Security Policy remains application-specific and should be installed
by middleware. Rendering does not provide authentication, CSRF defense, asset
serving, localization, or client-side hydration.

Renderers are immutable and safe for concurrent execution when caller
functions are also concurrency-safe. For development reload, parse a new
renderer and swap the application-owned instance only after parsing succeeds;
the package starts no watcher or goroutine.
