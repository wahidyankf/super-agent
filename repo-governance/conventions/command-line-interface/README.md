---
description: >-
  Indexes the ordered modules holding the command-line interface contract: exit statuses, streams, input, structured
  output, arguments, diagnostics, terminal behaviour, help, and the tier record for this repository.
when_to_use: >-
  Use to locate the module covering exit statuses, streams, standard input, machine-readable output, arguments,
  diagnostics, terminal and environment behaviour, help and discovery, or this repository's tier choices.
---

# Command-Line Interface Modules

Read in order. Together these hold the contract that the [Command-Line Interface](../command-line-interface.md)
entrypoint indexes.

## Directory Map

- [001 The Exit Status Contract](001-exit-status-contract.md): the closed status vocabulary and who may return which
  status.
- [002 Stream Discipline](002-stream-discipline.md): what goes to standard output, what goes to standard error, and how
  a closed pipe behaves.
- [003 Standard Input](003-standard-input.md): when a tool may read standard input and when it must not.
- [004 Machine-Readable Output](004-machine-readable-output.md): the explicit output flag, versioned documents, and the
  error body.
- [005 Arguments and Flags](005-arguments-and-flags.md): reserved flags, argument syntax, and option order.
- [006 Errors and Diagnostics](006-errors-and-diagnostics.md): the shape of a message and what it may never contain.
- [007 Terminal and Environment](007-terminal-and-environment.md): colour, escapes, and environment variables.
- [008 Help and Discovery](008-help-and-discovery.md): help spellings and what an invocation with no arguments does.
- [009 Tiers Here](009-tiers-here.md): the tier of `super-agent-cli` and the owner choices that fix its open points.
