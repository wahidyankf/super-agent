# Factual Validation Methodology Skill: Part 2 of 3

## Source Prioritization

### Tier 1: Official Documentation (Highest Authority)

**Examples**:

- Official language documentation (go.dev, nodejs.org)
- Official package documentation (npmjs.com package pages, pkg.go.dev)
- Official CLI references (git-scm.com, docs.npmjs.com, nx.dev)
- Official API documentation (developer.mozilla.org, official SDK docs)

**Characteristics**:

- Maintained by project creators
- Canonical source of truth
- Version-specific
- Updated with releases

**When to use**: For syntax, commands, API signatures, version numbers

### Tier 2: Package Registries (Version Truth)

**Examples**:

- npmjs.com (JavaScript/Node.js packages)
- pkg.go.dev (Go modules)

**Characteristics**:

- Authoritative version numbers
- Availability confirmation
- Dependency information
- Release dates

**When to use**: For version verification, package availability, dependency checks

### Tier 3: Official Release Notes (Change Truth)

**Examples**:

- GitHub Releases pages
- CHANGELOG.md files in official repos
- Official blog announcements
- Migration guides

**Characteristics**:

- Documents changes between versions
- Breaking change identification
- Deprecation notices
- Migration paths

**When to use**: For outdated content detection, breaking change verification

### Tier 4: Well-Maintained Community (Supplementary)

**Examples**:

- Stack Overflow accepted answers (with high votes)
- MDN Web Docs (Mozilla Developer Network)
- Official language tutorials for the languages the repository uses

**Characteristics**:

- Community-verified
- Practical examples
- Context-dependent correctness
- May lag behind latest versions

**When to use**: For practical patterns, edge cases, when official docs insufficient

## Common Validation Patterns

### Pattern 1: Command Syntax Validation

**Claim Type**: Bash/shell command syntax

**Validation Steps**:

1. Extract command and flags from content
2. WebSearch: "[command-name] official documentation"
3. WebFetch: Official CLI reference page
4. Compare: Flags, argument order, option names
5. Classify: [Verified] if exact match, [Error] if wrong

**Example**:

```text
Content: "git commit -am 'message'"
WebSearch: "git commit flags official documentation"
WebFetch: https://git-scm.com/docs/git-commit
Verify: -a flag exists, -m flag exists, combined -am valid
Result: [Verified]
```

### Pattern 2: Version Number Validation

**Claim Type**: Software/package version reference

**Validation Steps**:

1. Extract package name and version number
2. WebSearch: "[package-name] [version] release"
3. WebFetch: Package registry page (npmjs.com, pypi.org)
4. Compare: Version exists in registry
5. Check: Latest version, deprecation status
6. Classify: [Verified], [Outdated], or [Error]

### Pattern 3: Code Example Validation

**Claim Type**: Code snippet correctness

**Validation Steps**:

1. Extract code example from content
2. Identify language and version
3. WebSearch: "[language] [concept] official examples"
4. WebFetch: Official documentation or tutorial
5. Compare: Syntax, method signatures, patterns
6. Classify: [Verified], [Error], or [Outdated]

### Pattern 4: API Method Validation

**Claim Type**: API method existence and signature

**Validation Steps**:

1. Extract API method name, parameters, return type
2. WebSearch: "[library] [method] official api documentation"
3. WebFetch: Official API reference
4. Compare: Method name, parameter types, return type
5. Check: Deprecation status
6. Classify: [Verified], [Error], or [Outdated]

## Update Frequency Rules

### When to Re-validate Content

**MANDATORY re-validation triggers**:

- **6 months since last validation** - Standard refresh cycle
- **Major version release** of referenced software/library
- **Breaking change announced** in official release notes
- **User reports error** in documentation
- **Deprecation notice** for used API/method

**OPTIONAL re-validation triggers**:

- Minor version updates (if content specific to that version)
- Patch releases (usually safe to skip)
- Community feedback about potential issues
- Regular documentation review cycles
