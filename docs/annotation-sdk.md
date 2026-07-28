# Go-native annotation SDK

Spice annotation extensions are ordinary versioned Go source. The application
selects descriptor packages with file-scoped comment imports, and the
application module will authorize their executable handlers with Go `tool`
directives. Spice does not define a plugin manifest, dependency resolver,
binary registry, or runtime annotation container.

## File-scoped imports

Annotation imports remain valid Go comments:

```go
// @import { Application } from "github.com/StevenBuglione/spice/annotation/core"
// @import { Controller, Get as GET } from "github.com/StevenBuglione/spice/annotation/web"
// @import * as web from "github.com/StevenBuglione/spice/annotation/web"

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
is fail-closed: every annotation must resolve through a named or namespace
binding declared in that same file. Duplicate local names, malformed paths,
private descriptor symbols, missing imports, and missing descriptor source are
source-positioned errors. No built-in annotation registry or name-based
semantic fallback participates in product analysis.

The physical source always contains `// `. GoLand concealment is presentation
only, so `gofmt`, Go Run, debuggers, Git, and copied text operate on valid Go.
The retired `@spice.import` spelling fails closed and is never folded or
resolved. The shared compiler emits an exact source diagnostic and a
document-version-checked `@import` replacement, so migration cannot silently
reinterpret or broadly rewrite a file.

## Descriptor contract

Every annotation is one exported, documented function in its own `.go` file:

```go
import (
	"context"

	"github.com/StevenBuglione/spice/annotation/sdk"
	annotationtool "example.com/starter/annotationtool"
)

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
			Tool:     annotationtool.Path,
			Handler:  ControllerHandler,
			Protocol: sdk.ProtocolV1Alpha2,
		},
	}
}

// ControllerHandler is the actual generic contribution implementation.
func ControllerHandler(
	_ context.Context,
	invocation sdk.Invocation,
) (sdk.Result, error) {
	if err := invocation.RequireDescriptor(
		"example.com/starter/annotation/web",
		"Controller",
	); err != nil {
		return sdk.Result{}, err
	}
	arguments, err := sdk.BindArguments(invocation, "", "prefix")
	if err != nil {
		return sdk.Result{}, err
	}
	prefix, err := arguments.String("prefix", false)
	if err != nil {
		return sdk.Result{}, err
	}
	return sdk.OneContribution(sdk.Contribution{
		Kind: sdk.ContributionController,
		Controller: &sdk.ControllerContribution{
			Prefix: prefix,
		},
	})
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
fully qualified tool package, and a typed package-level `sdk.Handler`.
The descriptor and handler must be exported declarations in the same `.go`
file. The handler’s exact signature is
`func(context.Context, sdk.Invocation) (sdk.Result, error)`. The compiler
derives its package and symbol from Go type information; descriptor authors do
not repeat a handler ID or source-symbol string. This metadata is the common
compiler, LSP, and GoLand contract for completion, documentation, descriptor
navigation, and direct implementation navigation.

An argument may additionally declare a semantic `ValueDomain`. The initial
domain, `sdk.ValueDomainGoInterface`, says that an identifier argument denotes
a named runtime Go interface. The SDK metadata, not an annotation-name switch,
causes the shared compiler service to expose its typed interface catalog and
the LSP to offer exact package/type completions. Constraint-only, anonymous,
pointer-to-interface, inaccessible, and unresolved types remain invalid.

## Module and offline behavior

The typed compiler lexically discovers only import comments before its single
semantic load so descriptor packages can be included in that same Go type
universe. Semantic annotation resolution still examines only files selected by
the active Go build.

Normal analysis forces `GOPROXY=off`. It uses `-mod=vendor` when
`vendor/modules.txt` exists and `-mod=readonly` otherwise. Missing module-cache
or vendor content is therefore an actionable load diagnostic and never an
editor-triggered download.

Before a file imports a descriptor, editor completion can discover its public
function from the target module graph, active workspace modules, local
replacements, vendor source, and already-populated module cache. This is a
bounded lexical catalog only: it recognizes exported exact
`func() sdk.Definition` declarations and literal identity/provenance fields,
but the existing typed-program decoder remains authoritative after insertion.
The catalog runs offline, does not execute descriptors or tools, and marks
whether the target application's own `go.mod` declares the exact tool path.
Selecting a candidate always inserts a visible `@import`; discovery never
creates an implicit compiler binding.

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
environment mutation. The host negotiates exact v1alpha2
protocol/tool/module identity, requires `describe` to enumerate every public
descriptor package and descriptor-to-capability registration, rejects a
descriptor whose package or symbol is absent from that declaration, and
serializes calls over one persistent process per workspace and tool. Analyze
dispatch uses the descriptor’s Go symbol identity; arbitrary handler-name
strings are not part of the protocol.

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
explicit interface bindings, configuration, controllers, routes, modules,
named interfaces, lifecycle,
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

All 30 official descriptors have one public descriptor file, one declared
handler, a real implementation source symbol, rich GoDoc, compatibility
metadata, and examples. `@Service`, `@Controller`, and `@Repository` contribute
constructible stereotypes with deterministic ordinary Go constructor
selection. `@Implements` contributes typed named-interface expressions; the
compiler verifies exact method sets and plans a source-owned generated Go
assertion before adding an interface candidate. Its public descriptor marks
the positional argument with `ValueDomainGoInterface`, so a third-party
annotation can request the same compiler-owned authoring behavior without
being named `Implements`. Third-party architectural
stereotypes remain non-constructing unless their typed contribution explicitly
sets the construction contract.

Bean-selection annotations return the generic `bean-metadata` contribution:

- `@Qualifier("name")` is repeatable on beans and exact constructor
  parameters.
- `@Primary` and `@Fallback` control deterministic single-value selection.
- `@Order(value)` controls collection order.
- `@Singleton`, `@Prototype`, `@RequestScope`, and `@SessionScope` declare
  cleanup ownership.

The SDK contribution contains typed fields; handlers do not resolve candidates
or construct values. Parameter invocations carry their exact parameter index,
name, type identity, and physical source position through the same protocol.
The compiler applies all selection and ownership rules generically, so a
third-party annotation may contribute equivalent metadata without adding an
annotation-name switch.

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

## Author a third-party module

Keep each descriptor beside its real handler while leaving the executable tool
command visibly separate:

```text
example.com/acme/spice-mail
├── annotation/mail/send.go
├── internal/annotationtool/tool.go
└── cmd/spice-annotations/main.go
```

The public descriptor package imports only
`github.com/StevenBuglione/spice/annotation/sdk`. The tool command may
also import `github.com/StevenBuglione/spice/annotation/sdk/protocol`; they do
not import `compiler`, `internal/cli`, or an official handler package. A
descriptor's `Implementation.Handler` is a typed reference to the real
package-level handler in the same file, and the tool's `describe` response
must report that descriptor registration.

Implement `protocol.Tool` as an instance-owned value. `initialize` checks the
exact tool and protocol identities, `describe` returns stable handler metadata,
`analyze` decodes normalized invocation facts and returns typed contributions,
and `shutdown` releases owned resources. `protocol.Serve` owns framing and
panic containment. Handlers must honor the caller context; they must not retain
an invocation, write generated files, print to stdout, scan ambient packages,
or execute application declarations.

The committed independent proof is split between
[`testdata/annotationfixture`](../testdata/annotationfixture) and
[`testdata/annotationapp`](../testdata/annotationapp). It demonstrates a named
core import, an aliased third-party provider import, a namespace-qualified
policy import, plugin-owned diagnostics, real descriptor and handler source
navigation, provider contribution, deterministic generated Go, ownership
checking, build, and execution. The fixture plugin imports only the public SDK
and protocol.

## Publish and select a version

Tag the descriptor packages and tool command in the same Go module. Consumers
select that one version with standard Go commands:

```text
go get -tool example.com/acme/spice-mail/cmd/spice-annotations@v1.4.0
go mod tidy
spice annotations doctor ./...
```

`go get -tool` adds the executable package to the application module's `tool`
block and selects its module in the ordinary build list. The descriptor import
comment selects symbols from that same resolved module. Spice rejects a
descriptor and tool that differ in module path, version, or replacement
identity. Removing the integration uses the standard command:

```text
go get -tool example.com/acme/spice-mail/cmd/spice-annotations@none
go mod tidy
```

Because tool dependencies participate in minimal-version selection, extension
authors should keep their dependency surface small and publish compatibility
ranges honestly. Do not hide a second dependency solver or download path in
the annotation process.

In an LSP client such as GoLand, importing an annotation whose exact tool is
not declared offers a two-step quick fix. **Preview** runs the displayed
`go get -tool` command against a temporary sibling modfile, shows the exact
`go.mod`/`go.sum` unified diff, and changes no application file. A separate
**Apply previewed** action is the confirmation. It accepts only the
content-derived preview token and only while both original module-file hashes
still match; guarded staged replacement rolls back on failure. Re-previewing
replaces an older plan. Editor completion and ordinary analysis remain offline
and never invoke this path implicitly.

## Local development and workspaces

Use a normal replacement while developing an application and extension
together:

```go
require example.com/acme/spice-mail v0.0.0

replace example.com/acme/spice-mail => ../spice-mail

tool example.com/acme/spice-mail/cmd/spice-annotations
```

`go.work` can make both modules convenient to edit, but it does not authorize
the tool. The application module's own `go.mod` must retain the `tool`
directive. Editor documentation labels a local replacement explicitly and
shows its source directory; it never presents local source as checksum-verified
published content.

## Vendor and offline operation

Run the ordinary Go workflow:

```text
go mod tidy
go mod vendor
go test -mod=vendor ./...
spice annotations doctor ./...
spice generate --check ./...
```

When `vendor/modules.txt` exists, Spice uses `-mod=vendor`; otherwise it uses
`-mod=readonly`. It always sets `GOPROXY=off` for analysis and tool launch.
Therefore editor completion, hover, diagnostics, navigation, generation, and
verification cannot download missing code. Install or vendor the dependency
deliberately when the diagnostic says its source is unavailable.

## Trust and review

An annotation tool is a native executable with the developer's permissions.
It is not sandboxed. Before authorizing one, review its maintenance, license,
release provenance, dependencies, cancellation behavior, network/file access,
diagnostic quality, and generated-output requests. Capability declarations are
inspectable compatibility metadata, not a security boundary.

Spice narrows the execution surface by requiring an exact application-owned
`tool` directive, an exact full package path, offline Go resolution, protocol
and module identity negotiation, bounded framing and stderr, deadlines,
process-tree cancellation, no replay, and guarded generation. Those controls
do not make an untrusted native process safe.

## GoLand authoring loop

With the Spice plugin installed, type `@` or edit an `@import`. Completion
shows the descriptor package, selected version or replacement, tool, and
handler. Accepting a completion adds a visible named or namespace import when
needed. Modifier-click opens the one-file descriptor; **Go to Implementation**
opens the handler; Quick Documentation renders its GoDoc, arguments, examples,
compatibility, provenance, and protocol metadata.

Inside a Go-interface value domain, completion is generated by the same
`go/packages` type universe used by verification and generation. GoLand does
not enumerate its own index to decide what Spice may inject. Accepting an
external interface adds a namespace `@import` for its Go package when needed.
The Spice generator—not the editor—emits the mandatory compile-time assertion
in a source-owned generated shard. Native Implement Methods remains a useful
authoring and offline fallback, but it cannot register or select a bean; the
next Spice analysis is authoritative.

The editor does not install a missing tool silently. Use `go get -tool`
directly until the preview-and-confirm module edit action is available. Then
run `spice annotations doctor` to validate the executable identity and every
selected handler before generation.
