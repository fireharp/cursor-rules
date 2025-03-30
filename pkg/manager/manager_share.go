package manager

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// ShareableRule represents a rule that can be shared.
type ShareableRule struct {
	// The short "rule key" used in local .cursor/rules filenames
	Key string `json:"key"`

	// A short, machine-readable type: "built-in", "local-abs", "local-rel", "github-file", "github-dir"
	SourceType SourceType `json:"sourceType"`

	// The raw string that the user passed in (sanitized if necessary)
	Reference string `json:"reference"`

	// Category for built-in rules
	Category string `json:"category,omitempty"`

	// For Git sources, store explicit commit/tag/branch
	// e.g. "commit=0609329", "branch=main", etc.
	GitRef string `json:"gitRef,omitempty"`

	// Flag to indicate if this rule can't be easily shared
	Unshareable bool `json:"unshareable,omitempty"`

	// Optional embedded .mdc content for local rules
	Content string `json:"content,omitempty"`

	// Name of the original file for embedded content
	Filename string `json:"filename,omitempty"`
}

// ShareableLock represents a shareable version of the lockfile.
type ShareableLock struct {
	// Version of the shareable file format
	FormatVersion int `json:"formatVersion"`

	// Rules that can be shared
	Rules []ShareableRule `json:"rules"`
}

// Shareable is a helper struct for generating human-readable output.
type Shareable struct {
	GitHub      []string            `json:"github"`
	BuiltIn     map[string][]string `json:"builtIn"`
	Unshareable []string            `json:"unshareable"`
	Embedded    map[string]string   `json:"embedded"`
}

// ErrSkipRule is returned when a rule should be skipped during restore.
var ErrSkipRule = errors.New("rule skipped")

// ShareRules exports installed rules to a shareable JSON file.
func ShareRules(cursorDir string, shareFilePath string, embedContent bool) error {
	// Load the lockfile
	lock, err := LoadLockFile(cursorDir)
	if err != nil {
		return fmt.Errorf("failed to load lockfile: %w", err)
	}

	// Create shareable lock structure
	shareable := ShareableLock{
		FormatVersion: 1,
		Rules:         make([]ShareableRule, 0, len(lock.Rules)),
	}

	// Helper variables for human-readable output
	summary := Shareable{
		GitHub:      []string{},
		BuiltIn:     make(map[string][]string),
		Unshareable: []string{},
		Embedded:    make(map[string]string),
	}

	// Convert each rule to a shareable rule
	for _, rule := range lock.Rules {
		shareableRule := ShareableRule{
			Key:        rule.Key,
			SourceType: rule.SourceType,
			Reference:  rule.Reference,
			Category:   rule.Category,
			GitRef:     rule.GitRef,
		}

		switch rule.SourceType {
		case SourceTypeBuiltIn:
			// Built-in rules are easy to share
			if _, ok := summary.BuiltIn[rule.Category]; !ok {
				summary.BuiltIn[rule.Category] = []string{}
			}
			summary.BuiltIn[rule.Category] = append(summary.BuiltIn[rule.Category], rule.Key)

		case SourceTypeGitHubFile:
			// GitHub rules are easy to share
			summary.GitHub = append(summary.GitHub, rule.Reference)

		case SourceTypeLocalAbs, SourceTypeLocalRel:
			// Local files might need embedding
			if embedContent && len(rule.LocalFiles) > 0 {
				// Ensure we're using the full path
				filePath := rule.LocalFiles[0]
				if !filepath.IsAbs(filePath) {
					filePath = filepath.Join(cursorDir, filePath)
				}

				data, err := os.ReadFile(filePath)
				if err != nil {
					return fmt.Errorf("failed to read rule content: %w", err)
				}

				shareableRule.Content = string(data)
				shareableRule.Filename = filepath.Base(filePath)
				summary.Embedded[rule.Key] = filePath
			} else {
				// Mark as unshareable if we're not embedding
				shareableRule.Unshareable = true
				summary.Unshareable = append(summary.Unshareable, rule.Key)
			}

		default:
			// Unknown types are marked as unshareable
			shareableRule.Unshareable = true
			summary.Unshareable = append(summary.Unshareable, rule.Key)
		}

		shareable.Rules = append(shareable.Rules, shareableRule)
	}

	// Write the file
	data, err := json.MarshalIndent(shareable, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to serialize rules: %w", err)
	}

	err = os.WriteFile(shareFilePath, data, 0o644)
	if err != nil {
		return fmt.Errorf("failed to write share file: %w", err)
	}

	// Print summary
	fmt.Printf("Shared %d rules to %s\n", len(shareable.Rules), shareFilePath)
	if len(summary.GitHub) > 0 {
		fmt.Printf("- %d GitHub rules\n", len(summary.GitHub))
	}

	for category, rules := range summary.BuiltIn {
		fmt.Printf("- %d built-in %s rules\n", len(rules), category)
	}

	if len(summary.Embedded) > 0 {
		fmt.Printf("- %d local rules with embedded content\n", len(summary.Embedded))
	}

	if len(summary.Unshareable) > 0 {
		fmt.Printf("- %d unshareable rules (local files without embedded content)\n", len(summary.Unshareable))
	}

	return nil
}

// findAvailableKey finds an available key name for a rule when there are conflicts.
func findAvailableKey(baseKey string, existingRules map[string]bool) string {
	if !existingRules[baseKey] {
		return baseKey
	}

	// Try with a suffix (baseKey-1, baseKey-2, etc.)
	for i := 1; i < 100; i++ {
		newKey := fmt.Sprintf("%s-%d", baseKey, i)
		if !existingRules[newKey] {
			return newKey
		}
	}

	// Unlikely to get here, but just in case
	return fmt.Sprintf("%s-%d", baseKey, os.Getpid())
}

// loadShareableData loads data from a file or URL.
func loadShareableData(ctx context.Context, sharePath string) ([]byte, error) {
	// Check if it's a URL
	if strings.HasPrefix(sharePath, "http://") || strings.HasPrefix(sharePath, "https://") {
		return loadShareableFromURL(ctx, sharePath)
	}

	// Read from local file
	data, err := os.ReadFile(sharePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read share file: %w", err)
	}

	return data, nil
}

// loadShareableFromURL loads data from a URL.
func loadShareableFromURL(ctx context.Context, url string) ([]byte, error) {
	// Create request with context for cancellation
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Send request
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to download shareable file: %w", err)
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP error: %s", resp.Status)
	}

	// Read body
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	return data, nil
}

// parseShareableLock parses a ShareableLock from JSON data.
func parseShareableLock(data []byte) (*ShareableLock, error) {
	var lock ShareableLock
	err := json.Unmarshal(data, &lock)
	if err != nil {
		return nil, fmt.Errorf("failed to parse shareable file: %w", err)
	}

	// Validate
	if lock.FormatVersion != 1 {
		return nil, fmt.Errorf("unsupported shareable file format version: %d", lock.FormatVersion)
	}

	return &lock, nil
}

// buildExistingRuleSet builds a map of existing rule keys.
func buildExistingRuleSet(lock *LockFile) map[string]bool {
	existingRules := make(map[string]bool)
	for _, rule := range lock.Rules {
		existingRules[rule.Key] = true
	}
	return existingRules
}

// resolveConflict handles rule name conflicts during restore.
func resolveConflict(key, autoResolve string) (string, string, error) {
	if autoResolve == "" {
		// Prompt the user
		action := promptForConflictResolution(key)
		if action == ActionSkip {
			return "", "", ErrSkipRule
		}
		return key, action, nil
	}

	// Auto-resolve based on the flag
	switch autoResolve {
	case ActionSkip:
		return "", "", ErrSkipRule
	case ActionOverwrite:
		return key, ActionOverwrite, nil
	case ActionRename:
		// The new name will be determined later
		return "", ActionRename, nil
	default:
		return "", "", fmt.Errorf("invalid auto-resolve action: %s", autoResolve)
	}
}

// promptForConflictResolution prompts the user to resolve a conflict.
func promptForConflictResolution(key string) string {
	fmt.Printf("Rule '%s' already exists. Choose an action:\n", key)
	fmt.Println("  (s)kip - Skip this rule")
	fmt.Println("  (o)verwrite - Replace the existing rule")
	fmt.Println("  (r)ename - Install with a different name")
	fmt.Print("Enter choice [s/o/r]: ")

	var response string
	_, err := fmt.Scanln(&response)
	if err != nil {
		// Default to skip on error
		fmt.Println("Invalid input, defaulting to skip")
		return ActionSkip
	}

	response = strings.ToLower(strings.TrimSpace(response))
	switch response {
	case "o", "overwrite":
		return ActionOverwrite
	case "r", "rename":
		return ActionRename
	case "s", "skip":
		fallthrough
	default:
		return ActionSkip
	}
}

// processEmbeddedRuleContent processes a rule with embedded content.
func processEmbeddedRuleContent(cursorDir string, sr *ShareableRule, key string) error {
	if sr.Content == "" {
		return fmt.Errorf("embedded rule has no content")
	}

	// Determine the filename
	filename := key + ".mdc"
	if sr.Filename != "" && filepath.Ext(sr.Filename) == ".mdc" {
		filename = sr.Filename
	}

	// Write the file
	filePath := filepath.Join(cursorDir, filename)
	err := os.WriteFile(filePath, []byte(sr.Content), 0o644)
	if err != nil {
		return fmt.Errorf("failed to write rule file: %w", err)
	}

	// Create a RuleSource
	rule := RuleSource{
		Key:        key,
		SourceType: SourceTypeBuiltIn, // Simplify by treating as built-in
		Reference:  key,
		LocalFiles: []string{filePath},
		// We could store ContentSHA256 here for future upgrade checks
	}

	// Add to lockfile
	lock, err := LoadLockFile(cursorDir)
	if err != nil {
		return fmt.Errorf("failed to load lockfile: %w", err)
	}

	// Remove existing rule with the same key if any
	for i, r := range lock.Rules {
		if r.Key == key {
			// Remove any existing files
			for _, file := range r.LocalFiles {
				_ = os.Remove(file) // Ignore errors, file might not exist
			}
			// Remove from lockfile
			lock.Rules = append(lock.Rules[:i], lock.Rules[i+1:]...)
			break
		}
	}

	// Add the new rule
	lock.Rules = append(lock.Rules, rule)
	// For backwards compatibility
	lock.Installed = append(lock.Installed, key)

	// Save lockfile
	err = lock.Save(cursorDir)
	if err != nil {
		return fmt.Errorf("failed to update lockfile: %w", err)
	}

	return nil
}

// processGitHubRule processes a GitHub rule during restore.
func processGitHubRule(cursorDir string, sr *ShareableRule, key string) error {
	// GitHub rules can be installed via AddRuleByReference
	err := AddRuleByReference(cursorDir, sr.Reference)
	if err != nil {
		// If it's already installed, that's not an error here
		if strings.Contains(err.Error(), "already installed") {
			fmt.Printf("Rule %s is already installed\n", key)
			return nil
		}
		return fmt.Errorf("failed to install GitHub rule: %w", err)
	}

	return nil
}

// processBuiltInRule processes a built-in rule during restore.
func processBuiltInRule(cursorDir string, sr *ShareableRule, key string) error {
	// Built-in rules can be installed via AddRule
	return AddRule(cursorDir, sr.Category, key)
}

// processLocalRule processes a local rule during restore.
func processLocalRule(cursorDir string, sr *ShareableRule) error {
	// Local rules without embedded content can't be restored
	if sr.Content == "" {
		return fmt.Errorf("local rule has no embedded content and can't be restored")
	}

	// Otherwise, treat as embedded content
	return processEmbeddedRuleContent(cursorDir, sr, sr.Key)
}

// processRule processes a single rule during restore.
func processRule(cursorDir string, sr ShareableRule, existingRules map[string]bool, autoResolve string) error {
	// Skip unshareable rules
	if sr.Unshareable {
		fmt.Printf("Skipping unshareable rule: %s\n", sr.Key)
		return nil
	}

	// Check for conflicts
	key := sr.Key
	if existingRules[key] {
		// Resolve conflict
		var action string
		var err error
		key, action, err = resolveConflict(key, autoResolve)
		if err != nil {
			if err == ErrSkipRule {
				fmt.Printf("Skipping rule: %s\n", sr.Key)
				return nil
			}
			return err
		}

		// If renaming, find a new key
		if action == ActionRename {
			key = findAvailableKey(sr.Key, existingRules)
			fmt.Printf("Renaming rule to: %s\n", key)
		}
	}

	// Process based on source type
	var err error
	switch sr.SourceType {
	case SourceTypeBuiltIn:
		err = processBuiltInRule(cursorDir, &sr, key)
	case SourceTypeGitHubFile, SourceTypeGitHubDir:
		err = processGitHubRule(cursorDir, &sr, key)
	case SourceTypeLocalAbs, SourceTypeLocalRel:
		err = processLocalRule(cursorDir, &sr)
	default:
		err = fmt.Errorf("unsupported rule source type: %s", sr.SourceType)
	}

	if err != nil {
		return fmt.Errorf("failed to process rule %s: %w", sr.Key, err)
	}

	return nil
}

// RestoreFromShared restores rules from a shared file.
func RestoreFromShared(ctx context.Context, cursorDir, sharePath, autoResolve string) error {
	// Load and parse the shareable file
	data, err := loadShareableData(ctx, sharePath)
	if err != nil {
		return err
	}

	lock, err := parseShareableLock(data)
	if err != nil {
		return err
	}

	// Load the current lockfile to check for conflicts
	currentLock, err := LoadLockFile(cursorDir)
	if err != nil {
		return fmt.Errorf("failed to load lockfile: %w", err)
	}

	// Build a set of existing rules for conflict detection
	existingRules := buildExistingRuleSet(currentLock)

	// Process each rule
	processed := 0
	skipped := 0

	for _, sr := range lock.Rules {
		err := processRule(cursorDir, sr, existingRules, autoResolve)
		if err != nil {
			fmt.Printf("Error processing rule %s: %v\n", sr.Key, err)
			skipped++
		} else {
			processed++
			// Add to existing rules map to handle duplicates in the shared file
			existingRules[sr.Key] = true
		}
	}

	fmt.Printf("Restored %d rules, skipped %d rules\n", processed, skipped)
	return nil
}
