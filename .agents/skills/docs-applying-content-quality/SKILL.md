---
name: docs-applying-content-quality
description: >-
  Applies active voice, heading hierarchy, accessible alternatives, contrast guidance, and consistent formatting to
  repository Markdown.
when_to_use: >-
  Use before drafting or revising Markdown whose structure, accessibility, and prose quality must be reviewed.
---

# Applying Content Quality Standards

## Purpose

This Skill provides comprehensive guidance for applying **universal content quality standards** to all markdown content
in the repository. It ensures consistent writing quality, accessibility compliance, and professional presentation across
documentation, planning documents, and root files.

### When to use this Skill

- Creating or editing markdown content in docs/
- Creating planning documents in plans/
- Writing repository root files (README.md, CONTRIBUTING.md, etc.)
- Ensuring accessibility compliance (WCAG AA)
- Reviewing content for quality standards

## Core Quality Principles

### Writing Style and Tone

**Active Voice Required**: Use active voice for clarity and directness.

✅ **Good**: "The agent validates the content against the convention." ❌ **Avoid**: "The content is validated against
the convention by the agent."

**Professional Tone**: Maintain professional, welcoming tone without being overly formal.

**Clarity and Conciseness**: Write clear, direct sentences. Avoid jargon without context.

**Audience Awareness**: Consider reader's technical level and provide necessary context.

### Heading Hierarchy

**Single H1 Rule**: Each markdown file MUST have exactly one H1 heading (# Title).

**Proper Nesting**: Follow hierarchical structure without skipping levels:

- H1 (#) - Document title
- H2 (##) - Major sections
- H3 (###) - Subsections
- H4 (####) - Sub-subsections
- H5/H6 - Use sparingly

❌ **Invalid nesting** (skips level):

```markdown
# Title

### Subsection ← Skips H2!
```

✅ **Valid nesting**:

```markdown
# Title

## Section

### Subsection
```

### Accessibility Standards

**Alt Text Required**: All images MUST have descriptive alt text.

```markdown
✅ !Architecture diagram showing six-layer hierarchy ❌ ![](./diagram.png) ← Missing alt text
```

**WCAG AA Color Contrast**: Text must meet WCAG AA contrast ratios:

- Normal text: 4.5:1 minimum
- Large text (18pt+): 3:1 minimum

**Semantic Formatting**:

- Use **bold** for emphasis, not italics
- Use proper heading structure (not bold text as headers)
- Use lists for list content (not manual bullets)

**Screen Reader Support**: Content must be accessible to screen readers through proper HTML structure and ARIA labels
when needed.

### Formatting Conventions

**Code Blocks**: Always specify language for syntax highlighting.

````markdown
✅ Good:

```javascript
const x = 10;
```
````

❌ Bad:

```text
const x = 10;  ← No language specified
```

````text

**Paragraph Length**: Keep paragraphs concise (≤5 lines for readability).

**Line Length**: Keep every line within 120 characters, tables and code blocks included.

**Lists**: Use consistent formatting:
- Unordered lists: Use `-` (hyphen) for consistency
- Ordered lists: Use `1.` numbering
- Nested lists: Indent with 2 spaces per level

## No Time Estimates in Plan Documents

Plan documents — a plan's `README.md`, `brd.md`, `prd.md`, technical shape, and `delivery.md`, and idea briefs — state
no time estimates, durations, or dates as effort commitments. Schedule by dependency, order, and resource instead.

Tutorials, how-to guides, reference, other documentation, plan evidence, and `learnings.md` may carry an estimate when
it is labelled as an estimate. See No Time Estimates.


## Common Quality Checklist

Before publishing any markdown content, verify:

- [ ] Active voice used throughout
- [ ] Exactly one H1 heading
- [ ] Proper heading nesting (no skipped levels)
- [ ] All images have descriptive alt text
- [ ] Code blocks specify language
- [ ] No time estimates in plan documents; any estimate elsewhere labelled as one
- [ ] Professional, welcoming tone
- [ ] Paragraphs ≤5 lines
- [ ] Clear, jargon-free language (or jargon explained)
- [ ] WCAG AA color contrast for any custom colors
- [ ] Semantic formatting (bold for emphasis, proper lists)

## Common Mistakes

### Mistake 1: Missing alt text

**Wrong**: `![](./image.png)`
**Right**: `!Detailed description of image content`

### Mistake 2: Skipped heading levels

**Wrong**:
```markdown
# Title
### Subsection  ← Skips H2
````

**Right**:

```markdown
# Title

## Section

### Subsection
```

### Mistake 3: Time estimate in a plan document

**Wrong**: "Phase 2 takes three days." in `delivery.md`. **Right**: "Phase 2 starts after Phase 1's gate passes."

### Mistake 4: Passive voice overuse

**Wrong**: "The file is created by the command." **Right**: "The command creates the file."

### Mistake 5: Code blocks without language

**Wrong**:

```text
npm install
```

**Right**:

```bash
npm install
```

---

This Skill packages universal content quality standards for consistent, accessible, professional markdown content across
the repository.
