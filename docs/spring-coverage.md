# Spring Capability Coverage Map

Spice aims to cover as much of the practical Spring Boot and Spring Modulith platform as provides value to Go developers. This is a living product map, not a promise to port Java classes verbatim.

| Area | Spring capability | Spice direction | Status |
|---|---|---|---|
| Core | Application bootstrap and lifecycle | Context-aware provider cleanup metadata is available; generated invocation and lifecycle runtime follow | in-progress |
| Core | Dependency injection | Exact-type provider catalog, cleanup metadata, and dependency-graph validation; generated constructor wiring follows | in-progress |
| Core | Auto-configuration | Explicit starter manifests with compile-time conditions | planned |
| Core | Starters and dependency alignment | Versioned Go starter modules and compatibility manifest | planned |
| Configuration | External properties and profiles | Typed configuration binding, sources, profiles, validation | planned |
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
| Modulith | Module discovery | Package and declaration-derived module model | planned |
| Modulith | Module dependency verification | Cycles, internals, allowed dependencies, named interfaces | planned |
| Modulith | Module tests | Module-specific generated test harnesses | planned |
| Modulith | Event publication registry | Durable event publication and completion tracking | planned |
| Modulith | Documentation | Mermaid/PlantUML/JSON module diagrams and canvases | planned |
| Modulith | Runtime observations | Module-aware spans and interaction metrics | planned |
| Current bootstrap | Annotation parser, declaration scan, and target metadata | Valid comment syntax, source association, typed built-in definitions, and target diagnostics | available |
| Current bootstrap | Provider dependency validation | Exact-type missing-dependency diagnostics, cycle detection, and deterministic dependency-first metadata | available |
| Current bootstrap | Provider cleanup metadata | Named context-aware cleanup result validation retained for future generated rollback and shutdown | available |

## Coverage rules

Every new area must define:

1. The developer-facing API.
2. The generated or runtime behavior.
3. Failure diagnostics.
4. Security defaults.
5. Unit and integration tests.
6. A runnable example.
7. The intended Spring capability relationship.
