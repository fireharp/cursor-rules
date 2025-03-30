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

// Function type definitions for dependency injection and testing
type AddRuleByReferenceFunc func(cursorDir, ref string) error
type AddRuleFunc func(cursorDir, category, ruleKey string) error

// Default implementations
var AddRuleByReferenceFn AddRuleByReferenceFunc = addRuleByReferenceImpl
var AddRuleFn AddRuleFunc = addRuleImpl

// AddRule installs a built-in rule from a template.
func AddRule(cursorDir, category, ruleKey string) error {
	return AddRuleFn(cursorDir, category, ruleKey)
}

// addRuleImpl is the implementation of AddRule.
func addRuleImpl(cursorDir, category, ruleKey string) error {
	// Skip if already installed
	lock, err := LoadLockFile(cursorDir)
	if err != nil {
		return fmt.Errorf("failed to load lockfile: %w", err)
	}

	if lock.IsInstalled(ruleKey) {
		return fmt.Errorf("rule already installed: %s", ruleKey)
	}

	// Get template content
	content, err := templates.GetTemplate(category, ruleKey)
	if err != nil {
		return fmt.Errorf("template not found: %w", err)
	}

	// Write to .cursor/rules/{ruleKey}.mdc
	targetPath := filepath.Join(cursorDir, ruleKey+".mdc")

	// Ensure parent directories exist for hierarchical keys
	if err := ensureRuleDirectory(cursorDir, ruleKey); err != nil {
		return fmt.Errorf("failed preparing directory for rule '%s': %w", ruleKey, err)
	}

	err = os.WriteFile(targetPath, []byte(content), 0o644)
	if err != nil {
		return fmt.Errorf("failed to write rule file: %w", err)
	}

	// Update lockfile
	rule := RuleSource{
		Key:        ruleKey,
		SourceType: SourceTypeBuiltIn,
		Reference:  ruleKey,
		Category:   category,
		LocalFiles: []string{targetPath},
	}

	lock.Rules = append(lock.Rules, rule)
	// For backwards compatibility
	lock.Installed = append(lock.Installed, ruleKey)

	err = lock.Save(cursorDir)
	if err != nil {
		return fmt.Errorf("failed to update lockfile: %w", err)
	}

	return nil
}

// AddRuleByReference installs a rule from a file or URL reference.
func AddRuleByReference(cursorDir, ref string) error {
	return AddRuleByReferenceFn(cursorDir, ref)
}

// addRuleByReferenceImpl is the implementation of AddRuleByReference.
func addRuleByReferenceImpl(cursorDir, ref string) error {
	// Skip if already installed - we need to detect the key first
	var rule RuleSource
	var err error

	// Handle different reference types
	if isGlobPattern(ref) {
		// Handle glob pattern
		return handleGlobPattern(context.Background(), cursorDir, ref)
	} else if isGitHubBlobURL(ref) {
		rule, err = handleGitHubBlob(context.Background(), cursorDir, ref)
	} else if isGitHubTreeURL(ref) {
		rule, err = handleGitHubDir(cursorDir, ref)
	} else if isUsernameRuleWithSha(ref) {
		// Handle username/rule:sha format
		rule, err = handleUsernameRuleWithSha(context.Background(), cursorDir, ref)
	} else if isUsernameRuleWithTag(ref) {
		// Handle username/rule@tag format
		rule, err = handleUsernameRuleWithTag(context.Background(), cursorDir, ref)
	} else if isUsernameRule(ref) {
		// Handle username/rule format
		rule, err = handleUsernameRule(context.Background(), cursorDir, ref)
	} else if isUsernamePathRule(ref) {
		// Handle username/path/rule format
		rule, err = handleUsernamePathRule(context.Background(), cursorDir, ref)
	} else if isAbsolutePath(ref) {
		rule, err = handleLocalFile(cursorDir, ref, true)
	} else if isRelativePath(ref) {
		rule, err = handleLocalFile(cursorDir, ref, false)
	} else {
		// Check if there's a default username and this is a simple rule name
		defaultUsername := getDefaultUsername()
		if defaultUsername != "" {
			// Use the default username for resolution
			defaultRef := defaultUsername + "/" + ref
			// Directly construct GitHub URL to avoid recursive call
			githubURL := fmt.Sprintf("https://github.com/%s/cursor-rules-collection/blob/main/%s.mdc",
				defaultUsername, ref)
			rule, err = handleGitHubBlob(context.Background(), cursorDir, githubURL)

			if err == nil {
				// Found in the cursor-rules-collection repo
				rule.SourceType = SourceTypeGitHubShorthand
				rule.Reference = defaultRef // Store the resolved reference
				goto updateLockfile
			}

			// If not found, try with potential paths (could be nested)
			githubURL = fmt.Sprintf("https://github.com/%s/cursor-rules-collection/blob/main/%s/%s.mdc",
				defaultUsername, ref, ref)
			rule, err = handleGitHubBlob(context.Background(), cursorDir, githubURL)

			if err == nil {
				// Found in the cursor-rules-collection repo in a subdirectory
				rule.SourceType = SourceTypeGitHubShorthand
				rule.Reference = defaultRef // Store the resolved reference
				goto updateLockfile
			}
		}

		// Use the built-in template approach (search in templates)
		tmpl, err := templates.FindTemplateByName(ref)
		if err == nil && tmpl.Category != "" {
			return AddRule(cursorDir, tmpl.Category, ref)
		}
		return fmt.Errorf("unsupported reference format or rule not found: %s", ref)
	}

	if err != nil {
		return fmt.Errorf("failed to process reference: %w", err)
	}

updateLockfile:
	// Update lockfile with the new rule
	lock, err := LoadLockFile(cursorDir)
	if err != nil {
		return fmt.Errorf("failed to load lockfile: %w", err)
	}

	if lock.IsInstalled(rule.Key) {
		return fmt.Errorf("rule already installed: %s", rule.Key)
	}

	lock.Rules = append(lock.Rules, rule)
	// For backwards compatibility
	lock.Installed = append(lock.Installed, rule.Key)

	err = lock.Save(cursorDir)
	if err != nil {
		return fmt.Errorf("failed to update lockfile: %w", err)
	}

	return nil
}

// handleUsernameRule handles a reference in the username/rule format.
// This will look for the rule in the username/cursor-rules-collection repo.
func handleUsernameRule(ctx context.Context, cursorDir, ref string) (RuleSource, error) {
	username, ruleName, ok := parseUsernameRule(ref)
	if !ok {
		return RuleSource{}, fmt.Errorf("invalid username/rule format: %s", ref)
	}

	fmt.Printf("Debug: handleUsernameRule: username='%s', ruleName='%s'\n", username, ruleName)

	// First, try to find it in username/cursor-rules-collection repo
	githubURL := fmt.Sprintf("https://github.com/%s/cursor-rules-collection/blob/main/%s.mdc", username, ruleName)
	fmt.Printf("Debug: handleUsernameRule: trying URL: %s\n", githubURL)

	rule, err := handleGitHubBlob(ctx, cursorDir, githubURL)

	if err == nil {
		// Found in the cursor-rules-collection repo
		rule.SourceType = SourceTypeGitHubShorthand
		rule.Reference = ref // Store the original reference
		fmt.Printf("Debug: handleUsernameRule: Found rule at primary URL\n")
		return rule, nil
	}

	fmt.Printf("Debug: handleUsernameRule: Primary URL failed with error: %v\n", err)

	// If not found, try with potential paths (could be nested)
	githubURL = fmt.Sprintf("https://github.com/%s/cursor-rules-collection/blob/main/%s/%s.mdc", username, ruleName, ruleName)
	fmt.Printf("Debug: handleUsernameRule: trying fallback URL: %s\n", githubURL)

	rule, err = handleGitHubBlob(ctx, cursorDir, githubURL)

	if err == nil {
		// Found in the cursor-rules-collection repo in a subdirectory
		rule.SourceType = SourceTypeGitHubShorthand
		rule.Reference = ref // Store the original reference
		fmt.Printf("Debug: handleUsernameRule: Found rule at fallback URL\n")
		return rule, nil
	}

	fmt.Printf("Debug: handleUsernameRule: Fallback URL failed with error: %v\n", err)

	return RuleSource{}, fmt.Errorf("rule not found in username/cursor-rules-collection: %s", ref)
}

// handleUsernamePathRule handles a reference in the username/path/rule format.
// This will first try in username/cursor-rules-collection, then fallback to username/repo.
func handleUsernamePathRule(ctx context.Context, cursorDir, ref string) (RuleSource, error) {
	username, pathParts, ok := parseUsernamePathRule(ref)
	if !ok || len(pathParts) < 1 {
		return RuleSource{}, fmt.Errorf("invalid username/path/rule format: %s", ref)
	}

	fmt.Printf("Debug: handleUsernamePathRule: username='%s', pathParts=%v\n", username, pathParts)

	// First, try to interpret it as username/path/to/rule in cursor-rules-collection
	if len(pathParts) >= 1 {
		// Extract the last part as the rule name and construct the path
		ruleName := pathParts[len(pathParts)-1]
		pathToRule := strings.Join(pathParts[:len(pathParts)-1], "/")

		// If the rule already has .mdc extension, don't add it again
		ruleFile := ruleName
		if !strings.HasSuffix(ruleName, ".mdc") {
			ruleFile = ruleName + ".mdc"
		}

		var githubURL string
		if pathToRule == "" {
			githubURL = fmt.Sprintf("https://github.com/%s/cursor-rules-collection/blob/main/%s",
				username, ruleFile)
		} else {
			githubURL = fmt.Sprintf("https://github.com/%s/cursor-rules-collection/blob/main/%s/%s",
				username, pathToRule, ruleFile)
		}

		fmt.Printf("Debug: handleUsernamePathRule: trying URL: %s\n", githubURL)

		rule, err := handleGitHubBlob(ctx, cursorDir, githubURL)
		if err == nil {
			// Found in the cursor-rules-collection repo
			rule.SourceType = SourceTypeGitHubShorthand
			rule.Reference = ref // Store the original reference
			fmt.Printf("Debug: handleUsernamePathRule: Found rule in cursor-rules-collection\n")
			return rule, nil
		}

		fmt.Printf("Debug: handleUsernamePathRule: cursor-rules-collection URL failed with error: %v\n", err)
	}

	// As a fallback, attempt to interpret it as username/repo/path/to/rule.mdc
	if len(pathParts) >= 2 {
		repoName := pathParts[0]
		remainingPath := strings.Join(pathParts[1:], "/")

		// If the last part already has .mdc extension, don't add it again
		if !strings.HasSuffix(remainingPath, ".mdc") {
			remainingPath += ".mdc"
		}

		githubURL := fmt.Sprintf("https://github.com/%s/%s/blob/main/%s",
			username, repoName, remainingPath)

		fmt.Printf("Debug: handleUsernamePathRule: trying fallback URL (any repo): %s\n", githubURL)

		rule, err := handleGitHubBlob(ctx, cursorDir, githubURL)
		if err == nil {
			// Found in the other repo
			rule.SourceType = SourceTypeGitHubRepoPath
			rule.Reference = ref // Store the original reference
			fmt.Printf("Debug: handleUsernamePathRule: Found rule in other repo\n")
			return rule, nil
		}

		fmt.Printf("Debug: handleUsernamePathRule: fallback URL failed with error: %v\n", err)
	}

	return RuleSource{}, fmt.Errorf("rule not found in any repo: %s", ref)
}

// handleUsernameRuleWithSha handles a reference in the username/rule:sha format.
func handleUsernameRuleWithSha(ctx context.Context, cursorDir, ref string) (RuleSource, error) {
	username, ruleName, sha, ok := parseUsernameRuleWithSha(ref)
	if !ok {
		return RuleSource{}, fmt.Errorf("invalid username/rule:sha format: %s", ref)
	}

	// Build GitHub URL with specific commit
	githubURL := fmt.Sprintf("https://github.com/%s/cursor-rules-collection/blob/%s/%s.mdc",
		username, sha, ruleName)

	rule, err := handleGitHubBlob(ctx, cursorDir, githubURL)
	if err == nil {
		// Found in the cursor-rules-collection repo at specific commit
		rule.SourceType = SourceTypeGitHubShorthand
		rule.Reference = ref // Store the original reference
		rule.GitRef = "commit=" + sha
		return rule, nil
	}

	// Try in a subdirectory as well
	githubURL = fmt.Sprintf("https://github.com/%s/cursor-rules-collection/blob/%s/%s/%s.mdc",
		username, sha, ruleName, ruleName)

	rule, err = handleGitHubBlob(ctx, cursorDir, githubURL)
	if err == nil {
		// Found in the cursor-rules-collection repo in a subdirectory at specific commit
		rule.SourceType = SourceTypeGitHubShorthand
		rule.Reference = ref // Store the original reference
		rule.GitRef = "commit=" + sha
		return rule, nil
	}

	return RuleSource{}, fmt.Errorf("rule not found in username/cursor-rules-collection at commit %s: %s", sha, ref)
}

// handleUsernameRuleWithTag handles a reference in the username/rule@tag format.
func handleUsernameRuleWithTag(ctx context.Context, cursorDir, ref string) (RuleSource, error) {
	username, ruleName, tag, ok := parseUsernameRuleWithTag(ref)
	if !ok {
		return RuleSource{}, fmt.Errorf("invalid username/rule@tag format: %s", ref)
	}

	// Build GitHub URL with specific tag
	githubURL := fmt.Sprintf("https://github.com/%s/cursor-rules-collection/blob/%s/%s.mdc",
		username, tag, ruleName)

	rule, err := handleGitHubBlob(ctx, cursorDir, githubURL)
	if err == nil {
		// Found in the cursor-rules-collection repo at specific tag
		rule.SourceType = SourceTypeGitHubShorthand
		rule.Reference = ref // Store the original reference
		rule.GitRef = "tag=" + tag
		return rule, nil
	}

	// Try in a subdirectory as well
	githubURL = fmt.Sprintf("https://github.com/%s/cursor-rules-collection/blob/%s/%s/%s.mdc",
		username, tag, ruleName, ruleName)

	rule, err = handleGitHubBlob(ctx, cursorDir, githubURL)
	if err == nil {
		// Found in the cursor-rules-collection repo in a subdirectory at specific tag
		rule.SourceType = SourceTypeGitHubShorthand
		rule.Reference = ref // Store the original reference
		rule.GitRef = "tag=" + tag
		return rule, nil
	}

	return RuleSource{}, fmt.Errorf("rule not found in username/cursor-rules-collection at tag %s: %s", tag, ref)
}

// RemoveRule uninstalls a rule and removes its files.
func RemoveRule(cursorDir string, ruleKey string) error {
	// Load the lockfile
	lock, err := LoadLockFile(cursorDir)
	if err != nil {
		return fmt.Errorf("failed to load lockfile: %w", err)
	}

	// Find the rule
	var ruleIndex = -1
	var rule RuleSource

	for i, r := range lock.Rules {
		if r.Key == ruleKey {
			ruleIndex = i
			rule = r
			break
		}
	}

	if ruleIndex == -1 {
		// Legacy support - look for simple rule keys in the Installed list
		for i, key := range lock.Installed {
			if key == ruleKey {
				// Old style, so we need to create a fake RuleSource
				ruleIndex = i
				rule = RuleSource{
					Key:        ruleKey,
					SourceType: SourceTypeBuiltIn,
					Reference:  ruleKey,
					LocalFiles: []string{filepath.Join(cursorDir, ruleKey+".mdc")},
				}
				break
			}
		}
	}

	if ruleIndex == -1 {
		return fmt.Errorf("rule not found: %s", ruleKey)
	}

	// Remove the rule files
	for _, file := range rule.LocalFiles {
		// Ensure the path is absolute for files that might be relative
		filePath := file
		if !filepath.IsAbs(file) {
			filePath = filepath.Join(cursorDir, file)
		}

		// Remove the file if it exists
		if fileExists(filePath) {
			err = os.Remove(filePath)
			if err != nil {
				return fmt.Errorf("failed to remove rule file %s: %w", filePath, err)
			}
		}
	}

	// Remove from the lockfile
	if ruleIndex < len(lock.Rules) {
		lock.Rules = append(lock.Rules[:ruleIndex], lock.Rules[ruleIndex+1:]...)
	}

	// For backwards compatibility, remove from the Installed list too
	for i, key := range lock.Installed {
		if key == ruleKey {
			lock.Installed = append(lock.Installed[:i], lock.Installed[i+1:]...)
			break
		}
	}

	// Save the lockfile
	err = lock.Save(cursorDir)
	if err != nil {
		return fmt.Errorf("failed to update lockfile: %w", err)
	}

	return nil
}

// ListInstalledRules returns the list of installed rules from the lockfile.
func ListInstalledRules(cursorDir string) ([]string, error) {
	lock, err := LoadLockFile(cursorDir)
	if err != nil {
		return nil, err
	}

	// Prefer enhanced structure if available
	if len(lock.Rules) > 0 {
		var rules []string
		for _, rule := range lock.Rules {
			rules = append(rules, rule.Key)
		}
		return rules, nil
	}

	// Fall back to legacy format
	return lock.Installed, nil
}

// GetInstalledRules returns the full RuleSource objects for installed rules.
func GetInstalledRules(cursorDir string) ([]RuleSource, error) {
	lock, err := LoadLockFile(cursorDir)
	if err != nil {
		return nil, fmt.Errorf("failed to load lockfile: %w", err)
	}
	return lock.Rules, nil
}

// SyncLocalRules scans the cursor/rules directory for .mdc files and updates the lockfile accordingly.
func SyncLocalRules(cursorDir string) error {
	// Load the lockfile
	lock, err := LoadLockFile(cursorDir)
	if err != nil {
		return fmt.Errorf("failed to load lockfile: %w", err)
	}

	// Create a map of installed rules for quick lookup
	installedRules := make(map[string]bool)
	for _, rule := range lock.Rules {
		installedRules[rule.Key] = true
	}

	// Scan the directory for .mdc files
	foundRules := make(map[string]bool)
	err = filepath.Walk(cursorDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Only process .mdc files
		if !info.IsDir() && strings.HasSuffix(path, ".mdc") {
			relativePath, err := filepath.Rel(cursorDir, path)
			if err != nil {
				return fmt.Errorf("failed to get relative path: %w", err)
			}

			// Extract the rule key from the filename
			key := strings.TrimSuffix(relativePath, ".mdc")
			foundRules[key] = true

			// Add to lockfile if not already there
			if !installedRules[key] {
				rule := RuleSource{
					Key:        key,
					SourceType: SourceTypeBuiltIn, // Default to built-in for simplicity
					Reference:  key,
					LocalFiles: []string{path},
				}
				lock.Rules = append(lock.Rules, rule)
				// For backwards compatibility
				lock.Installed = append(lock.Installed, key)
			}
		}

		return nil
	})

	if err != nil {
		return fmt.Errorf("failed to scan directory: %w", err)
	}

	// Check for rules in the lockfile that don't exist on disk
	var updatedRules []RuleSource
	for _, rule := range lock.Rules {
		if foundRules[rule.Key] {
			updatedRules = append(updatedRules, rule)
		}
	}

	// Update the lockfile with the new list
	if len(updatedRules) != len(lock.Rules) {
		lock.Rules = updatedRules

		// For backwards compatibility, update the Installed list too
		var updatedInstalled []string
		for _, key := range lock.Installed {
			if foundRules[key] {
				updatedInstalled = append(updatedInstalled, key)
			}
		}
		lock.Installed = updatedInstalled

		// Save the lockfile
		err = lock.Save(cursorDir)
		if err != nil {
			return fmt.Errorf("failed to update lockfile: %w", err)
		}
	}

	return nil
}

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
	fmt.Printf("Debug: Processing local glob pattern: %s\n", pattern)

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

	fmt.Printf("Debug: Using absolute glob pattern: %s\n", absolutePattern)

	// Use filepath.Glob to expand the pattern
	matchedFiles, err := filepath.Glob(absolutePattern)
	if err != nil {
		return fmt.Errorf("failed to expand glob pattern: %w", err)
	}

	fmt.Printf("Debug: Found %d matching files\n", len(matchedFiles))

	if len(matchedFiles) == 0 {
		return fmt.Errorf("no files matched the pattern: %s", pattern)
	}

	// Track success/failure
	successCount := 0
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
			fmt.Printf("Rule already installed: %s\n", rule.Key)
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
	if successCount == 0 {
		return fmt.Errorf("no valid rules found for pattern: %s", pattern)
	}

	fmt.Printf("Added %d rules matching pattern %s (errors: %d)\n", successCount, pattern, errorCount)
	return nil
}

// processLocalFile processes a local file and returns a RuleSource without updating the lockfile.
// This is a helper function extracted from handleLocalFile to be used with glob patterns.
func processLocalFile(cursorDir, filePath string, isAbs bool) (RuleSource, error) {
	// 1. Validate path and ensure it's readable
	var fullPath string
	if isAbs {
		fullPath = filePath
	} else {
		var err error
		fullPath, err = filepath.Abs(filePath)
		if err != nil {
			return RuleSource{}, fmt.Errorf("failed to resolve path %s: %w", filePath, err)
		}
	}

	// Check if the file exists and is readable
	info, err := os.Stat(fullPath)
	if err != nil {
		return RuleSource{}, fmt.Errorf("failed to access file %s: %w", fullPath, err)
	}

	if info.IsDir() {
		return RuleSource{}, fmt.Errorf("%s is a directory, not a file", fullPath)
	}

	// 2. Read the file
	data, err := os.ReadFile(fullPath)
	if err != nil {
		return RuleSource{}, fmt.Errorf("failed to read file %s: %w", fullPath, err)
	}

	// 3. Generate rule key and determine destination filename
	ruleKey := generateRuleKey(filePath)
	destFilename := ruleKey + ".mdc"
	destPath := filepath.Join(cursorDir, destFilename)

	// Ensure parent directories exist for hierarchical keys
	if err := ensureRuleDirectory(cursorDir, ruleKey); err != nil {
		return RuleSource{}, fmt.Errorf("failed preparing directory for rule '%s': %w", ruleKey, err)
	}

	// 4. Write to .cursor/rules
	err = os.WriteFile(destPath, data, 0o600)
	if err != nil {
		return RuleSource{}, fmt.Errorf("failed to write to %s: %w", destPath, err)
	}

	// 5. Create and return RuleSource
	sourceType := SourceTypeLocalAbs
	if !isAbs {
		sourceType = SourceTypeLocalRel
		// Use relative path if possible for portability
		// Get current working directory
		cwd, err := os.Getwd()
		if err == nil {
			rel, err := filepath.Rel(cwd, fullPath)
			if err == nil {
				fullPath = rel
			}
		}
	}

	// Create the rule source
	result := RuleSource{
		Key:        ruleKey,
		SourceType: sourceType,
		Reference:  filePath,
		LocalFiles: []string{destFilename},
		// Calculate and store content hash for future modification checks
		ContentSHA256: calculateSHA256(data),
		// Store the original glob pattern
		GlobPattern: filePath,
	}

	return result, nil
}

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
