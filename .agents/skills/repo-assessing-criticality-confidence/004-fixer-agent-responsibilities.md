# Criticality-Confidence System Skill: Part 4 of 6

## Fixer Agent Responsibilities

### Re-Validation Process

**CRITICAL**: Fixer agents MUST re-validate all findings before applying fixes.

### Never trust checker findings blindly

**Process**:

The fixer starts from the checker report, reads each finding, re-executes its validation, assesses confidence, and then
applies, skips, or reports the finding.

```text
+----------------+     +--------------+     +------------+     +------------+     +-------------------+
| Checker Report |---->| Read Finding |---->| Re-execute |---->| Assess     |---->| Apply/Skip/Report |
+----------------+     +--------------+     | Validation |     | Confidence |     +-------------------+
                                            +------------+     +------------+
```

**Re-validation methods**:

- Check file existence for broken links
- Count objective metrics (lines, headings)
- Verify patterns (date format, naming)
- Analyze context (content type, directory)

### Confidence Assessment

### Step 1: Classify issue type

- **Objective** (missing field, wrong format) → Potentially HIGH confidence
- **Subjective** (narrative flow, tone) → MEDIUM confidence

### Step 2: Re-validate finding

- **Confirms issue** → Continue to Step 3
- **Disproves issue** → FALSE_POSITIVE

### Step 3: Assess fix safety

- **Safe and unambiguous** → HIGH confidence (auto-fix)
- **Unsafe or ambiguous** → MEDIUM confidence (manual review)

### Priority-Based Execution

**P0 fixes first** (CRITICAL + HIGH confidence):

```text
for finding in critical_high_confidence:
    apply_fix(finding)  # Auto-fix immediately
    if fix_failed:
        block_deployment()  # Stop if P0 fails
```

**P1 fixes second**:

```text
# AUTO: HIGH criticality + HIGH confidence
for finding in high_high_confidence:
    apply_fix(finding)

# FLAG: CRITICAL + MEDIUM confidence (urgent review)
for finding in critical_medium_confidence:
    flag_for_urgent_review(finding)
```

**P2 fixes third**:

```text
# AUTO if approved: MEDIUM + HIGH
if user_approved_batch_mode:
    for finding in medium_high_confidence:
        apply_fix(finding)

# FLAG: HIGH + MEDIUM (standard review)
for finding in high_medium_confidence:
    flag_for_standard_review(finding)
```

**P3-P4 last**:

```text
# Include in summary only
for finding in low_priority:
    include_in_summary(finding)
```

### Fix Report Format

```markdown
# [Agent Name] Fix Report

**Source Audit**: {agent-family}**{uuid}**{timestamp}\_\_audit.md **Fix Date**: YYYY-MM-DDTHH:MM:SS+07:00

---

## Execution Summary

- **P0 Fixes Applied**: X (CRITICAL + HIGH confidence)
- **P1 Fixes Applied**: Y (HIGH + HIGH confidence)
- **P1 Flagged for Urgent Review**: Z (CRITICAL + MEDIUM confidence)
- **P2 Fixes Applied**: W (MEDIUM + HIGH confidence)
- **P2 Flagged for Standard Review**: V (HIGH + MEDIUM confidence)
- **P3-P4 Suggestions**: U (LOW priority)
- **False Positives Detected**: T

---

## P0 Fixes Applied (CRITICAL + HIGH Confidence)

### 1. [Issue Title]

**File**: `path/to/file.md` **Criticality**: CRITICAL - [Why critical] **Confidence**: HIGH - [Why confident] **Fix
Applied**: [What was changed]

**Before**: [broken state] **After**: [fixed state]

---

[... P1, P2, P3-P4 sections ...]

---

## False Positives Detected

### 1. [Issue Title]

**File**: `path/to/file.md` **Checker Finding**: [What checker reported] **Re-validation**: [What fixer found]
**Conclusion**: FALSE_POSITIVE **Reason**: [Why checker was wrong]

---
```
