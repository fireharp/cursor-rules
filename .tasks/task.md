# Task: Implement Package Manager-style API for Cursor Rules

## TS: 2025-03-22 19:36:33 CET

## PROBLEM:

Need to create a lower-level "package manager-style" API for cursor rules, while maintaining compatibility with existing functionality. Based on review feedback, several improvements are needed.

## WHAT WAS DONE:

- Analyzed requirements for a package manager approach
- Created task outline with implementation steps
- Implemented the pkg/manager package with core functions
- Updated main.go to use the manager package and added new subcommands
- Received code review with suggestions for improvements
- Fixed the bug where template modifications weren't updating the global map
- Added comprehensive tests for the manager package
- Updated README.md with documentation for the new package manager commands

## MEMO:

This implementation introduced a two-layer architecture:

1. Low-level manager package with add/remove/upgrade functions
2. High-level CLI commands built on top of these functions

All tests pass for the manager package, and the code now correctly updates templates in the global map before writing them to disk.

## Task Steps

1. [x] Create pkg/manager package

   - [x] Design LockFile structure for tracking installed rules
   - [x] Implement LoadLockFile and Save functions
   - [x] Create core functions: AddRule, RemoveRule, UpgradeRule, ListInstalledRules

2. [x] Update main.go to support new subcommands

   - [x] Add "add" subcommand with category flag
   - [x] Add "remove" subcommand
   - [x] Add "upgrade" subcommand
   - [x] Add "list" subcommand
   - [x] Implement showHelp() function with updated usage info

3. [x] Refactor existing CLI commands

   - [x] Modify init command to use manager.AddRule
   - [x] Modify setup command to use manager.AddRule for each detected rule
   - [x] Ensure backward compatibility with flag-style syntax

4. [x] Fix bugs and implementation issues

   - [x] Fix bug: Update init template in global map after modifying content
   - [x] Review other template modifications for similar issues (fixed in setupProject)
   - [x] Ensure consistent error handling throughout codebase
   - [ ] Check for typos in template text (e.g., "cursor-rulse")

5. [x] Add tests for the manager package

   - [x] Test LockFile operations (create, load, save)
   - [x] Test rule management functions (add, remove, upgrade, list)
   - [x] Test edge cases (e.g., adding a rule twice, removing a non-existent rule)
   - [x] Test CLI integration

6. [ ] Future enhancements (optional)

   - [ ] Enhance LockFile to store more metadata (version, filename, category)
   - [ ] Add version pinning capability
   - [ ] Support for remote rule repositories

7. [x] Documentation and examples

   - [x] Update README.md with new subcommands
   - [x] Add examples for package manager usage
   - [x] Document migration path for existing users
   - [ ] Expand or unify documentation in LOCAL.md
