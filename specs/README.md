# Specifications

This tree holds the executable description of what retriever does. Code and tests implement these specifications, and a
change to behaviour starts here. The layout follows the
[specs directory structure](../repo-governance/conventions/structure/specs-directory-structure.md).

## What lives here

```text
specs/
├── README.md              this index
├── behaviours/            Gherkin feature files, one folder per behaviour area
│   ├── cli/               the command-line contract: flags, exit statuses, failure bodies
│   ├── catalog/           the sources catalog and how a command finds it
│   ├── credentials/       where each source's key comes from, and where it may go
│   ├── transport/         HTTPS, redirects, timeouts, retries, and reply limits
│   ├── outline/           Outline reads, writes, critical actions, and reply redaction
│   ├── kiwi/              Kiwi product, plan, and case operations
│   ├── jira/              Jira Cloud reads and their rendering
│   ├── export/            export to plain Markdown files in a chosen folder
│   ├── graphql/           the read-only GraphQL server started by `serve`
│   └── mcp/               the MCP tool server started by `mcp`
└── fixtures/              synthetic data and conformance data that tests and feature files read
    └── cli-conformance/   the published contract checks that run before merge
```

- [behaviours/](behaviours/) holds the feature files, and [fixtures/](fixtures/) holds the synthetic data they read.
- **Feature files** cover one behaviour area each, and each file holds one `Feature:`. Every scenario carries a tag for
  each level it runs at: `@unit`, `@integration`, or `@e2e`.
- **Fixtures** are small and synthetic. Hosts are `example.test` names, keys are `EXAMPLE_*` names, and issue keys are
  `EXAMPLE-n`.
- **Run by** the runners in [apps/retriever-cli/tests](../apps/retriever-cli/tests/README.md). The step vocabulary is
  documented there.

## Status

Every feature is written to the public contract. A scenario that a build does not yet satisfy fails; it is not removed
or marked pending.

## Related

- [Apps](../apps/retriever-cli/README.md) holds the implementation.
- [Documentation](../docs/README.md) explains the project for people who use it.
