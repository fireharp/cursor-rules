package manager

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/fireharp/cursor-rules/pkg/templates"
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
// This function has been refactored to use the Strategy Pattern instead of a large
// if/else chain. It now uses a registry of handlers to find the appropriate handler
// for the given reference type, and delegates the processing to that handler.
// The original goto statement has been replaced with proper function extraction.
func addRuleByReferenceImpl(cursorDir, ref string) error {
	// Create a registry with all reference handlers
	registry := NewReferenceHandlerRegistry()

	// Find a handler for this reference
	handler := registry.FindHandler(ref)
	if handler == nil {
		// Check if it's a built-in template as a last resort
		tmpl, err := templates.FindTemplateByName(ref)
		if err == nil && tmpl.Category != "" {
			return AddRule(cursorDir, tmpl.Category, ref)
		}

		return fmt.Errorf("unsupported reference format or rule not found: %s", ref)
	}

	// Process the reference
	rule, err := handler.Process(context.Background(), cursorDir, ref)

	// Handle specific error cases
	if err != nil {
		// Check if this is a special error indicating a template was found
		if strings.HasPrefix(err.Error(), "template_found:") {
			parts := strings.Split(err.Error(), ":")
			if len(parts) == 3 {
				category := parts[1]
				templateName := parts[2]
				return AddRule(cursorDir, category, templateName)
			}
		}

		// Pass through other errors
		return fmt.Errorf("failed to process reference: %w", err)
	}

	// For glob patterns, the handler updates the lockfile directly and returns an empty RuleSource
	if handler.CanHandle(ref) && isGlobPattern(ref) {
		// Glob patterns are handled specially and already update the lockfile
		return nil
	}

	// Update the lockfile with the processed rule
	return updateLockfileWithRule(cursorDir, rule)
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
