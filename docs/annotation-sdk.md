# Go-native annotation SDK

Spice annotation extensions are ordinary versioned Go source. The application
selects descriptor packages with file-scoped comment imports, and the
application module will authorize their executable handlers with Go `tool`
directives. Spice does not define a plugin manifest, dependency resolver,
binary registry, or runtime annotation container.

## File-scoped imports

Annotation imports remain valid Go comments:

```go
// @spice.import { Application } from "github.com/StevenBuglione/spice/annotation/core"
// @spice.import { Controller, Get as GET } from "github.com/StevenBuglione/spice/annotation/web"
// @spice.import * as web from "github.com/StevenBuglione/spice/annotation/web"

// @Application
// @Controller
// @web.Get(path="/orders")
// @GET(path="/orders/{id}")
func main() {}
```

Named imports make framework-wide concepts concise. Aliases let an application
choose a local spelling. Namespace imports retain visible provenance where two
starters expose similar concepts.

Imports apply to the complete file regardless of their textual order. A file
that declares at least one annotation import is fail-closed: every annotation
in that file must resolve through a named or namespace binding. Duplicate local
names, malformed paths, private descriptor symbols, and missing descriptor
source are source-positioned errors. Files with no import declaration retain
the pre-1.0 built-in spelling compatibility path. New application code should
use explicit imports; the compatibility path has no third-party resolution and
is scheduled for removal before 1.0.

The physical source always contains `// `. GoLand concealment is presentation
only, so `gofmt`, Go Run, debuggers, Git, and copied text operate on valid Go.

## Descriptor contract

Every annotation is one exported, documented function in its own `.go` file:

```go
// Controller marks a type whose methods are exposed through generated
// net/http adapters.
func Controller() sdk.Definition {
	return sdk.Definition{
		Name:    "web.Controller",
		Summary: "Marks an HTTP controller.",
		Targets: []sdk.Target{sdk.TargetType},
		Arguments: []sdk.Argument{{
			Name:        "prefix",
			Kinds:       []sdk.Kind{sdk.KindString},
			Description: "Optional route prefix.",
		}},
		Examples: []sdk.Example{{
			Title: "Controller",
			Code:  "// @Controller",
		}},
		Compatibility: sdk.Compatibility{
			Since:        "0.1.0",
			MinimumSpice: "0.1.0",
		},
		Implementation: sdk.Implementation{
			Tool:     "example.com/starter/cmd/spice-annotations",
			Handler:  "web/controller",
			Protocol: sdk.ProtocolV1Alpha1,
			Source: sdk.Symbol{
				Package: "example.com/starter/internal/annotations",
				Name:    "ControllerHandler",
			},
		},
	}
}
```

The compiler loads application and selected descriptor packages in one
`go/packages` operation and statically decodes the returned composite literal.
It does not execute the function, package initialization, or any provider.
Descriptor bodies may contain only one return statement with a keyed
`sdk.Definition` composite literal. Nested metadata must also be keyed static
composite literals. Scalar values must be literals or exported SDK constants;
local constants, calls, control flow, computed expressions, and mutable
initialization fail.

Definitions require a summary, target, compatibility range, documented
example, documented arguments, supported value kinds, an exact protocol, a
fully qualified tool package, a stable handler identity, and a real Go source
symbol identity. This metadata is the common compiler, LSP, and GoLand contract
for completion, parameter information, documentation, definition navigation,
and implementation navigation.

## Module and offline behavior

The typed compiler lexically discovers only import comments before its single
semantic load so descriptor packages can be included in that same Go type
universe. Semantic annotation resolution still examines only files selected by
the active Go build.

Normal analysis forces `GOPROXY=off`. It uses `-mod=vendor` when
`vendor/modules.txt` exists and `-mod=readonly` otherwise. Missing module-cache
or vendor content is therefore an actionable load diagnostic and never an
editor-triggered download.

The public SDK defines bounded `Content-Length` JSON-RPC framing and typed
`initialize`, `describe`, `analyze`, and `shutdown` messages. `protocol.Serve`
provides the matching panic-contained server loop so extension authors do not
reimplement framing or method dispatch.

The compiler parses the target root's exact `go.mod` and rejects tools not
listed by an exact `tool` directive; a parent module or another `go.work`
member cannot authorize the process. It resolves package and module provenance
with offline `go list`, preserving selected versions and local or versioned
replacement identity. Descriptor and executable packages must resolve from the
same module version and replacement.

Authorized tools launch through the fixed command shape:

```text
go tool <full-package-path> --spice-stdio
```

The descriptor cannot supply a binary path, shell, command-line fragment, or
environment mutation. The host negotiates exact protocol/tool/module identity,
requires each descriptor handler and implementation symbol in `describe`, and
serializes calls over one persistent process per workspace and tool.

Calls have bounded startup and request deadlines. Framing corruption, stdout
contamination, a crash, a timeout, or cancellation fails the operation and
terminates the complete process tree without replay. Windows processes are
contained in kill-on-close Job Objects; Unix processes use dedicated process
groups. Stderr is bounded and diagnostic-only.

## Typed contributions

Every explicitly imported occurrence is sent to its descriptor's declared
handler after static descriptor validation. The invocation contains the
canonical descriptor identity, normalized typed literal arguments, declaration
target, stable Go symbol ID, package path, exact type identity when available,
and non-executable declaration facts. It never contains a provider value or an
instruction to execute application code.

Handlers return the public `sdk.Contribution` discriminated union. The current
typed capabilities cover application roots, service stereotypes, providers,
configuration, controllers, routes, modules, named interfaces, lifecycle,
bootstrap features, scheduling, async execution, transactions, event topics
and listeners, caching, authorization, and guarded generated files. Both the
tool-side encoder and compiler-side decoder validate the selected kind and its
one matching payload. Unknown fields, unknown kinds, ambiguous payloads,
trailing JSON, malformed values, and duplicate contribution kinds fail before
the immutable application IR is built.

The compiler consumes contribution kinds and typed payloads, not annotation
names. Any authorized third-party descriptor can contribute a supported
capability without a compiler switch for that descriptor's package or name.
Official descriptors use exactly the same protocol path through:

```text
github.com/StevenBuglione/spice/cmd/spice-annotation-core
```

All 20 official descriptors have one public descriptor file, one declared
handler, a real implementation source symbol, rich GoDoc, compatibility
metadata, and examples. `@Service` is intentionally an architectural
stereotype; construction remains an explicit `@Bean` provider so Spice does
not invent a hidden container or zero-value construction rule.

Use the read-only inspection commands:

```text
spice annotations list ./...
spice annotations doctor ./...
```

`list` reports explicit descriptors and whether their tools are authorized.
`doctor` launches authorized tools, negotiates provenance, checks handlers and
source symbols, shuts them down, and reports every problem. Neither command
installs dependencies or changes module files.

`spice verify`, `generate`, `build`, `run`, `dev`, and `lsp` now share this
tool-aware compiler service. Verification uses validation mode, so it includes
committed generated files and does not require an application target;
generation mode excludes generated files while producing the guarded plan.
