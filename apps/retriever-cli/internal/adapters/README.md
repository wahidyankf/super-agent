# internal/adapters

The inbound adapters of the app. Each one parses a request from outside and calls the use cases of the
[runtime](../../../../libs/retriever-runtime/README.md) and the
[contract](../../../../libs/retriever-contract/README.md).

- [in/](./in/README.md) holds the command-line interface and the MCP server.
- [graphql/](./graphql/README.md) is the read-only GraphQL service that `retriever serve` runs.
