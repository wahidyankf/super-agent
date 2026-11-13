---
description: >-
  Organises the docs/ tree into four Diataxis categories: tutorials, how-to guides, reference, and explanation.
when_to_use: >-
  Use when deciding where new documentation belongs, or when moving a page into or out of docs/.
---

# Diataxis Framework

The `docs/` tree uses the [Diataxis](https://diataxis.fr/) framework. Diataxis sorts documentation by the reader's need:
learning, doing a task, looking up a fact, or understanding a subject. Each page serves one of these four needs.

| Category     | Reader's need               | The page gives                           | Folder              |
| ------------ | --------------------------- | ---------------------------------------- | ------------------- |
| Tutorial     | "Teach me by doing."        | a first success, step by step            | `docs/tutorials/`   |
| How-to guide | "Help me finish this task." | the steps for one goal                   | `docs/how-to/`      |
| Reference    | "Give me the exact facts."  | commands, flags, codes, and file formats | `docs/reference/`   |
| Explanation  | "Help me understand why."   | design, background, and trade-offs       | `docs/explanation/` |

## Rules

- One page serves one category. When a page mixes two, split it and link the halves.
- A tutorial uses one fixed example and never asks the reader to choose between paths.
- A how-to guide assumes the reader knows the basics. It links to reference for exact values.
- Reference states facts and gives no opinion. Keep reference in step with the source it describes.
- Explanation may compare options and argue for a choice. It links to reference for exact values.
- Each folder under `docs/` has a README.md that indexes its pages, with one annotated line for each page.

## Layout

```text
docs/
  README.md         index of the four categories
  tutorials/        first-success paths
  how-to/           task guides
  reference/        commands, flags, exit codes, environment variables, file formats
  explanation/      design notes, stability, and background
```

The top-level `docs/README.md` links each category folder and says in one line what it holds.
