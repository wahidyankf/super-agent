# Changelog

All notable changes to super-agent are listed here. The format follows
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/).

super-agent is alpha. Breaking changes are allowed in any release, including minor and patch releases. Each breaking
change is listed under its release with a **Breaking** label, so you can pin a version and read what changed before you
upgrade.

## Unreleased

## v0.1.0

Alpha pre-release. Expect breaking changes: commands, flags, file formats, connector specs, JSON output, and APIs can
change in any release, including minor and patch releases, until a stable 1.0. Read the
[migration table](docs/reference/retriever-cli-migration-from-butler.md) before you move from Butler, and the
[scenario-gap page](docs/reference/retriever-cli-scenario-gap.md) for the behaviours that no scenario covers yet. This is
the first release, so it has no earlier version to break.

### Added

- Repository skeleton: Nx workspace configuration, HIPPO and RHINO pins, Husky hooks, public-safety gates, and the
  governance and docs trees.
- `retriever`, the sourcing command-line tool in `apps/retriever-cli`. It lists the sources of a catalog, reads
  documents
  from Outline, Kiwi, and Jira, writes Markdown exports to a folder, and serves a read-only GraphQL API (`serve`) and a
  Model Context Protocol server (`mcp`). It is alpha. See the
  [command reference](docs/reference/retriever-cli-commands.md).
- Documentation for `retriever`: a how-to for a first source, and references for the commands, exit codes, JSON
  schemas, configuration, and catalog format. The
  [migration table](docs/reference/retriever-cli-migration-from-butler.md)
  maps each behaviour of the older Butler tool to its `retriever` counterpart.
  - `serve` and `mcp` now share one signal rule: `SIGINT` exits `130` and `SIGTERM` exits `143`, and a closed standard
    output pipe ends the process by `SIGPIPE` (shell status `141`).
  - The migration table now lists four deliberate departures from Butler: refusals before reads, the missing-catalog
    code, the HTTPS refusal at load, and the unknown outcome of a Kiwi write. `retriever` supports macOS and Linux only.
- A release build, `build:release`, that writes one archive for each of `darwin` and `linux` on `amd64` and `arm64`, and
  a `SHA256SUMS` file, from an annotated tag.
