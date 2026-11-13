---
description: >-
  Requires diagrams in repository markdown to be ASCII art inside text blocks, so they render the same on every host and
  stay readable in a diff.
when_to_use: >-
  Use when adding or changing a diagram in a markdown file, including architecture and flow diagrams.
---

# Diagrams

Draw every diagram as ASCII art inside a fenced block tagged `text`. A `text` block shows as plain monospace text on
every host, needs no renderer, and shows its change line by line in a diff.

## Rules

- Use a `text` fence. Do not use a bare fence or a renderer-specific language.
- Keep each line within 120 columns, the same limit as prose.
- Put a one-sentence caption above the diagram.
- Put every relationship the diagram shows in prose below it. A reader using a screen reader or a text search cannot see
  the arrows.
- Update the diagram in the same change as the structure it depicts. A stale diagram misleads more than no diagram.
- Do not draw what has not been built. Mark a planned element as planned in its label.

## Example

The caption sits above the block, and the prose below names each arrow.

```text
  +-----------+         +-------------+        +-------------+
  |  Harness  | ------> | super-agent | -----> |  Connector  |
  +-----------+   CLI   |     -cli    |        +-------------+
                        +-------------+
```

The harness calls the command-line tool. The tool calls one connector for the source it names.
