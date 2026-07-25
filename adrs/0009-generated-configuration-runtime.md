# ADR 0009: Generated configuration with explicit sources

Status: Accepted

## Context

Production services need external configuration, profiles, validation,
provenance, and safe diagnostics. A reflection-based generic struct binder
would weaken compile-time feedback and repeat work at startup.

## Decision

Generated Spice binders will target the public `config` runtime. Generated
metadata is a sorted schema of exact keys, scalar kinds, defaults, required and
secret flags, owning modules, descriptions, and optional environment names.
Generated decoders use typed snapshot accessors and ordinary Go assignments;
the runtime does not inspect structs with reflection.

Resolution starts with schema defaults and applies explicitly ordered sources
from lowest to highest precedence. Every winning value retains its source.
Source names are unique. Unknown keys fail closed unless the caller
deliberately enables them. Required and scalar validation happens before typed
decode. Typed validators execute afterward in declaration order.

Active profile identities are validated, deduplicated, order-preserving, and
passed to every source. Sources decide how profiles affect their own values.

The standard JSON source opens one caller-selected directory through `os.Root`.
It reads `<base>.json`, then optional `<base>-<profile>.json` files in active
profile order. Each file has a caller-configurable size bound (1 MiB by
default). Objects flatten to dotted keys; strings, Booleans, and JSON numbers
become scalar text. Duplicate object keys, flattened collisions, arrays, nulls,
invalid keys, non-object roots, and trailing values fail closed.

The environment source reads only schema-declared variables. Explicit
environment names win over deterministic prefix plus upper-snake key mapping,
and collisions fail rather than silently alias.

Secret values remain available to the generated decoder but are replaced with
`<redacted>` by every safe snapshot representation. Scalar and validation
errors identify keys and sources without including raw values.

## Boundaries

- Core configuration performs no hidden network access.
- Source order is explicit caller policy.
- Runtime schema and snapshots are immutable by convention and return
  defensive copies.
- Generated binders, not reflection, own struct construction.
- Compiler generation builds on this runtime contract.

## Consequences

- Configuration startup behavior is deterministic and inspectable.
- Tests can use immutable map sources and injected environment lookup.
- Provenance and module ownership are available for future actuator and
  observability metadata.
- Custom sources implement one small context-aware interface.
