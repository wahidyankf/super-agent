# retriever-cli

`retriever` is the sourcing command-line tool of super-agent. It lists the sources in a catalog, reads documents from
Outline wikis, Kiwi test management, and Jira Cloud, and writes Markdown exports to a folder. It also serves a read-only
GraphQL API (`serve`) and a Model Context Protocol server (`mcp`). The tool is alpha, and breaking changes are allowed
in any release.

The Go module path is `github.com/wahidyankf/super-agent/apps/retriever-cli`. The binary is `retriever`.

## Prerequisites

- A Go toolchain that can build the module. The `go` and `toolchain` lines of `go.mod` name the version.
- Node.js and npm, when you run the targets through Nx from the repository root.
- A sources catalog and an API key for a service you may read. The repository ships no real catalog and no key.
- macOS or Linux. Windows is not supported. See
  [supported platforms](../../docs/reference/retriever-cli-exit-codes.md#supported-platforms).

## Build and run

```sh
cd apps/retriever-cli
go build -trimpath -o dist/retriever ./cmd/retriever
./dist/retriever --help
```

The first run needs a catalog. Copy `examples/sources.example.yml` to `sources.yml`, or pass `--catalog`. The
[first-source guide](../../docs/how-to/add-a-source-from-the-example-catalog.md) walks through the whole path.

## Nx targets

Run each target from the repository root with `npx nx run retriever-cli:<target>`. The targets are defined in
`project.json`.

- `install`: runs `go mod download`.
- `build`: runs `go build -trimpath -o dist/retriever ./cmd/retriever`.
- `run`: runs the built binary, `./dist/retriever`. It depends on `build`.
- `lint`: runs `go vet` with the integration and e2e tags, and checks that `gofmt -l` reports no file.
- `typecheck`: runs `go build ./...`, and compiles every test package with the integration and e2e tags, without running
  it.
- `format`: runs `gofmt -w .`.
- `format:check`: checks that `gofmt -l` reports no file.
- `test:unit`: refuses a test that calls `.Skip(`, then runs the coverage gate, then the self-test of the release build.
  The gate measures unit-level statement coverage of the whole module, with a 99 percent floor.
- `test:integration`: runs `go test -tags integration ./...`.
- `test:e2e`: runs the end-to-end suite in `tests/behaviour` against the binary that the suite builds.
- `test:conformance`: runs the conformance and architecture tests, with the `integration` tag.
- `bdd:check`: checks that each scenario has its bindings or a justified exemption, and that `behaviour-coverage.json`
  matches the feature files.
- `test:coverage`: runs the coverage gate on its own.
- `test:quick`: runs `lint`, `typecheck`, and `test:unit`, in order.
- `deps:audit`: runs `govulncheck`, which reads the Go vulnerability database and needs the network.
- `build:release`: builds the release archives and `SHA256SUMS` for the annotated tag named by `RELEASE_VERSION`, into
  the empty directory named by `RELEASE_OUT`. The [alpha pre-release how-to](../../docs/how-to/cut-an-alpha-pre-release.md)
  runs it. CI never runs it, and nothing is published by it.

The [test gates](../../docs/reference/retriever-cli-test-gates.md) page describes what each target runs, the coverage
gate rules, the binding rules, and the platforms and HIPPO resource tiers that each target runs under in CI.

The `.golangci.yml` file is present, but no target runs `golangci-lint`, because it is not a tool directive in `go.mod`.
The CI cross-compile check builds four platforms and publishes nothing.

## Layout

- `cmd/` holds the composition root. See [cmd](./cmd/README.md).
- `examples/` holds the example sources catalog, `sources.example.yml`. See [examples](./examples/README.md).
- `internal/` holds the core, the adapters, and their unit tests. See [internal](./internal/README.md).
- `tests/` holds the behaviour tests, the conformance and architecture tests, and the stub servers that the tests use.
- `scripts/coverage-gate.sh` is the coverage gate that `test:unit` and `test:coverage` run.
- `scripts/build-release.sh` builds the release archives for `build:release`, and `scripts/build-release_test.sh` tests it.
- `behaviour-coverage.json` records the scenario counts that the binding check compares with the feature files.
- `.golangci.yml` is the lint configuration, written for a later golangci-lint adoption.
- `.gitignore` ignores `dist/`, `sources.yml`, and the env files.
- `go.mod` and `go.sum` pin the module and its dependencies.
- `project.json` declares the Nx targets above.

## Related

- [Command reference](../../docs/reference/retriever-cli-commands.md)
- [Exit codes](../../docs/reference/retriever-cli-exit-codes.md)
- [JSON schemas](../../docs/reference/retriever-cli-json-schemas.md)
- [Configuration](../../docs/reference/retriever-cli-configuration.md)
- [Catalog format](../../docs/reference/retriever-cli-catalog-format.md)
- [Test gates](../../docs/reference/retriever-cli-test-gates.md)
- [Migrating from Butler](../../docs/reference/retriever-cli-migration-from-butler.md)
- [Scenario gap](../../docs/reference/retriever-cli-scenario-gap.md)
