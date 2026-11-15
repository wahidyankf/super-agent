---
title: "E22: the issue document layout"
description: A declarative Markdown layout over many fields of one reply, for single-document reads.
category: explanation
tags:
  - explanation
  - security
  - decision
  - planned
---

# E22: the issue document layout

This page belongs to [ADR 004](../004-jira-gaps.md). E22 is built by pull request 36. It covers J6:
`issues.get` assembles Markdown from many fields (a title, a metadata list, labels, a due date, a project
line, and the description). A converter acts on one field and cannot do this. E22 is a generic layout with
closed line kinds; it bakes in no service.

```yaml
response:
  document:
    title: summary
    lines:
      - { label: Key, verbatim: key }
      - { label: Project, pair: { name: project.name, key: key } }
      - { label: Type, field: issuetype }
      - { label: Resolution, field: resolution, skip_empty: true }
      - { label: Assignee, field: assignee, default: Unassigned }
      - { label: Labels, field: labels, join: ", ", skip_empty: true }
      - { label: Parent, field: parent, valid: issue_key, skip_empty: true }
    body: { field: description, convert: markdown-from-adf, heading: Description, empty: "(no description)" }
```

- **Rule.** The title is one collapsed line rendered as `# <title>`. Each line renders `- <label>: <value>`,
  with the label a constant of 1 to 32 characters. A `field` line escapes its value with the moved
  `escapeText` algorithm; a `verbatim` line writes a shape-checked key or date as is. A `pair` line renders
  `name (key)` escaped, or the bare key when the name is empty. `join` renders a string list with the fixed
  separator `", "`. `default` fills an empty value with its closed constant; `skip_empty` omits the line when
  the value is still empty; otherwise the line is emitted even when empty, as the Go code emits Reporter. The
  body renders `## <heading>` plus the converted description, or the `empty` constant when it is blank. Every
  value is cleaned before mapping, and planted markup stays literal text.
- **Why not E14.** E14 renders without escaping, has no defaults, no joins, and no pairs. Jira needs all
  three, and its scenarios pin the exact lines. Two layouts with different escaping is honest: each matches
  the Go code it retires.
- **L25. Markup in a value renders as structure, or a line leaks when empty.** Control: fixed line kinds,
  the moved escape algorithm, and the skip rules above. Proof: `TestDocLayoutPlantStaysText` (plants a
  heading and a link in the summary and the labels and finds both literal), `TestDocLayoutSkipsEmptyLines`
  (an empty resolution, parent, and due leave no line), and `TestDocLayoutDefaultsExact` (`Unassigned` and
  `(no description)` byte-identical).
- **Cannot do.** A label from a value, a heading from a value, another separator, a second document per
  read, or rendering without cleaning.
