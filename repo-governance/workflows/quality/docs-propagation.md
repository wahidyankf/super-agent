---
description: >-
  Requires the documentation in docs/ to change in the same pull request as the behaviour it describes, and defines the
  checks that keep the four Diataxis categories accurate.
when_to_use: >-
  Use when a change alters a command, a flag, an exit status, a configuration key, a file format, or any other behaviour
  a reader can observe.
---

# Docs Propagation

Documentation describes what the code does. When behaviour changes, the documentation that describes it changes in the
same pull request. A page that stays behind the code misleads the reader it was written for.

## What Changes With What

| The change                              | Pages that must change                                    |
| --------------------------------------- | --------------------------------------------------------- |
| a command, subcommand, or flag          | reference for the command, and any tutorial that uses it  |
| an exit status or error code            | reference for exit codes and error codes                  |
| a configuration key or environment name | reference for configuration, and the environment template |
| a file or JSON format                   | reference for the format, with its schema version         |
| a workflow or a release step            | the how-to guide for that task                            |
| a design choice or boundary             | the explanation page, and the architecture specification  |

## Steps

1. **Find the affected pages.** Search `docs/` for the name of each changed item.
2. **Update the reference first.** Reference states the exact facts, so change it before any guide that links to it.
3. **Update each guide and tutorial.** Check that every step still works, by running it.
4. **Update the indexes.** A new page is listed in its folder's README with one annotated line.
5. **Check the links.** Every link resolves, and no link points to a page that was moved without updating its
   references.

## Diataxis Discipline

A change does not move a page to another category to avoid editing it. Each page keeps its category, as the [Diataxis
framework](../../conventions/writing/diataxis-framework.md) requires. A tutorial that no longer works is fixed or
removed, never left to fail.

## Generated Facts

Where a reference page can be generated from the source, such as the help text of a command, generate it and check it in
a gate. Where it cannot, write it by hand and review it against the source in the same change.

## Enforcement

The markdown gates check the headings, the links, the naming, and the index of each folder. Reviewers check that the
affected pages changed with the behaviour.
