# Factual Validation Methodology Skill: Part 3 of 3

## Integration with Checker Agents

### Dual-Label Pattern

Factual validation findings require BOTH verification label AND criticality level:

```markdown
### 1. [Error] - Command Syntax Incorrect in Installation Guide

**File**: `docs/quick-start.md:42` **Verification**: [Error] - Command syntax verified incorrect via WebSearch
**Criticality**: CRITICAL - Breaks user quick start experience **Category**: Factual Error - Command Syntax

**Finding**: [description] **Impact**: [consequences] **Recommendation**: [fix] **Verification Source**: [URL]
**Confidence**: HIGH
```

### Why dual labels?

- **Verification** ([Verified]/[Error]/[Outdated]/[Unverified]) describes FACTUAL STATE
- **Criticality** (CRITICAL/HIGH/MEDIUM/LOW) describes URGENCY/IMPORTANCE

Both dimensions provide complementary information.

### Confidence Assessment

**HIGH confidence** when:

- Official documentation clearly contradicts claim
- Package registry shows version doesn't exist
- Code example fails to compile (verified locally)
- Multiple authoritative sources agree

**MEDIUM confidence** when:

- Sources partially contradict (some say yes, some say no)
- Claim is version-specific but version unclear
- Documentation ambiguous or incomplete
- Cannot find definitive authoritative source

### Delegate Research to `web-researcher` for Context Isolation

Delegate to `web-researcher` whenever verifying a single claim requires 2+ `WebSearch` calls or 3+ `WebFetch` calls;
in-context work remains correct for single-shot verification against a known authoritative URL and for fixer agents
re-validating a single audit finding.

### Delegation pattern

```text
Agent tool call → subagent_type: web-researcher
Prompt: "Verify whether <specific claim>. Return cited findings with confidence tags."
```

`web-researcher` returns a synthesised, cited summary using the same
`[Verified]/[Outdated]/[Unverified]/[Needs Verification]` confidence tags defined in this skill. You then translate
those tags into dual-labelled findings (`[Verified]/[Error]/[Outdated]/[Unverified]` + `CRITICAL/HIGH/MEDIUM/LOW`) for
your audit report.

## Common Mistakes

### Mistake 1: Using outdated sources

**Wrong**: Citing 5-year-old blog post for current syntax

**Right**: Check official documentation for latest version

### Mistake 2: Missing version context

**Wrong**: "Go generics were introduced recently"

**Right**: "Go generics were introduced in Go 1.18 (March 2022)"

### Mistake 3: Trusting unofficial sources

**Wrong**: Using random Stack Overflow answer as sole source

**Right**: Verify with official docs, use SO for supplementary context

### Mistake 4: Not documenting verification source

**Wrong**: Marking [Verified] without citing source

**Right**: Always include verification source URL in finding

### Mistake 5: Conflating verification with subjective quality

**Wrong**: [Error] for "code style not following best practices"

**Right**: Use [Error] only for objective incorrectness (won't compile, wrong syntax)

## Best Practices

### Validation Checklist

Before marking content as validated:

- [ ] Identified all objective, verifiable claims
- [ ] Used authoritative sources (official docs, registries)
- [ ] Documented verification source URLs
- [ ] Applied correct confidence classification
- [ ] Recorded validation date and expiry
- [ ] Classified criticality level
- [ ] Provided clear remediation steps for errors

### Batch Validation Workflow

1. **Extract claims**: Scan content for all verifiable claims
2. **Group by type**: Commands, versions, code examples, APIs
3. **Prioritize**: Critical paths first (install commands, quick starts)
4. **Validate systematically**: One claim type at a time
5. **Document findings**: Use standardized format

### Tool Usage Pattern

```text
Step 1: WebSearch to find authoritative source
  Query: "npm install flags official documentation"
  Result: Multiple sources, identify official docs

Step 2: WebFetch to retrieve content
  URL: https://docs.npmjs.com/cli/v9/commands/npm-install
  Extract: Flag list, examples, usage patterns

Step 3: Compare and classify
  Claim vs Source → Determine classification
  Document finding with source citation
```

---

This Skill packages critical factual validation methodology for maintaining accuracy in technical documentation.
