# Task: Resolve Linting Issues

## TS: 2025-03-24 06:11:20 CET

## PROBLEM

The codebase has several linting issues flagged by golangci-lint, including HTTP context issues, unwrapped errors, long lines, complex nested blocks, and high cognitive complexity. These issues need to be addressed to improve code quality and maintainability.

## WHAT WAS DONE

- Created task plan for resolving linting issues identified by golangci-lint
- Prioritized issues based on severity (HTTP context, unwrapped errors, long lines)
- Planned approach for complex refactoring tasks to maintain code stability

## MEMO

Addressing linting issues will improve code robustness and readability. The implementation will focus on highest priority issues first, followed by complex refactoring. This systematic approach ensures that each change can be tested independently.

## Completed Work

We've successfully addressed the high-priority linting issues:

1. Fixed HTTP context issues (noctx):

   - Added context parameters to network-related functions in manager.go
   - Modified HTTP requests to use http.NewRequestWithContext
   - Updated function calls across the codebase to pass context.Background()

2. Fixed unwrapped errors (wrapcheck):

   - Added proper error wrapping with fmt.Errorf and %w in filepath.Walk
   - Improved error messages for filepath.Rel errors
   - Added context to various error returns

3. Fixed long lines (lll):
   - Broke up long flag description lines in main.go
   - Split parameter descriptions into multiple lines

## Remaining Issues

The following issues still need to be addressed:

1. Complex nested blocks (nestif):

   - main.go: if len(args) > 0 (complexity: 51)
   - manager.go: Various functions with complexity > 5

2. High cognitive complexity (gocognit):

   - main.go: main() function (complexity: 138)
   - manager.go: RestoreFromShared() function (complexity: 98)
   - manager.go: UpgradeRule() function (complexity: 67)
   - Several other functions with complexity > 20

3. Long function (funlen):
   - templates/custom.go: CreateCustomTemplate() (53 > 50 statements)

These issues require more extensive refactoring and should be approached carefully to avoid introducing bugs. The recommended approach is to:

1. Break down large functions into smaller helper functions
2. Use guard clauses to reduce nesting
3. Extract complex logic into separate functions
4. Apply consistent patterns for error handling and flow control

Given the complexity of these changes, they should be addressed in a separate PR after thorough testing of the current fixes.


## Task Steps

1. [x] Fix High-Priority HTTP Context Issues (noctx)

   - [x] Modify manager.go to use http.NewRequestWithContext
   - [x] Add context parameter to functions making HTTP requests
   - [x] Propagate context from caller or use context.Background()

2. [x] Fix Unwrapped Errors (wrapcheck)

   - [x] Wrap filepath.Walk errors with context
   - [x] Wrap filepath.Rel errors
   - [x] Review and wrap other direct error returns

3. [x] Fix Long Lines in Go Code (lll)

   - [x] Break up long lines using string concatenation
   - [x] Use multi-line formats for long constructs

4. [ ] Refactor Complex Nested Blocks (nestif)

   - [ ] Identify functions with excessive nesting
   - [ ] Apply guard clauses to reduce nesting
   - [ ] Extract nested logic to helper functions

5. [ ] Reduce High Cognitive Complexity (gocognit)

   - [ ] Break main() into subcommand handlers
   - [ ] Split RestoreFromShared() into smaller functions
   - [ ] Refactor other high-complexity functions

6. [ ] Fix Low-Priority Issues

   - [ ] Split long CreateCustomTemplate function
   - [ ] Address line length issues in markdown files

7. [ ] Verify Fixes with Linter

   - [ ] Run golangci-lint after each set of changes
   - [ ] Ensure no regressions are introduced
   - [ ] Document any remaining issues that require exemptions

8. [x] Update Documentation
   - [x] Document coding standards applied
   - [x] Update PROGRESS.md with completed work

## Initial Request

Below is a step-by-step plan for resolving the linter issues you've identified, organized by priority. We'll cover the most important concerns first (HTTP context, unwrapped errors, long lines), then tackle the more complex tasks (function refactoring for nested blocks and high cognitive complexity), and finally address any cosmetic/low-priority tasks.

<br />

1. High-Priority Issues

A. HTTP Context Issues (noctx)

Symptoms:
• Linter complains that http.Get(...) calls in manager.go have no context.
• Golang best practices recommend using http.NewRequestWithContext so your requests can be canceled or time out properly.

Where:
• manager.go lines 443, 1146 (or near them).

Fix: 1. Create or accept a context.Context from upstream (e.g., pass it from your CLI or use context.Background() as a fallback). 2. Replace http.Get(url) with:

req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, http.NoBody)
if err != nil {
return RuleSource{}, fmt.Errorf("failed to create request: %w", err)
}
resp, err := http.DefaultClient.Do(req)
...

    3.	Handle cancellation or timeouts as needed. Typically, you might keep it simple with context.Background() if you don't have a more advanced context.

Example:

func handleGitHubBlob(ctx context.Context, cursorDir, ref string) (RuleSource, error) {
...
req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, http.NoBody)
if err != nil {
return RuleSource{}, fmt.Errorf("failed to create request: %w", err)
}
resp, err := http.DefaultClient.Do(req)
...
}

Then in your CLI, you can do:

ctx := context.Background()
ruleSource, err := handleGitHubBlob(ctx, cursorDir, ref)
...

<br />

B. Unwrapped Errors (wrapcheck)

Symptoms:
• Certain operations (e.g., filepath.Walk, filepath.Rel) return errors you might be returning directly, without wrapping.
• The linter says each error should be wrapped with context, e.g. fmt.Errorf("problem at step X: %w", err).

Where:
• manager.go lines 947, 955, etc.

Fix:

err = filepath.Walk(cursorDir, func(path string, info os.FileInfo, err error) error {
if err != nil {
return fmt.Errorf("error walking path %q: %w", path, err)
}
...
})

relPath, err := filepath.Rel(cursorDir, path)
if err != nil {
return fmt.Errorf("filepath.Rel failed for %s: %w", path, err)
}

Why: Wrapping ensures the final user sees the entire chain of error context.

<br />

C. Long Lines in Go Code (lll)

Symptoms:
• Some lines exceed a certain character limit (often 100 or 120 chars).

Where:
• Possibly in cmd/cursor-rules/main.go lines 41, 49 (just hypothetical line numbers from your note).

Fix:
• Simply break them up or use string concatenation. For example:

// Before:
fmt.Println("some extremely long line with more than 120 characters that the linter complains about")

// After (broken line):
fmt.Println("some extremely long line with more than 120 characters that the " +
"linter complains about, but is now split")

Or use backtick strings with newlines, or short variables to reduce the line length.

<br />

2. Medium-Priority Issues

A. Complex Nested Blocks (nestif)

Symptoms:
• Functions with many nested ifs, especially in manager.go and main.go, hamper readability.

Approach: 1. Identify large blocks of nested logic. 2. Extract them to helper functions with clear names. 3. Invert conditions or "guard clauses" to reduce nesting.

Example:
Instead of:

if A {
if B {
if C {
// deep logic
}
}
}

Use guard statements:

if !A {
return
}
if !B {
return
}
if !C {
return
}
// shallow logic

Or isolate in separate functions:

func doNestedLogic() error {
if err := stepOne(); err != nil {
return err
}
if err := stepTwo(); err != nil {
return err
}
...
}

Where:
• manager.go: 104, 791, 1252
• main.go: 103 (some monstrous 51-line or 51-nested block)

Fix:
• Break them up. For example, in main.go, the huge switch len(args) block for subcommands can be simplified by an approach that checks the subcommand early and calls a helper function.

<br />

B. High Cognitive Complexity (gocognit)

Symptoms:
• The linter sees functions with many branching paths and big nested structures.
• E.g., main() has a 138 complexity, RestoreFromShared() is 96, etc.

Approach: 1. Identify repeated logic. 2. Break the function into multiple smaller functions. 3. Potentially move some logic into separate packages or specialized "commands."

Example:
For main(), you could:
• Move subcommand handling into separate short functions.
• e.g. handleAddCmd(...), handleRemoveCmd(...), etc., each returning an error if something goes wrong.

<br />

3. Low-Priority Issues

A. Long Function in custom.go (funlen)

Symptoms:
• CreateCustomTemplate is 53 lines or more.

Fix:
• Consider splitting out sub-steps: 1. Gather user input for template metadata (gatherTemplateMetadata()) 2. Gather multi-line content (readTemplateContent()) 3. Construct the Template struct 4. Call CreateTemplate

Why:
• Smaller functions are easier to read and test.

<br />

B. Long Lines in Markdown (lll in .md Files)

Symptoms:
• Markdown files exceed line length limit.

Options: 1. Wrap lines at 80 or 100 characters. 2. Or disable this lint for .md if your team prefers not to wrap.

Example:
• If you want to keep wrapping, do:

This is some text in the README that we want to limit to 80 characters so that
linters do not complain about line length.

    •	Or, configure markdownlint to allow up to 120 or more.

<br />

4. Summary of the Approach
   1. Add HTTP contexts to your network calls to avoid noctx warnings:
      • Introduce a context.Context parameter in key functions (handleGitHubBlob, getHeadCommitForBranch) and pass context.Background() from main() at minimum.
   2. Wrap errors when returning them, especially from filepath calls. This addresses wrapcheck.
   3. Split long lines in both .go and .md files. For .go, you might break strings or rename variables. For .md, either wrap lines or disable the rule.
   4. Refactor nested blocks to reduce complexity:
      • For instance, break main() into smaller subcommand handler functions.
      • In manager.go, separate large blocks like UpgradeRule or RestoreFromShared into subfunctions that do each step.
   5. Reduce function length:
      • In custom.go, split CreateCustomTemplate into smaller pieces for reading input, reading content, building a struct, etc.
   6. Consider deferring some major changes (like big refactors) if they'd break existing logic. Possibly do them in a separate branch or version.

<br />

Example of Refactoring a Complex Function

Before

func main() {
// ~200 lines of subcommand parsing in one huge function
}

After

func main() {
// parse flags
// pick subcommand
switch cmd {
case "add":
if err := runAddCmd(); err != nil {
fmt.Fprintf(os.Stderr, "add command failed: %v\n", err)
os.Exit(1)
}
...
default:
showHelp()
}
}

func runAddCmd() error {
// everything related to "add" subcommand
return nil
}

This drastically reduces main() complexity.

<br />

Conclusion
• By focusing on context for HTTP requests and wrapping errors first, you'll satisfy your highest-priority linter checks (security/robustness).
• Then tackle line-length issues for immediate readability gains.
• For larger refactors (nesting, cognitive complexity), create a series of small, focused refactoring steps so you don't break existing functionality all at once.
• Adjust your code incrementally, verifying it still compiles and passes tests, to keep your codebase stable.

Following these recommendations should clean up the code and make future maintenance and expansions easier, while also satisfying your linter's more advanced checks. Good luck!
