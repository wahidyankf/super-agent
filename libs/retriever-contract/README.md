# retriever-contract

The contract that the app, the runtime, the testkit, and the connectors build against. It holds the plain types, the
ports that the core depends on, and the operation catalog. It imports no other library of this repository, and its
production code imports none of the packages that reach outside the process, such as `os` and `net/http`.

- Module path: `github.com/wahidyankf/super-agent/libs/retriever-contract`.
- Version: the `ContractVersion` constant in [contract.go](./contract.go), which is `0.3.0`.
- Nx project `retriever-contract`, with the tags `type:contract`, `lang:go`, and `domain:retriever`.

## Folders

- [domain/](./domain/README.md) holds the plain types, the shared error values, and their validators.
- [ports/](./ports/README.md) declares the interfaces that the core depends on, such as `SourceGateway` and
  `Credentials`.
- [operations/](./operations/README.md) holds the operation catalog, with one row for each operation.
- [scripts/](./scripts/README.md) holds the coverage exclusions that this module's gate reads.

## Files

- [contract.go](./contract.go) holds the package doc and `ContractVersion`. The libraries release in lockstep, so the
  value matches their tags, `libs/<name>/v0.3.0`.
- `go.mod` names the module. It requires no other module of this repository.
- `project.json` declares the Nx project and its targets.

Read [library layering](../../docs/explanation/library-layering.md) for the rules that this module must keep, and
[Go modules and tags](../../docs/reference/go-modules-and-tags.md) for its version and tag.
