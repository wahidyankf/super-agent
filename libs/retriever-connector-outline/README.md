# retriever-connector-outline

The source adapter for one Outline wiki. It reads collections, documents, and comments, and it changes the wiki through
the same client. The operations come from the operation catalog in the contract. This module makes the calls and
decodes the replies.

- Module path: `github.com/wahidyankf/super-agent/libs/retriever-connector-outline`.
- Nx project `retriever-connector-outline`, with the tags `type:connector`, `lang:go`, `domain:retriever`, and
  `source:outline`.
- `go.mod` requires the contract and the testkit at `v0.3.0`. The testkit serves the tests only. The production code
  imports the contract alone.

## Folders

- [outline/](./outline/README.md) holds the package: the source, its client, its endpoints, its reader, its writer, and
  its decoders.
- [scripts/](./scripts/README.md) holds the coverage exclusions that this module's gate reads.

Read [add a connector](../../docs/how-to/add-a-connector.md) for the steps that a connector follows, and
[library layering](../../docs/explanation/library-layering.md) for the imports it may make.
