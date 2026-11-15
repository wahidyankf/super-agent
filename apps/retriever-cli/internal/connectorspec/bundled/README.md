# Bundled specs

The embedded specs of the built-in kinds: `outline`, `kiwi`, and `jira`. Each file of this folder is one spec that
the embed loader reads through `go:embed`, so no file, flag, or catalog field reaches the `ParseBundled` exception.
Each operation of a bundled spec claims its catalog row with `catalog_id`, and the catalog row stays the source of
the CLI name, the MCP tool name, the flags, the descriptions, the input schema, and the hints.

`outline.yml` is the port of pull request 16: the 66 catalog reads with a `catalog_id` and an example each, and the
fallback target `documents.export` with both surfaces off. Each file must claim every read row of its kind: the loader
refuses a read row with no operation and an operation with no row. A change row may stay unclaimed until its port,
because the composite builder routes it to the Go gateway meanwhile.
