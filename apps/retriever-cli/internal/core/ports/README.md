# internal/core/ports

The outbound contracts that the core depends on. An adapter implements each interface.

- `ports.go` declares `FileSystem`, `ExportSink`, and the `Ports` struct that carries them.
- `sources.go` declares the source contracts: `SourceGateway`, `SourceReader`, `SourceWriter`, `SourceAdmin`,
  `Credentials`, `Identities`, `GatewayResolver`, and `CatalogLoader`.
