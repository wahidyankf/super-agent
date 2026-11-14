# retriever-cli

`retriever` is the sourcing command-line tool of super-agent. It lists the sources in a catalog, reads documents from
Outline wikis, Kiwi test management, and Jira Cloud, and writes Markdown exports to a folder. It also serves a read-only
GraphQL API (`serve`) and a Model Context Protocol server (`mcp`). The tool is alpha, and breaking changes are allowed
in any release.

The Go module path is `github.com/wahidyankf/super-agent/apps/retriever-cli`. The binary is `retriever`. The module
requires the six libraries under [libs/](../../libs/README.md) at `v0.4.0`, and it replaces each one with its local
folder.

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
- `test:unit`: refuses a test that calls `.Skip(`, then runs the gate's self-test, then the coverage gate of this
  module with its 99 percent floor, then the self-test of the release build.
- `test:integration`: runs `go test -tags integration ./...`.
- `test:e2e`: runs the end-to-end suite in `tests/behaviour` against the binary that the suite builds.
- `test:conformance`: runs the conformance and architecture tests, with the `integration` tag.
- `bdd:check`: checks that each scenario has its bindings or a justified exemption, and that `behaviour-coverage.json`
  matches the feature files.
- `test:coverage`: runs the coverage gate on its own.
- `test:quick`: runs `lint`, `typecheck`, and `test:unit`, in order.
- `deps:audit`: runs `govulncheck`, which reads the Go vulnerability database and needs the network.
- `build:release`: builds the release archives and `SHA256SUMS` for the annotated tag named by `RELEASE_VERSION`, into
  the empty directory named by `RELEASE_OUT`. The
  [alpha pre-release how-to](../../docs/how-to/cut-an-alpha-pre-release.md) runs it. CI never runs it, and nothing is
  published by it.

The [test gates](../../docs/reference/retriever-cli-test-gates.md) page describes what each target runs, the coverage
gate rules, the binding rules, and the platforms and HIPPO resource tiers that each target runs under in CI.

The `.golangci.yml` file is present, but no target runs `golangci-lint`, because it is not a tool directive in `go.mod`.
The CI cross-compile check builds four platforms and publishes nothing.

## Layout

- `cmd/retriever/` holds the composition root: the real wrappers of the process, and the wiring of the connectors. See
  [cmd](./cmd/README.md).
- `internal/adapters/` holds the inbound adapters: the command-line interface, the MCP server, and the GraphQL service.
  See [internal](./internal/README.md).
- `examples/` holds the example sources catalog, `sources.example.yml`. See [examples](./examples/README.md).
- `tests/` holds the behaviour tests, the conformance tests, and the architecture tests. See [tests](./tests/README.md).
- `scripts/` holds the release build, its self-test, and the coverage exclusions of the app. See
  [scripts](./scripts/README.md). The shared coverage gate is `scripts/go-coverage-gate.sh` at the repository root.
- `behaviour-coverage.json` records the scenario counts that the binding check compares with the feature files.
- `.golangci.yml` is the lint configuration, written for a later golangci-lint adoption.
- `.gitignore` ignores `dist/`, `sources.yml`, and the env files.
- `go.mod` and `go.sum` pin the module and its dependencies.
- `project.json` declares the Nx targets above.

The domain types, the ports, the operation catalog, the use cases, the source connectors, and the testkit are not in
this folder. They are libraries under [libs/](../../libs/README.md). See
[library layering](../../docs/explanation/library-layering.md).

## Related

- [Command reference](../../docs/reference/retriever-cli-commands.md)
- [Exit codes](../../docs/reference/retriever-cli-exit-codes.md)
- [JSON schemas](../../docs/reference/retriever-cli-json-schemas.md)
- [Configuration](../../docs/reference/retriever-cli-configuration.md)
- [Catalog format](../../docs/reference/retriever-cli-catalog-format.md)
- [Test gates](../../docs/reference/retriever-cli-test-gates.md)
- [Go modules and tags](../../docs/reference/go-modules-and-tags.md)
- [Migrating from Butler](../../docs/reference/retriever-cli-migration-from-butler.md)
- [Scenario gap](../../docs/reference/retriever-cli-scenario-gap.md)
