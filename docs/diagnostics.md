# Diagnostics

Spice uses one shared diagnostic contract for command, development-loop, and
editor integrations. Stage-local compiler metadata is adapted into this model;
renderers do not parse human error strings.

Each diagnostic contains:

- a stable namespaced code such as
  `spice.validation.unknown-argument`;
- error, warning, information, or hint severity;
- an actionable message;
- a canonical physical file URI, path, and half-open start/end range;
- an optional developer-facing display location for `//line` mappings;
- deterministically ordered related information;
- optional narrowly safe text edits with the analyzed document version.

Physical identity controls ordering and editor edits. Display mappings control
human text rendering. This prevents an adjusted filename from becoming
filesystem authority while retaining useful generated-source diagnostics.
Coordinates are one-based in the Spice protocol. An LSP adapter converts them
to LSP's zero-based coordinates at its boundary.

Sets defensively copy nested related information, fixes, edits, and document
versions. They sort by physical URI and source offset, then stable code and
message. The compiler never stores a mutable global diagnostic collection.

## CLI

Text verification includes the stable code:

```text
main.go:8:3: [spice.application.invalid-target] application marker is invalid
```

Machine consumers use:

```text
spice verify --format=json ./...
```

The result is always the `spice.diagnostics/v1` envelope on standard output.
It contains `success`, a summary, and a deterministic `diagnostics` array.
Diagnostic verification failures return exit code 1 while still emitting valid
JSON. Invalid CLI usage returns exit code 2 on the ordinary error stream.
Successful verification emits an empty JSON array, not `null`.

Suggested fixes are data, not permission to rewrite a file. A client must match
the document URI and version, validate every half-open range, refuse ambiguous
or overlapping edits, apply all edits atomically, and preserve valid Go.
