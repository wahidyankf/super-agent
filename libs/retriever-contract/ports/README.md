# ports

The outbound contracts that the core depends on. An adapter implements each interface, and a test replaces it with a
fake.

- `ports.go` declares `FileSystem`, `ExportSink`, and the `Ports` struct that carries them.
- `sources.go` declares the source contracts: `SourceGateway`, `SourceReader`, `SourceWriter`, `Credentials`,
  `Identities`, `GatewayResolver`, and `CatalogLoader`.
