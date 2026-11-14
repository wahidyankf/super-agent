---
title: Sources catalog format
description: The fields, rules, and limits of the sources catalog that the retriever command-line tool reads.
category: reference
tags:
  - reference
  - cli
  - retriever
  - catalog
---

# Sources catalog format

The sources catalog is a YAML file. It names each source the tool may read, the host that receives its key, and the
environment variable that holds the key. It never holds a key. The repository ships a synthetic example at
[`apps/retriever-cli/examples/sources.example.yml`](../../apps/retriever-cli/examples/sources.example.yml). Copy it to
`sources.yml`, or point `--catalog` at it. A real catalog names real hosts, so it stays outside the repository. The
repository ignores `sources.yml` by name.

## Shape

```yaml
version: 1
sources:
  - id: example-outline
    kind: outline
    tags: [wiki, docs]
    endpoint: https://outline.example.test
    credential_env: EXAMPLE_OUTLINE_API_KEY
```

The top level has two keys, and no other key is allowed:

- `version`: required. It must be `1`. No other version is read.
- `sources`: required. It is a list of one to 64 sources.

## Source fields

- `id`: required. A lowercase slug of letters, digits, and single hyphens, starting with a letter. It is unique in the
  catalog.
- `kind`: required. It is `outline`, `kiwi`, or `jira`, or the kind that a loaded connector spec declares. Any other kind
  is refused, and so is a kind that no loaded spec declares.
- `tags`: optional. A list of lowercase slugs, at most 16 distinct tags. Tags label a source. They never change what the
  source allows.
- `endpoint`: required. It is `https://` followed by a host name only: no path, no port, and no user information. A
  trailing slash is allowed.
- `credential_env`: required. It is the name of the variable that holds the key. It must match
  `^[A-Z][A-Z0-9_]*_API_(KEY|TOKEN)$`, and it is at most 64 characters.
- `identity_env`: required for Jira only. It is the name of the variable that holds the account e-mail address. It must
  match `^[A-Z][A-Z0-9_]*_EMAIL$`, and it is 7 to 64 characters.
- `spec_sha256`: optional, and allowed only on a source whose kind a loaded spec declares. It is the SHA-256 of the spec
  file's bytes, as 64 lower-case hex characters. A mismatch fails the run with `retriever.connector.spec_pin_mismatch`,
  and no request goes out.

Each rule states a requirement, and no error message repeats a value. For example, a bad endpoint reports that it must
be an `https` host name, and it does not print the endpoint.

## Rules across sources

- **Unique IDs.** Two sources cannot share an `id`.
- **One credential, one host.** A credential variable serves one endpoint host only. Two sources may not name the same
  variable for different hosts.
- **No access key.** The `access` key was removed. A catalog that still holds it is refused when it loads, and the error
  names the key. Limit what a source may do with its API token instead.
- **Identity by kind.** `jira` requires `identity_env`. `outline` and `kiwi` refuse it.
- **One document.** The file must hold exactly one YAML document. It is at most 64 KiB.
- **No unknown fields.** A misspelled key is an error, so a typo cannot silently drop a source setting.

Any violation stops the load, and the CLI reports the source number and the field, such as
`sources catalog: source 2 endpoint: ...`. The text never includes the file path.

## Example

The example catalog in the repository has three sources, one for each kind:

- `example-outline`: kind `outline`, endpoint `https://outline.example.test`, key variable `EXAMPLE_OUTLINE_API_KEY`,
  and no identity variable.
- `example-kiwi`: kind `kiwi`, endpoint `https://kiwi.example.test`, key variable `EXAMPLE_KIWI_API_KEY`, and no
  identity variable.
- `example-jira`: kind `jira`, endpoint `https://jira.example.test`, key variable `EXAMPLE_JIRA_API_TOKEN`, and identity
  variable `EXAMPLE_JIRA_EMAIL`.

These hosts and names are examples. They do not resolve to a real service.

## Related

- [Configuration](./retriever-cli-configuration.md) gives the rules that choose which catalog file is read.
- [Add a source from the example catalog](../how-to/add-a-source-from-the-example-catalog.md) is the step-by-step guide.
- [Environment variables](./environment-variables.md) lists the credential variable names the catalog refers to.
