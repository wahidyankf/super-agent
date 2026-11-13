# Maker-Checker-Fixer Pattern (Comprehensive): Part 2 of 7

### CRITICAL FIRST STEP - Execute before any validation begins

```bash
# 1. Generate 6-char UUID
MY_UUID=$(uuidgen | tr '[:upper:]' '[:lower:]' | head -c 6)

# 2. Determine UUID chain (scope-based)
SCOPE="${EXECUTION_SCOPE:-agent-family}"
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

# 3. Generate UTC+7 timestamp
TIMESTAMP=$(TZ='Asia/Jakarta' date +"%Y-%m-%d--%H-%M")

# 4. Create report filename
REPORT_FILE="generated-reports/${AGENT_FAMILY}__${UUID_CHAIN}__${TIMESTAMP}__audit.md"

# 5. Initialize report with header
cat > "$REPORT_FILE" << 'HEADER'
# Validation Report: {Agent Name}

**Status**: In Progress
**Agent**: {agent-name}
**Scope**: {scope-description}
**Timestamp**: {YYYY-MM-DD--HH-MM UTC+7}
**UUID Chain**: {uuid-chain}

---

## Findings

[Findings will be written progressively during validation]
HEADER
```

### Why Initialize Early?

- Creates file before validation begins (survives context compaction)
- Enables progressive writing (append findings as discovered)
- Provides audit trail even if validation interrupted
- File is readable throughout execution

### Steps 1-N: Validate Content (Domain-Specific)

**Pattern**: Each checker has domain-specific validation steps, but all follow progressive writing.

**Progressive Writing Requirements**:

- Write each finding to report file immediately after discovery
- Don't buffer findings in memory
- Use append mode for file writes
- Include all finding details (file, line, criticality, issue, recommendation)

**Finding Format**:

```markdown
### Finding {N}: {Title}

**File**: path/to/file.md **Line**: {line-number} (if applicable) **Criticality**: {CRITICAL/HIGH/MEDIUM/LOW}
**Category**: {category-name}

**Issue**: {Description of what's wrong}

**Recommendation**: {How to fix it}

---
```

### Final Step: Finalize Report

### Final update to existing report file

```bash
# Update report status
cat >> "$REPORT_FILE" << 'SUMMARY'

## Summary

**Total Findings**: {N}

**By Criticality**:
- CRITICAL: {count}
- HIGH: {count}
- MEDIUM: {count}
- LOW: {count}

**Status**: Complete
**Completed**: {YYYY-MM-DD--HH-MM UTC+7}
SUMMARY
```

**Finalization Checklist**:

1. Update status: "In Progress" → "Complete"
2. Count findings by criticality level
3. Add completion timestamp
4. Ensure all findings written to file (progressive writing)
5. Report file path to user

### Progressive Writing Methodology

**CRITICAL REQUIREMENT**: All checker agents MUST write findings progressively.

**Why?** Context compaction during long validation runs can lose buffered findings. Progressive writing ensures audit
history survives.
