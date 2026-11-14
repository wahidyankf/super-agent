# Shared libraries

`libs/` holds the Go libraries that the `retriever-cli` app builds on, and that the connectors and the testkit build on
too. Each library is its own Go module, with its own `go.mod`, its own Nx project, and its own tag.

## Libraries

- [retriever-contract/](./retriever-contract/) holds the plain types, the ports, the operation catalog, and the
  contract version. It imports no other library.
- [retriever-runtime/](./retriever-runtime/) holds the use cases, the catalog and env-file loaders, the config
  paths, the export sink, and the gateway registry. It imports the contract.
- [retriever-testkit/](./retriever-testkit/) holds the stub servers, the in-memory file tree, the shared source
  contract suite, and the synthetic replies. Tests import it, and production code does not.
- [retriever-connector-outline/](./retriever-connector-outline/) is the Outline wiki source.
- [retriever-connector-kiwi/](./retriever-connector-kiwi/) is the Kiwi test-management source.
- [retriever-connector-jira/](./retriever-connector-jira/) is the Jira Cloud source.
- [retriever-spec/](./retriever-spec/) holds the model of a declarative connector spec, its JSON Schema, the validation
  that refuses a bad spec, and the review of a valid one. It imports the contract.

Each library folder is one level deep, directly under `libs/`. The root `package.json` lists `libs/*` as an npm
workspace pattern. Go does not read that pattern, so each module resolves its siblings through its own `replace`
directives.

## Before you change a library

- [Library layering](../docs/explanation/library-layering.md) states which module may import which, and why.
- [Go modules and tags](../docs/reference/go-modules-and-tags.md) lists the module paths, the Nx tags, the versions, and
  the tag format.
- [Add a connector](../docs/how-to/add-a-connector.md) lists the steps for a new source.
