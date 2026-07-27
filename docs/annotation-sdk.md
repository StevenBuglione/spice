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
the pre-1.0 built-in spelling compatibility path while the commerce example
and built-ins migrate.

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

The public SDK also defines bounded `Content-Length` JSON-RPC framing and typed
`initialize`, `describe`, `analyze`, and `shutdown` messages. The compiler
parses the target root's exact `go.mod` and rejects tools not listed by an exact
`tool` directive; a parent module or another `go.work` member cannot authorize
the process. Process launching, resolved-module handshake validation, and
generic contribution incorporation are the next delivery slice. Until that
host is connected, this document describes the landed
source/import/descriptor/protocol contract rather than promising third-party
semantic contributions.
