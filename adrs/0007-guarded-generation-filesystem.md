# ADR 0007: Guarded Generated-File Ownership

Status: Accepted

## Decision

`internal/genfs` is the only normal filesystem application layer for a
`compiler/generate.Plan`. It uses Go's rooted filesystem API and treats the
manifest as an ownership record, not a cache.

Before writing, it validates:

- an absolute selected module root and clean module-relative slash paths;
- no parent traversal, Windows device names, case-fold collisions, or
  manifest/output overlap;
- exact agreement between plan files, hashes, and manifest entries;
- no symlink in an existing output or manifest path component;
- a supported manifest schema and exact target identity;
- prior owned-file hashes and generated markers;
- absence of unowned expected-path collisions and unexpected unowned Spice
  generated markers.

Normal generation refuses to replace a manually edited owned file, refuses even
a byte-identical file without manifest ownership, and removes a stale file only
when its bytes still match the prior manifest and retain the generated marker.

## Apply protocol

One target lock is created with exclusive creation under `.spice/`. After
locking, state is re-read and validated. Changed files are written to
same-directory exclusive temporary files, synced, parsed when Go source,
hash-verified, and replaced with backup restoration on failure. Unchanged files
and manifests are not rewritten, preserving mtimes. The manifest is replaced
last as the commit marker.

The protocol is guarded and recoverable but is not described as a globally
atomic multi-file transaction.

## Commands

- `spice generate` renders and safely applies one selected application.
- `spice generate --check` is read-only and exits nonzero for any difference.
- `spice generate --diff` is read-only and prints a bounded deterministic diff.
- `spice build` safely generates, then runs `go build -trimpath ./...` in the
  selected module.

Generated files use `//go:build !spice_generate`. Spice reserves the
`spice_generate` tag for generation analysis and merges it with
caller/`GOFLAGS` tags so targeted regeneration can exclude stale output.
Verification and annotation listing load the ordinary committed program,
including generated dependencies imported by commands.
