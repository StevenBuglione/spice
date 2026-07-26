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

Completion is derived from the current compiler result:

- built-in and explicitly selected starter annotations;
- annotation arguments and required-argument snippets;
- bootstrap allowed values such as management endpoint names;
- exact module IDs and named-interface identities;
- generated configuration property keys.

Typing `@` on an otherwise empty declaration line may complete to a valid
comment such as:

```go
// @management.Enable(expose=["health"])
```

Completion is refused when `@` appears in an unrelated Go expression. Hover
describes annotation targets and argument types, module summaries, and
configuration types, environment names, requirements, and non-secret defaults.
Secret defaults never enter compiler metadata or hover text.

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

The server also provides full-document semantic tokens. The
`@qualified.Annotation` portion of each valid declaration comment is reported
as the standard `decorator` token; the `// ` punctuation remains ordinary Go
comment syntax. Editors choose whether and how to combine semantic tokens with
their native Go grammar.

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
refresh, diagnostics, completion, hover, quick fixes, and full semantic tokens.
Shutdown cancels active analyses. Caller context cancellation interrupts a
blocked closable input stream. Multiple workspaces never share services,
overlays, results, or caches.

The first-party Zed adapter and its setup/fixture are documented in
[`zed.md`](zed.md).
