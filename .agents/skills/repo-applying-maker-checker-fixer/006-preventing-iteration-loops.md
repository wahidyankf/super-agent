# Maker-Checker-Fixer Pattern (Comprehensive): Part 6 of 7

## Preventing Iteration Loops

Without explicit mechanisms to track accepted decisions, checker-fixer workflows can enter infinite or very long
iteration loops. Four structural safeguards prevent this:

Inside a `<family>-quality-gate` run, safeguards 2 and 4 do not apply: every audit covers the whole frozen scope, and a
disputed row is recorded `needs-decision`, per the
[Quality Gate Contract](../../../repo-governance/development/workflow/quality-gate-contract.md).

### 1. FALSE_POSITIVE Persistence (`.known-false-positives.md`)

**Problem**: Checker re-flags the same accepted FALSE_POSITIVE findings every iteration.

**Solution**: Fixer writes all accepted FALSE_POSITIVE findings to `generated-reports/.known-false-positives.md`.
Checker reads this file at the start of every run and skips matching entries.

**Checker behavior**: Match findings using stable key `[category] | [file] | [brief-description]`. If matched, log as
`[PREVIOUSLY ACCEPTED FALSE_POSITIVE — skipped]`, do NOT count in findings total.

**Fixer behavior**: After every fix run, append each FALSE_POSITIVE to `.known-false-positives.md`:

```bash
cat >> generated-reports/.known-false-positives.md << 'EOF'
## FALSE_POSITIVE: [category] | [file] | [brief-description]

**Accepted**: [YYYY-MM-DD--HH-MM]
**Category**: [finding category]
**File**: [path/to/file.md]
**Finding**: [Brief description]
**Reason**: [Why accepted as false positive]

---
EOF
```

### 2. Scoped Re-validation (Changed Files Only)

**Problem**: Full-repo scan on every iteration re-validates all files even when fixer only changed 3-4 files.

**Solution**: Fixer captures changed files after applying fixes:

```bash
git diff --name-only HEAD
```

Includes list in fix report under `## Changed Files (for Scoped Re-validation)`. Checker in re-validation mode
(multi-part UUID chain like `abc123_def456`) runs validation only on changed files.

### 3. Self-Verification After Bash Edits

**Problem**: `sed -i` exits 0 even when pattern doesn't match. Fixer logs "fixed" for non-applied changes. Next checker
re-flags. Infinite loop.

**Solution**: Verify after every bash edit:

```bash
sed -i 's/old-pattern/new-pattern/' file.md
grep -q "new-pattern" file.md || echo "WARNING: sed pattern did not match — fix NOT applied"
```

Log as **FAILED (not applied)** if verification fails.

### 4. Escalation After 2+ Iteration Disagreements

If checker and fixer disagree on the same finding for 2+ iterations, escalate:

1. Fixer marks finding as `ESCALATED` (not FALSE_POSITIVE, not applied)
2. Notify user: "This finding has been re-flagged after a FALSE_POSITIVE acceptance. Manual review required."
3. Maker resolves the root ambiguity

**Ceiling**: A `<family>-quality-gate` runs at most 3 cycles and ends `FAIL` when the open blocking count stops falling,
per the [Quality Gate Contract](../../../repo-governance/development/workflow/quality-gate-contract.md); no run adds a
fourth cycle.

## Common Mistakes

### All Roles

- **Using fixer for content creation** - Use maker instead
- **Skipping checker validation** - Always validate before publication
- **Manual fixes for mechanical issues** - Use fixer for efficiency
- **Auto-applying MEDIUM confidence fixes** - Needs manual review
- **Not re-validating before fixing** - Prevents false positive fixes

### Checker-Specific

- **Buffering findings**: Don't collect all findings in memory and write at end
- **Wrong timestamp format**: Don't use `YYYY-MM-DD HH:MM` (spaces in filenames)
- **Missing UUID chain**: Don't use timestamp alone for uniqueness
- **Generic scope**: Don't use same scope for all agents

### Fixer-Specific

- **Skipping re-validation**: Don't trust checker, apply fix directly
- **Ignoring priority order**: Don't fix findings in discovery order
- **File-level confidence instead of per-finding**: Each finding assessed independently
