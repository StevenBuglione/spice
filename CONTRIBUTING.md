# Contributing

Spice welcomes focused contributions that advance the roadmap without sacrificing Go interoperability, deterministic behavior, or developer ergonomics.

## Maintainer workflow

The active completion program uses one local writer directly on `main`:

1. Define a bounded developer outcome and public invariants.
2. Record an RFC or ADR when public contracts or architecture change.
3. Implement tests and documentation with the code.
4. Run issue-specific executable and integration paths.
5. Run `make verify` on the exact tree.
6. Fetch `origin/main`, commit the green slice, and push without overwriting unexpected work.

External contributors should open an issue or pull request in the conventional GitHub workflow. The maintainer will reconcile accepted work into the single-writer mainline and run the same local gate.

## Change expectations

Every change should state:

- The developer outcome and exact scope.
- The design choice and important alternatives.
- Public compatibility, security, and operational implications.
- Exact verification commands actually run and their outcomes.
- Runnable smoke or integration evidence.
- Documentation, example, coverage-map, and benchmark effects.
- Known limitations and separately scoped follow-up work.

## Generated and vendored code

Generated output must be deterministic, ownership-tracked, and mechanically reproducible. Never hand-edit generated files or `vendor/`. Tool dependencies belong in the isolated `tools` module.
