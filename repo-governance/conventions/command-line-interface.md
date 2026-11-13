---
description: >-
  Fixes the interface a command-line tool presents to its callers: one closed exit vocabulary, stream discipline,
  machine-readable output, argument syntax, and terminal behaviour, with precision carried in the body rather than in
  the exit number.
when_to_use: >-
  Use when building, reviewing, or changing any command-line entry point, and before adding a new exit status, output
  mode, or environment variable to one.
---

# Command-Line Interface

A command-line tool has two audiences that cannot both be served by the same channel. A person reads what it prints; a
script or a harness branches on what it returns. The interface is what makes the second audience possible, and it is a
contract: once a caller branches on a status, changing what that status means breaks the caller silently, usually in
production rather than in review.

## The Two Layers

The contract has exactly two layers, and the split is the whole design.

| Layer                   | Carries                               | Size                      |
| ----------------------- | ------------------------------------- | ------------------------- |
| The process exit status | What class of thing happened          | Small, closed, universal  |
| A machine-readable body | Exactly what happened, and about what | Open, namespaced, growing |

An integer is a poor place to put a taxonomy. It has no room for context, no way to name the thing at fault, and no way
to grow without colliding with an existing meaning. So the number answers one question, may the caller trust the result,
and everything else moves into a body the caller can parse.

A wide numeric vocabulary, where each number names one failure, has been tried. A legacy Unix header reserved a block of
such codes, and its upstream now records that "the choice of an appropriate exit value is often ambiguous". That
ambiguity is what happens when one integer is asked to do two jobs.

## Tiers

Not every tool owes every rule, so the convention binds at two tiers. An adopter records which of its own tools sits at
which tier.

- **The floor tier** binds anything invoked from a shell, including wrappers, hooks, and shipped scripts. It owes the
  exit vocabulary, stream discipline, and the closed-pipe and signal rules.
- **The full bar** binds a tool that a person or a script calls directly as a product surface. It owes the floor plus
  machine-readable output, argument syntax, diagnostics, and terminal behaviour.

A tool that starts other programs additionally owes the supervisor statuses. A tool that starts none never returns them.

## Modules

1. [The Exit Status Contract](command-line-interface/001-exit-status-contract.md)
2. [Stream Discipline](command-line-interface/002-stream-discipline.md)
3. [Standard Input](command-line-interface/003-standard-input.md)
4. [Machine-Readable Output](command-line-interface/004-machine-readable-output.md)
5. [Arguments and Flags](command-line-interface/005-arguments-and-flags.md)
6. [Errors and Diagnostics](command-line-interface/006-errors-and-diagnostics.md)
7. [Terminal and Environment](command-line-interface/007-terminal-and-environment.md)
8. [Help and Discovery](command-line-interface/008-help-and-discovery.md)
9. [Tiers Here](command-line-interface/009-tiers-here.md), the tier record and owner choices for this repository

## What This Convention Does Not Decide

It does not choose a subcommand tree, a flag vocabulary beyond the reserved names, an output schema, or a configuration
format. Those belong to the tool. The convention fixes only the surface a caller must rely on without reading the tool's
source.

It also does not promise that conformance is free. Every runtime breaks some part of this contract by default, so a
conforming tool overrides its runtime rather than inheriting from it. Each module names the defaults it corrects.

## Tiers Here

| Surface                                   | Tier     | Why                                                              |
| ----------------------------------------- | -------- | ---------------------------------------------------------------- |
| `retriever-cli`, built in `apps/`         | Full bar | Called by a person and a harness; choices recorded in module 009 |
| `./hippo` and `./rhino`                   | Floor    | Shell-invoked wrappers; each starts the pinned tool it installs  |
| `.husky/{commit-msg,pre-commit,pre-push}` | Floor    | Git invokes them and branches on what they return                |
| `scripts/*.sh` and `scripts/*`            | Floor    | Shipped scripts that a gate or a hook calls                      |

The wrappers return `125` when they refuse to start the tool. No surface here starts another program except the
wrappers, which start their pinned tool, and the gate runner, which starts gate children.
