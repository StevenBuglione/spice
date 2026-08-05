# GoLand integration

The primary Spice editor integration is independently versioned at
[`spice-framework/goland`](https://github.com/spice-framework/goland).

That repository owns installation, compatibility, valid-Go annotation editing,
zero-width prefix concealment, theme-aware syntax colors, PSI navigation,
completion, documentation, package Run/Debug, Plugin Verifier, and the real
installed-IDE visual and interaction gates on Windows and Linux.

Spice core owns the compiler, LSP, annotation SDK, and generated-code contracts
consumed by the plugin. The plugin never makes naked `@` source valid and never
performs dependency injection or compiler resolution itself.
