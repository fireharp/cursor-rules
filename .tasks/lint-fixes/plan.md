# Lint Fixes Plan

## Completed Fixes

- ✅ Octal literals updated to new style (0o755)
- ✅ Empty block removed in manager.go
- ✅ Empty string check using equality instead of length comparison
- ✅ Parameter type combination in function declarations
- ✅ Exhaustive switch cases added for SourceType
- ✅ Unused parameters renamed to \_
- ✅ String constants created for common string values
- ✅ Nesting reduced in manager_test.go
- ✅ Indent-error-flow issues fixed by restructuring code blocks
- ✅ Nil error handling improved in managers.go

## Remaining Issues

### High Priority

1. **HTTP Context Issues**

   - **Files**: pkg/manager/manager.go
   - **Lines**: 443, 1146
   - **Fix**: Add context to HTTP requests using http.NewRequestWithContext
   - **Lint Rule**: noctx

2. **Unwrapped Errors**

   - **Files**: pkg/manager/manager.go
   - **Lines**: 947, 955
   - **Fix**: Wrap errors from filepath.Walk and filepath.Rel with proper context
   - **Lint Rule**: wrapcheck

3. **Long Lines in Go Code**
   - **Files**: cmd/cursor-rules/main.go
   - **Lines**: 41, 49
   - **Fix**: Break lines into multiple lines or use string concatenation
   - **Lint Rule**: lll

### Medium Priority

4. **Complex Nested Blocks**

   - **Files**: pkg/manager/manager.go, cmd/cursor-rules/main.go
   - **High Complexity**:
     - manager.go:104 (os.IsNotExist - complexity: 21)
     - manager.go:791 (!isGitCommitHash - complexity: 20)
     - manager.go:1252 (sr.Content != "" && sr.Filename != "" - complexity: 19)
     - cmd/cursor-rules/main.go:103 (len(args) > 0 - complexity: 51)
   - **Fix**: Break up these functions into smaller, more focused functions
   - **Lint Rule**: nestif

5. **High Cognitive Complexity**
   - **Files**: pkg/manager/manager.go, cmd/cursor-rules/main.go
   - **Highest Complexity**:
     - main.go:22 (main - complexity: 138)
     - manager.go:1139 (RestoreFromShared - complexity: 96)
     - manager.go:739 (UpgradeRule - complexity: 67)
   - **Fix**: Extract helper functions for logical blocks and reduce nesting
   - **Lint Rule**: gocognit

### Low Priority

6. **Long Function in Custom.go**

   - **Files**: pkg/templates/custom.go
   - **Lines**: 12
   - **Fix**: Split 'CreateCustomTemplate' (53 statements) into smaller functions
   - **Lint Rule**: funlen

7. **Long Lines in Markdown Files**
   - **Files**: PROGRESS.md, README.md
   - **Fix**: Consider wrapping lines at 80 characters or disabling this check for markdown
   - **Lint Rule**: lll (markdownlint)

## Approach for Remaining Issues

1. **Focus on Security First**:

   - Address HTTP context issues (noctx) as these are security concerns
   - Fix unwrapped errors to improve error handling

2. **Readability Improvements**:

   - Fix long lines in code to improve readability
   - Address simple nested if structures

3. **Deferred Complex Refactoring**:
   - The high cognitive complexity (gocognit) issues will require significant refactoring
   - Consider creating separate tasks for each complex function
   - Document potential breaking changes that might result from refactoring

## Affected Files (Remaining Issues)

- cmd/cursor-rules/main.go
- pkg/manager/manager.go
- pkg/templates/custom.go
- PROGRESS.md
- README.md
