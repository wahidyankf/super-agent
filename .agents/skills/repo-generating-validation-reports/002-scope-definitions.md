# Generating Validation Reports: Part 2 of 2

### Scope Definitions

Common scopes for execution tracking:

| Agent Family | Scope  | Tracking File           |
| ------------ | ------ | ----------------------- |
| docs-checker | `docs` | `.execution-chain-docs` |

### Tool Requirements

Agents using this Skill MUST have:

- **Write tool**: Required for creating report files
- **Bash tool**: Required for UUID generation and UTC+7 timestamps

**Example frontmatter**:

```yaml
---
name: example-checker
tools: [Read, Glob, Grep, Write, Bash]
skills: [repo-generating-validation-reports]
---
```

## Reference Documentation

Complete specifications in:

- Temporary Files Convention (if present in repo)

## Usage Example

**Checker Agent Startup**:

```bash
# Generate UUID and determine chain
MY_UUID=$(uuidgen | tr '[:upper:]' '[:lower:]' | head -c 6)
SCOPE="${EXECUTION_SCOPE:-docs}"
CHAIN_FILE="generated-reports/.execution-chain-${SCOPE}"

if [ -f "$CHAIN_FILE" ]; then
  read PARENT_TIME PARENT_CHAIN < "$CHAIN_FILE"
  CURRENT_TIME=$(date +%s)
  TIME_DIFF=$((CURRENT_TIME - PARENT_TIME))

  if [ $TIME_DIFF -lt 300 ]; then
    UUID_CHAIN="${PARENT_CHAIN}_${MY_UUID}"
  else
    UUID_CHAIN="$MY_UUID"
  fi
else
  UUID_CHAIN="$MY_UUID"
fi

echo "$(date +%s) $UUID_CHAIN" > "$CHAIN_FILE"

# Generate timestamp
TIMESTAMP=$(TZ='Asia/Jakarta' date +"%Y-%m-%d--%H-%M")

# Create report filename
REPORT_FILE="generated-reports/docs__${UUID_CHAIN}__${TIMESTAMP}__audit.md"

# Initialize report (progressive writing starts here)
cat > "$REPORT_FILE" << 'REPORT_HEADER'
# Validation Report: docs-checker

**Status**: In Progress
**Agent**: docs-checker
**Scope**: Documentation validation
**Timestamp**: [timestamp]
**UUID Chain**: [uuid-chain]

---

## Findings

REPORT_HEADER

# Continue with validation, writing findings progressively...
```

## Key Principles

1. **Generate UUID early**: First thing at agent startup
2. **Initialize report immediately**: Before any validation begins
3. **Write progressively**: Append findings as you discover them
4. **Use UTC+7 timestamps**: Consistent timezone across all reports
5. **Follow 4-part naming**: Agent-family, UUID chain, timestamp, type
6. **Track execution scope**: Enable parent-child hierarchy for workflows
7. **Require Write+Bash tools**: Essential for report generation

## Common Mistakes to Avoid

❌ **Buffering findings**: Don't collect all findings in memory and write at end (context compaction risk) ✅
**Progressive writing**: Write each finding immediately after discovery

❌ **Wrong timestamp format**: Don't use `YYYY-MM-DD HH:MM` (spaces in filenames) ✅ **Correct format**: Use
`YYYY-MM-DD--HH-MM` (double dash separator)

❌ **Missing UUID chain**: Don't use timestamp alone for uniqueness ✅ **UUID chain**: Enables parallel execution
without collisions

❌ **Generic scope**: Don't use same scope for all agents ✅ **Specific scope**: Use agent-family or language-specific
scope

## Integration with Other Skills

Works alongside:

- `repo-assessing-criticality-confidence` - Categorize findings by severity
- `repo-applying-maker-checker-fixer` - Fixer agents read these reports

## Benefits

1. **Parallelization-safe**: UUID chains prevent file collisions
2. **Traceable**: Can track parent-child execution hierarchy
3. **Resilient**: Progressive writing survives context compaction
4. **Consistent**: Standard naming across all checker agents
5. **Debuggable**: Timestamp and UUID chain aid troubleshooting
