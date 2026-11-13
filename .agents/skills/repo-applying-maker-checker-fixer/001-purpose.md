# Maker-Checker-Fixer Pattern (Comprehensive): Part 1 of 7

## Purpose

Use this Skill when:

- Implementing content quality workflows
- Working with maker/checker/fixer agents
- Validating content against conventions
- Applying validated fixes from audit reports
- Understanding agent family structures
- Deciding when to use maker vs fixer for content changes
- Creating new checker or fixer agents
- Executing validation or fix workflows

## The Three Stages

### Stage 1: Maker (Content Creation & Updates)

**Role**: Creates NEW content and updates EXISTING content with all dependencies

**Characteristics**:

- User-driven operation (responds to "create" or "update" requests)
- Comprehensive scope (creates target content AND updates related files)
- Cascading changes (adjusts indices, cross-references, dependencies)
- Proactive management (anticipates what needs updating)

**Tool Pattern**: `Write`, `Edit` (content modification)

**Color**: Blue (writer agents) or Yellow (special case)

**When to Use Maker**:

- User explicitly requests content creation or updates
- Creating NEW content from scratch
- Making significant changes to EXISTING content
- Need comprehensive dependency management
- User-driven workflow (user says "create" or "update")

**Example Workflow**:

```markdown
User: "Create a new plan for adding Tailscale monitoring"

Maker:

1. Grills user for requirements (grill-me skill)
2. Creates all six plan documents (README.md, brd.md, prd.md, one technical shape, delivery.md, learnings.md)
3. Updates plans/backlog/README.md to list the new plan
4. Ensures worktree section is present
5. Validates all internal links
6. Delivers complete, ready-to-execute plan
```

### Stage 2: Checker (Validation) - Detailed Workflow

**Role**: Validates content against conventions and generates audit reports

**Characteristics**:

- Validation-driven (analyzes existing content)
- Non-destructive (does NOT modify files being checked)
- Comprehensive reporting (generates detailed audit in `generated-reports/`)
- Evidence-based (re-validation in fixer prevents false positives)

**Tool Pattern**: `Read`, `Glob`, `Grep`, `Write`, `Bash` (read-only + report generation)

- `Write` needed for audit report files
- `Bash` needed for UTC+7 timestamps

**Color**: Green (checker agents)

**When to Use Checker**:

- New content created from scratch
- Major refactoring or updates
- Before starting plan execution
- Critical files (agent definitions, conventions)

**Criticality Categorization**:

Checkers categorize findings by importance/urgency:

- **CRITICAL** - Breaks functionality, blocks users (must fix before publication)
- **HIGH** - Significant quality degradation, convention violations (should fix)
- **MEDIUM** - Minor quality issues, style inconsistencies (fix when convenient)
- **LOW** - Suggestions, optional improvements (consider for future)

**Report Format**: Findings grouped by criticality with clear labels

#### Checker Workflow: 5-Step Process

Checker agents follow a consistent 5-step workflow:

A checker initializes its report at Step 0, runs domain-specific validation in Steps 1 to N, and ends by finalizing the
report.

```text
+---------------------------+
| Step 0: Initialize Report |
+---------------------------+
    |
    v
+----------------------------+
| Step 1-N: Validate Content |
| (domain-specific)          |
+----------------------------+
    |
    v
+-----------------------------+
| Final Step: Finalize Report |
+-----------------------------+
```

### Step 0: Initialize Report File
