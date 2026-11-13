# Factual Validation Methodology Skill: Part 1 of 3

## Purpose

This Skill provides a universal methodology for verifying factual correctness in technical documentation, educational
content, and code examples using WebSearch and WebFetch tools.

### When to use this Skill

- Verifying command syntax and flags
- Checking version numbers and compatibility
- Validating code examples compile/run correctly
- Confirming API methods and signatures
- Verifying external claims and references
- Classifying confidence levels for documentation
- Maintaining factual accuracy in plans and docs

## Core Concepts

### What is Factual Validation?

**Factual validation** is the systematic process of verifying objective, verifiable claims in documentation against
authoritative sources using web search and retrieval tools.

**Validates**:

- Command syntax (bash, npm, git, go commands)
- Software version numbers
- API method names and signatures
- Code example correctness
- Configuration file formats
- Library/package availability
- External factual claims

**Does NOT validate**:

- Subjective quality assessments
- Narrative flow or writing style
- Architectural decisions or opinions
- Future predictions or speculation

### The Four Confidence Classifications

All validation findings use one of four confidence labels:

**[Verified]** - Objectively correct according to authoritative sources

- Command syntax matches official documentation
- Version number confirmed via package registry
- Code example compiles/runs as shown
- API method exists with correct signature

**[Error]** - Objectively incorrect, breaks functionality

- Command syntax wrong (fails when executed)
- Code example won't compile/run
- API method doesn't exist or has wrong signature
- Incorrect version number that doesn't exist

**[Outdated]** - Was correct but now superseded by newer version

- References old major version with breaking changes
- Uses deprecated API (still works but replacement exists)
- Command syntax changed in recent release
- Configuration format updated

**[Unverified]** - Cannot confirm correctness (insufficient evidence)

- No authoritative source found
- Multiple conflicting sources
- Documentation ambiguous or incomplete
- Claim too specific to verify externally

## Validation Workflow

### Step-by-Step Process

### 1. Identify Claims to Validate

Extract objective, verifiable claims from content:

```markdown
Content: "Install using `npm install --save-deps prettier`" Claim: npm flag "--save-deps" is valid syntax Type: Command
syntax
```

### 2. Determine Source Priority

Use authoritative sources in priority order:

1. **Official documentation** (docs.npmjs.com, official GitHub repos)
2. **Package registries** (npmjs.com, pypi.org, crates.io)
3. **Official release notes** (CHANGELOG, GitHub releases)
4. **Well-maintained community sources** (Stack Overflow official answers, MDN)

### 3. Execute Web Search

```text
WebSearch query: "npm install flags official documentation"
Target: Find official npm CLI documentation
```

### 4. Fetch Authoritative Content

```text
WebFetch: https://docs.npmjs.com/cli/v9/commands/npm-install
Extract: List of valid flags and their purposes
```

### 5. Compare and Classify

```text
Claim: "--save-deps" flag
Source: Official npm docs list "--save-dev" (NOT "--save-deps")
Classification: [Error]
Confidence: HIGH (official docs contradict claim)
```

### 6. Document Finding

```markdown
**File**: `docs/quick-start.md:42` **Verification**: [Error] - Command syntax incorrect **Criticality**: CRITICAL -
Breaks user quick start experience **Category**: Factual Error - Command Syntax

**Finding**: Installation command uses incorrect npm flag `--save-deps` (should be `--save-dev`)

**Impact**: Users following tutorial get command error, cannot complete setup

**Recommendation**: Change `npm install --save-deps prettier` to `npm install --save-dev prettier`

**Verification Source**: Official npm documentation confirms `--save-dev` is correct flag
https://docs.npmjs.com/cli/v9/commands/npm-install

**Confidence**: HIGH
```
