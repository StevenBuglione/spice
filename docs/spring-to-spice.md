# Spring-to-Spice guide

Spice targets the useful outcomes of Spring Boot and Spring Modulith, not Java
API compatibility. Go packages, functions, interfaces, contexts, and generated
direct calls replace runtime classpath discovery, proxies, and a mutable
application context.

## Concept map

| Spring concept | Spice contract | Important difference |
| --- | --- | --- |
| `@SpringBootApplication` | `// @Application` on `func main()` | The marker body remains ordinary Go; generated `spiceMain` owns process conventions. |
| `@Component`, `@Service`, `@Repository`, `@Controller` | Explicit Spice stereotypes | Discovery is bounded to the selected package scope and validated at compile time. |
| Constructor injection | Constructor parameters | This is the only injection style; fields and setters are not injected. |
| Interface autowiring | `@Implements(pkg.Interface)` or an exact interface-returning bean | General Go assignability does not silently create candidates. |
| `@Qualifier`, `@Primary`, fallback beans | Qualifier parameter comments and bean metadata | Selection is deterministic and ambiguity is a compile error. |
| `ObjectProvider<T>` | `bean.Optional[T]`, `bean.Lazy[T]`, `bean.Provider[T]` | Handles are generic Go values with explicit acquisition and cleanup. |
| Singleton/prototype/request/session scopes | Explicit bean scope annotations | Cleanup belongs to a typed scope; there is no global container. |
| `@ConfigurationProperties` and profiles | Typed configuration declarations and explicit sources | Precedence and provenance are generated; secrets are redacted. |
| Spring MVC controllers | `@Controller`, route annotations, typed request DTOs | Adapters are generated `net/http` code with no reflection. |
| `ResponseEntity`, validation, problem details | Typed response values, `validation.Result`, RFC 9457 errors | Binding and validation failures have bounded source-safe representations. |
| Spring Security method/route policies | `@security.Authorize` and explicit authentication middleware | Authorization is generated deny-by-default; identity still comes from an explicit starter or caller. |
| `@Transactional` | `@data.Transactional` generated boundaries and `data.Manager` | Transaction ownership is visible and executor-based, not proxy interception. |
| Spring Data repositories | Ordinary repository interfaces plus explicit implementations | SQL and row decoding remain visible Go; Spice does not derive queries from method names. |
| Actuator | Explicit `@management.Enable` allowlist | Nothing is exposed by dependency presence; access is public or direct-peer loopback by declaration. |
| Application events | Typed `event.Topic[T]` and event annotations | Publication is instance-owned and payload types are compile-time exact. |
| Spring Modulith modules | Package `@Module`, named interfaces, allowed dependencies | Real Go import edges and `internal` rules are checked before generation. |
| Test application context and slices | `spicetest.NewContext`, `NewHTTP`, `NewSQL`, `spice test --module` | Tests receive typed generated values, never a mutable bean lookup API. |
| DevTools and IDE metadata | `spice dev`, `spice lsp`, GoLand plugin | One compiler service owns diagnostics; the physical file always remains valid Go. |

## Dependency injection example

Spring commonly selects one implementation of an interface through component
scanning and qualifiers. Spice makes the candidate edge explicit while leaving
the interface and constructor idiomatic:

```go
// @import { Implements, Primary, Qualifier, Service } from "github.com/StevenBuglione/spice/annotation/core"
// @import * as payments from "example.com/shop/payments"

// @Service(name="stripeProcessor")
// @Implements(payments.Processor)
// @Qualifier("stripe")
// @Primary
type StripeProcessor struct{}

func NewStripeProcessor(config Config) (*StripeProcessor, error) {
	return &StripeProcessor{}, nil
}
```

```go
func NewCheckout(
	// @Qualifier("stripe")
	processor payments.Processor,
) (*Checkout, error) {
	return &Checkout{processor: processor}, nil
}
```

Spice resolves `payments.Processor` with the same `go/types` program used for
the rest of compilation. It verifies pointer/value and generic method sets and
writes the ordinary compile-time assertion into an owned shard beside the
implementation. The application wiring contains a direct constructor call and
typed assignment. There is no IDE-provided resolution and no runtime lookup.

## What intentionally does not translate

- Component discovery does not depend on an imported module merely being on
  the classpath/module graph.
- There is no reflection container, bean-name lookup, runtime package scan, or
  mutable global application context.
- There are no field injection, circular proxy, or hidden transactional method
  calls. Cycles fail compilation and transaction boundaries are generated.
- Interface assignability alone is not a bean declaration. `@Implements`
  documents the architectural intent and makes selection inspectable.
- Annotation comments never execute Go code during analysis. Descriptors are
  statically decoded and handlers contribute versioned typed data.
- Starter dependencies are opt-in and reviewed. Package presence cannot open a
  port, read the environment, contact a service, or activate behavior.

## Migration sequence

For a Spring service moving to Spice:

1. Preserve domain types and ports as small Go packages and interfaces.
2. Convert constructors first; remove field/setter injection and cycles.
3. Add explicit stereotype and `@Implements` declarations.
4. Move properties into typed configuration and make source precedence
   explicit.
5. Port controllers to typed request/response contracts.
6. Put transaction ownership at generated HTTP or explicit service
   boundaries; keep repositories executor-oriented.
7. Declare Modulith package roots and allowed dependencies.
8. Add generated context, HTTP, SQL, and focused module tests.
9. Enable only required management, authentication, telemetry, and external
   starters.
10. Inspect and commit generated Go before replacing the production process.

The executable Petclinic port demonstrates that sequence with one named type
per file, explicit repository interfaces, in-memory/PostgreSQL/MySQL graphs,
generated web adapters, localization, management, and ordinary Go tests.

The detailed implementation status for each capability is maintained in
[spring-coverage.md](spring-coverage.md). `available` means the documented
Spice contract is executable; `integration` means a useful core exists but a
broader ecosystem or operational path remains explicit; `not-planned` records
a deliberate Java-specific non-goal and the Go-native replacement. Release
verification rejects unresolved `planned` and `in-progress` rows.
