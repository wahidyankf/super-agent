# Changelog

All notable changes to super-agent are listed here. The format follows
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/).

super-agent is alpha. Breaking changes are allowed in any release, including minor and patch releases. Each breaking
change is listed under its release with a **Breaking** label, so you can pin a version and read what changed before you
upgrade.

## Unreleased

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
  - Open question: `serve` exits `0` on `SIGINT` and `SIGTERM`, but `mcp` exits `130` and `143` on the same signals.
    The owner decides whether the two commands keep different statuses.
