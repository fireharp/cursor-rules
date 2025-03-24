# Task: Refactor high complexity functions in manager.go

## Problem:

The functions `RestoreFromShared()` and `UpgradeRule()` in `pkg/manager/manager.go` have high cognitive complexity that makes them difficult to understand and maintain. They also trigger linter warnings for complexity.

- `RestoreFromShared()` has a complexity of 98 ✓ FIXED
- `UpgradeRule()` has a complexity of 67 ✓ FIXED

These functions have:

- High cognitive complexity (exceeding threshold of 20)
- Multiple nested conditionals
- Inconsistent error handling
- Multiple levels of responsibilities

## Strategy:

The refactoring approach will focus on:

1. Decomposing large functions into smaller, single-responsibility helper functions
2. Using guard clauses to simplify conditionals and flatten nested structures
3. Ensuring consistent error handling with proper context
4. Using descriptive naming to make the code self-documenting
5. Making incremental changes that can be tested individually

## Implementation:

### RestoreFromShared() Refactoring ✓ COMPLETED

1. ✓ Extract `loadShareableData(ctx, sharePath)` function to handle data loading
2. ✓ Extract `loadShareableFromURL(ctx, url)` for HTTP requests
3. ✓ Extract `parseShareableLock(data)` for unmarshaling and validation
4. ✓ Extract `buildExistingRuleSet(lock)` for conflict detection
5. ✓ Extract `resolveConflict(key, autoResolve)` for conflict resolution
6. ✓ Extract `promptForConflictResolution(key)` for user interaction
7. ✓ Extract rule processing functions:
   - ✓ `processEmbeddedRuleContent()`
   - ✓ `processGitHubRule()`
   - ✓ `processBuiltInRule()`
   - ✓ `processLocalRule()`
   - ✓ `processRule()` to handle a single rule
8. ✓ Add consistent error handling and function documentation

### UpgradeRule() Refactoring ✓ COMPLETED

1. ✓ Extract `findRuleToUpgrade(lock, ruleKey)` to locate rules
2. ✓ Extract `upgradeBuiltInRule(cursorDir, rule)` for built-in rules
3. ✓ Extract `checkLocalModifications(rule, cursorDir)` for file change detection
4. ✓ Extract `promptForLocalModifications(filePath)` for user interaction
5. ✓ Extract functions for different source types:
   - ✓ `upgradeGitHubBranchRule(cursorDir, rule, gitRef, owner, repo)`
   - ✓ `upgradeGitHubPinnedRule(cursorDir, rule, gitRef)`
   - ✓ `upgradeLocalRule(cursorDir, rule)`
6. ✓ Fix errors and improve error handling throughout

## Testing:

1. ✓ Run `golangci-lint run --disable-all --enable gocognit,nestif` to check for complexity warnings
2. ✓ Verify that `RestoreFromShared()` and `UpgradeRule()` no longer appear in warnings
3. ☐ Test the share and restore functionality with `cursor-rules share` and `cursor-rules restore`
4. ☐ Test rule upgrades with `cursor-rules upgrade <rule>` for different rule types

## Success Criteria:

1. ✓ Linter no longer shows cognitive complexity warnings for these functions
2. ✓ Code passes all linter checks for nestif
3. ✓ Functions have clear, single responsibilities
4. ✓ Code has improved error handling with proper context
5. ☐ All existing functionality continues to work correctly

## Remaining Work:

- Reduce complexity in other functions:
  - LoadLockFile (complexity: 42)
  - ShareRules (complexity: 33)
  - RemoveRule (complexity: 31)
  - processRule (complexity: 22)
- Test all functionality to ensure no regressions
