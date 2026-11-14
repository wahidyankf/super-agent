# internal

The private packages of the app module. Go does not let another module import them.

- [adapters/](./adapters/README.md) holds the inbound adapters of the app: the command-line interface, the MCP server,
  and the GraphQL service.

The domain types, the ports, the use cases, the source connectors, and the runtime loaders are not here. They are
libraries under [libs/](../../../libs/README.md).
