# Product Principles

## Be broader than a router

Spice is an application platform. Routing is only one capability. The value comes from an integrated, consistent developer experience across configuration, lifecycle, security, data, events, observability, testing, and architecture.

## Prefer capability parity to implementation parity

Spice should cover valuable Spring Boot outcomes but implement them with Go-native mechanisms. Generated wrappers replace subclass proxies. Goroutines replace reactive abstractions where appropriate. Go packages and `internal` boundaries become inputs to module verification.

## Make the easy path excellent

A new developer should receive:

- Clear project structure.
- Strong defaults.
- Source-positioned diagnostics.
- Generated code that can be inspected.
- One verification command.
- Examples that actually execute.

## Keep escape hatches explicit

The framework should enforce standards while allowing reviewed exceptions. Escape hatches must be documented, visible, and analyzable—not accidental bypasses.

## Test the developer experience

Every major feature should include:

- A minimal example.
- A modular reference application.
- Failure-case diagnostics.
- Build and startup benchmarks.
- A first-use workflow test.
