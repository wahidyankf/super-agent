# Behaviours

Gherkin feature files that describe what retriever does at its public boundary: a command, a protocol server, or a
connector contract. A feature never describes an internal function.

## Files

- [cli/](cli/README.md): the command-line contract that every command follows.
- [catalog/](catalog/README.md): the sources catalog, how a command finds it, and how a bad catalog is refused.
- [credentials/](credentials/README.md): where each key comes from, and where it may be sent.
- [transport/](transport/README.md): HTTPS, redirects, timeouts, retries, and reply limits for every source kind.
- [outline/](outline/README.md): Outline reads, writes, critical actions, and reply redaction.
- [kiwi/](kiwi/README.md): Kiwi products, plans, and test cases.
- [jira/](jira/README.md): Jira Cloud reads, with descriptions rendered as escaped Markdown.
- [export/](export/README.md): exports to plain Markdown files in a folder the caller names.
- [graphql/](graphql/README.md): the read-only GraphQL server.
- [mcp/](mcp/README.md): the MCP tool server and the tools it offers.

## Conventions

- A scenario has a stable name, and a rename is a change to the contract.
- Each scenario carries a tag for every level it runs at. Unit scenarios run in process with no network. Integration
  scenarios run against local stub servers. End-to-end scenarios run the built binary, and only a small set of them.
- Values are synthetic. See the [public repository data policy](../../repo-governance/conventions/security/README.md).
