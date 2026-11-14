# retriever-runtime

The use cases and the loaders that the app runs. The runtime runs the source use cases, loads the sources catalog and
the env file, decides where those files are read from, writes the export files, and builds one gateway for each source.
It imports the contract only.

- Module path: `github.com/wahidyankf/super-agent/libs/retriever-runtime`.
- Nx project `retriever-runtime`, with the tags `type:runtime`, `lang:go`, and `domain:retriever`.
- `go.mod` requires the contract at `v0.4.0`, and `go.yaml.in/yaml/v3`, which the catalog loader parses with.

## Packages

- [sources/](./sources/README.md) holds the use cases. Each one resolves a source and calls its gateway through a port.
  Its production code imports no package that reaches outside the process.
- [registry/](./registry/README.md) builds one gateway for each source ID, on first use, from the builders the app
  supplies.
- [catalog/](./catalog/README.md) loads and validates the sources catalog file.
- [envfile/](./envfile/README.md) reads source keys and account e-mails from the process environment and a local env
  file.
- [configpaths/](./configpaths/README.md) decides where the catalog and the env file are read from. It reads no file.
- [markdownfiles/](./markdownfiles/README.md) is the default export sink. It writes one Markdown file for each document.
- [scripts/](./scripts/README.md) holds the coverage exclusions that this module's gate reads.

The packages take the file system, the environment, and the clock as parameters or fields. The app's composition root
passes the real ones, and a unit test passes fakes. See [test gates](../../docs/reference/retriever-cli-test-gates.md).
