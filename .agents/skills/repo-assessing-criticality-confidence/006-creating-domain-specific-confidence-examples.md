# Criticality-Confidence System Skill: Part 6 of 6

## Creating Domain-Specific Confidence Examples

Fixer agents should include domain-specific examples of HIGH/MEDIUM/FALSE_POSITIVE confidence assessments to guide
re-validation decisions.

**Where to include**: In fixer agent files (not in this Skill - keep examples domain-specific)

### Example Structure Template

```markdown
### Domain-Specific Confidence Examples

**HIGH Confidence** (Apply automatically):

- [Objective error type 1] verified by [verification method]
- [Objective error type 2] verified by [verification method]

**MEDIUM Confidence** (Manual review):

- [Subjective issue 1] that may be [context-dependent reason]
- [Ambiguous issue] where [ambiguity explanation]

**FALSE_POSITIVE** (Report to checker):

- Checker flagged [correct thing] as incorrect ([reason for false positive])
- Checker reported [missing thing] that actually exists ([reason])
```

## Related Skills

- `repo-applying-maker-checker-fixer` - Deep dive into the three-stage workflow
- `repo-generating-validation-reports` - UUID chain generation, report format, progressive writing

---

This Skill packages the critical criticality-confidence classification system for maintaining consistent quality
validation and automated fixing.
