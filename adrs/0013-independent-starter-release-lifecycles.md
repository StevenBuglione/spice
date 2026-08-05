# ADR 0013: Independent starter release lifecycles

Status: Accepted

## Context

ADR 0012 established that external-system integrations do not belong in the
standard-library-first core module. Its initial repository table grouped
OpenTelemetry under `starter-observability` and OAuth2/OIDC under
`starter-security`. Implementation and acceptance exposed different dependency,
security, and compatibility lifecycles inside both proposed groups:

- OpenTelemetry adapters evolve with the OpenTelemetry API and provider model;
- OAuth2 client credentials depend on outbound HTTP token acquisition policy;
- OIDC resource servers depend on discovery, JWKS, JWT, and identity-provider
  interoperability;
- gRPC, WebSocket, and Kafka each require a different real-system matrix.

Keeping unrelated integrations together would reproduce the module-graph and
release coupling that the multi-repository migration is intended to remove.

## Decision

Every production starter is an independently versioned Go module when it owns
a distinct external dependency or interoperability lifecycle. The initial
repositories are:

- `spice-framework/starter-smtp`;
- `spice-framework/starter-postgres`;
- `spice-framework/starter-mysql`;
- `spice-framework/starter-redis`;
- `spice-framework/starter-otel`;
- `spice-framework/starter-oauth2client`;
- `spice-framework/starter-oidc`;
- `spice-framework/starter-websocket`;
- `spice-framework/starter-grpc`;
- `spice-framework/starter-kafka`.

Each module depends only on public Spice contracts, declares a strict minimum
and current compatible core revision, owns its direct dependency review and
vendor graph, and proves its relevant real-system behavior. Repositories are
not combined merely to reduce repository count.

The `spice-framework/development` compatibility catalog is the authoritative
tested ecosystem inventory. It coordinates workspaces and verification but is
not a package resolver or BOM. Applications continue to select exact versions
through ordinary `go.mod`, `go.sum`, MVS, and optional `go.work` files.

Core may retain dependency-free portable starter metadata and static
auto-configuration contracts. It does not retain a compatibility forwarding
package for an extracted external client. An external starter becomes
authoritative only after filtered history, standalone verification, hosted
acceptance, and a durable organization repository are green.

## Consequences

Core consumers no longer select client libraries they do not use. A security
or compatibility release in one starter does not force unrelated starter
releases. Each repository has a smaller and faster local feedback loop and an
acceptance matrix matched to its system.

There are more repositories to govern. That cost is controlled by reusable
organization workflows, identical quality contracts, machine-readable
compatibility metadata, and the development catalog. Cross-starter features
compose through public Spice interfaces rather than private repository access
or lockstep versions.

This decision refines the starter names in ADR 0012; its foundation, editor,
reference-application, module, and dependency-tier decisions remain in force.
