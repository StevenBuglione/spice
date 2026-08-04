# ADR 0001: Begin with One Monorepo

Status: Superseded by ADR 0012

Spice begins in one repository so compiler, runtime, docs, examples, starters, and agent instructions can evolve atomically. Repositories will split only after independent release cadence, ownership, or scale creates a measurable need.

That measurable need is now established. The accepted replacement and the
history-preserving transition are recorded in
[ADR 0012](0012-multi-repository-product-boundaries.md).
