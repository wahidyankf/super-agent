---
description: >-
  Sets the procedure a swe-developer follows in Apply Findings mode: re-validate each finding against the current code,
  fix it test-first or record why not, and report a disposition for every finding.
when_to_use: >-
  Use when a review, a tester, or a frozen quality-gate ledger has returned findings to apply, or when checking a fix
  report for completeness.
---

# SWE Developer Apply Findings

Apply Findings mode takes findings from a review, a tester, or a frozen quality-gate ledger, and applies them. The
developer does not trust a finding because it was reported. Each one is checked against the code as it is now.

## Apply Findings

1. **Take the list whole.** Read every finding, row, or ledger entry before changing anything. Number them so each
   disposition can name its finding.
2. **Re-validate each finding.** Read the code the finding names. Confirm that the problem still exists, is described
   correctly, and has the severity the report gives. A finding that no longer holds is closed with the evidence that
   shows it.
3. **Apply only what the rule allows.** Fix the findings the caller asks for, at the confidence the caller sets. A
   finding rated `MEDIUM` is not applied by this mode. It is reported with its evidence for the caller to decide.
4. **Fix test-first.** For each applied finding, write a test that fails on the current code, record the red result,
   make the change, and record the green result. A finding that cannot be tested is reported with the reason.
5. **Check the changed projects.** Run the checks that apply to the changed code. A failing check is fixed, never
   suppressed.
6. **Record a disposition for each finding.** Each one is `fixed`, `closed as not valid` with evidence, `not applied`
   with the reason, or `needs a decision` with the question.
7. **Hand off.** Report the disposition table, the red and green results for each fix, the checks run, and the open
   questions.

## Stopping Rule

Apply Findings stops when every finding has a disposition and the fix report is complete. It also stops early when a
finding needs a decision that nobody supplied. In that case it reports the finding and the work done so far.

## What It Does Not Do

It never applies a finding it has not re-validated. It never decides when a check-and-repair loop ends, and it never
commits. A finding about a running interface belongs to the tester role, and a finding that calls for a new design
belongs to the architect role.
