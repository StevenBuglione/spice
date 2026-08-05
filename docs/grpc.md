# gRPC integration

The independently versioned
[`github.com/spice-framework/starter-grpc`](https://github.com/spice-framework/starter-grpc)
module integrates the standard grpc-go and protobuf toolchain without creating
a second IDL, code generator, service registry, or resolver. Applications
continue to define `.proto` contracts and commit ordinary generated Go clients
and servers. Spice owns only explicit construction, transport policy,
lifecycle, limits, and payload-free observation.

```text
go get github.com/spice-framework/starter-grpc@latest
```

## Server

`OpenServer` performs no bind or network operation. It sorts and validates a
bounded set of named registrations, constructs one native `grpc.Server`, invokes
each registration exactly once, and returns lifecycle cleanup. A registration
normally delegates to generated protobuf code:

```go
server, cleanup, err := grpcstarter.OpenServer(
    grpcstarter.ServerConfig{
        TLSConfig:    serviceTLS,
        EnableHealth: true,
    },
    []grpcstarter.Registration{
        {
            Service: "petclinic.OwnerService",
            Register: func(registrar grpc.ServiceRegistrar) error {
                petclinicv1.RegisterOwnerServiceServer(registrar, owners)
                return nil
            },
        },
    },
    rpcObserver,
)
```

The application creates a `net.Listener`, calls `Serve`, and registers cleanup
immediately with its generated lifecycle. `Close` first performs
`GracefulStop`; cancellation force-stops active RPCs and returns the caller's
cause. The optional standard gRPC health service is disabled unless selected.
`SetServing` controls its aggregate status.

Server construction requires a certificate-bearing TLS configuration with TLS
1.2 or newer. Plaintext local development requires `AllowInsecure: true` and
cannot be combined with TLS. Receive/send messages default to 4 MiB and cannot
exceed 16 MiB. Concurrent streams default to 128 and cannot exceed 4096.
Registration names and counts are bounded, duplicate services fail before
serving, and registration panics are converted to payload-free errors.

## Client

`OpenClient` accepts an exact `host:port`, returns a lazy native
`*grpc.ClientConn`, and performs no connection during construction. Generated
protobuf clients use that connection directly:

```go
connection, cleanup, err := grpcstarter.OpenClient(
    grpcstarter.ClientConfig{
        Target:    "owners.internal.example:8443",
        TLSConfig: clientTLS,
    },
    rpcObserver,
)
owners := petclinicv1.NewOwnerServiceClient(connection)
```

Verified TLS with the target hostname and system trust roots is the default.
Custom TLS is cloned defensively, certificate verification cannot be disabled,
and insecure transport requires an explicit opt-out. Callers own RPC contexts,
deadlines, generated request/response types, credentials, retry policy, and
service discovery.

## Observability and boundaries

Observers receive only direction, unary/stream kind, full method, status code,
and elapsed time. Request, response, protobuf metadata, and credentials are not
part of the observation contract. Both unary and stream interceptors propagate
observer contexts.

Spice does not enable gRPC reflection, retries, proxy discovery, authentication,
authorization, or load balancing implicitly. Applications compose those
policies explicitly. The starter never scans packages or registers a global
codec, resolver, connection, or server.

The starter repository owns the canonical [dependency
review](https://github.com/spice-framework/starter-grpc/blob/main/docs/dependency-review.md),
[support policy](https://github.com/spice-framework/starter-grpc/blob/main/docs/support.md),
compatibility manifest, and real TLS/mTLS acceptance evidence. This core
document remains the ecosystem composition guide.
