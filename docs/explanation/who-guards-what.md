---
title: Who guards what
description: Which checks retriever makes itself, and which checks the API token of a source makes.
category: explanation
tags:
  - explanation
  - cli
  - retriever
  - security
---

# Who guards what

`retriever` does not decide what a source may do. The API token of each source decides that, and the service behind the
token enforces it. `retriever` guards three things itself: the keys, the connection, and what it prints.

## The API token decides actions

A write, a delete, or an admin change runs the change that its command names. `retriever` has no access level, no
ceiling for a kind of source, and no confirmation flag to pass first. If a token may delete a document, the delete goes
through. If the token lacks the right, the service refuses the change, and `retriever` reports that refusal with exit
status `2`.

To limit what the tool can do, limit the token. A token that can only read makes a read-only setup.

## What retriever guards

- **Keys.** A key comes from the process environment or from a local env file that Git ignores. The catalog names the
  variable that holds a key, and it never holds the key. A missing key is reported by its variable name, and no value is
  printed.
- **Connections.** An endpoint must be an `https://` host name. Redirects and timeouts have fixed limits, and a reply
  larger than its cap is refused.
- **The catalog.** Only the sources in the catalog are reached. A command that names an unknown source sends no request.
- **Writes.** A write sends one request. The Kiwi adapter calls only the methods named in its code, and it never retries
  a write whose outcome is unknown, so a change is never sent twice.
- **Output.** MCP tool results are bounded and redacted, and control characters are removed from them. The source list
  never prints an endpoint or a key name.

## Jira is read-only in the code

The Jira adapter has no write path. It reads issues, projects, and search results, and it exports them to Markdown.

## Related

- [The sourcing layer](./sourcing-layer.md) explains why retriever reads sources through one catalog.
- [Commands](../reference/retriever-cli-commands.md) lists the commands, including the MCP server.
- [Sources catalog format](../reference/retriever-cli-catalog-format.md) gives the fields of a source.
