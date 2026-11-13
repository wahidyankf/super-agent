# Maker-Checker-Fixer Pattern (Comprehensive): Part 7 of 7

## Tool Requirements

### Checkers

Checkers typically need:

- **Read**: Read files to validate
- **Glob**: Find files by pattern
- **Grep**: Extract content patterns
- **Write**: Initialize and update report file
- **Bash**: Generate UUID, timestamp, file operations
- **WebFetch**: (Optional) Access official documentation
- **WebSearch**: (Optional) Find authoritative sources

**Bash Tool Critical**: Required for UUID generation and report initialization.

### Fixers

Fixers typically need:

- **Read**: Read audit reports and files to fix
- **Edit**: Apply fixes to files
- **Bash**: Apply fixes (sed, awk), generate timestamps
- **Write**: Generate fix reports
- **Glob/Grep**: Optional - for pattern matching and validation

## Related Skills

- `repo-assessing-criticality-confidence` - Deep dive into criticality/confidence levels and priority matrix
- `repo-generating-validation-reports` - UUID chain generation, report format, progressive writing

---

**Note**: This Skill provides comprehensive action-oriented guidance combining pattern overview with detailed execution
workflows.
