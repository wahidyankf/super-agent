# retriever-connector-kiwi

The source adapter for one Kiwi TCMS server, a test-management system. The adapter speaks JSON-RPC 2.0 over HTTPS. The
operations come from the operation catalog in the contract. This module makes the calls and decodes the replies.

- Module path: `github.com/wahidyankf/super-agent/libs/retriever-connector-kiwi`.
- Nx project `retriever-connector-kiwi`, with the tags `type:connector`, `lang:go`, `domain:retriever`, and
  `source:kiwi`.
- `go.mod` requires the contract and the testkit at `v0.4.0`. The testkit serves the tests only. The production code
  imports the contract alone.

## Folders

- [kiwi/](./kiwi/README.md) holds the package: the source, its client, its reader, its writer, and its decoders.
- [scripts/](./scripts/README.md) holds the coverage exclusions that this module's gate reads.

Read [add a connector](../../docs/how-to/add-a-connector.md) for the steps that a connector follows, and
[library layering](../../docs/explanation/library-layering.md) for the imports it may make.
