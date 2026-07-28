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

Petclinic is a real consuming Go module rather than a package inside the
framework module. Its own `go.mod` authorizes the Spice annotation tool, uses a
local `replace` only while this repository is under development, and owns its
generated source, ownership manifest, and compact vendor tree. The repository
quality gate verifies this module independently, including shuffled/race tests,
security analysis, vendor-only tests, generation freshness, and an executable
application check.

Implemented foundations:

- owners, pets, visits, pet types, veterinarians, and specialties;
- immutable validation results with deterministic field ordering;
- concurrency-safe, cancellation-aware in-memory repositories;
- defensive aggregate boundaries and stable query ordering;
- the canonical Petclinic sample data;
- generated direct-call dependency injection and interface assertions;
- a complete-package executable serving the welcome and management routes;
- the complete owner web workflow: find, paginated results, details, create,
  edit, validation, redirects, and missing-owner problem responses;
- nested pet creation/editing and visit registration with pet-type reference
  data, duplicate-name checks, aggregate-owned identities, validation, and
  owner/pet not-found problem responses;
- paginated veterinarian HTML plus a stable JSON collection with canonical
  lower-camel-case fields and ordered specialties.
- a PostgreSQL target with required redacted configuration, reviewed pgx pool
  ownership, module-owned schema/seed migrations, explicit repository
  interface bindings, aggregate-safe transactions, and a real PostgreSQL 18.3
  workflow test.

The in-memory target remains the zero-network default. PostgreSQL is a separate
compile-time application graph rather than a runtime service-locator branch:
its selected concrete repositories are visible in generated Go and debugger
stacks. MySQL, internationalization, security, and the installed-IDE workflow
are delivered as subsequent bounded slices.

Current application routes:

| Method | Path | Purpose |
| --- | --- | --- |
| `GET` | `/` | Welcome page |
| `GET` | `/owners/find` | Owner search form |
| `GET` | `/owners` | Prefix search and paginated results |
| `GET`, `POST` | `/owners/new` | Create an owner |
| `GET` | `/owners/{ownerId}` | Owner, pets, and visits |
| `GET`, `POST` | `/owners/{ownerId}/edit` | Edit an owner |
| `GET`, `POST` | `/owners/{ownerId}/pets/new` | Add a pet |
| `GET`, `POST` | `/owners/{ownerId}/pets/{petId}/edit` | Edit a pet |
| `GET`, `POST` | `/owners/{ownerId}/pets/{petId}/visits/new` | Add a visit |
| `GET` | `/vets.html` | Paginated veterinarian browser view |
| `GET` | `/vets` | Veterinarian JSON collection |
| `GET` | `/actuator/*` | Generated management endpoints |

Generate and exercise the current target:

```text
go build -trimpath -o ./bin/spice ./cmd/spice
cd examples/petclinic
../../bin/spice generate --check --target Petclinic . ./memory ./model ./owner ./presentation ./system ./vet
../../bin/spice run --target Petclinic . ./memory ./model ./owner ./presentation ./system ./vet -- -check
```

Generate the PostgreSQL graph:

```text
cd examples/petclinic
../../bin/spice generate --check --target Postgres ./cmd/postgres ./owner ./postgres ./presentation ./system ./vet
set SPICE_PETCLINIC_POSTGRES_URL=postgres://petclinic:petclinic@127.0.0.1:5432/petclinic?sslmode=disable
set SPICE_PETCLINIC_POSTGRES_ALLOW_INSECURE=true
../../bin/spice run --target Postgres ./cmd/postgres ./owner ./postgres ./presentation ./system ./vet
```

The environment names are the same on every platform; the example above uses
Windows `set` syntax only for brevity. Disabled TLS is accepted solely through
the explicit local-development opt-in. Production URLs default to verified
TLS.

Run the real PostgreSQL repository workflow against an already-started test
database:

```text
set SPICE_POSTGRES_TEST_URL=postgres://petclinic:petclinic@127.0.0.1:5432/petclinic?sslmode=disable
go test -tags=integration -count=1 ./postgres
```
