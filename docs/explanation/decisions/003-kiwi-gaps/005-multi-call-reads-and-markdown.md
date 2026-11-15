---
title: "E13 and E14: multi-call reads and Markdown"
description: Ordered read steps with all-or-nothing assembly, and a declarative Markdown layout over merged fields.
category: explanation
tags:
  - explanation
  - security
  - decision
  - planned
---

# E13 and E14: multi-call reads and Markdown

This page belongs to [ADR 003](../003-kiwi-gaps.md). E13 and E14 are built by pull request 30. They cover K4:
`documents.get` reads a plan in three calls and a case in four, and assembles Markdown from many fields. E8 allows
one second read, and a converter acts on one field. The name-to-ID lookups of the writes keep riding the existing
`resolve`; the ID-to-name direction for reads rides the E13 pick below, so K4 needs no new lookup feature.

## E13: ordered steps

```yaml
- id: kiwi.documents.get
  assemble:
    steps:
      - { operation: kiwi.case.row, as: case }
      - { operation: kiwi.lookups.categories, as: categories }
    pick:
      - { from: categories, match: { id: case.category }, take: name, as: category }
```

- **Rule.** At most four steps, each a read of the same spec. Each step takes only the operation's own declared
  arguments, and its result merges under `as`. A `pick` takes one row from a step list by matching a canonical-number
  field and keeps one named field. Any step failure fails the whole read; no partial record is returned. All steps
  share the 30 second limit, the pacing of E17, and the safeguards of the engine.
- **L15 (assembly). A step loops, doubles load, or returns a partial record.** Control: the step cap, the reads-only
  rule, the shared deadline, and the all-or-nothing rule above. Proof: `TestAssembleStepCapIsFour`,
  `TestAssembleFailsWhenAnyStepFails`, `TestAssembleStepMustBeRead`, and `TestAssembleSharesTimeLimit`.
- **Cannot do.** A fifth step, a step that is a write, step params from another step's reply text, or a partial
  record on failure.

## E14: Markdown layout

```yaml
markdown:
  title: summary
  meta: [status, priority, category]
  body: text
  sections: [setup, action, effect, breakdown]
```

- **Rule.** The layout names fields only. The title is one collapsed line, the metadata is one `- Key: value` line
  per set field, and each section is a fixed heading with its field body, as `markdown.go` renders today. A list
  shape renders `- <id>: <one-line>` lines from two named fields. Values are inserted as text; the engine escapes
  nothing and renders no markup itself, exactly as the Go code does today.
- **L15 (Markdown). Markup in a value is trusted.** Control: headings and shapes are spec constants, values stay
  text, and cleaning with `clean-text` runs first on each field. Proof: `TestMarkdownPlantStaysText` (plants a
  heading and a link in a value and finds them as literal text) and `TestMarkdownSkipsEmptySections`.
- **Cannot do.** A heading from a value, a free shape, a second document per read, or rendering without cleaning.
