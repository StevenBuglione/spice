# Contributing

Spice welcomes focused contributions that advance the active roadmap without sacrificing Go interoperability or developer ergonomics.

## Workflow

1. Open or select a bounded issue.
2. Discuss architecture through an RFC when the change alters public contracts or compiler behavior.
3. Create a branch.
4. Implement tests and documentation with the code.
5. Run `make verify`.
6. Open a pull request using the template.

## Pull request expectations

Every pull request must state:

- The issue and user outcome.
- The design choice and alternatives considered.
- Exact verification commands run.
- Actual results.
- Relevant runnable smoke behavior.
- Any known limitation or follow-up.

## Generated code

Generated output must be deterministic and committed only when the owning RFC requires committed output. Never hand-edit generated files.
