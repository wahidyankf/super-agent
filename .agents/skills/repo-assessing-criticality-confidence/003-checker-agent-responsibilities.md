# Criticality-Confidence System Skill: Part 3 of 6

## Checker Agent Responsibilities

### Categorizing Findings by Criticality

**Decision tree**:

Four yes-or-no questions are asked in order, and the first YES sets the finding's criticality to CRITICAL, HIGH, MEDIUM,
or LOW.

```mermaid
flowchart TD
    accTitle: Criticality decision tree
    accDescr: Four yes-or-no questions run in order, and the first YES sets the criticality of the finding.
    q1{"1. Does it BREAK<br/>functionality or<br/>BLOCK users?"}
    q2{"2. Does it cause SIGNIFICANT<br/>quality degradation or<br/>violate DOCUMENTED conventions?"}
    q3{"3. Is it a MINOR<br/>quality issue or<br/>style inconsistency?"}
    q4{"4. Is it a suggestion,<br/>optimization, or<br/>future consideration?"}
    critical["CRITICAL"]
    high["HIGH"]
    medium["MEDIUM"]
    low["LOW"]
    q1 -->|YES| critical
    q1 -->|NO| q2
    q2 -->|YES| high
    q2 -->|NO| q3
    q3 -->|YES| medium
    q3 -->|NO| q4
    q4 -->|YES| low
    classDef default fill:#0173B2,stroke:#000000,color:#FFFFFF
```

### Standardized Report Format

**Report header**:

```markdown
# [Agent Name] Audit Report

**Audit ID**: {uuid-chain}\_\_{timestamp} **Scope**: {scope-description} **Files Checked**: N files **Audit Start**:
YYYY-MM-DDTHH:MM:SS+07:00 **Audit End**: YYYY-MM-DDTHH:MM:SS+07:00

---

## Executive Summary

- **CRITICAL Issues**: X (must fix before publication)
- **HIGH Issues**: Y (should fix before publication)
- **MEDIUM Issues**: Z (improve when time permits)
- **LOW Issues**: W (optional enhancements)

**Total Issues**: X + Y + Z + W = TOTAL

**Overall Status**: [PASS | PASS WITH WARNINGS | FAIL]

---
```

**Issue sections**:

````markdown
## CRITICAL Issues (Must Fix)

**Count**: X issues found

---

### 1. [Issue Title]

**File**: `path/to/file.md:line` **Criticality**: CRITICAL - [Why critical] **Category**: [Category name]

**Finding**: [What's wrong] **Impact**: [What breaks if not fixed] **Recommendation**: [How to fix]

**Example**:

```yaml
# Current (broken)
[show broken state]

# Expected (fixed)
[show fixed state]
```
````

**Confidence**: [Will be assessed by fixer]

---

### Dual-Label Pattern

**Agents that require BOTH verification/status AND criticality**:

Format:

```markdown
### 1. [Verification] - Issue Title

**File**: `path/to/file.md:line` **Verification**: [Error] - [Reason for verification status] **Criticality**:
CRITICAL - [Reason for criticality level] **Category**: [Category name]

**Finding**: [Description] **Impact**: [Consequences] **Recommendation**: [Fix] **Verification Source**: [URL]

**Confidence**: [Will be assessed by fixer]
```

### Why dual labels?

- **Verification** describes FACTUAL STATE ([Verified], [Error], etc.)
- **Criticality** describes URGENCY/IMPORTANCE (CRITICAL, HIGH, etc.)
- Both provide complementary information
