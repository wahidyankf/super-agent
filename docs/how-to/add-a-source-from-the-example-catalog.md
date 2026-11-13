---
title: Add a source from the example catalog
description: The steps to build retriever, point it at a catalog, give it a key, and read one document.
category: how-to
tags:
  - how-to
  - cli
  - retriever
  - catalog
---

# Add a source from the example catalog

This guide adds one source to `retriever` and reads a document from it. The example catalog names example hosts, so you
replace them with the host of a service you are allowed to read. You need a Go toolchain that can build the module. The
version is in `apps/retriever-cli/go.mod`.

## 1. Build the binary

From the repository root, run:

```sh
cd apps/retriever-cli
go build -trimpath -o dist/retriever ./cmd/retriever
```

The `build` target in `apps/retriever-cli/project.json` runs the same command. Git ignores `dist/`, so the binary stays
out of commits. Stay in `apps/retriever-cli` for the rest of this guide.

## 2. Copy the example catalog

The catalog is a file that the tool reads. It is not part of the code. Copy the example into the folder where you run
the tool:

```sh
cp examples/sources.example.yml sources.yml
```

Git ignores `sources.yml` by name. If you prefer to keep the catalog somewhere else, pass its path with `--catalog`
instead.

Open the copy and replace the host in `endpoint` with the host name of your service. Use the host name only: no path,
no port, and no user information. Keep the `https://` prefix. The other fields are described in
[the catalog format](../reference/retriever-cli-catalog-format.md).

## 3. Give the tool the key

The tool reads each key from a variable, and the catalog names the variable in `credential_env`. The first source in
the example uses `EXAMPLE_OUTLINE_API_KEY`. The command below reads the value without echoing it, and then exports it to
the tool:

```sh
read -r -s EXAMPLE_OUTLINE_API_KEY && export EXAMPLE_OUTLINE_API_KEY
```

You can also add a line `EXAMPLE_OUTLINE_API_KEY=` followed by the key to a file named `.env.local` in the same
folder. Git ignores that file. A key never goes in the catalog, in a tracked file, or in a command you paste into a
chat.

## 4. List the sources

```sh
./dist/retriever sources list
```

With the unchanged example catalog, the output is:

```text
3 sources
  example-jira     jira     read  tickets
  example-kiwi     kiwi     read  tests
  example-outline  outline  read  docs,wiki
```

The list shows each source's kind, its access level, and its tags. It never shows the endpoint or the key name.

## 5. Read a document

Find a document ID in the wiki first. The list command prints one page at a time:

```sh
./dist/retriever outline documents list --source example-outline
```

Then read one document. Its Markdown body prints to standard output:

```sh
./dist/retriever document get <documentId> --source example-outline
```

Add `-o json` before or after the command to get the JSON document described in
[the JSON schemas](../reference/retriever-cli-json-schemas.md).

## 6. Export a document to a folder

Pick a folder outside the repository, or one that Git ignores. The export writes one Markdown file with no front matter:

```sh
./dist/retriever outline export <documentId> --out ./exports --source example-outline
```

The text output ends with `Wrote <path>`, and the JSON output prints the same path as `location`.

## 7. Read a failure

A failed command exits `2`, writes its message to standard error, and prints nothing to standard output. These are the
failures that a first run usually meets:

- `sources catalog: not found: ...` means the tool found no catalog. Create `sources.yml`, or pass `--catalog`.
- `EXAMPLE_OUTLINE_API_KEY is not set` means the key is in neither the environment nor the env file. Set it as in
  step 3.
- An error code that ends in `ambiguous_source` means that more than one source of the kind exists. Add `--source <id>`.
- An error code that ends in `access_level_too_low` means the source's `access` is below what the command needs. Use a
  read command, or raise `access` only if the service grants that level.

With `-o json`, read `error.code` from the failure body. The exit codes are listed in
[the exit codes](../reference/retriever-cli-exit-codes.md).

## Related

- [The command reference](../reference/retriever-cli-commands.md) lists every command and flag.
- [Configuration](../reference/retriever-cli-configuration.md) explains which catalog and env file the tool picks.
- [The sourcing layer](../explanation/sourcing-layer.md) explains why the tool works this way.
