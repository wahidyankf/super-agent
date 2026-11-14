# super-agent

> **Status: alpha.** Expect breaking changes: commands, flags, file formats, connector specs, JSON output, and APIs can
> change in any release, including minor and patch releases, until a stable 1.0. Do not depend on it in production.

super-agent connects coding harnesses and people to data sources and channels. It is a set of source connectors for
catalogs, document systems, and messaging, exposed through a command-line tool (`retriever-cli`) and a Model Context
Protocol (MCP) server. Every connector keeps keys and connections safe, and each source's API token decides what a
change may do.

The apps share one theme: the human summons one creature per app, and the retriever (`retriever-cli`) fetches data from
sources and brings it back.

## What the name means

The human is the super agent. You juggle many resources and channels, at work and outside work, and you use many coding
agents to help. super-agent amplifies that person and their harnesses: it is an orchestrator of many agents and of
information from many data sources.

super-agent is not a single catch-all agent. It does not decide for you and it does not replace your agents. It gives
every harness and every person one catalog of sources and one credential model, so the same connectors serve many agents.
Orchestrating the agents themselves is a later direction, described in the
[software-factory note](./docs/explanation/software-factory-note.md), and it is out of scope now.

## Documentation

Start at the [documentation index](./docs/README.md). Reference material for the repository layout, environment
variable names, and quality gates is in [docs/reference](./docs/reference/README.md). The
[stability and versioning](./docs/explanation/stability-and-versioning.md) page explains what alpha means for you.

## Go modules

The repository holds seven Go modules. The app `retriever-cli` lives in `apps/retriever-cli`. Six libraries live under
`libs/`: `retriever-contract`, `retriever-runtime`, `retriever-testkit`, `retriever-connector-outline`,
`retriever-connector-kiwi`, and `retriever-connector-jira`. Each module has its own `go.mod`, and each library has its
own tag format, `libs/<name>/vX.Y.Z`. Read [library layering](./docs/explanation/library-layering.md) for what each
module may import, and [Go modules and tags](./docs/reference/go-modules-and-tags.md) for the paths and versions.

## Status

The latest release is `v0.2.0`, an alpha pre-release. It follows `v0.1.0`. The archives and checksum lists are on the
GitHub Releases page. Each release names its breaking changes in [CHANGELOG.md](./CHANGELOG.md).

## Repository

- `worktrees/` holds local task checkouts for pull requests. It is not committed.
- `local-tmp/` holds local working state. It is not committed.
- `.env.example` names no variables, because each source in a catalog names its own. Values stay empty there.

Real catalogs, connector specs, and credentials are never committed to this repository. Use `.env.local`
for local values; that file is ignored by Git.

## Contributing

This repository is under construction and does not yet accept external contributions. Contributor rules are in
[AGENTS.md](./AGENTS.md).

## License

MIT. See [LICENSE](./LICENSE).
