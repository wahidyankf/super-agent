---
title: "E8 and E9: the fallback read and bundled kinds"
description: A declared second read when the first reply is empty, and the exception for embedded specs.
category: explanation
tags:
  - explanation
  - security
  - decision
  - planned
---

# E8 and E9: the fallback read and bundled kinds

This page belongs to [ADR 002](../002-spec-language-extensions.md). E8 is built by pull request 14; E9 is still
planned.

## E8: the fallback read

`documents.get` reads the metadata of a document. Only when the reply holds no text does it read the body with
`documents.export`.

```yaml
- id: outline.documents.get
  access: read
  method: POST
  read_over_post: true
  fallback:
    operation: outline.documents.export
    when:
      empty: markdown # or: status: [404]
    fill: [markdown]
```

- **Rule.** The Go code falls back when the reply of `documents.info` holds no text (`outline.go`, `decode.go`). The
  closed triggers `empty` and `status` are proposals that the owner can still veto at review. The target is a read of
  the same spec. It declares only parameters that the first operation also declares,
  and it receives their values. `when` is one closed trigger: a mapped output field that is empty, or a list of HTTP
  statuses. `fill` names the output fields that take the target's values.
- **Limits.** At most one fallback request for each call, and the target has no fallback of its own. Both requests share
  the 30 second limit and run under every safeguard of the engine: the same host, the same key, no redirect, and the
  same caps. If the fallback fails, the read fails with the error of the fallback.
- **L8. A fallback loops, doubles load, reaches another host, or runs a write.** Control: the limits above, and a target
  that must be a read. This extends S3 and S4. Proof: `TestFallbackRunsOnceWhenFieldIsEmpty`,
  `TestFallbackSkippedWhenFieldIsPresent`, `TestFallbackChainIsRefused`, `TestFallbackTargetMustBeRead`,
  `TestFallbackSharesTheTimeLimit`, and `TestFallbackFollowsNoRedirect`.
- **Cannot do.** Pick the target from reply data, chain a second fallback, retry a failed read, or fall back to a write.

## E9: bundled kinds and the ID mapping

After the ports, `outline`, `kiwi`, and `jira` are bundled specs. Two things block that today. The loader refuses a
built-in kind (S10). And a spec ID such as `outline.documents.get` differs from the catalog ID `documents.get`, which
the CLI and MCP names come from.

**The exception.** The spec module has two entry points. `Parse` serves every file of the user folder, and it refuses a
built-in kind and the field `catalog_id`. `ParseBundled` accepts both, and only the loader of the embedded specs calls
it. The embedded bytes come from `go:embed`, so no file, flag, or catalog field reaches the exception.

**The mapping.** A bundled operation may carry `catalog_id: documents.get`. The catalog row stays the source of the CLI
name, the MCP tool name, the flags, the descriptions, the input schema, and the hints. The spec supplies only the
request and the reply rules. So the MCP golden `outline-read-tools.golden` stays byte-identical. A table in the app was
rejected, because it would be a second place to drift.

- **Limits.** Each `catalog_id` names one existing row of the same built-in kind, once. An operation that no row names,
  such as the fallback target `outline.documents.export`, has no `catalog_id` and sets `cli: false` and `mcp: false`.
  A spec may not claim a row that another spec of the kind already claims.
- **L9. A user spec claims a built-in kind or shadows a bundled spec.** Control: `Parse` for the folder, `ParseBundled`
  for the embed, and an architecture test on who may call it. Proof: `TestUserSpecCannotClaimBuiltinKind` (a byte
  copy of `outline.yml` in the user folder is refused), `TestCatalogIDIsRefusedInUserSpec`,
  `TestUserFolderCannotShadowBundledSpec`, and `TestParseBundledHasOneCaller`.
- **L10. A catalog ID maps to the wrong operation.** Control: a two-way check. Proof:
  `TestBundledSpecCoversEveryCatalogRow` fails on a row with no operation or an operation with no row.
  `TestBundledParamsMatchTheCatalogRow` compares names, types, and required flags. The existing golden test stays
  unedited.
- **Cannot do.** Add a kind to the built-in set, let a user spec bind to a catalog row, or lift a limit.
