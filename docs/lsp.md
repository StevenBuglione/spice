# Spice Language Server

`spice lsp` serves editor-neutral Language Server Protocol 3.x features over
standard JSON-RPC on stdin/stdout. Stdout contains protocol frames only; command
or transport failures go to stderr after the connection ends.

## Editor command

Configure an editor LSP client to launch the repository-built or installed
Spice executable with one argument:

```text
spice lsp
```

The client should identify Go documents and pass a local workspace folder URI.
If it does not, opening a file makes Spice walk upward to the nearest `go.mod`.
Remote document and workspace URIs are rejected.

Spice advertises UTF-16 positions and full-document synchronization. Open,
change, save, and close notifications maintain bounded defensive overlay
copies. Files on disk remain ordinary Go; annotation completion always inserts
or preserves the `// @...` representation.

## Analysis

Each workspace owns an isolated `compiler/service.Service`. After a 150 ms
debounce, the server submits all open document versions as one overlay request.
A newer version cancels older work, advances the service sequence, and prevents
the older result from being published. Analysis is read-only: it does not
generate files or update ownership manifests.

The server publishes the same stable diagnostic codes and physical source
locations used by the CLI for annotation syntax and validation, exact provider
wiring, lifecycle hooks, application features, configuration declarations, and
module ownership, boundary, dependency, and cycle failures. LSP ranges are
converted from compiler byte columns to zero-based UTF-16 positions. A
diagnostic publication includes the analyzed document version so an editor can
also reject an obsolete message.

Application-global failures that do not have a source location use
`window/showMessage`; they are never attached to an arbitrary file.

## Language features

Completion and navigation are derived from the shared compiler service:

- statically decoded descriptors selected by explicit annotation imports;
- exported descriptor candidates discovered from source already present in the
  target module graph, workspace modules, vendor tree, local replacements, and
  module cache;
- descriptor package paths and symbols inside `@import` declarations;
- annotation arguments and required-argument snippets;
- bootstrap allowed values such as management endpoint names;
- exact module IDs and named-interface identities;
- generated configuration property keys.

The earlier `@spice.import` spelling is a hard error, not an alias. The
diagnostic is anchored to the retired token and carries a version-checked
workspace edit that replaces only it with `@import`.

Typing `@` on an otherwise empty declaration line may complete to a valid
comment and add the corresponding explicit import as a versioned additional
text edit:

```go
// @management.Enable(expose=["health"])
```

Existing named aliases and namespace imports are preserved. Completion detail
identifies the descriptor package, selected module version or replacement,
implementation tool, and whether the target `go.mod` authorizes that exact
tool, so the inserted source has inspectable provenance. Catalog discovery
forces `GOPROXY=off`, never executes a descriptor or tool, never changes module
files, and keeps fully typed imported descriptors authoritative over lexical
pre-import candidates. Completion is refused when `@` appears in an unrelated
Go expression.

Hover renders the descriptor summary and GoDoc, typed arguments, descriptions,
defaults and allowed values, targets, examples, compatibility, resolved module
provenance, tool, handler, protocol, and implementation symbol. Signature help
uses the same argument model and tracks the active annotation argument.
Configuration hover still omits secret defaults.

Definition and document-link requests return the exact one-file Go descriptor
function selected by the current file's named, aliased, or namespace import.
`textDocument/implementation` returns the real Go handler source symbol declared
by that descriptor. The compiler resolves both locations offline through the
same target module graph used for analysis, including vendor and local
replacement source. Unknown annotations and `@` text in strings or ordinary
comments never become links. Unimported annotations fail closed and do not
receive synthetic documentation or navigation targets.

For an SDK argument with `ValueDomainGoInterface`, completion and navigation
use the same loaded Go type universe as dependency validation. An interface
from any package in the module graph receives a namespace `@import`; definition
and hover resolve the operand to the real Go interface declaration and method
set. This path does not depend on the editor's Go index, and it never inserts an
ordinary import used only by an annotation or a handwritten assertion.

Code actions come from `compiler/diagnostic.SuggestedFix`. The server returns an
action only when every edit names an open document, carries the exact current
document version, and intersects the requested range. The first available fix
converts an accidental raw annotation line:

```text
@Application
```

to:

```go
// @Application
```

The edit is a precise prefix insertion rather than a file rewrite.

An explicitly imported descriptor whose tool is absent from the target
application's own `go.mod` also receives a command-backed quick fix:

1. **Preview** runs the exact displayed `go get -tool package@version` against
   a temporary sibling modfile and shows the complete `go.mod`/`go.sum` diff;
   the application files remain unchanged.
2. **Apply previewed** is offered only after that preview and acts as the
   explicit confirmation. The server checks the content-derived token and the
   original hashes of both module files before staged replacement, and rolls
   back on a write failure.

The preview is bounded, tied to an unauthorized descriptor from the shared
offline catalog, and replaced when the user previews again. Execute commands
run as tracked asynchronous LSP requests so the server continues reading
`$/cancelRequest`; cancellation terminates the `go get` subprocess and returns
the standard LSP request-cancelled error. The language server advertises only
these two exact execute-command IDs. It does not accept an arbitrary shell
command, package path, workspace, or unpreviewed mutation.

The server also provides full-document semantic tokens. It reports
`@qualified.Annotation` as `decorator`, argument names as `parameter`, quoted
values as `string`, integers as `number`, booleans and unquoted values as
`keyword`, and annotation delimiters as `operator`. The `// ` prefix remains
ordinary Go comment syntax. Editors choose whether and how to combine semantic
tokens with their native Go grammar.

## Workspace settings

Clients may select a target and bounded package patterns during initialization:

```json
{
  "initializationOptions": {
    "target": "Commerce",
    "patterns": ["./examples/commerce/..."]
  }
}
```

The same values may be refreshed without restarting through
`workspace/didChangeConfiguration`:

```json
{
  "settings": {
    "spice": {
      "target": "Commerce",
      "patterns": ["./examples/commerce/..."]
    }
  }
}
```

Omitting patterns uses the compiler service default `./...`. Values must be
trimmed; invalid settings fail closed without replacing the last complete
metadata result.

## Protocol and resource boundaries

The stdlib-only transport bounds a message at 16 MiB, a header line at 8 KiB,
and a header block at 64 lines. Duplicate or invalid `Content-Length` framing
is fatal because the stream cannot be safely resynchronized. A length-delimited
invalid JSON body receives the standard JSON-RPC parse error and the session
continues.

The server supports initialize, initialized, shutdown, exit, cancellation,
document open/change/save/close, workspace-folder changes, configuration
refresh, diagnostics, completion, signature help, hover, definition,
implementation and document-link navigation, quick fixes, and full semantic
tokens. Its execute-command provider is limited to confirmed annotation-tool
preview/apply. A clean first analysis still publishes an empty diagnostic set
for each open document, allowing clients to finish synchronization without a
sentinel error. Shutdown cancels active analyses and command requests. Caller
context cancellation interrupts a blocked closable input stream. Multiple
workspaces never share services, overlays, results, caches, or installation
previews.

The first-party Zed adapter and its setup/fixture are documented in
[`zed.md`](zed.md).
