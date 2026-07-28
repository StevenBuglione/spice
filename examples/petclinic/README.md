# Spice Petclinic

This application is Spice's behavior-first port of Spring Petclinic. The
reference source is
[`spring-projects/spring-petclinic`](https://github.com/spring-projects/spring-petclinic)
at commit `f182358d02e4a68e52bdbabf55ca7800288511e7`.

The port preserves the recognizable domain vocabulary, validation rules,
sample data, routes, screens, and persistence profiles while expressing the
application as idiomatic Go with explicit Spice annotations and inspectable
generated code. Production source follows one named type per file so
navigation and debugging remain direct.

Implemented foundations:

- owners, pets, visits, pet types, veterinarians, and specialties;
- immutable validation results with deterministic field ordering;
- concurrency-safe, cancellation-aware in-memory repositories;
- defensive aggregate boundaries and stable query ordering;
- the canonical Petclinic sample data;
- generated direct-call dependency injection and interface assertions;
- a complete-package executable serving the welcome and management routes.

Owner, pet, visit, and veterinarian workflows, SQL persistence profiles,
internationalization, security, and the installed-IDE workflow are delivered
as subsequent bounded slices. The in-memory profile remains the zero-network
default.

Generate and exercise the current target:

```text
go run ./cmd/spice generate --check --target Petclinic ./examples/petclinic/...
go run ./cmd/spice run --target Petclinic ./examples/petclinic/... -- -check
```
