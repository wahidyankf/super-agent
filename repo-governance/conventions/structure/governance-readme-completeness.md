---
description: >-
  Requires the README.md of every governed directory to link each direct child, and states what the readme-index gate
  checks and what it leaves to review.
when_to_use: >-
  Use when adding a file or folder to a governed tree, or when the readme-index gate fails.
---

# Governance README Completeness

Each governed tree has a root README.md that links every direct child file and every direct subdirectory. A subdirectory
counts as linked only when the link uses its folder path, such as `./development/`.

## Rules

- A new file or folder appears in the README of its parent folder, in the same change.
- Each link carries a one-line annotation that says what the child holds.
- A subdirectory's README links its own children, so every file is reachable from the root by links alone.
- A README index lists the children that exist. It does not list planned files.

## What the Gate Checks

The `md-readme-index` gate reads the trees that `repo-config.yml` declares under `policies.markdown.readme-index`. For
each tree it checks that the root README links every direct child. It does not check that each link carries an
annotation, so review checks the annotations.

## When the Gate Fails

Read the message, which names the missing child. Add the link and its annotation to the parent README, then run the gate
again. Do not add an empty placeholder file to satisfy the gate.
