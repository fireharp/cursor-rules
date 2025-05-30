# Task: Implement Rule Add Functionality with Path Resolution

## Task Steps

- [x] Step 1: Create a new branch `feature/enhance-rule-add`
- [x] Step 2: Analyze current rule add functionality in the codebase
  - Found implementation in `go-claude/cmd/cursor-rules/main.go` and `go-claude/pkg/manager/manager_rules.go`
  - Current implementation handles: local files (absolute/relative paths), GitHub blob URLs, GitHub tree URLs, and built-in templates
  - Missing path resolution options for username/repo patterns and glob support
- [x] Step 3: Implement path resolution logic for rule locations:
  - [x] Check username/cursor-rules-collection repo first
  - [x] Support username/rule path resolution
  - [x] Support username/[path_in_repo]/rule path resolution
  - [x] Fallback to username/repo/[path_in_repo]/rule checking
  - [x] Add support for SHA and tag references (username/rule:[sha1], username/rule@[tag])
  - [x] Implement shorthand syntax for default username configuration
- [x] Step 4: Add glob pattern support for rule paths
  - [x] Implement basic glob support (e.g., path/\*)
  - [x] Add advanced glob support (e.g., path/\*_/_ for recursive)
- [x] Step 5: Fix linting issues and integration bugs
  - [x] Resolve initialization cycle for AddRuleByReferenceFn
  - [x] Add templates.ListTemplates() implementation
  - [x] Ensure proper glob functionality without recursive definition
- [ ] Step 6: Add GitHub API implementation for repo file listing
  - [ ] Implement listGitHubRepoFiles() to enable glob pattern support
  - [ ] Test against real repositories
- [ ] Step 7: Update documentation
- [ ] Step 8: Create PR and request review

## Task Description

The goal is to enhance the rule add functionality to support more flexible path resolution strategies. Currently, we need to implement the following path resolution order:

1. Check by default in username/cursor-rules-collection repo:

   - `cursor-rules add username/rule`
   - `cursor-rules add username/[path_in_repo]/rule`

2. If not found, check:

   - `cursor-rules add username/repo/[path_in_repo]/rule`

3. Support SHA and tag references:

   - `cursor-rules add username/rule:[sha1 full or truncated]`
   - `cursor-rules add username/rule@[tag]`

4. Support default username from configuration:

   - `cursor-rules add rule` → Resolves to username/cursor-rules-collection/rule if default username is set

5. Add glob pattern support:
   - Basic glob: `cursor-rules add go/*`
   - Advanced glob: `cursor-rules add go/**/important/*`

## Project Considerations

- This feature needs to integrate with the existing rule management system
- We should maintain backward compatibility
- Default username configuration should be read from ~/.cursor-rules/ configuration
- Error handling should be clear about which resolution strategies were attempted
- Consider caching mechanisms for previously resolved rules

## Initial Code Analysis Findings

Current implementation in `addRuleByReferenceImpl` handles:

1. GitHub blob URLs via `handleGitHubBlob`
2. GitHub tree URLs via `handleGitHubDir`
3. Absolute local paths via `handleLocalFile`
4. Relative local paths via `handleLocalFile`
5. Built-in templates via `AddRule`

Need to extend this to handle:

1. GitHub references in format `username/rule`
2. GitHub references with path `username/path/rule`
3. GitHub references with repo `username/repo/path/rule`
4. References with SHA or tag
5. Default username configuration

## Current Implementation Status

We have successfully implemented:

1. Username/rule path resolution for cursor-rules-collection repos
2. Username/path/rule path resolution with fallback to repositories
3. SHA and tag reference support
4. Default username configuration for simple rule names
5. Glob pattern matching functionality in principle

Fixed the following issues:

1. Initialization cycle for AddRuleByReferenceFn by breaking the circular dependency
2. Added templates.ListTemplates() function
3. Removed recursive calls to prevent cycles

## Remaining Issues

1. The GitHub API implementation for repository file listing is still a stub
2. We need to implement the actual GitHub API client to fetch repository contents
3. Testing with real patterns against the cursor-rules-collection repository
4. Documentation for the new functionality
5. Additional test coverage for the new features
