---
title: retriever configuration
description: How the retriever command-line tool finds its sources catalog and env file, and in what order.
category: reference
tags:
  - reference
  - cli
  - retriever
  - configuration
---

# retriever configuration

`retriever` reads two files. The **sources catalog** names the sources and their endpoints. The **env file** holds the
keys of those sources. Neither file is part of the repository, and the repository ignores both by name.

## Precedence

For each file, the first rule that applies wins:

1. The flag: `--catalog <path>` or `--env-file <path>`.
2. The environment variable: `RETRIEVER_CATALOG` or `RETRIEVER_ENV_FILE`, when it is set and not empty.
3. The file of the same name in the working directory: `./sources.yml` or `./.env.local`, when it exists as a regular
   file.
4. The user configuration folder: `$XDG_CONFIG_HOME/retriever`, or `~/.config/retriever` when `XDG_CONFIG_HOME` is unset
   or empty.

A relative path given by a flag or a variable is made absolute against the working directory. The rules read no file
themselves. A file at rule 4 is used even when it does not exist yet, and a missing catalog then fails with exit status
`2`.

```text
  --catalog flag ---> set? --yes--> use it
        |
        no
        v
  RETRIEVER_CATALOG ---> set and not empty? --yes--> use it
        |
        no
        v
  ./sources.yml ---> regular file exists? --yes--> use it
        |
        no
        v
  $XDG_CONFIG_HOME/retriever/sources.yml, or ~/.config/retriever/sources.yml
```

The env file follows the same order with `--env-file`, `RETRIEVER_ENV_FILE`, `./.env.local`, and
`retriever/.env.local` in the user folder.

## Missing files

- **A missing catalog** fails every source command with exit status `2`. The message is
  `sources catalog: not found: set --catalog or RETRIEVER_CATALOG, or create ./sources.yml`. Commands that read no
  source, such as `--version` and `--help`, still run.
- **A missing env file** is not an error. A key that the env file would hold is then reported as missing, with
  `retriever.<area>.key_missing`, when a command needs it.

## What the catalog loader refuses

The catalog must be a regular file. A symbolic link or a folder is refused. It must be at most 64 KiB. It must hold one
YAML document. The loader refuses unknown and repeated fields, and it reports the file's path nowhere. The rules of the
catalog itself are in [the catalog format](./retriever-cli-catalog-format.md).

## Key lookup

A source's key is read from the process environment first, then from the env file. An empty value counts as missing in
both places. The value is never printed. A diagnostic names the variable, such as `EXAMPLE_OUTLINE_API_KEY is not set`.

The env file uses the `KEY=VALUE` format, one pair per line:

- Blank lines and lines that start with `#` are skipped.
- An optional `export` word, followed by a space, is ignored.
- A value in single or double quotes is taken as written between the quotes. There are no escapes and no expansion.
- An unquoted value loses trailing spaces, and any comment that starts with a space and a `#`.
- When a key repeats, the last value wins.
- A malformed line stops the read with an error that gives its line number only, never its text.

## Related

- [Environment variables](./environment-variables.md) lists the variable names.
- [Catalog format](./retriever-cli-catalog-format.md) describes the fields of the sources file.
- [Add a source from the example catalog](../how-to/add-a-source-from-the-example-catalog.md)
  is the first-run walkthrough.
