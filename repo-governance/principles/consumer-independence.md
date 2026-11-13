---
description: >-
  States that a tool or artifact shared by many consumers holds none of their answers: it requires what only a consumer
  knows, refuses rather than guessing, and treats its surface as a contract.
when_to_use: >-
  Use when designing a reusable tool, generator, validator, or shared artifact, or when a default taken from one
  consumer is proposed for it.
---

# Consumer Independence

A tool built for many consumers knows nothing about any one of them. What only a consumer can know, the consumer
supplies. What the consumer does not supply, the tool does not guess.

## A Default Is a Guess That Cannot Be Retracted

Every consumer-specific default in a shared tool was correct somewhere, usually in the first repository the tool was
written for. Everywhere else it is a value that nobody stated, nobody can see, and the tool cannot verify.

The failure is quiet. A tool that assumes a directory layout, a command name, or a build system produces a plausible
result against a repository that has none of them. Plausible is worse than an error, because nobody investigates a run
that looked fine.

For a control it is worse again. A guard or validator is worth exactly what its clean result means. A validator that
inferred what to check has a clean result that means only that the inference was not obviously wrong. See [Fail
Closed](fail-closed.md).

## What It Requires

- No consumer's value ships in the tool: no repository, directory, command, host, or organisation, and no special case
  for one ecosystem.
- A value the tool needs and cannot derive from its own contract is a required input. Without it, the tool refuses with
  a stable, documented outcome, unless its published contract defines what the absence means.
- No heuristic recognises what a consumer probably is and quietly adjusts behaviour to match.

## Shapes, Not Values

What this permits is mechanism that a consumer parameterises: an argument, an environment variable, a configuration key,
or an extension namespace. Each is a shape, and the value stays with the repository that knows it.

The tool's own documented behaviour is not a guess. A fixed rule that applies identically to every consumer and is
published as part of the contract, such as a file name it reads or an order it runs things in, is the tool describing
itself. The test is whose answer the value is: the tool's, or one consumer's.

An absence passes the same test only when the published contract says what it means, identically for every consumer. An
absence that the contract leaves undefined is a missing input, and the tool refuses.

## The Surface Is a Contract

Consumers reach the tool only through its inputs and outcomes, so those form the whole agreement: commands, flags,
configuration keys, exit codes, and machine-readable output. The [public contract](../development/public-contract.md)
lists them.

Adding to that surface is compatible. Renaming, removing, or changing the meaning of any part breaks every consumer that
relied on it. Such a change ships only under a version that visibly marks the break.

## Where It Is Already Load-Bearing

- [Plans](../conventions/structure/plans.md): a plan names no repository and depends on none.
- [Quality Gate Contract](../development/workflow/quality-gate-contract.md): a gate declares its own inputs, because a
  runner would otherwise guess them.
- [Multi-Harness Binding](../conventions/structure/multi-harness-binding.md): an absent tier mapping emits nothing
  rather than a guess frozen in place.
- [Related Repositories](../conventions/structure/related-repositories.md): a copy of a shared convention belongs to
  this repository, and nothing reaches back into it.

## Proving It

Independence is demonstrated rather than claimed. A tool that publishes such a contract enforces it in its own test
suite: run the tool against a fixture that supplies none of the required inputs, and assert that it refuses with the
documented outcome instead of producing a result.
