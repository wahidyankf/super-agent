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
- **File size.** This was an estimate. The bundled `example-wiki.yml` takes about 0.7 KB for each of its four operations,
  with one-line parameters and one or two reply fields. An Outline operation has about 2.4 parameters (about 400 in
  all), about six reply fields, a body template, and at least one example with a request, a reply, and the expected
  fields. That is 1.4 to 2.0 KB per operation, or about 230 to 330 KB for 166 operations. The cap is 256 KiB, which is
  about 262 KB.

Pull request 16 measured the real spec: `outline.yml` holds all 67 operations (66 catalog reads + the export fallback
target) in 115,907 bytes as committed — 44 percent of the cap, about 1.7 KB per operation. It fits with room to spare,
so no cap change, no second file, and no new ADR. Per the standing owner rule, every byte is there because the
operation needs it: parameters carry help text, and each operation keeps one example.

Pull request 17 measured again: `outline.yml` holds 159 operations (66 catalog reads, 92 claimed writes, and the
export fallback target) in 228,340 bytes as committed — 87 percent of the cap, about 1.4 KB per operation. The 7
unclaimed writes have no spec entry, and each operation keeps one minimal example. It fits, so no cap change, no
second file, and no new ADR. The 200-operation cap holds 159 operations with 41 to spare.

The owner then decided to extend the language in the same pull request so the remaining 11 operations move, and
measured a third time: `outline.yml` holds 166 operations (66 catalog reads, all 99 claimed writes, and the export
fallback target) in 244,480 bytes — 93 percent of the cap, about 1.5 KB per operation. Every catalog row is claimed
and each operation keeps one minimal example, so the unclaimed set is empty. It fits, so still no cap change, no
second file, and no new ADR. The 200-operation cap holds 166 operations with 34 to spare.

## What ADR 001 lets us do

ADR 001 fixes the loader at one regular file of at most 256 KiB, one YAML document, and no repeated kind (S10). It does
not allow a second file for one kind. This is an open question for the owner, and this ADR decides none of it:

- **Raise the file cap for embedded specs only.** The embedded bytes are trusted by the compile step (B2e). The cap
  would still guard the user folder. This changes S10 and needs a named test.
- **Allow several files for one bundled kind.** This is a new language feature, so it needs its own ADR and threats.
- **Write the shortest possible example for each operation.** This may be enough, and costs nothing in the design.
- **Raise the 200-operation cap.** It is not needed for Outline alone.

Pull request 16 measures the real size first. If it fits, nothing changes.
