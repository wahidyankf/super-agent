---
description: >-
  States that a defect is fixed at its cause, not at the symptom that exposed it, and that a workaround is recorded as a
  workaround until its cause is removed.
when_to_use: >-
  Use when diagnosing a failure, deciding whether to patch a symptom, or reviewing a fix that adds a special case.
---

# Root Cause Orientation

A symptom shows where a problem appeared. The cause is where it starts. A fix applied only at the symptom leaves the
cause in place, and the cause will produce the next symptom.

## Practice

1. **Reproduce first.** Make the failure happen with a test or a command, and record it. A fix that cannot be reproduced
   cannot be proven.
2. **Trace back.** Ask why at each step until you reach a decision, an input, or a missing check that caused the
   failure.
3. **Fix where the cause is.** Change the code, configuration, or rule that produced the wrong result.
4. **Prove the fix.** The test that reproduced the failure now passes, and the cause cannot produce it again without a
   test failing.

## Workarounds

Sometimes the cause lies outside the repository, in an upstream tool or a service. Then a workaround is acceptable, but
it must be labelled as one. The label names the cause, the upstream reference, and the condition for removing the
workaround. An unlabelled workaround becomes permanent.

## Special Cases

A fix that adds a special case for one input is a symptom fix. Before adding it, check whether the general rule is
wrong. Most special cases exist because a rule was never stated clearly.

## Related

- [Minimal Sufficiency](minimal-sufficiency.md) keeps the fix to the cause, with no unrelated changes.
- [Reproducibility](reproducibility.md) makes the reproduction repeatable.
