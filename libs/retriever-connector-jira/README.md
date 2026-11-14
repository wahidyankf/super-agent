# retriever-connector-jira

The source adapter for Jira Cloud. It reads projects, issues found by JQL, and the comments of an issue. The operations
come from the operation catalog in the contract. This module makes the calls and converts the replies. The adapter has
no write path.

- Module path: `github.com/wahidyankf/super-agent/libs/retriever-connector-jira`.
- Nx project `retriever-connector-jira`, with the tags `type:connector`, `lang:go`, `domain:retriever`, and
  `source:jira`.
- `go.mod` requires the contract and the testkit at `v0.4.0`. The testkit serves the tests only. The production code
  imports the contract alone.

## Folders

- [jira/](./jira/README.md) holds the package: the source, its client, its reader, and its Atlassian Document Format
  to Markdown conversion.
- [scripts/](./scripts/README.md) holds the coverage exclusions that this module's gate reads.

Read [add a connector](../../docs/how-to/add-a-connector.md) for the steps that a connector follows, and
[library layering](../../docs/explanation/library-layering.md) for the imports it may make.
