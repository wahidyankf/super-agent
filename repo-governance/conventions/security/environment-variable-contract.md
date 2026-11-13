---
description: >-
  Defines how environment variables are named, declared, read, and kept out of Git: one committed template with empty
  values, real values outside the repository, and a closed set of names each program may read.
when_to_use: >-
  Use when adding, renaming, or reading an environment variable, or when changing the committed environment template.
---

# Environment Variable Contract

Environment variables carry settings that differ between machines, and secrets that must never be committed. This
contract says where each variable is declared, how it is named, and where its real value lives.

## The Template

- `.env.example` is the only committed environment file. It lists every name the program reads, with an empty value and
  a comment that says what the variable is for.
- Real values live in `.env.local`, which Git ignores, or in a platform secret store.
- Until connectors ship, the template holds only placeholder names that begin with `EXAMPLE_`.
- A name that a program reads but the template does not declare is a defect.

## Naming

- Use upper snake case.
- Prefix every setting the tool owns with `RETRIEVER_`.
- Use the `EXAMPLE_` prefix only for placeholders in documentation, fixtures, and the template.
- Name a credential for its purpose and its source kind, never with a number or a generic word.

## Reading

- Read each variable in one configuration adapter at startup. The rest of the program receives typed values.
- Validate a value where it is read. A missing required variable fails with a coded error that names the variable.
- Never print a value in a diagnostic, a log, or a machine-readable body. Name the variable instead.
- Precedence is a flag, then the environment, then a file, then a default. The [command-line
  interface](../command-line-interface/005-arguments-and-flags.md) module states the rule for flags.

## Remote Binding

`REMOTE_DISPATCH_HOME` names the local binding for remote compute. The variable is read by the practice that governs
dispatch, and its value stays on the machine that holds the binding. The repository never records the value.

## Checks

The `env-validate` gate checks the template against the keys declared in `repo-config.yml`. The `env` staged-file rule
refuses any real environment file at commit. [No Machine-Specific Values](no-machine-specific-values.md) governs values
that differ between machines.
