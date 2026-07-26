# Dependency review: OpenTelemetry Go

- Decision: approved for the isolated `starter/otel` package.
- Version: `go.opentelemetry.io/otel` and SDK v1.43.0.
- Upstream: <https://github.com/open-telemetry/opentelemetry-go>.
- License: Apache-2.0; retained in vendored module license files.
- Maintenance: vendor-neutral CNCF/OpenTelemetry project with signed immutable
  releases and an explicit supported-Go policy.
- Compatibility: v1.43.0 explicitly supports Go 1.26. Traces and metrics are
  stable; the beta logs API is deliberately not adopted.
- Security: exporter packages are not included by this starter. Applications
  choose and configure transport/export separately. `govulncheck` and `gosec`
  remain mandatory for the reachable repository graph.
- Cancellation: all span and measurement operations use the request context.
  Provider shutdown and exporter deadlines remain application-owned.
- Observability: the adapter emits server spans and four bounded metrics using
  generated route templates, stable symbol IDs, module paths, method, and
  status. It never labels telemetry with raw paths, queries, headers, or body
  values.
- Configuration: providers are explicit constructor inputs; the starter does
  not read environment variables, install globals, contact a collector, or
  select an exporter.
- Activation: the committed manifest must be explicitly selected and
  `@otel.Enable` must occur on the application marker. The compiler validates
  the exact HTTP-observer output and reachable mux capability before generated
  code directly composes the constructor result.
- Transitive scope: OpenTelemetry API/SDK support libraries only. No collector,
  network exporter, protobuf, gRPC, or Prometheus dependency is accepted by
  this slice.

Primary references:

- <https://github.com/open-telemetry/opentelemetry-go/releases>
- <https://opentelemetry.io/docs/languages/go/>
