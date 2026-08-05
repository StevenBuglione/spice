# Releasing Spice core

This repository publishes the standard-library-only
`github.com/spice-framework/spice` library module. It does not build the Spice
CLI or own release-construction implementation; those responsibilities belong
to [`spice-framework/toolchain`](https://github.com/spice-framework/toolchain).

## Core release contract

A core version may be tagged only from an exact clean commit that:

1. passes `make verify` under Go 1.26.5 on the supported host matrix;
2. preserves the public API maturity policy and compatibility report;
3. has no compiler, CLI, LSP, bootstrap, generated-toolchain, fixture, or
   external-client dependency in the module;
4. passes coordinated exact-revision compatibility with the toolchain,
   supported editors, reference applications, and advertised starters;
5. remains pre-1.0 until preview contracts and compatibility policy are
   intentionally frozen.

The Go module tag and source commit are the library identity. Checksums,
source/SBOM provenance, signing, and release publication are coordinated by the
organization development/release workflow using the exact committed tree.
Private signing material never enters this repository.

## Toolchain releases

CLI archives, cross-platform binaries, deterministic source archives, SPDX
documents, checksums, signatures, and bootstrap recovery are constructed and
verified in the toolchain repository. Consumers install or authorize that
module independently:

```text
go get -tool github.com/spice-framework/toolchain/cmd/spice@<version>
go get -tool github.com/spice-framework/toolchain/cmd/spice-annotation-core@<version>
go tool github.com/spice-framework/toolchain/cmd/spice version
```

Application and starter modules choose compatible core/toolchain revisions
through ordinary Go module requirements and tool directives. No Spice-specific
artifact resolver or hidden download step is introduced.
