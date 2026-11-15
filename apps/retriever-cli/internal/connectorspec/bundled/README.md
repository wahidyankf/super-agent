# Bundled specs

The embedded specs of the built-in kinds: `outline`, `kiwi`, and `jira`. Each file of this folder is one spec that
the embed loader reads through `go:embed`, so no file, flag, or catalog field reaches the `ParseBundled` exception.
Each operation of a bundled spec claims its catalog row with `catalog_id`, and the catalog row stays the source of
the CLI name, the MCP tool name, the flags, the descriptions, the input schema, and the hints.

E9 carries no spec yet, so the folder holds only this note. The ports add one spec file per bundled kind, and each
file must claim every row of its kind: the loader refuses a row with no operation and an operation with no row.
