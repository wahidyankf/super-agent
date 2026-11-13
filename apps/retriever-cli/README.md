# retriever-cli

`retriever` is the sourcing command-line tool of super-agent. It lists the sources in a catalog, reads documents from
Outline wikis, Kiwi test management, and Jira Cloud, and writes Markdown exports to a folder. It also serves a
read-only GraphQL API (`serve`) and a Model Context Protocol server (`mcp`). The tool is alpha, and breaking changes
are allowed in any release.

The Go module path is `github.com/wahidyankf/super-agent/apps/retriever-cli`. The binary is `retriever`.

## Prerequisites

- A Go toolchain that can build the module. The version is in `go.mod`.
- Node.js and npm, when you run the targets through Nx from the repository root.
- A sources catalog and an API key for a service you may read. The repository ships no real catalog and no key.

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
- `lint`: runs `go vet ./...`, and checks that `gofmt -l` reports no file.
- `typecheck`: runs `go build ./...`, and compiles every test package without running it.
- `format`: runs `gofmt -w .`.
- `format:check`: runs the formatting check that `lint` also runs.
- `test:unit`: runs `go test -cover ./...`, after a check that no test calls `.Skip(`.
- `test:integration`: runs `go test -tags integration ./...`.
- `test:coverage`: runs unit and integration coverage, and checks that each `internal/core` package reaches 99 percent.
  It fails today, because two core packages are below 99 percent.
- `test:quick`: runs `lint`, `typecheck`, and `test:unit`, in order. The pull-request gate runs this set for affected
  projects.

Some checks do not exist yet. There is no `test:e2e` target. There is no dependency audit target, because no
`govulncheck` version is pinned. The `.golangci.yml` file is present, but no target runs `golangci-lint`, because it is
not a tool directive in `go.mod`. A release that publishes archives for each platform has no build step yet.

## Layout

- `cmd/` holds the composition root. See [cmd](./cmd/README.md).
- `examples/` holds the example sources catalog, `sources.example.yml`. See [examples](./examples/README.md).
- `internal/` holds the core, the adapters, and their unit tests. See [internal](./internal/README.md).
- `tests/` holds the behaviour tests and the stub servers that the tests use.
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
- [Migrating from Butler](../../docs/reference/retriever-cli-migration-from-butler.md)
