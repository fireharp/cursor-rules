package manager

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/fireharp/cursor-rules/pkg/templates"
)

// LockFileName is the file that tracks installed rules (similar to a package-lock.json)
const LockFileName = "cursor-rules.lock"

// UseRootLockFile is a flag to indicate whether to use the lockfile in the project root
// This can be set through an environment variable or config file
var UseRootLockFile = false

// getRootDirectory returns the project root directory from the cursor rules directory
func getRootDirectory(cursorDir string) string {
	// cursorDir is typically /path/to/project/.cursor/rules
	// We need to go up two levels to get the project root
	return filepath.Dir(filepath.Dir(cursorDir))
}

// getLockFilePath returns the path to the lockfile based on the UseRootLockFile setting
func getLockFilePath(cursorDir string) string {
	if UseRootLockFile {
		rootDir := getRootDirectory(cursorDir)
		return filepath.Join(rootDir, LockFileName)
	}
	return filepath.Join(cursorDir, LockFileName)
}

// SourceType represents the type of rule source
type SourceType string

// Source types for rules
const (
	SourceTypeBuiltIn    SourceType = "built-in"
	SourceTypeLocalAbs   SourceType = "local-abs"
	SourceTypeLocalRel   SourceType = "local-rel"
	SourceTypeGitHubFile SourceType = "github-file"
	SourceTypeGitHubDir  SourceType = "github-dir"
)

// RuleSource represents a source for a rule file
type RuleSource struct {
	// The short "rule key" used in local .cursor/rules filenames
	Key string `json:"key"`

	// A short, machine-readable type: "built-in", "local-abs", "local-rel", "github-file", "github-dir"
	SourceType SourceType `json:"sourceType"`

	// The raw string that the user passed in (e.g. "/Users/.../file.mdc" or "https://github.com/.../blob/commit/file.mdc")
	Reference string `json:"reference"`

	// Category for built-in rules
	Category string `json:"category,omitempty"`

	// For Git sources, store explicit commit/tag/branch
	// e.g. "commit=0609329", "branch=main", etc.
	GitRef string `json:"gitRef,omitempty"`

	// The exact file path(s) that ended up in .cursor/rules
	// If you download multiple .mdc files from a directory, you might store a slice here
	LocalFiles []string `json:"localFiles"`

	// Optionally, store a "resolved" commit if user used a branch
	// e.g. user says "main," but you lock it to a commit hash for reproducibility
	ResolvedCommit string `json:"resolvedCommit,omitempty"`
}

// LockFile represents the structure of the lockfile on disk.
type LockFile struct {
	// For backwards compatibility
	Installed []string `json:"installed,omitempty"`

	// Enhanced structure for tracking rule sources
	Rules []RuleSource `json:"rules"`
}

// LoadLockFile loads the lockfile from the specified directory (if it exists).
func LoadLockFile(cursorDir string) (*LockFile, error) {
	lockFilePath := getLockFilePath(cursorDir)

	// If there's no lockfile yet, check if there's one in the other location
	if _, err := os.Stat(lockFilePath); os.IsNotExist(err) {
		// If using root lockfile but it doesn't exist, check in .cursor/rules
		if UseRootLockFile {
			altPath := filepath.Join(cursorDir, LockFileName)
			if info, err := os.Stat(altPath); err == nil && !info.IsDir() {
				// Found in .cursor/rules, migrate it
				data, err := os.ReadFile(altPath)
				if err == nil {
					var lock LockFile
					if json.Unmarshal(data, &lock) == nil {
						// Successfully loaded from alternate location
						// Save to the new location
						lockFilePath = altPath
					}
				}
			}
		} else {
			// If using .cursor/rules but it doesn't exist, check in project root
			rootDir := getRootDirectory(cursorDir)
			altPath := filepath.Join(rootDir, LockFileName)
			if info, err := os.Stat(altPath); err == nil && !info.IsDir() {
				// Found in project root, migrate it
				data, err := os.ReadFile(altPath)
				if err == nil {
					var lock LockFile
					if json.Unmarshal(data, &lock) == nil {
						// Successfully loaded from alternate location
						// For migration, save to .cursor/rules location
						lockFilePath = altPath
					}
				}
			}
		}

		// If still no lockfile, return an empty one
		if _, err := os.Stat(lockFilePath); os.IsNotExist(err) {
			return &LockFile{Rules: []RuleSource{}}, nil
		}
	}

	data, err := os.ReadFile(lockFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read lockfile: %w", err)
	}

	var lock LockFile
	if err := json.Unmarshal(data, &lock); err != nil {
		return nil, fmt.Errorf("failed to parse lockfile: %w", err)
	}

	// Handle legacy lockfiles with only Installed field
	if len(lock.Installed) > 0 && len(lock.Rules) == 0 {
		// Migrate from old format to new format
		for _, key := range lock.Installed {
			// Find rule in templates to get category
			var category string
			for catName, cat := range templates.Categories {
				if _, ok := cat.Templates[key]; ok {
					category = catName
					break
				}
			}

			lock.Rules = append(lock.Rules, RuleSource{
				Key:        key,
				SourceType: SourceTypeBuiltIn,
				Reference:  key,
				Category:   category,
				LocalFiles: []string{key + ".mdc"},
			})
		}
	}

	return &lock, nil
}

// Save writes the lockfile back to disk.
func (lock *LockFile) Save(cursorDir string) error {
	lockFilePath := getLockFilePath(cursorDir)
	data, err := json.MarshalIndent(lock, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal lockfile: %w", err)
	}

	// Ensure directory exists for the lockfile
	if UseRootLockFile {
		// For root lockfile, the directory should already exist
	} else {
		// For .cursor/rules lockfile, make sure the directory exists
		if err := os.MkdirAll(cursorDir, 0755); err != nil {
			return fmt.Errorf("failed to create directory for lockfile: %w", err)
		}
	}

	if err := os.WriteFile(lockFilePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write lockfile: %w", err)
	}

	return nil
}

// SetLockFileLocation sets the location for the lockfile (project root or .cursor/rules).
// Returns the new path to the lockfile.
func SetLockFileLocation(cursorDir string, useRoot bool) (string, error) {
	// Load the existing lockfile from wherever it might be
	oldUseRootLockFile := UseRootLockFile
	UseRootLockFile = false // Temporarily set to check .cursor/rules first
	lockFromCursor, errCursor := LoadLockFile(cursorDir)

	UseRootLockFile = true // Now check project root
	lockFromRoot, errRoot := LoadLockFile(cursorDir)

	// Determine which one to use
	var lock *LockFile

	if errCursor == nil && errRoot == nil {
		// Both exist, merge them
		for _, rule := range lockFromRoot.Rules {
			if !containsRule(lockFromCursor.Rules, rule.Key) {
				lockFromCursor.Rules = append(lockFromCursor.Rules, rule)
			}
		}
		lock = lockFromCursor
	} else if errCursor == nil {
		lock = lockFromCursor
	} else if errRoot == nil {
		lock = lockFromRoot
	} else if errCursor != nil && errRoot != nil {
		// Neither exists, create a new one
		lock = &LockFile{Rules: []RuleSource{}}
	}

	// Set the location for future operations
	UseRootLockFile = useRoot

	// Save the lock file to the new location
	if err := lock.Save(cursorDir); err != nil {
		return "", fmt.Errorf("failed to save lockfile to new location: %w", err)
	}

	// If we migrated, delete the old lockfile
	if oldUseRootLockFile != useRoot {
		oldPath := ""
		if oldUseRootLockFile {
			oldPath = filepath.Join(getRootDirectory(cursorDir), LockFileName)
		} else {
			oldPath = filepath.Join(cursorDir, LockFileName)
		}

		if _, err := os.Stat(oldPath); err == nil {
			if err := os.Remove(oldPath); err != nil {
				return getLockFilePath(cursorDir), fmt.Errorf("warning: could not remove old lockfile at %s: %w", oldPath, err)
			}
		}
	}

	return getLockFilePath(cursorDir), nil
}

// Helper function to check if a rule key exists in a slice of RuleSources
func containsRule(rules []RuleSource, key string) bool {
	for _, rule := range rules {
		if rule.Key == key {
			return true
		}
	}
	return false
}

// IsInstalled checks if a rule is already in the lockfile.
func (lock *LockFile) IsInstalled(ruleKey string) bool {
	// First check in the enhanced Rules structure
	for _, r := range lock.Rules {
		if r.Key == ruleKey {
			return true
		}
	}

	// For backward compatibility, also check Installed
	for _, r := range lock.Installed {
		if r == ruleKey {
			return true
		}
	}

	return false
}

// isAbsolutePath checks if a path is absolute
func isAbsolutePath(path string) bool {
	return filepath.IsAbs(path)
}

// isRelativePath checks if a path is relative
func isRelativePath(path string) bool {
	return strings.HasPrefix(path, "./") || strings.HasPrefix(path, "../")
}

// Pattern for GitHub blob URLs
var githubBlobPattern = regexp.MustCompile(`^https://github\.com/([^/]+)/([^/]+)/blob/([^/]+)/(.+)$`)

// Pattern for GitHub tree URLs
var githubTreePattern = regexp.MustCompile(`^https://github\.com/([^/]+)/([^/]+)/tree/([^/]+)/(.+)$`)

// isGitHubBlobURL checks if a URL is a GitHub blob URL
func isGitHubBlobURL(ref string) bool {
	return githubBlobPattern.MatchString(ref)
}

// isGitHubTreeURL checks if a URL is a GitHub tree URL
func isGitHubTreeURL(ref string) bool {
	return githubTreePattern.MatchString(ref)
}

// generateRuleKey creates a unique key from a reference
func generateRuleKey(ref string) string {
	// For GitHub URLs, use the filename without extension
	if isGitHubBlobURL(ref) {
		matches := githubBlobPattern.FindStringSubmatch(ref)
		if len(matches) > 4 {
			path := matches[4]
			base := filepath.Base(path)
			return strings.TrimSuffix(base, filepath.Ext(base))
		}
	}

	// For local files, use the filename without extension
	if isAbsolutePath(ref) || isRelativePath(ref) {
		base := filepath.Base(ref)
		return strings.TrimSuffix(base, filepath.Ext(base))
	}

	// Fallback
	return ref
}

// AddRuleByReference adds a rule from a reference (local path, GitHub URL, etc.)
func AddRuleByReference(cursorDir, ref string) error {
	// 1. Load the lockfile
	lock, err := LoadLockFile(cursorDir)
	if err != nil {
		return err
	}

	// 2. Generate a rule key
	ruleKey := generateRuleKey(ref)

	// 3. Check if it's already installed
	if lock.IsInstalled(ruleKey) {
		return fmt.Errorf("rule %q is already installed", ruleKey)
	}

	// 4. Determine the type of reference and handle accordingly
	var ruleSource RuleSource
	var handleErr error

	switch {
	case isAbsolutePath(ref):
		ruleSource, handleErr = handleLocalFile(cursorDir, ref, true)
	case isRelativePath(ref):
		ruleSource, handleErr = handleLocalFile(cursorDir, ref, false)
	case isGitHubBlobURL(ref):
		ruleSource, handleErr = handleGitHubBlob(cursorDir, ref)
	case isGitHubTreeURL(ref):
		ruleSource, handleErr = handleGitHubDir(cursorDir, ref)
	default:
		// Try to handle as a built-in template reference (category/key format)
		parts := strings.Split(ref, "/")
		if len(parts) == 2 {
			return AddRule(cursorDir, parts[0], parts[1])
		}
		return fmt.Errorf("unrecognized rule reference: %s", ref)
	}

	if handleErr != nil {
		return handleErr
	}

	// 5. Update lockfile
	lock.Rules = append(lock.Rules, ruleSource)
	err = lock.Save(cursorDir)
	if err != nil {
		return fmt.Errorf("failed to save lockfile: %w", err)
	}

	return nil
}

// handleLocalFile copies a local .mdc file to .cursor/rules
func handleLocalFile(cursorDir, ref string, isAbs bool) (RuleSource, error) {
	// 1. Validate path and ensure it's readable
	var fullPath string
	if isAbs {
		fullPath = ref
	} else {
		var err error
		fullPath, err = filepath.Abs(ref)
		if err != nil {
			return RuleSource{}, fmt.Errorf("failed to resolve path %s: %w", ref, err)
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
	ruleKey := generateRuleKey(ref)
	destFilename := ruleKey + ".mdc"
	destPath := filepath.Join(cursorDir, destFilename)

	// 4. Write to .cursor/rules
	err = os.WriteFile(destPath, data, 0644)
	if err != nil {
		return RuleSource{}, fmt.Errorf("failed to write to %s: %w", destPath, err)
	}

	// 5. Create and return RuleSource
	sourceType := SourceTypeLocalAbs
	if !isAbs {
		sourceType = SourceTypeLocalRel
	}

	return RuleSource{
		Key:        ruleKey,
		SourceType: sourceType,
		Reference:  ref,
		LocalFiles: []string{destFilename},
	}, nil
}

// handleGitHubBlob downloads a single file from GitHub
func handleGitHubBlob(cursorDir, ref string) (RuleSource, error) {
	// 1. Parse the GitHub URL
	matches := githubBlobPattern.FindStringSubmatch(ref)
	if len(matches) < 5 {
		return RuleSource{}, fmt.Errorf("invalid GitHub blob URL: %s", ref)
	}

	owner := matches[1]
	repo := matches[2]
	gitRef := matches[3]
	path := matches[4]

	// 2. Construct raw URL
	rawURL := fmt.Sprintf("https://raw.githubusercontent.com/%s/%s/%s/%s",
		owner, repo, gitRef, path)

	// 3. Download the file
	resp, err := http.Get(rawURL)
	if err != nil {
		return RuleSource{}, fmt.Errorf("failed to download file from %s: %w", rawURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return RuleSource{}, fmt.Errorf("failed to download file from %s: status %d", rawURL, resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return RuleSource{}, fmt.Errorf("failed to read response body: %w", err)
	}

	// 4. Generate rule key and determine destination filename
	ruleKey := generateRuleKey(ref)
	destFilename := ruleKey + ".mdc"
	destPath := filepath.Join(cursorDir, destFilename)

	// 5. Write to .cursor/rules
	err = os.WriteFile(destPath, data, 0644)
	if err != nil {
		return RuleSource{}, fmt.Errorf("failed to write to %s: %w", destPath, err)
	}

	// 6. Create and return RuleSource
	return RuleSource{
		Key:        ruleKey,
		SourceType: SourceTypeGitHubFile,
		Reference:  ref,
		GitRef:     gitRef,
		LocalFiles: []string{destFilename},
	}, nil
}

// handleGitHubDir is a placeholder for downloading multiple files from GitHub dir
// In a real implementation, this would use the GitHub API to list files in the directory
func handleGitHubDir(cursorDir, ref string) (RuleSource, error) {
	// This is a simplified version; in reality would need to:
	// 1. Use GitHub API to list all .mdc files in the directory
	// 2. Download each file
	// 3. Store all files in the LocalFiles field

	return RuleSource{}, fmt.Errorf("GitHub directory references not yet implemented")
}

// AddRule installs the rule with the given key from a specified category into .cursor/rules.
// It also updates the lockfile to track what is installed.
func AddRule(cursorDir string, category string, ruleKey string) error {
	// 1. Find the template in the global templates map
	cat, ok := templates.Categories[category]
	if !ok {
		return fmt.Errorf("category %q not found", category)
	}

	tmpl, ok := cat.Templates[ruleKey]
	if !ok {
		return fmt.Errorf("rule %q not found in category %q", ruleKey, category)
	}

	// 2. Load the lockfile
	lock, err := LoadLockFile(cursorDir)
	if err != nil {
		return err
	}

	// 3. Check if it's already installed
	if lock.IsInstalled(ruleKey) {
		return fmt.Errorf("rule %q is already installed", ruleKey)
	}

	// 4. Actually install (write .mdc) to .cursor/rules
	err = templates.CreateTemplate(cursorDir, tmpl)
	if err != nil {
		return fmt.Errorf("failed to create template: %w", err)
	}

	// 5. Update lockfile with enhanced structure
	ruleSource := RuleSource{
		Key:        ruleKey,
		SourceType: SourceTypeBuiltIn,
		Reference:  ruleKey,
		Category:   category,
		LocalFiles: []string{tmpl.Filename},
	}

	lock.Rules = append(lock.Rules, ruleSource)

	// For backward compatibility, also update Installed
	lock.Installed = append(lock.Installed, ruleKey)

	err = lock.Save(cursorDir)
	if err != nil {
		return fmt.Errorf("failed to save lockfile: %w", err)
	}

	return nil
}

// RemoveRule uninstalls the rule by removing the .mdc file from .cursor/rules and
// removing the entry from the lockfile.
func RemoveRule(cursorDir string, ruleKey string) error {
	// 1. Load the lockfile
	lock, err := LoadLockFile(cursorDir)
	if err != nil {
		return err
	}

	if !lock.IsInstalled(ruleKey) {
		return fmt.Errorf("rule %q is not installed", ruleKey)
	}

	// 2. Find the rule in the enhanced structure
	var filesToRemove []string
	var newRules []RuleSource

	for _, rule := range lock.Rules {
		if rule.Key == ruleKey {
			filesToRemove = rule.LocalFiles
		} else {
			newRules = append(newRules, rule)
		}
	}

	// If not found in enhanced structure, look in old format
	if len(filesToRemove) == 0 {
		// Look up the template to get the proper filename
		var ruleFilename string
		for _, category := range templates.Categories {
			for key, tmpl := range category.Templates {
				if key == ruleKey {
					ruleFilename = tmpl.Filename
					break
				}
			}
			if ruleFilename != "" {
				break
			}
		}

		if ruleFilename == "" {
			// If we couldn't find the template (maybe it was manually added),
			// assume the filename is ruleKey.mdc
			ruleFilename = ruleKey + ".mdc"
		}

		filesToRemove = append(filesToRemove, ruleFilename)
	}

	// 3. Remove all files associated with the rule
	for _, filename := range filesToRemove {
		filePath := filepath.Join(cursorDir, filename)
		if _, err := os.Stat(filePath); err == nil {
			err = os.Remove(filePath)
			if err != nil {
				return fmt.Errorf("failed to remove %s: %w", filePath, err)
			}
		}
	}

	// 4. Update both formats in the lockfile
	lock.Rules = newRules

	// For backward compatibility, also update Installed
	var newInstalled []string
	for _, r := range lock.Installed {
		if r != ruleKey {
			newInstalled = append(newInstalled, r)
		}
	}
	lock.Installed = newInstalled

	// 5. Save lockfile
	err = lock.Save(cursorDir)
	if err != nil {
		return fmt.Errorf("failed to save lockfile after removal: %w", err)
	}

	return nil
}

// UpgradeRule re-installs the template from the "latest" version in the built-in templates.
func UpgradeRule(cursorDir string, ruleKey string) error {
	// 1. Ensure the rule is actually installed
	lock, err := LoadLockFile(cursorDir)
	if err != nil {
		return err
	}
	if !lock.IsInstalled(ruleKey) {
		return fmt.Errorf("rule %q is not installed, can't upgrade", ruleKey)
	}

	// 2. Find the rule in the enhanced structure
	var ruleToUpgrade RuleSource
	for _, rule := range lock.Rules {
		if rule.Key == ruleKey {
			ruleToUpgrade = rule
			break
		}
	}

	// 3. Handle upgrade based on source type
	switch ruleToUpgrade.SourceType {
	case SourceTypeBuiltIn:
		// For built-in rules, use the category from the rule
		cat, ok := templates.Categories[ruleToUpgrade.Category]
		if !ok {
			return fmt.Errorf("category %q not found", ruleToUpgrade.Category)
		}
		tmpl, ok := cat.Templates[ruleKey]
		if !ok {
			return fmt.Errorf("rule %q not found in category %q", ruleKey, ruleToUpgrade.Category)
		}

		err = templates.CreateTemplate(cursorDir, tmpl)
		if err != nil {
			return fmt.Errorf("failed to create template: %w", err)
		}

	case SourceTypeGitHubFile, SourceTypeGitHubDir:
		// For GitHub references, re-download the file(s)
		err := AddRuleByReference(cursorDir, ruleToUpgrade.Reference)
		if err != nil {
			return fmt.Errorf("failed to upgrade GitHub reference: %w", err)
		}

	case SourceTypeLocalAbs, SourceTypeLocalRel:
		// For local files, re-copy the file
		_, err := handleLocalFile(cursorDir, ruleToUpgrade.Reference,
			ruleToUpgrade.SourceType == SourceTypeLocalAbs)
		if err != nil {
			return fmt.Errorf("failed to upgrade local file: %w", err)
		}
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

// GetInstalledRules returns the full RuleSource structs for all installed rules
func GetInstalledRules(cursorDir string) ([]RuleSource, error) {
	lock, err := LoadLockFile(cursorDir)
	if err != nil {
		return nil, err
	}

	return lock.Rules, nil
}

// SyncLocalRules scans the .cursor/rules directory for .mdc files that are not in the lockfile
// and adds them to the lockfile as locally installed rules.
func SyncLocalRules(cursorDir string) error {
	lock, err := LoadLockFile(cursorDir)
	if err != nil {
		return err
	}

	// Get all .mdc files including those directly in the rules directory
	ruleFiles := []string{}
	err = filepath.Walk(cursorDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip if it's a directory, non-mdc file, or the lockfile itself
		if info.IsDir() ||
			!strings.HasSuffix(info.Name(), ".mdc") ||
			info.Name() == "cursor-rules.lock" {
			return nil
		}

		// Add to the list
		ruleFiles = append(ruleFiles, path)
		return nil
	})

	if err != nil {
		return err
	}

	// Now process each rule file and add if not already in lockfile
	for _, path := range ruleFiles {
		// Get the relative path for display/reference
		relPath, err := filepath.Rel(cursorDir, path)
		if err != nil {
			return err
		}

		// Extract rule key from filename (without extension)
		ruleKey := strings.TrimSuffix(filepath.Base(path), ".mdc")

		// Check if rule is already in lockfile
		if lock.IsInstalled(ruleKey) {
			continue
		}

		// Add rule to lockfile
		rule := RuleSource{
			Key:        ruleKey,
			SourceType: SourceTypeLocalRel,
			Reference:  relPath,
			LocalFiles: []string{relPath},
		}
		lock.Rules = append(lock.Rules, rule)
		fmt.Printf("Adding local rule: %s\n", ruleKey)
	}

	// Save the updated lockfile
	return lock.Save(cursorDir)
}

// Helper function to check if file exists
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
