# Zed integration

The repository-owned extension in `editors/zed` attaches the editor-neutral
`spice lsp` server to Go buffers. It adds Spice annotation completion,
versioned diagnostics, hover, quick fixes, module/configuration metadata, and
semantic annotation highlighting while leaving `gopls` responsible for
ordinary Go language features.

The extension contains no compiler and downloads nothing. It resolves an
explicitly configured executable first and otherwise looks for `spice` on the
worktree `PATH`. It launches exactly that executable with the `lsp` argument.
Its structure follows Zed's current
[language-extension contract](https://zed.dev/docs/extensions/languages) and
the pinned
[`zed_extension_api` contract](https://docs.rs/zed_extension_api/0.7.0/zed_extension_api/trait.Extension.html).

## Install a development build

Install the Spice command from the repository root:

```text
go install ./cmd/spice
spice version
```

Open Zed's command palette, choose `zed: install dev extension`, and select the
repository's `editors/zed` directory. Zed compiles the pinned Rust extension to
WebAssembly. The repository validates the same extension on Windows and Linux
with:

```text
make zed
```

`make verify` also runs that target. It requires the pinned Rust 1.93.0
toolchain, the `rustfmt` and `clippy` components, and the `wasm32-wasip2`
target described by `editors/zed/rust-toolchain.toml`.

Zed may not inherit the interactive terminal's `PATH`, especially when
launched from a desktop shortcut. In that case set the executable explicitly
in `.zed/settings.json`. Forward slashes work in a Windows JSON path:

```json
{
  "lsp": {
    "spice": {
      "binary": {
        "path": "C:/Users/me/go/bin/spice.exe"
      },
      "initialization_options": {
        "target": "Commerce",
        "patterns": ["./..."]
      }
    }
  }
}
```

On Linux, the corresponding path is commonly
`/home/me/go/bin/spice`. Omit `initialization_options` for the default target
and package scope. Explicit `binary.arguments` replace the extension's default
`["lsp"]`; include `lsp` when overriding them.

## Go and Spice language servers

Keep both servers enabled:

```json
{
  "languages": {
    "Go": {
      "language_servers": ["gopls", "spice", "..."],
      "semantic_tokens": "combined"
    }
  }
}
```

Spice advertises `@` as a completion trigger. At a declaration-comment
position, accepting an annotation completion inserts valid Go such as:

```go
// @management.Enable(expose=["health"])
```

If the buffer temporarily contains a raw `@Application` line, the Spice quick
fix inserts the missing `// ` prefix as a precise version-aware edit. The
compiler service supplies all validation, module identities, allowed values,
configuration keys, and hover data; the extension does not maintain parallel
metadata.

Zed's extension manifest currently attaches language servers by language, not
by an arbitrary project activation predicate. The extension therefore starts
alongside Go for the worktree, and Spice's bounded typed analysis recognizes
actual Spice annotations and module roots. Recognition never relies on
`go.mod` presence or dependency presence alone and never executes providers.

## Annotation presentation

`spice lsp` emits the standard semantic `decorator` token for the
`@qualified.Annotation` portion of a valid `// @...` declaration. With
`semantic_tokens: "combined"`, Zed can emphasize that token while its Go
grammar continues to render the punctuation as a comment.

The current public Zed extension API does not expose an arbitrary text
decoration or concealment hook that can hide only the `// ` prefix of a
built-in Go comment. Spice therefore does not fork the Go grammar or place
invalid raw annotations on disk to simulate concealment. The source-of-truth
representation remains ordinary, inspectable Go.

## Dependency review

The adapter pins the official `zed_extension_api` 0.7.0 crate maintained and
Apache-2.0 licensed by Zed Industries. It is the only direct dependency and is
required to implement Zed's WebAssembly host contract. `Cargo.lock` fixes its
generated bindings and serialization graph. The adapter performs no network,
filesystem, credential, process-discovery, or telemetry work of its own:
binary and environment resolution use only Zed's worktree/settings APIs, and
the returned process remains owned by Zed. Cancellation and analysis isolation
stay inside the editor-neutral LSP process.

## Diagnostic fixture

Open `editors/zed/fixture` as a Zed project. Its local `replace` points at the
current Spice checkout. In `main.go`:

1. Change `"health"` to `"healthy"` and save; the editor should publish
   `spice.bootstrap.unsupported-item` at that annotation.
2. Restore the value using completion inside the quotes.
3. Add a new raw `@Application` line, request completion or the quick fix, and
   confirm the resulting source is `// @Application`.
4. Hover an annotation to inspect its compiler-derived target and arguments.

Protocol behavior is tested independently of Zed in `internal/lsp`; the
fixture is deliberately small so the visible editor loop is easy to inspect.
