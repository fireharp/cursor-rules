# Task: Split manager.go into focused smaller files

## Task Steps

- [ ] Step 1: Analyze manager.go structure and identify logical groupings
- [ ] Step 2: Create new files for each logical group
- [ ] Step 3: Move code to new files
- [ ] Step 4: Update imports and ensure no circular dependencies
- [ ] Step 5: Run tests to verify functionality
- [ ] Step 6: Run linters to check for issues

## Problem

The `manager.go` file is large (1531 lines) and contains multiple responsibilities, making it difficult to maintain and understand. By splitting it into smaller, focused files, we can improve:

- Code organization and maintainability
- Separation of concerns
- Developer experience and readability
- Easier code navigation and contribution

## Implementation Strategy

### Logical Groupings

1. **Lockfile Operations** (`manager_lockfile.go`)

   - LoadLockFile
   - Save
   - Constants and types related to lockfile

2. **Rule Management** (`manager_rules.go`)

   - AddRule
   - RemoveRule
   - ListInstalledRules
   - GetInstalledRules

3. **Upgrade Operations** (`manager_upgrade.go`)

   - UpgradeRule
   - Helper functions for upgrades

4. **Sharing Operations** (`manager_share.go`)

   - ShareRules
   - RestoreFromShared
   - ShareableRule/ShareableLock types

5. **Utility Functions** (`manager_utils.go`)
   - Path handling (isAbsolutePath, isRelativePath)
   - Key generation (generateRuleKey)
   - Shared types and constants

### Implementation Plan

1. Create new files in pkg/manager directory
2. Move related code while maintaining package structure
3. Ensure no breaking changes to public API
4. Verify tests pass after the refactoring

## Testing

- Run existing tests to ensure functionality remains intact
- Run linting tools to ensure code quality
- Manually test key features

## Success Criteria

1. Code is split into multiple focused files
2. All tests pass
3. No regressions in functionality
4. No new linting issues
5. Improved code organization and readability

## Original Task Description

Splitting large Go files into smaller, more focused files to reduce complexity, organize code, and make maintenance easier. The approach follows:

1. Identify distinct feature sets within the file
2. Group each feature set into dedicated files
3. Create new .go files in the pkg/manager directory
4. Move related functions while maintaining package structure
5. Update imports and ensure no circular dependencies
6. Test thoroughly to ensure no breaking changes
