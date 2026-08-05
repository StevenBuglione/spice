# Zed integration

Zed is Spice's supported secondary editor integration. GoLand is the primary
target because its native folding and PSI APIs can provide exact zero-width
prefix concealment and virtual declarations; see [`goland.md`](goland.md).
The independently versioned
[`spice-framework/zed`](https://github.com/spice-framework/zed) extension
attaches the same editor-neutral `spice lsp` server to Go buffers. It adds
annotation completion, versioned diagnostics, hover, definition and
document-link navigation, quick fixes, module/configuration metadata, and
semantic annotation highlighting while leaving `gopls` responsible for
ordinary Go features.

The extension contains no compiler and downloads nothing. It resolves an
explicitly configured executable first and otherwise looks for `spice` on the
worktree `PATH`. It launches exactly that executable with the `lsp` argument.
The standalone repository owns the Rust source, locked dependency graph,
fixture, release artifact, and compatibility declaration.

## Install a development build

Install a compatible Spice command, then clone the extension:

```text
go install github.com/spice-framework/toolchain/cmd/spice@<compatible-version>
git clone https://github.com/spice-framework/zed.git
spice version
```

Open Zed's command palette, choose `zed: install dev extension`, and select the
cloned `zed` repository root. Zed compiles its pinned Rust extension to
WebAssembly. That repository's CI verifies Rust formatting, Clippy, tests, the
`wasm32-wasip2` release build, and a canonical offline Spice fixture. Core's
`make verify` deliberately does not duplicate those independently versioned
checks.

Zed may not inherit the interactive terminal's `PATH`, especially when
launched from a desktop shortcut. In that case add an explicit project-local
`.zed/settings.json`. Forward slashes work in a Windows JSON path:

```json
{
  "lsp": {
    "spice": {
      "binary": {
        "path": "C:/Users/me/go/bin/spice.exe",
        "arguments": ["lsp"]
      },
      "initialization_options": {
        "target": "Commerce",
        "patterns": ["./..."]
      }
    }
  },
  "lsp_document_links": true,
  "languages": {
    "Go": {
      "language_servers": ["gopls", "spice", "..."],
      "semantic_tokens": "combined"
    }
  }
}
```

On Linux, the corresponding path is commonly `/home/me/go/bin/spice`. Omit
`initialization_options` for the default target and package scope. Explicit
`binary.arguments` replace the extension's default `["lsp"]`; retain `lsp`
when overriding them.

## Navigation and completion

Keep both `gopls` and `spice` enabled. Hold the platform modifier (`Ctrl` on
Windows/Linux or `Cmd` on macOS) while hovering an annotation to see its exact
token underlined, then click to open the descriptor. Spice advertises `@` as a
completion trigger. At a declaration-comment position, accepting a completion
inserts valid Go such as:

```go
// @management.Enable(expose=["health"])
```

If the buffer temporarily contains a raw `@Application` line, the Spice quick
fix inserts the missing `// ` prefix as a precise version-aware edit. The
compiler service supplies validation, module identities, allowed values,
configuration keys, and hover data; the extension does not maintain a parallel
annotation registry.

Zed's extension manifest currently attaches language servers by language, not
by an arbitrary project activation predicate. The extension therefore starts
alongside Go for the worktree, and Spice's bounded typed analysis recognizes
actual Spice annotations and module roots. Recognition never executes
providers.

## Honest presentation boundary

`spice lsp` emits standard semantic tokens for the full annotation structure:
`decorator` for `@qualified.Annotation`, `parameter` for argument names,
`string` and `number` for typed values, `keyword` for booleans and identifiers,
and `operator` for delimiters. With `semantic_tokens: "combined"`, Zed can
render annotations as structured source while its Go grammar keeps the comment
prefix visually subordinate.

The current public Zed extension API does not expose an arbitrary buffer
decoration or concealment hook that can hide only the `// ` prefix of a
built-in Go comment. Semantic tokens can restyle ranges but cannot remove
source characters. Spice therefore does not fork the Go grammar or place
invalid raw annotations on disk. The source of truth remains ordinary,
inspectable Go.

## Dependency and verification ownership

The standalone adapter pins the official `zed_extension_api` crate maintained
and Apache-2.0 licensed by Zed Industries. Its `Cargo.lock` fixes the generated
bindings and serialization graph. The adapter performs no network,
credential, process-discovery, or telemetry work of its own; binary and
environment resolution use only Zed's worktree/settings APIs, and the returned
process remains owned by Zed.

Use the fixture and exact commands documented in the
[`spice-framework/zed` README](https://github.com/spice-framework/zed#compatibility-and-verification)
for extension acceptance. Protocol behavior remains independently tested in
core under `internal/lsp`; the standalone fixture proves that the released
adapter and its exact public Spice dependency interoperate without a local
`replace` directive.
