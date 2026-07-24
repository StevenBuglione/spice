# ADR 0002: Preserve Valid Go Source

Status: Accepted

Spice annotations use declaration comments rather than raw `@` syntax or a Go compiler fork. This preserves interoperability with `gofmt`, `go test`, `go vet`, debuggers, profilers, security scanners, and standard editors.
