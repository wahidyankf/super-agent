---
description: >-
  States that when designs differ, the one that most helps the consumer be correct wins: errors that name the cause
  and the fix, names that match across surfaces, and no silent behaviour.
when_to_use: >-
  Use when choosing between two designs for an error, a name, a default, or any surface a consumer must use correctly.
---

# Consumer Correctness

When options differ, choose the one that most helps the consumer be correct. Consumers cannot read maintainer intent;
ambiguity becomes misuse, and misuse becomes incidents.

## What It Requires

- **Descriptive errors.** A failure names its cause and the fix. The cause goes in the message; the fix follows as
  clearly marked advice, per [Errors and
  Diagnostics](../conventions/command-line-interface/006-errors-and-diagnostics.md), so a machine matching on the
  message never mistakes advice for diagnosis.
- **Exact names across surfaces.** A thing keeps one name everywhere it appears: flags, variables, fields, documents,
  and messages. A second spelling is a second identity, and the consumer will use the wrong one.
- **No silent behaviour.** Nothing passes quietly that the consumer should notice. A tolerated misuse teaches the wrong
  lesson and fails later, far from the cause.

## Terseness Loses

A shorter surface that lets misuse pass quietly is the wrong choice. Brevity serves the writer once; correctness
serves every consumer every time.

## Followed and Violated

Followed: a rejected input produces an error naming the element at fault and the accepted form; the same identifier
appears in the flag, the variable, and the document; an unchecked condition refuses rather than passing.

Violated: an error says only that the input is invalid; two surfaces spell one concept differently; a questionable
value is accepted without notice.

## Related

- [Fail Closed](fail-closed.md): a control that cannot decide refuses; silence is never consent.
- [Explicit Over Implicit](explicit-over-implicit.md): behaviour is stated where it applies.
- [Consumer Independence](consumer-independence.md): a shared tool holds none of its consumers' answers.
- [Errors and Diagnostics](../conventions/command-line-interface/006-errors-and-diagnostics.md): the shape a
  diagnostic takes.
