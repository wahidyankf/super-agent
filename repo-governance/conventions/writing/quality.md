---
description: >-
  Sets the content quality standard for every markdown file in the repository: clear active English, one H1 per file,
  descriptive headings, accessible links and images, and no time estimates in plans.
when_to_use: >-
  Use before writing or reviewing any markdown content in this repository.
---

# Content Quality

Write clear, simple English that a reader can act on the first time. Plain words beat formal ones. Short sentences beat
long ones. Say what a thing does before saying how it is built.

## Voice

- Use the active voice: "The gate refuses the file", not "The file is refused by the gate".
- Address the reader directly when giving instructions. Write "Run the gate", not "One should run the gate".
- Keep the tone plain and professional. Avoid hype and filler such as "robust", "seamless", and "powerful".

## Clarity and Length

- State the purpose in the first paragraph.
- Use one idea per paragraph and one action per list item.
- Keep a rule in one place and link to it from elsewhere.
- Prefer a table to a long list of parallel facts.
- Define a project term at first use, or link to the [glossary](../../glossary.md).

## Headings

- Every markdown file has exactly one H1, the document title.
- Do not skip heading levels. Each H2 sits under the H1, each H3 under an H2, and so on. Nesting shows structure for
  screen readers and document outlines.
- Headings are descriptive. "Publish a Release" says more than "Overview" when the section does one job.
- Use headings for structure only, never for emphasis.

## Links and Images

- Link text names the destination. Write "the [exit status
  contract](../command-line-interface/001-exit-status-contract.md)", not "click here".
- Every image has alt text that says what the image shows. Prefer text to images.
- Use relative links inside the repository. The `md-internal-link` gate checks that each one resolves.

## Formatting

- Put code in fenced blocks with a language tag, or `text` when no language fits.
- Use bold for a key term where it is defined, and inline code for names, paths, flags, and values.
- Use a blank line before and after every heading, list, table, and fenced block.

## Time Estimates

Documentation may give a time estimate when it is labelled as an estimate and states its assumption. Plans do not carry
time estimates. A plan sequences work by dependency.

## Checklist

Before committing a markdown file, confirm that it has one H1, no skipped heading levels, and no broken link. Confirm
that it is within its word limit and that it holds no private value.
