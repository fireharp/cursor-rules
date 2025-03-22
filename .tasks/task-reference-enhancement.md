# Task: Enhance Manager/Lockfile to Support Multiple Types of Rule References

## TS: 2025-03-22 22:23:15 CET

## PROBLEM:

The current cursor-rules lockfile implementation only supports built-in templates referenced by category and ruleKey. Users need a more flexible system that supports various types of rule references similar to package managers like npm and pip, including:

1. Local absolute/relative paths
2. Direct GitHub commit/tags
3. Branch references
4. Directory references

## WHAT WAS DONE:

- Created task plan for enhancing the lockfile structure
- Identified key components to implement
- Outlined implementation approach for different reference types
- Implemented enhanced LockFile structure with RuleSource for tracking metadata
- Added functions for handling different reference types (local files, GitHub URLs)
- Updated CLI commands to support the new reference types
- Maintained backward compatibility with existing functionality
- Added basic tests for the AddRuleByReference function
- Updated documentation in README.md with examples for the new reference types

## MEMO:

This enhancement will transform cursor-rules into a true package manager, allowing users to reference rules from various sources while maintaining reproducibility and upgrade paths. Future work includes completing the GitHub directory reference implementation and adding commit resolution for branch references.

## Task Steps

1. [x] Enhance LockFile Structure

   - [x] Replace `Installed []string` with `Rules []RuleSource` to store metadata
   - [x] Define `RuleSource` struct with fields for source type, reference, git ref, local files, etc.
   - [x] Update serialization/deserialization logic for the new structure

2. [x] Implement Reference Parsing

   - [x] Create `AddRuleByReference` function to handle various references
   - [x] Add helpers for detecting reference types (local paths, GitHub URLs)
   - [x] Implement parsers for different GitHub URL formats (blob, tree)

3. [x] Add Handlers for Different Reference Types

   - [x] Local files (absolute/relative paths)
   - [x] GitHub single file (blob URLs)
   - [ ] GitHub directory (tree URLs) - Placeholder implementation
   - [x] Fallback to built-in templates for backward compatibility

4. [x] Enhance CLI Commands

   - [x] Add `add-ref` subcommand for handling references
   - [x] Update `upgrade` to handle different reference types
   - [x] Ensure `remove` and `list` work with the enhanced lockfile

5. [x] Implement File Installation Logic

   - [x] Download/copy files to .cursor/rules with appropriate naming
   - [x] Track relationship between references and local files
   - [ ] Add git commit resolution for branch references

6. [x] Update Documentation and Examples

   - [x] Update README.md with examples for different reference types
   - [x] Document upgrade behavior for different source types
   - [x] Provide migration guidance for existing users

7. [x] Add Tests

   - [x] Test parsers for different reference formats
   - [x] Test file installation from various sources
   - [x] Test upgrade behavior for different reference types
   - [x] Test backward compatibility with existing logic
