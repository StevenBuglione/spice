# Spring Core parity

Spice measures core parity against the documented technology areas in
[Spring Framework 7.0.8 Core Technologies](https://docs.spring.io/spring-framework/reference/core.html).
Parity means that the valuable developer outcome has an executable Go-native
contract, or that a Java-specific mechanism is deliberately replaced and
classified. It does not mean reproducing Java reflection, class loaders,
runtime proxies, or the Spring API.

## Capability matrix

| Spring Core area | Spice contract | Classification |
| --- | --- | --- |
| IoC container and bean configuration | Compile-time provider discovery, exact constructor injection, explicit interface bindings, qualifiers, primary/fallback selection, ordered collections, typed handles, generated scopes, lifecycle ownership, and direct ordinary-Go construction | available |
| Environment, profiles, and property sources | Generated typed schemas and binders over explicitly ordered JSON/profile/environment sources with provenance, validation, and secret redaction | available |
| Application context lifecycle | Generated construction/start/stop/run, dependency-first startup, reverse shutdown, rollback, caller-owned contexts/signals, typed component snapshots, application events, and message catalogs | available |
| Resources | Instance-owned `resource.Loader` resolves canonical `spice://mount/path` locations over explicit caller-owned `fs.FS` mounts. Reads are context-aware, bounded, rooted by the supplied filesystem, and never perform hidden network access | available |
| Validation | Layer-neutral `validation.Validator[T]`, composable `ValidatorFunc[T]`, immutable rejected-value-free violations, deterministic ordering, cancellation, and bounded aggregation | available |
| Data binding | Compiler-generated typed configuration and HTTP binders use exact Go fields and source positions. Dynamic JavaBean property mutation and reflection-based `BeanWrapper` behavior are intentionally replaced by generated code | available |
| Type conversion and formatting | `conversion.Converter[S,T]`, typed composition, custom codecs, and safe built-in Boolean/integer/float/duration/time/URL codecs are reflection-free. Configuration and HTTP binding use the same scalar conversions | available |
| Events | Generated typed topics, exact publisher interfaces, deterministic synchronous subscribers, module identities, observations, and durable outbox integration | available |
| Internationalization | Immutable bounded UTF-8 property catalogs, deterministic locale negotiation, fallback, and explicit injection | available |
| Spring Expression Language | No runtime expression language. Typed Go functions, generated configuration access, and descriptor literals replace string evaluation, bean-name access, reflective method invocation, and expression injection risk | not-planned |
| Spring AOP and AspectJ | Security, transactions, caching, observations, HTTP middleware, lifecycle, retry, and events use explicit generated or typed decorators. Universal runtime pointcuts, subclass proxies, load-time weaving, and self-invocation magic are not planned | integration |
| Resilience | Explicit finite retry policies, classification, capped backoff, cancellation, exhaustion, observations, and panic-safe async/event boundaries | available |
| Null safety | Go's type system, explicit optional handles, compile-time interface checks, NilAway, vet, and generated fail-closed diagnostics replace Java nullability annotations | available |
| Data buffers and codecs | Standard `io` ownership plus bounded JSON, MIME, HTTP, WebSocket, gRPC, configuration, and resource codecs cover application boundaries. A universal reference-counted pooled-buffer API is not introduced | integration |
| Ahead-of-time optimization | One immutable typed program produces deterministic, inspectable, trimpath ordinary Go with no runtime scanner, reflection container, or compiler dependency | available |
| Bean definition inheritance and runtime `BeanFactory` mutation | Explicit constructor composition, typed factory functions, generated test overrides, and library auto-configuration replace mutable definition inheritance and runtime registration | not-planned |
| Class loader and load-time weaving facilities | Go modules, imports, static linking, `go tool` annotation handlers, and generated direct calls are the replacement; JVM class-loader APIs have no useful Go equivalent | not-planned |

## Core invariants

- No package-level mutable container or global converter/resource registry.
- No implicit network access, package scanning, or dependency-by-presence
  activation.
- No rejected configuration, request, or conversion value is placed into a
  framework error.
- Every resource filesystem, converter, validator, event topic, catalog, and
  lifecycle is explicitly owned and injectable.
- Compiler-visible behavior remains valid Go and generated runtime behavior
  remains ordinary inspectable Go.

## Executable evidence

The `conversion`, `resource`, and `validation` packages contain positive,
negative, boundary, cancellation, safe-error, composition, deterministic-order,
and runnable example tests. Existing `config`, `web`, `event`, `i18n`, `bean`,
`lifecycle`, `retry`, compiler, generated-code, Commerce, and Petclinic suites
prove the remaining rows. `make verify` runs them shuffled, race-enabled,
coverage-gated, vendor-offline, and through executable generated applications.
