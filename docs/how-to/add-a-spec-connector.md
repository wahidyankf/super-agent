---
title: Add a spec connector
description: Steps for a user to add a source kind by writing one declarative connector spec, without a Go change.
category: how-to
tags:
  - how-to
  - connector
  - spec
  - retriever
---

# Add a spec connector

A spec connector is a YAML file that describes one service: its endpoint, its key variable, and its read and write
operations. The `retriever` tool runs every spec through one generic engine, so a new service needs no Go code and no
release. For a connector written in Go, see [add a connector](./add-a-connector.md).

The design is in [ADR 001](../explanation/decisions/001-user-defined-connectors-and-profiles.md).

## 1. Put the spec in the connectors folder

A spec file ends in `.yml` or `.yaml`, and it is at most 256 KiB. The tool reads the folder named by `--connectors-dir`,
or the default folder `retriever/connectors` in your user configuration folder. A file that is a link, or a folder that
is not a file, is refused.

A starting point is the bundled example, [example-wiki.yml](../../apps/retriever-cli/examples/connectors/example-wiki.yml).
Copy it into the connectors folder. Its endpoint is `example.test`, and its key comes from `EXAMPLE_WIKI_API_KEY`.

## 2. Name the spec's kind in a source

A source in the sources catalog names the kind that the spec declares, and the key variable that the spec names:

```yaml
version: 1
sources:
  - id: wiki
    kind: example-wiki
    endpoint: https://wiki.example.test
    credential_env: EXAMPLE_WIKI_API_KEY
```

The endpoint host must end with one of the `host_suffix` values of the spec. The key itself stays in the env file or the
process environment, never in the catalog.

## 3. Pin the spec (optional)

A pin stops a changed spec file from running unnoticed. Add the SHA-256 of the file's bytes to the source:

```yaml
sources:
  - id: wiki
    kind: example-wiki
    spec_sha256: <64 lower-case hex characters>
```

On a mismatch, every command that needs sources fails with exit status `2` and the code
`retriever.connector.spec_pin_mismatch`. No request goes out. A pin is allowed only on a source whose kind is a spec kind.

## 4. Run the commands

Each operation of the spec becomes a command, under a group named by the kind. The example's list operation is
`example-wiki docs list`:

```sh
retriever --connectors-dir ~/connectors example-wiki docs list --source wiki
```

An operation whose spec turns `cli` off has no command. Its MCP tool stays in the list, when the spec turns `mcp` on.

## 5. Check the spec when it fails

A spec that the tool refuses stops start-up for every command that reads sources. The failure names the file, the field
path, and a fixed rule, and it never prints a value of the spec:

```text
connector spec file bad.yml: /kind: collides with a built-in kind or a command word
```

The code is `retriever.connector.spec_invalid`, with exit status `2`. The same status applies to an explicit
`--connectors-dir` that does not exist or cannot be read, with the code `retriever.connector.folder_unavailable`. A
default folder that does not exist is not an error.

## Rules a spec must keep

- The kind is not a built-in kind (`outline`, `kiwi`, `jira`), not a command word, and not the kind of another spec.
- Two operations never share a command path or an MCP tool name.
- A spec never sets `Authorization`, `Cookie`, `Host`, or `Content-Length` headers, and it never names a credential
  value. It names a variable only.
- A write has no confirmation step. The API token's own access limits what the source allows, so give a spec's token
  only the access that the spec needs.
- A field may name a Markdown converter, `markdown-from-html` or `markdown-from-adf`, but the engine does not serve one
  yet. Such an operation fails with `retriever.<kind>.unsupported_operation` and sends no request. Use `plain` until then.

## Related

- [Configuration](../reference/retriever-cli-configuration.md) lists the connectors folder and its flag.
- [Catalog format](../reference/retriever-cli-catalog-format.md) lists the fields of a source, including `spec_sha256`.
- [Exit codes](../reference/retriever-cli-exit-codes.md) lists the statuses and codes of a refused spec.
