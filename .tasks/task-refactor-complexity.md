# Task: Refactor Code to Reduce Complexity

## TS: 2025-03-24 06:40:30 CET

## PROBLEM

The codebase has multiple functions with high cyclomatic and cognitive complexity, as reported by golangci-lint. The complexity makes the code harder to understand, test, and maintain. Key offenders include:

- main() in cmd/cursor-rules/main.go (complexity: 138)
- RestoreFromShared() in pkg/manager/manager.go (complexity: 98)
- UpgradeRule() in pkg/manager/manager.go (complexity: 67)
- Several functions with nested if blocks (nestif issues)

## WHAT WAS DONE

- Created a detailed refactoring plan using techniques like:
  - Guard clauses to flatten nested conditionals
  - Extraction of helper functions to reduce complexity
  - Separation of subcommands into dedicated handlers

## MEMO

The refactoring will focus on reducing complexity while maintaining the same functionality. Each major function should be refactored in separate PRs to minimize risk. This approach will make the code more maintainable, easier to test, and less prone to bugs when adding new features.

## Task Steps

1. [ ] Refactor main() in cmd/cursor-rules/main.go

   - [ ] Create separate handler functions for each subcommand
   - [ ] Use guard clauses for early returns on flags
   - [ ] Extract common initialization logic
   - [ ] Verify functionality after refactoring

2. [ ] Refactor RestoreFromShared() in pkg/manager/manager.go

   - [ ] Extract loadShareData() function for URL/local file handling
   - [ ] Extract resolveConflict() function for conflict resolution
   - [ ] Extract handleEmbeddedOrExternalRule() function
   - [ ] Update tests to ensure behavior is preserved

3. [ ] Refactor UpgradeRule() in pkg/manager/manager.go

   - [ ] Break down logic into smaller functions
   - [ ] Use guard clauses to reduce nesting
   - [ ] Improve error handling flow

4. [ ] Refactor setupProject() in cmd/cursor-rules/main.go

   - [ ] Extract template writing functions
   - [ ] Create separate functions for detecting project types
   - [ ] Simplify the main flow

5. [ ] Refactor CreateCustomTemplate() in pkg/templates/custom.go

   - [ ] Extract gatherTemplateMetadata() function
   - [ ] Extract readTemplateContent() function
   - [ ] Simplify the main flow

6. [ ] Fix remaining nestif issues

   - [ ] Identify other functions with high nesting
   - [ ] Use guard clauses and extraction to reduce nesting

7. [ ] Verify all tests pass

   - [ ] Run test suite to confirm refactoring preserves functionality
   - [ ] Add tests for new extracted functions as appropriate

8. [ ] Update documentation
   - [ ] Update code comments for new functions
   - [ ] Update PROGRESS.md with completed refactoring work

