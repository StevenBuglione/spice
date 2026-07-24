# Scheduled Agent System

Spice uses three independent hourly tasks staggered across each hour:

- `:00` Researcher and Planner.
- `:20` Implementer.
- `:45` Verifier and Merger.

The prompts in this directory are the version-controlled source of truth. The live scheduled tasks should be updated whenever these files materially change.

The pipeline intentionally separates requirements, implementation, and approval. All durable output belongs in GitHub, and all implementation work must run `make verify` plus issue-specific executable behavior.
