# internal/adapters

The adapters of the hexagonal layout. Each adapter implements a port from `internal/core/ports`, or it drives the core
from outside: a source service, a file, the terminal, or a protocol.

- [`catalog/`](./catalog/README.md) loads and validates the sources catalog file.
- [`configpaths/`](./configpaths/README.md) decides which catalog and env file the tool reads.
- [`envfile/`](./envfile/README.md) reads source keys and account e-mails from the environment and from the env file.
- [`graphql/`](./graphql/README.md) is the read-only GraphQL service that `retriever serve` runs.
- [`in/`](./in/README.md) holds the inbound adapters: the command line and the MCP server.
- [`jira/`](./jira/README.md) is the Jira Cloud source.
- [`kiwi/`](./kiwi/README.md) is the Kiwi test-management source.
- [`out/`](./out/README.md) holds the outbound sinks, which write exports.
- [`outline/`](./outline/README.md) is the Outline wiki source.
- [`registry/`](./registry/README.md) builds one gateway for each source.
