---
description: >-
  Sets the naming rules for files and folders: lowercase kebab-case, numbered companion sets, fixed tool names, and no
  names that differ only by case.
when_to_use: >-
  Use when creating a file or folder, or when a path must be predicted from its name.
---

# File Naming

A reader should be able to predict a path before opening it. These rules make that possible.

## Rules

- Use lowercase kebab-case for every file and folder: `exit-status-contract.md`, not `ExitStatusContract.md`.
- Use only lowercase letters, digits, and hyphens in the name, with one dot before the extension.
- Name a file for what it is, not for where it sits.
- Name a folder for the topic it holds, and let it share the name of the page beside it when it holds that page's
  companions: `resource-aware-development.md` and `resource-aware-development/`.
- Prefix an ordered companion set with `001-`, `002-`, and so on, in reading order. Use three digits. Each set has a
  README.md that indexes it.
- Put companions in the folder named after their parent page. Do not create a sibling folder with a longer name, such as
  `-details` or `-modules`.
- Do not rely on case to tell two names apart. Names that differ only by case collide on macOS and Windows.
- Keep the spelling of a name that a tool imposes: `README.md`, `LICENSE`, `CHANGELOG.md`, `AGENTS.md`, `CLAUDE.md`, and
  `SKILL.md`.

## Examples

| Path                                                             | Verdict                  |
| ---------------------------------------------------------------- | ------------------------ |
| `conventions/command-line-interface/001-exit-status-contract.md` | Good                     |
| `conventions/CommandLineInterface.md`                            | Bad: mixed case          |
| `conventions/cli-details/001-exit.md`                            | Bad: sibling folder name |
| `workflows/maintenance/release_notes.md`                         | Bad: underscore          |

## Enforcement

The `md-naming` gate checks kebab-case for markdown file names. Its exemptions are the tool-imposed names listed in
`repo-config.yml`.
