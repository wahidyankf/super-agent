---
description: >-
  Limits every markdown line to 120 columns, tables included, so that diffs and reviews stay readable.
when_to_use: >-
  Use when writing or wrapping markdown prose, list items, headings, tables, or diagrams.
---

# Markdown Line Length

Keep each markdown line within 120 columns. The limit covers prose, list items, headings, and table rows.

## Wrapping

- Wrap prose at a sentence or clause boundary, not inside a link.
- Indent a wrapped list item to line up with its text.
- Keep a table within 120 columns by shortening the cell text, not by removing a column.

## Exemptions

- Fenced code blocks, where a line keeps its shape: generated output and long commands.
- A link destination that cannot break across a line.
- `CHANGELOG.md`, where each release note is one line.

## Enforcement

The markdown linter disables its own line-length rule, so this convention is checked by review and by the formatter. The
formatter settings live in `.prettierrc.json`. Keep the width there in step with this page.
