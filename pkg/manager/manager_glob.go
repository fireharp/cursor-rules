package manager

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/fireharp/cursor-rules/pkg/templates"
	"github.com/gobwas/glob"
)

// handleGlobPattern handles references with glob patterns.
// This processes multiple rules that match the pattern.
func handleGlobPattern(ctx context.Context, cursorDir, ref string) error {
	// Parse the glob pattern
	username, pattern, ok := parseGlobPattern(ref)
	if !ok {
		return fmt.Errorf("invalid glob pattern: %s", ref)
	}

	// Compile the glob pattern
	g, err := compileGlob(pattern)
	if err != nil {
		return fmt.Errorf("invalid glob pattern: %w", err)
	}

	// Check if the pattern is a local path (contains ./ or / at start, or no username)
	if username == "" && (strings.HasPrefix(pattern, "./") || strings.HasPrefix(pattern, "/") || !strings.Contains(pattern, "/")) {
		// Handle local file glob pattern
		return handleLocalGlobPattern(ctx, cursorDir, pattern, g)
	}

	// Check if we have a username
	if username != "" {
		// Try to find matching rules in username/cursor-rules-collection
		return handleUsernameGlobPattern(ctx, cursorDir, username, pattern, g)
	} else {
		// Try to find matching templates
		return handleTemplateGlobPattern(cursorDir, pattern, g)
	}
}

// handleLocalGlobPattern handles glob patterns for local filesystem.
func handleLocalGlobPattern(ctx context.Context, cursorDir, pattern string, g glob.Glob) error {
	Debugf("Processing local glob pattern: %s\n", pattern)

	// Load lockfile once at the beginning
	lock, err := LoadLockFile(cursorDir)
	if err != nil {
		return fmt.Errorf("failed to load lockfile: %w", err)
	}

	// Resolve pattern to absolute pattern if it's relative
	var absolutePattern string
	if strings.HasPrefix(pattern, "/") {
		// Already absolute
		absolutePattern = pattern
	} else {
		// Get current directory and join with pattern
		cwd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get current working directory: %w", err)
		}

		// If pattern starts with ./, remove it
		if strings.HasPrefix(pattern, "./") {
			pattern = pattern[2:]
		}

		absolutePattern = filepath.Join(cwd, pattern)
	}

	Debugf("Using absolute glob pattern: %s\n", absolutePattern)

	// Use filepath.Glob to expand the pattern
	matchedFiles, err := filepath.Glob(absolutePattern)
	if err != nil {
		return fmt.Errorf("failed to expand glob pattern: %w", err)
	}

	Debugf("Found %d matching files\n", len(matchedFiles))

	if len(matchedFiles) == 0 {
		return fmt.Errorf("no files matched the pattern: %s", pattern)
	}

	// Track success/failure
	successCount := 0
	skippedCount := 0
	errorCount := 0

	// Collection of new rules
	newRules := []RuleSource{}

	// Process each matching file
	for _, filePath := range matchedFiles {
		// Skip directories
		info, err := os.Stat(filePath)
		if err != nil {
			fmt.Printf("Warning: Could not stat file %s: %v\n", filePath, err)
			errorCount++
			continue
		}

		if info.IsDir() {
			fmt.Printf("Skipping directory: %s\n", filePath)
			continue
		}

		// Skip non-mdc files
		if !strings.HasSuffix(filePath, ".mdc") {
			fmt.Printf("Skipping non-mdc file: %s\n", filePath)
			continue
		}

		fmt.Printf("Processing file: %s\n", filePath)

		// Process the file - create a helper function that returns the RuleSource without updating lockfile
		rule, err := processLocalFile(cursorDir, filePath, true)
		if err != nil {
			fmt.Printf("Warning: Could not process file %s: %v\n", filePath, err)
			errorCount++
			continue
		}

		// Check if rule is already installed
		if lock.IsInstalled(rule.Key) {
			// Find the existing rule
			var existingRule RuleSource
			for _, r := range lock.Rules {
				if r.Key == rule.Key {
					existingRule = r
					break
				}
			}

			// If content hash is available, compare that
			if rule.ContentSHA256 != "" && existingRule.ContentSHA256 != "" &&
				rule.ContentSHA256 != existingRule.ContentSHA256 {
				fmt.Printf("Rule '%s' is already installed but has different content.\n", rule.Key)
				fmt.Printf("To update, use: cursor-rules upgrade %s\n", rule.Key)
			} else {
				fmt.Printf("Rule '%s' is already installed and up-to-date.\n", rule.Key)
			}

			skippedCount++
			continue
		}

		// Check if we've already added this rule in the current operation
		alreadyAdded := false
		for _, r := range newRules {
			if r.Key == rule.Key {
				alreadyAdded = true
				break
			}
		}

		if alreadyAdded {
			fmt.Printf("Rule already added in this operation: %s\n", rule.Key)
			continue
		}

		// Add rule to our collection
		newRules = append(newRules, rule)
		successCount++
	}

	// Update lockfile with all new rules
	if len(newRules) > 0 {
		lock.Rules = append(lock.Rules, newRules...)

		// For backwards compatibility
		for _, rule := range newRules {
			lock.Installed = append(lock.Installed, rule.Key)
		}

		err = lock.Save(cursorDir)
		if err != nil {
			return fmt.Errorf("failed to update lockfile: %w", err)
		}
	}

	// Report results
	if successCount == 0 && skippedCount == 0 {
		return fmt.Errorf("no valid rules found for pattern: %s", pattern)
	}

	fmt.Printf("Added %d rules matching pattern %s (skipped: %d, errors: %d)\n",
		successCount, pattern, skippedCount, errorCount)
	return nil
}

// processLocalFile has been moved to manager_local_handlers.go - this improves organization
// by keeping all local file handling functions in the same file.

// handleUsernameGlobPattern handles glob patterns with a username.
// This looks for matching rules in the username's cursor-rules-collection repo.
func handleUsernameGlobPattern(ctx context.Context, cursorDir, username, pattern string, g glob.Glob) error {
	// Try to get a list of files from the repo
	owner := username
	repo := "cursor-rules-collection"
	branch := "main" // Default to main branch

	// Get list of files from GitHub
	files, err := listGitHubRepoFiles(ctx, owner, repo, branch, pattern)
	if err != nil {
		// If we can't list, tell the user and suggest alternatives
		fmt.Printf("Could not list files matching pattern: %v\n", err)
		fmt.Println("You may need to add rules individually instead of using glob patterns.")
		return fmt.Errorf("failed to list files matching pattern: %w", err)
	}

	// Track success/failure
	successCount := 0
	errorCount := 0

	// Process each matching file
	for _, file := range files {
		// Check if it matches our pattern
		if !matchGlob(g, file) {
			continue
		}

		// Only process .mdc files
		if !strings.HasSuffix(file, ".mdc") {
			continue
		}

		// Construct a new reference without the glob
		fileRef := fmt.Sprintf("%s/%s", username, file)

		// Directly construct GitHub URL to avoid recursive call to AddRuleByReference
		githubURL := fmt.Sprintf("https://github.com/%s/cursor-rules-collection/blob/main/%s",
			username, file)

		rule, err := handleGitHubBlob(ctx, cursorDir, githubURL)
		if err != nil {
			fmt.Printf("Warning: Could not add rule %s: %v\n", fileRef, err)
			errorCount++
		} else {
			// Found in the cursor-rules-collection repo
			rule.SourceType = SourceTypeGitHubShorthand
			rule.Reference = fileRef // Store the original reference

			// Update lockfile with the new rule
			lock, err := LoadLockFile(cursorDir)
			if err != nil {
				fmt.Printf("Warning: Could not load lockfile for %s: %v\n", fileRef, err)
				errorCount++
				continue
			}

			if lock.IsInstalled(rule.Key) {
				fmt.Printf("Rule already installed: %s\n", rule.Key)
				continue
			}

			lock.Rules = append(lock.Rules, rule)
			// For backwards compatibility
			lock.Installed = append(lock.Installed, rule.Key)

			err = lock.Save(cursorDir)
			if err != nil {
				fmt.Printf("Warning: Could not update lockfile for %s: %v\n", fileRef, err)
				errorCount++
				continue
			}

			successCount++
		}
	}

	// Report results
	if successCount == 0 {
		return fmt.Errorf("no matching rules found for pattern: %s", pattern)
	}

	fmt.Printf("Added %d rules matching pattern %s (errors: %d)\n", successCount, pattern, errorCount)
	return nil
}

// handleTemplateGlobPattern handles glob patterns for built-in templates.
func handleTemplateGlobPattern(cursorDir, pattern string, g glob.Glob) error {
	// Get list of available templates
	allTemplates, err := templates.ListTemplates()
	if err != nil {
		return fmt.Errorf("failed to list templates: %w", err)
	}

	// Track success/failure
	successCount := 0
	errorCount := 0

	// Process each matching template
	for _, tmpl := range allTemplates {
		// Check if the template name matches our pattern
		if !matchGlob(g, tmpl.Name) {
			continue
		}

		// Try to add this template directly instead of calling AddRule
		content := tmpl.Content
		targetPath := filepath.Join(cursorDir, tmpl.Name+".mdc")

		// Write to .cursor/rules/{ruleName}.mdc
		err := os.WriteFile(targetPath, []byte(content), 0o644)
		if err != nil {
			fmt.Printf("Warning: Could not write template %s: %v\n", tmpl.Name, err)
			errorCount++
			continue
		}

		// Update lockfile
		lock, err := LoadLockFile(cursorDir)
		if err != nil {
			fmt.Printf("Warning: Could not load lockfile for %s: %v\n", tmpl.Name, err)
			errorCount++
			continue
		}

		if lock.IsInstalled(tmpl.Name) {
			fmt.Printf("Template already installed: %s\n", tmpl.Name)
			continue
		}

		rule := RuleSource{
			Key:        tmpl.Name,
			SourceType: SourceTypeBuiltIn,
			Reference:  tmpl.Name,
			Category:   tmpl.Category,
			LocalFiles: []string{targetPath},
		}

		lock.Rules = append(lock.Rules, rule)
		// For backwards compatibility
		lock.Installed = append(lock.Installed, tmpl.Name)

		err = lock.Save(cursorDir)
		if err != nil {
			fmt.Printf("Warning: Could not update lockfile for %s: %v\n", tmpl.Name, err)
			errorCount++
			continue
		}

		successCount++
	}

	// Report results
	if successCount == 0 {
		return fmt.Errorf("no matching templates found for pattern: %s", pattern)
	}

	fmt.Printf("Added %d templates matching pattern %s (errors: %d)\n", successCount, pattern, errorCount)
	return nil
}
