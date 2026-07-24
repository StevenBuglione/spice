# ADR 0003: Prefer Generation Over Reflection

Status: Accepted

Dependency wiring, routes, configuration binders, and cross-cutting wrappers should be generated when feasible. The runtime remains small and explicit. Reflection may be used only when it provides clear value that cannot reasonably be achieved at compile time.
