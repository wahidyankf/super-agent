# Applications

`apps/` holds the runnable parts of super-agent. It holds one application so far:

- [retriever-cli/](./retriever-cli/) is the `retriever` command-line tool. Its Go module path is
  `github.com/wahidyankf/super-agent/apps/retriever-cli`. The app holds the composition root, the command-line,
  MCP, and GraphQL adapters, and the app-level tests. Its domain, use cases, and source connectors are libraries under
  [libs/](../libs/README.md).

Each application gets its own README, with its prerequisites, commands, and tests.
