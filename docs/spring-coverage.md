# Spring Capability Coverage Map

Spice aims to cover as much of the practical Spring Boot and Spring Modulith platform as provides value to Go developers. This is a living product map, not a promise to port Java classes verbatim.

| Area | Spring capability | Spice direction | Status |
|---|---|---|---|
| Core | Application bootstrap and lifecycle | Exact roots/hooks/cleanup, immutable IR, generated construction/start/stop/run, rollback, idempotency, and caller-owned context policy | available |
| Core | Dependency injection | Exact-type catalog/graph, deterministic direct-call constructor/rollback, and guarded generated-file application | available |
| Core | Auto-configuration | Explicit starter manifests with compile-time conditions | planned |
| Core | Starters and dependency alignment | Versioned Go starter modules and compatibility manifest | planned |
| Configuration | External properties and profiles | Reflection-free schema/snapshot runtime, ordered defaults/sources, environment mapping, profiles, provenance, validation, and redaction are available; file sources and generated binders follow | in-progress |
| Web | MVC/REST controllers | Generated `net/http` adapters from controller annotations | planned |
| Web | Request binding and validation | Typed path, query, header, and body binding | planned |
| Web | Error handling and content negotiation | Standard policy interfaces and generated adapters | planned |
| Web | OpenAPI and REST documentation | Generated contracts and examples | planned |
| Web | WebSocket, GraphQL, gRPC | Go-native integrations and starters | planned |
| Security | Authentication and authorization | Middleware, generated method guards, OAuth2/OIDC starters | planned |
| Security | Resource server and client | Standards-based Go integrations | planned |
| Data | JDBC and transaction management | `database/sql`/driver integrations and generated transaction decorators | planned |
| Data | Repository abstractions | Optional generated repositories and query integration | planned |
| Data | JPA/Hibernate | Go-native persistence patterns; no transparent entity runtime | planned |
| Data | Redis, MongoDB, Elasticsearch, Cassandra, Neo4j | Curated starters around mature Go clients | planned |
| Migration | Flyway/Liquibase | Migration-tool starters and module ownership checks | planned |
| Messaging | Kafka, RabbitMQ, JMS-like APIs, Pulsar | Typed publishers/listeners with client-specific starters | planned |
| Events | Application events | Typed in-process and durable event publication | planned |
| Scheduling | Scheduled and asynchronous work | Generated scheduler registration and lifecycle | planned |
| Batch | Batch jobs | Job/step abstraction with restart and observability support | planned |
| Cache | Cache abstraction | Typed cache interfaces and generated decorators | planned |
| Observability | Actuator endpoints | Health, info, metrics, routes, modules, config metadata | planned |
| Observability | Metrics and tracing | OpenTelemetry-first integrations | planned |
| Testing | Application context and test slices | Generated test application graphs and focused module/web/data tests | planned |
| Development | Devtools and reload | Fast generate/test loop and optional reload integration | planned |
| Modulith | Module discovery | Import-path roots, longest-root package ownership, root APIs, named interfaces, and unassigned-package metadata | available |
| Modulith | Module dependency verification | Real Go import edges, explicit allowed APIs, internal rejection, and deterministic strongly connected cycle paths | available |
| Modulith | Module tests | Focused transitive dependency graphs and dependency-first composition are available; generated test harnesses follow | in-progress |
| Modulith | Event publication registry | Durable event publication and completion tracking | planned |
| Modulith | Documentation | Read-only JSON canvases plus deterministic Mermaid and PlantUML module diagrams | available |
| Modulith | Runtime observations | Generated module ownership and synchronous lifecycle observation seam are available; OpenTelemetry spans and interaction metrics follow | in-progress |
| Current bootstrap | Annotation parser, declaration scan, and target metadata | Valid comment syntax, source association, typed built-in definitions, and target diagnostics | available |
| Current bootstrap | Provider dependency validation | Exact-type missing-dependency diagnostics, cycle detection, and deterministic dependency-first metadata | available |
| Current bootstrap | Provider cleanup metadata | Named context-aware cleanup result validation retained for future generated rollback and shutdown | available |
| Current bootstrap | Lifecycle hook metadata | Exact provider-owned `@OnStart`/`@OnStop` validation plus direct generated execution through the lifecycle coordinator | available |
| Current bootstrap | Application roots and immutable IR | Exact `@Application` parameter roots plus ordered provider, graph, cleanup, and lifecycle metadata without executing marker bodies | available |
| Current bootstrap | Lifecycle runtime coordination | Explicit callback state machine with dependency-order start, reverse stop/cleanup, startup rollback, idempotency, and caller-owned contexts | available |
| Current bootstrap | Deterministic generation plan | Pure target-scoped Go rendering with direct calls, stable imports, canonical SHA-256 manifest metadata, and executable generated fixtures | available |
| Current bootstrap | Safe generation and build commands | Rooted ownership checks, manual-edit refusal, no-op preservation, check/diff, stale recovery, target locks, and trimpath build | available |
| Current bootstrap | Executable generated reference | Committed provider graph, generated application, live HTTP behavior, process-signal ownership, freshness checks, and graceful drain | available |
| Current bootstrap | Engineering quality and reproducible builds | Go-owned cross-platform verification with pinned formatting, lint, nil-safety, security, fuzz, race, coverage, vendor, and executable checks | available |

## Coverage rules

Every new area must define:

1. The developer-facing API.
2. The generated or runtime behavior.
3. Failure diagnostics.
4. Security defaults.
5. Unit and integration tests.
6. A runnable example.
7. The intended Spring capability relationship.
