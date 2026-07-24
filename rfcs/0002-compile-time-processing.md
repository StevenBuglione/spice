# RFC 0002: Compile-Time Processing Pipeline

Status: Proposed

## Summary

Spice will load Go packages, parse annotations, resolve Go symbols and types, build a typed intermediate representation, validate application and module rules, and generate deterministic ordinary Go.

## Principles

- No runtime classpath scanning analogue.
- No reflection-heavy dependency container by default.
- Generated code must be readable and source-mapped.
- Generation must be deterministic and testable with golden files.
- `spice build` will eventually run generation, verification, and the standard Go build.

## Initial phases

1. Annotation parsing.
2. Declaration association.
3. Annotation definition registry.
4. Go symbol and type resolution.
5. Typed IR.
6. Dependency and module graph validation.
7. Code generation.
8. Standard Go compilation.
