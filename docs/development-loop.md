# Development Loop

`spice dev` is a repository-owned development supervisor. It is not a shell
loop and it never builds over the executable that is currently running.

## Run

From a Go module containing one preferred package-main `@Application`:

```text
spice dev [--target name] [dev-option ...] [package-pattern ...] [-- application-argument ...]
```

The defaults are:

```text
--quiet=150ms
--max-delay=2s
--poll=500ms
--stop-timeout=15s
```

`--include pattern` extends the default relevant-file policy and may be
repeated. `--exclude pattern` removes paths and may also be repeated. Patterns
are workspace-relative, accept ordinary Go path globs, and support `**`
segments.

Application arguments after `--` are passed directly to every candidate
process without a shell.

## Candidate pipeline

Every accepted change batch has one monotonically increasing revision:

1. load and validate the existing typed compiler pipeline with cancellation;
2. report the same source-positioned compiler failures used by other CLI
   commands;
3. render and apply generation through the ownership guard;
4. build the exact package-main target with `go build -trimpath`;
5. keep the result in a unique temporary directory;
6. request bounded graceful stop only after the candidate is complete;
7. launch the candidate in an isolated process group;
8. release obsolete candidate artifacts idempotently.

If analysis, generation, or compilation fails, the current process is not
stopped. The supervisor reports that it remains on the last-known-good
revision and continues watching. An initial failure also leaves the watcher
active, so saving a correction can produce the first process.

The reusable engine has explicit contracts for file events, clocks/timers,
candidate preparation, process launch, graceful stop, and structured events.
Tests use synthetic events and a fake clock; the command-level recovery test
builds and restarts a real generated package-main application under the race
detector.

## Watch policy

The portable watcher recursively scans the workspace on Windows and Linux.
It discovers new and renamed directory trees without registration races and
hashes relevant regular-file contents, so an editor save is detected even
when size and modification time are unchanged. Polling is the correctness
baseline and will remain the bounded fallback when a native notification
accelerator is added.

Defaults observe:

- Go source and `go.mod`, `go.sum`, `go.work`, and `go.work.sum`;
- JSON, YAML, TOML, SQL, HTML, and common template files;
- additional assets named by `--include`.

Defaults ignore:

- `.git`, `vendor`, `node_modules`, editor/tool directories, and conventional
  build output;
- temporary editor files;
- `.spice` build/dev artifacts;
- generated `zz_spice_gen.go`, generated `openapi.json`, ownership manifests,
  and legacy `internal/spicegen` output.

Paths are normalized to workspace-relative slash form. Unsafe absolute or
parent-traversing include/exclude patterns fail before watching starts.

## Process and output policy

Each child owns a process group. Graceful replacement sends the platform
interrupt (`CTRL_BREAK` on Windows and a group signal on Unix), waits for the
generated lifecycle and HTTP drain, and escalates only after
`--stop-timeout`. Cancellation of the supervisor uses the same bounded stop
path and does not leave an intentionally detached child.

Development observations are concise and structured around change detection,
analysis, generation, build, failure, last-known-good retention, restart,
start, exit, watcher recovery, and artifact cleanup. The supervisor never
prints environment contents, authorization tokens, SQL arguments, message
bodies, or attachment contents. Compiler/build output is bounded before it is
attached to a failure.

## Current boundary

This slice provides the complete CLI supervisor and portable fallback watcher.
The overlay-aware compiler service, editor/LSP debounce, native notification
accelerator, and Zed integration are separate remaining developer-proof
deliverables. Generated application code remains ordinary committed Go and
requires no Spice compiler at runtime.
