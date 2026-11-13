# Maker-Checker-Fixer Pattern (Comprehensive): Part 5 of 7

## Best Practices

### For All Roles

1. **Always run checker before execution** - Catches issues early
2. **Review audit reports before fixing** - Understand what will change
3. **Use maker for user-driven creation** - Not fixer
4. **Use fixer for validation-driven fixes** - Not maker
5. **Re-run checker after major fixes** - Verify fixes worked
6. **Report false positives** - Improves checker accuracy over time

### For Checkers

**DO**:

- Initialize report file before validation begins (Step 0)
- Write findings progressively during execution
- Use decision tree for consistent criticality assessment
- Document specific impact for each finding
- Provide clear, actionable recommendations

**DON'T**:

- Buffer findings in memory (context compaction risk)
- Mix criticality levels in same report section
- Skip impact description
- Provide vague recommendations

### For Fixers

**DO**:

- ALWAYS re-validate before applying fixes
- Process findings in strict priority order (P0 → P1 → P2 → P3)
- Document confidence assessment reasoning
- Report false positives with improvement suggestions
- Trust checker's documented verification work
- Respect mode parameter thresholds

**DON'T**:

- Trust checker findings without re-validation
- Apply fixes in discovery order (ignore priority)
- Skip MEDIUM confidence manual review flagging
- Apply P2 fixes without user approval
