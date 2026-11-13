---
description: >-
  Fixes the closed exit-status vocabulary a conforming tool may return, which tools return the bounded-wait, refusal,
  and supervisor statuses, and the rule that an internal crash and a death by signal are never reported as results.
when_to_use: >-
  Use when choosing or changing any exit status, when adding a failure path, or when deciding what a caller may branch
  on.
---

# The Exit Status Contract

## The Vocabulary

These are the only statuses a conforming tool returns. The set is closed: a new failure mode gets a new `error.code` in
the body, never a new number.

| Status  | Means                                                                                |
| ------- | ------------------------------------------------------------------------------------ |
| `0`     | The work completed and the result is affirmative                                     |
| `1`     | The work completed and the result is negative: no match, nothing to do               |
| `2`     | The work did not complete: bad usage, bad configuration, or internal failure         |
| `124`   | A bounded wait elapsed before the work finished                                      |
| `125`   | The environment refused; the work never started                                      |
| `126`   | A child was found but could not be executed                                          |
| `127`   | A child was not found                                                                |
| `128+N` | The process was terminated by signal `N`; `130` is an interrupt, `141` a closed pipe |

## Why `1` Is a Result and Not an Error

The most valuable distinction here is between `1` and `2`, and it is the one most often lost.

`1` means the tool ran correctly and the answer was no. A search that matched nothing, a check that found nothing to
fix, and a query that returned an empty set all succeeded. `2` means the answer is unknown, because the tool could not
do the work.

Collapsing them hides broken runs. A caller that treats every non-zero status alike cannot tell an empty result from a
crash, so it carries on as if nothing failed.

## Which Tools Return Which Statuses

`124` and `125` belong to any tool that has a bounded wait, or that checks its environment before work starts.

- `124` is a timeout: a bounded wait elapsed. It is the retryable status, because contention is the usual cause.
- `125` is a refusal before work starts: an unsupported platform, a malformed pin, or an unmet precondition. Retrying
  changes nothing.

`126` and `127` belong to a program that starts another program. They tell a child that exists but cannot run apart from
one that is absent, which is the difference between a permission or format problem and a missing dependency. A tool that
starts no child process returns neither, and a caller may read their absence as meaning exactly that.

A supervisor passes a child's own status through unchanged when the child ran, except for a check child (below). It
substitutes its own status only when the child never produced one.

## An Internal Crash Is a Failure to Complete

An unhandled panic, an uncaught exception, or an abort exits `2`, and a conforming tool installs a top-level handler
that guarantees it. No runtime does this unaided. The common defaults are `1`, the status reserved for a result, and
`101`, which means nothing to any caller.

The stack trace is not part of the interface. It appears only behind a documented flag or variable, never by default.

## A Death by Signal Is Never a Result

A process terminated by signal `N` reports `128+N`. Nothing in the tool may translate that into a status from the result
range.

Reporting `1` for an interrupt or a closed pipe is the substitution a caller is least likely to check. A script that
branches on `1` to mean "no results" reads a truncated run as an empty one. Two cases occur in practice, `130` for an
interrupt and `141` for a closed pipe, and neither is negotiable.

One exception is narrow. A check child's non-zero exit or death by signal is the finding `1`, and a child that cannot
start is `126` or `127`. A mutation child's death by signal is a refusal, `2`, not a finding. The exception covers check
children only. A signal that ends the tool itself is still `128+N`.

## Publication

Every status a tool can return is listed in its own `--help` output with the meaning it carries. A vocabulary published
only in a specification is one every caller has to guess at.
