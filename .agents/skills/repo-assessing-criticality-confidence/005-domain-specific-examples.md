# Criticality-Confidence System Skill: Part 5 of 6

## Domain-Specific Examples

### Repository / Infrastructure Plans

**CRITICAL**:

- Missing required sections in delivery checklist (plan execution breaks)
- Broken internal links in plan files (navigation fails)
- Missing acceptance criteria (no way to verify completion)

**HIGH**:

- Wrong folder placement (backlog item in in-progress/)
- Missing Gherkin scenarios for user stories
- Execution-grade clarity violations in delivery checklist

**MEDIUM**:

- Missing optional cross-reference
- Suboptimal section ordering
- Diagram could be clearer

**LOW**:

- Suggest adding related links
- Consider alternative organization

### Documentation Quality

**CRITICAL**:

- [Error] Command syntax incorrect (verified via WebSearch)
- [BROKEN] Internal link to non-existent file
- Security vulnerability in code example

**HIGH**:

- [Outdated] Major version with breaking changes
- Passive voice in step-by-step instructions
- Wrong heading nesting (H1 → H3)

**MEDIUM**:

- [Unverified] External claim needs verification
- Missing optional code fence language tag

**LOW**:

- Suggest additional examples
- Consider adding diagram

## Common Patterns

### Pattern 1: Checker categorizing finding

```markdown
### 1. Missing Required Frontmatter Field

**Criticality**: CRITICAL - Breaks page rendering **Category**: Missing Required Field

**Finding**: Required `draft` field missing from frontmatter **Impact**: Page fails to render with missing required
field **Recommendation**: Add `draft: false` to frontmatter
```

### Pattern 2: Fixer assessing confidence

```text
# Read checker finding
finding = "Missing `draft` field"

# Re-validate
frontmatter = extract_frontmatter(file)
draft_exists = "draft" in frontmatter  # Result: False (confirmed)

# Assess confidence
issue_type = "objective"  # Field either exists or doesn't
re_validation = "confirmed"  # Field is indeed missing
fix_safety = "safe"  # Adding missing field is straightforward

confidence = "HIGH"  # Objective, confirmed, safe → HIGH
priority = determine_priority("CRITICAL", "HIGH")  # → P0

# Apply fix
apply_fix(finding)
```

## Best Practices

### For Checker Agents

**DO**:

- Use decision tree for consistent criticality assessment
- Document specific impact for each finding
- Provide clear, actionable recommendations
- Include examples showing broken vs fixed state
- Write findings progressively during execution

**DON'T**:

- Mix criticality levels in same report section
- Skip impact description
- Provide vague recommendations
- Forget to document verification source (for dual-label)

### For Fixer Agents

**DO**:

- ALWAYS re-validate before applying fixes
- Process findings in strict priority order (P0 → P1 → P2 → P3)
- Document confidence assessment reasoning
- Report false positives with improvement suggestions
- Group fixes by priority in report

**DON'T**:

- Trust checker findings without re-validation
- Apply fixes in discovery order (ignore priority)
- Skip MEDIUM confidence manual review flagging
- Apply P2 fixes without user approval

## Common Mistakes

### Mistake 1: Conflating verification with criticality

**Wrong**: [Error] is CRITICAL, [Verified] is LOW

**Right**: Verification describes WHAT (factual state), criticality describes HOW URGENT

### Mistake 2: File-level confidence instead of per-finding

**Wrong**: Overall file confidence HIGH

**Right**: Each finding assessed independently

### Mistake 3: Skipping re-validation

**Wrong**: Trust checker, apply fix directly

**Right**: Re-validate finding first, then assess confidence

### Mistake 4: Ignoring priority order

**Wrong**: Fix findings in discovery order

**Right**: Fix P0 first, then P1, then P2, then P3-P4
