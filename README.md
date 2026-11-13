# super-agent

super-agent connects coding harnesses and people to data sources. It provides source connectors for
catalogs and document systems, and exposes them through a command-line tool (`super-agent-cli`) and a
Model Context Protocol (MCP) server. Every connector checks access before it reads or writes data.

## Status

This is a pre-release 0.x project, under construction. Commands, file formats, and APIs will change
without notice until a stable 1.0 release. There is no released binary yet.

## Repository

- `worktrees/` holds local task checkouts for pull requests. It is not committed.
- `local-tmp/` holds local working state. It is not committed.
- `.env.example` lists the environment variable names that connectors read. Values stay empty here.

Real catalogs, connector specs, and credentials are never committed to this repository. Use `.env.local`
for local values; that file is ignored by Git.

## Contributing

This repository is under construction and does not yet accept external contributions.

## License

MIT. See [LICENSE](./LICENSE).
