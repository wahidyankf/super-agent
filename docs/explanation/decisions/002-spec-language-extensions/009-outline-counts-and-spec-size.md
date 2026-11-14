---
title: Outline operation counts and the size of a bundled spec
description: The checked count of Outline reads, the 65 and the 66, and an estimate of the size of one Outline spec.
category: explanation
tags:
  - explanation
  - decision
  - planned
---

# Outline operation counts and the size of a bundled spec

This page belongs to [ADR 002](../002-spec-language-extensions.md). The counts come from reading the code on `main`. The
size is an estimate, and pull request 16 measures it.

## The 65 and the 66, checked in code

The probe counted 66 Outline reads. The golden `outline-read-tools.golden` lists 65 MCP tools. Both are right:

- The contract catalog holds 165 Outline operations. 66 are reads, and 99 are writes, updates, or deletes.
- 65 of the 66 reads have an MCP tool. The one without a tool is `revisions.export`, which only the CLI offers.
- `documents.export` is a read that is not in the catalog. The Go constant `opExport` in `endpoints.go` names it. The
  only caller is the text fallback of `documents.get`, which runs when the reply of `documents.info` holds no text. A
  comment also names the vault export, but no code outside the fallback calls `exportText`.

So the Go code makes 67 distinct read calls, 66 from the catalog and one beside it. The earlier guess, 65 catalog reads
plus the export call, was wrong. The catalog already holds 66.

Two things follow for this ADR:

- The fallback target of E8, `outline.documents.export`, has no catalog row. It carries no `catalog_id` and sets
  `cli: false` and `mcp: false`, so it adds no tool. The golden is unaffected.
- Pull request 16 ports 66 catalog reads and the one fallback target.

## Does one Outline spec fit the caps?

The caps are 200 operations and 256 KiB for one spec file.

- **Operations.** The catalog holds 165, and the fallback target makes 166. That fits, with 34 to spare.
- **File size.** This is an estimate. The bundled `example-wiki.yml` takes about 0.7 KB for each of its four operations,
  with one-line parameters and one or two reply fields. An Outline operation has about 2.4 parameters (about 400 in
  all), about six reply fields, a body template, and at least one example with a request, a reply, and the expected
  fields. That is 1.4 to 2.0 KB per operation, or about 230 to 330 KB for 166 operations. The cap is 256 KiB, which is
  about 262 KB.

So the spec may be too large. Examples are the largest part, and each operation must have at least one.

## What ADR 001 lets us do

ADR 001 fixes the loader at one regular file of at most 256 KiB, one YAML document, and no repeated kind (S10). It does
not allow a second file for one kind. This is an open question for the owner, and this ADR decides none of it:

- **Raise the file cap for embedded specs only.** The embedded bytes are trusted by the compile step (B2e). The cap
  would still guard the user folder. This changes S10 and needs a named test.
- **Allow several files for one bundled kind.** This is a new language feature, so it needs its own ADR and threats.
- **Write the shortest possible example for each operation.** This may be enough, and costs nothing in the design.
- **Raise the 200-operation cap.** It is not needed for Outline alone.

Pull request 16 measures the real size first. If it fits, nothing changes.
