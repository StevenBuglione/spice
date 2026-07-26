# Compiler Service

`compiler/service` is the reusable read-only analysis boundary for Spice
commands, development supervision, tests, and editor tooling. It executes the
existing typed compiler stages; it is not a second parser or model.

## Request

One request supplies:

- a workspace or Go module root;
- optional package patterns and application target;
- versioned in-memory file contents keyed by workspace-relative path, absolute
  path, or local `file:` URI;
- a cancellation context;
- an optional monotonic workspace sequence;
- an optional caller-owned content hash for bounded cache reuse.

Overlay paths are normalized to absolute files inside the workspace. Remote
URIs, parent traversal, excessive document counts, and excessive aggregate
bytes fail before package loading. Overlay bytes are defensively copied.
Analysis never applies its generated plan or writes an ownership manifest.

The content hash must identify every relevant disk and overlay input, not only
the currently open document. When it is empty, result caching is disabled.
This prevents a CLI or watcher request from accidentally reusing analysis
after an unrepresented filesystem change.

## Pipeline and result

One load and type universe feeds:

```text
load -> resolve -> validate -> modules -> application IR -> pure generation
```

No provider or marker body executes. Source failures become the same immutable
`compiler/diagnostic.Set` consumed by `spice verify`; cancellation and invalid
service requests remain ordinary Go errors.

An explicitly selected starter catalog is part of the service configuration,
not command-specific post-processing. The service composes its annotation and
bootstrap definitions, loads only its declared constructor packages, validates
only dependencies activated by the application source against the caller's Go
module graph, and adds those constructors to the same provider graph. Missing
or mismatched reviewed versions fail before application generation.

The result exposes defensive:

- resolved annotation summaries and physical/source-mapped locations;
- exact provider nodes, dependencies, and graph edges;
- the immutable application IR;
- module ownership, named APIs, allowed and observed dependencies, and
  unassigned packages;
- generated configuration property metadata with secret defaults omitted;
- completion-safe annotation definitions;
- version-aware safe fixes already attached to diagnostics;
- the selected target name, pure generation plan, and readiness flag.

Consumers never parse rendered diagnostic text and never rebuild metadata from
declaration comments.

## Concurrency and cache policy

A service instance may analyze independent workspaces concurrently. A nonzero
request sequence is tracked per normalized workspace identity. Starting a
newer sequence cancels the older in-flight load, and an older result is
returned as `service.ErrStaleAnalysis` instead of being published.

The service owns a small bounded LRU. Entries are keyed by the caller's
complete content hash together with normalized workspace, target, patterns,
overlays and versions, Go environment/build flags, selected auxiliary
packages, annotation definitions, and extension namespace. Custom compiler
extensions disable caching unless their owner supplies a namespace that
changes whenever extension metadata changes.

There is no global workspace, cache, registry mutation, filesystem write, or
network policy override.

`spice generate`, `spice build`, `spice run`, `spice dev`, and `spice lsp` all
consume this service. The development supervisor reuses one instance and submits its
monotonic change revision as the analysis sequence, so obsolete work cannot be
published after a newer save. The language server uses the same API with
versioned overlays and projects the returned metadata rather than reconstructing
compiler behavior.
