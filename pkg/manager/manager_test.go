package manager

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/fireharp/cursor-rules/pkg/templates"
)

// setupTestDir creates a temporary directory for testing
func setupTestDir(t *testing.T) string {
	t.Helper()
	tempDir, err := os.MkdirTemp("", "cursor-rules-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	return tempDir
}

// cleanupTestDir removes the temporary directory
func cleanupTestDir(t *testing.T, dir string) {
	t.Helper()
	if err := os.RemoveAll(dir); err != nil {
		t.Fatalf("Failed to remove temp directory: %v", err)
	}
}

// setupTestTemplates creates mock templates for testing
func setupTestTemplates() {
	// Initialize templates.Categories if it doesn't exist
	if templates.Categories == nil {
		templates.Categories = make(map[string]*templates.TemplateCategory)
	}

	// Create test categories and templates
	templates.Categories["general"] = &templates.TemplateCategory{
		Name:        "General",
		Description: "General templates for all projects",
		Templates:   make(map[string]templates.Template),
	}

	templates.Categories["languages"] = &templates.TemplateCategory{
		Name:        "Languages",
		Description: "Templates for programming languages",
		Templates:   make(map[string]templates.Template),
	}

	// Add test templates
	templates.Categories["general"].Templates["test-rule"] = templates.Template{
		Name:        "Test Rule",
		Description: "A test rule for testing",
		Globs:       []string{"*.test"},
		AlwaysApply: false,
		Content:     "This is a test rule content",
		Filename:    "test-rule.mdc",
		Category:    "general",
	}

	templates.Categories["languages"].Templates["python"] = templates.Template{
		Name:        "Python",
		Description: "Python language rule",
		Globs:       []string{"*.py"},
		AlwaysApply: false,
		Content:     "This is a Python rule content",
		Filename:    "python.mdc",
		Category:    "languages",
	}
}

// TestLoadLockFile_New tests loading a lock file that doesn't exist yet
func TestLoadLockFile_New(t *testing.T) {
	tempDir := setupTestDir(t)
	defer cleanupTestDir(t, tempDir)

	lock, err := LoadLockFile(tempDir)
	if err != nil {
		t.Fatalf("LoadLockFile returned error: %v", err)
	}

	if len(lock.Installed) != 0 {
		t.Errorf("Expected empty installed rules, got %d", len(lock.Installed))
	}
}

// TestLockFileSaveLoad tests saving and loading a lock file
func TestLockFileSaveLoad(t *testing.T) {
	tempDir := setupTestDir(t)
	defer cleanupTestDir(t, tempDir)

	// Create and save a lock file
	lock := &LockFile{
		Installed: []string{"test-rule", "python"},
	}

	if err := lock.Save(tempDir); err != nil {
		t.Fatalf("Save returned error: %v", err)
	}

	// Verify the file exists
	lockPath := filepath.Join(tempDir, LockFileName)
	if _, err := os.Stat(lockPath); os.IsNotExist(err) {
		t.Fatalf("Lock file was not created at %s", lockPath)
	}

	// Read the file and check its contents
	data, err := os.ReadFile(lockPath)
	if err != nil {
		t.Fatalf("Failed to read lock file: %v", err)
	}

	var savedLock LockFile
	if err := json.Unmarshal(data, &savedLock); err != nil {
		t.Fatalf("Failed to parse lock file JSON: %v", err)
	}

	if !reflect.DeepEqual(savedLock.Installed, lock.Installed) {
		t.Errorf("Saved lock file contents don't match. Expected %v, got %v", lock.Installed, savedLock.Installed)
	}

	// Test loading the file
	loadedLock, err := LoadLockFile(tempDir)
	if err != nil {
		t.Fatalf("LoadLockFile returned error: %v", err)
	}

	if !reflect.DeepEqual(loadedLock.Installed, lock.Installed) {
		t.Errorf("Loaded lock file contents don't match. Expected %v, got %v", lock.Installed, loadedLock.Installed)
	}
}

// TestIsInstalled tests the IsInstalled method
func TestIsInstalled(t *testing.T) {
	lock := &LockFile{
		Installed: []string{"test-rule", "python"},
	}

	tests := []struct {
		name     string
		ruleKey  string
		expected bool
	}{
		{"Installed rule", "test-rule", true},
		{"Another installed rule", "python", true},
		{"Not installed rule", "golang", false},
		{"Empty rule key", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if result := lock.IsInstalled(tt.ruleKey); result != tt.expected {
				t.Errorf("IsInstalled(%q) = %v, want %v", tt.ruleKey, result, tt.expected)
			}
		})
	}
}

// TestAddRule tests the AddRule function
func TestAddRule(t *testing.T) {
	tempDir := setupTestDir(t)
	defer cleanupTestDir(t, tempDir)

	// Setup test templates
	setupTestTemplates()

	// Create templates directory
	templatesDir := filepath.Join(tempDir, ".cursor", "rules")
	if err := os.MkdirAll(templatesDir, 0755); err != nil {
		t.Fatalf("Failed to create templates directory: %v", err)
	}

	// Test adding a rule
	if err := AddRule(templatesDir, "general", "test-rule"); err != nil {
		t.Fatalf("AddRule returned error: %v", err)
	}

	// Check if the rule file was created
	rulePath := filepath.Join(templatesDir, "test-rule.mdc")
	if _, err := os.Stat(rulePath); os.IsNotExist(err) {
		t.Fatalf("Rule file was not created at %s", rulePath)
	}

	// Check if the rule was added to the lockfile
	lock, err := LoadLockFile(templatesDir)
	if err != nil {
		t.Fatalf("LoadLockFile returned error: %v", err)
	}

	if !lock.IsInstalled("test-rule") {
		t.Errorf("Rule was not added to lockfile")
	}

	// Test adding the same rule again (should fail)
	if err := AddRule(templatesDir, "general", "test-rule"); err == nil {
		t.Errorf("Expected error when adding the same rule twice, but got nil")
	}
}

// TestRemoveRule tests the RemoveRule function
func TestRemoveRule(t *testing.T) {
	tempDir := setupTestDir(t)
	defer cleanupTestDir(t, tempDir)

	// Setup test templates
	setupTestTemplates()

	// Create templates directory
	templatesDir := filepath.Join(tempDir, ".cursor", "rules")
	if err := os.MkdirAll(templatesDir, 0755); err != nil {
		t.Fatalf("Failed to create templates directory: %v", err)
	}

	// Add a rule first
	if err := AddRule(templatesDir, "general", "test-rule"); err != nil {
		t.Fatalf("AddRule returned error: %v", err)
	}

	// Test removing the rule
	if err := RemoveRule(templatesDir, "test-rule"); err != nil {
		t.Fatalf("RemoveRule returned error: %v", err)
	}

	// Check if the rule file was removed
	rulePath := filepath.Join(templatesDir, "test-rule.mdc")
	if _, err := os.Stat(rulePath); !os.IsNotExist(err) {
		t.Errorf("Rule file was not removed from %s", rulePath)
	}

	// Check if the rule was removed from the lockfile
	lock, err := LoadLockFile(templatesDir)
	if err != nil {
		t.Fatalf("LoadLockFile returned error: %v", err)
	}

	if lock.IsInstalled("test-rule") {
		t.Errorf("Rule was not removed from lockfile")
	}

	// Test removing a rule that doesn't exist (should fail)
	if err := RemoveRule(templatesDir, "non-existent-rule"); err == nil {
		t.Errorf("Expected error when removing non-existent rule, but got nil")
	}
}

// TestUpgradeRule tests the UpgradeRule function
func TestUpgradeRule(t *testing.T) {
	// Create a temporary directory
	tempDir, err := os.MkdirTemp("", "cursor-rules-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Prepare a lockfile
	lock := &LockFile{
		Installed: []string{"test-rule"},
		Rules: []RuleSource{
			{
				Key:        "test-rule",
				SourceType: SourceTypeBuiltIn,
				Category:   "general",
				Reference:  "test-rule",
				LocalFiles: []string{"test-rule.mdc"},
			},
		},
	}
	err = lock.Save(tempDir)
	if err != nil {
		t.Fatalf("Failed to save lockfile: %v", err)
	}

	// Create a test template file
	testTemplateContent := "# Test Rule\n\nThis is a test rule."
	testTemplatePath := filepath.Join(tempDir, "test-rule.mdc")
	err = os.WriteFile(testTemplatePath, []byte(testTemplateContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write test template: %v", err)
	}

	// Try to upgrade a rule that doesn't exist
	err = UpgradeRule(tempDir, "nonexistent-rule")
	if err == nil {
		t.Errorf("Expected error when upgrading nonexistent rule, got nil")
	}

	// Add a mock template to the global categories
	if templates.Categories["general"] == nil {
		templates.Categories["general"] = &templates.TemplateCategory{
			Name:        "General",
			Description: "General templates for all projects",
			Templates:   make(map[string]templates.Template),
		}
	}

	templates.Categories["general"].Templates["test-rule"] = templates.Template{
		Name:        "Test Rule",
		Description: "A test rule for testing upgrades",
		Filename:    "test-rule.mdc",
		Content:     "# Upgraded Test Rule\n\nThis rule has been upgraded.",
		Category:    "general",
	}

	// Upgrade the rule
	err = UpgradeRule(tempDir, "test-rule")
	if err != nil {
		t.Fatalf("Failed to upgrade rule: %v", err)
	}

	// Check that the file was updated
	content, err := os.ReadFile(testTemplatePath)
	if err != nil {
		t.Fatalf("Failed to read template after upgrade: %v", err)
	}

	// Just check if the content contains the upgraded text, as the template system adds frontmatter
	if !strings.Contains(string(content), "# Upgraded Test Rule\n\nThis rule has been upgraded.") {
		t.Errorf("Template content was not updated:\n%s", string(content))
	}

	// Clean up the test template from global categories
	delete(templates.Categories["general"].Templates, "test-rule")
}

// TestListInstalledRules tests the ListInstalledRules function
func TestListInstalledRules(t *testing.T) {
	tempDir := setupTestDir(t)
	defer cleanupTestDir(t, tempDir)

	// Setup test templates
	setupTestTemplates()

	// Create templates directory
	templatesDir := filepath.Join(tempDir, ".cursor", "rules")
	if err := os.MkdirAll(templatesDir, 0755); err != nil {
		t.Fatalf("Failed to create templates directory: %v", err)
	}

	// Add some rules
	if err := AddRule(templatesDir, "general", "test-rule"); err != nil {
		t.Fatalf("AddRule returned error: %v", err)
	}

	if err := AddRule(templatesDir, "languages", "python"); err != nil {
		t.Fatalf("AddRule returned error: %v", err)
	}

	// Test listing installed rules
	rules, err := ListInstalledRules(templatesDir)
	if err != nil {
		t.Fatalf("ListInstalledRules returned error: %v", err)
	}

	expectedRules := []string{"test-rule", "python"}
	if !reflect.DeepEqual(rules, expectedRules) {
		t.Errorf("ListInstalledRules returned %v, want %v", rules, expectedRules)
	}

	// Remove a rule
	if err := RemoveRule(templatesDir, "test-rule"); err != nil {
		t.Fatalf("RemoveRule returned error: %v", err)
	}

	// Test listing again
	rules, err = ListInstalledRules(templatesDir)
	if err != nil {
		t.Fatalf("ListInstalledRules returned error: %v", err)
	}

	expectedRules = []string{"python"}
	if !reflect.DeepEqual(rules, expectedRules) {
		t.Errorf("ListInstalledRules returned %v, want %v", rules, expectedRules)
	}
}

func TestAddRuleByReference(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "cursor-rules-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a test .mdc file
	testRuleContent := `# Test Rule

This is a test rule created for testing AddRuleByReference.`

	testFilePath := filepath.Join(tempDir, "test-rule.mdc")
	err = os.WriteFile(testFilePath, []byte(testRuleContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	// Test adding the rule by reference
	err = AddRuleByReference(tempDir, testFilePath)
	if err != nil {
		t.Fatalf("AddRuleByReference failed: %v", err)
	}

	// Check that the rule was added to the lockfile
	lock, err := LoadLockFile(tempDir)
	if err != nil {
		t.Fatalf("Failed to load lockfile: %v", err)
	}

	// Verify the rule is in the lockfile
	found := false
	for _, rule := range lock.Rules {
		if rule.Key == "test-rule" {
			found = true
			if rule.SourceType != SourceTypeLocalAbs {
				t.Errorf("Expected source type %s, got %s", SourceTypeLocalAbs, rule.SourceType)
			}
			if rule.Reference != testFilePath {
				t.Errorf("Expected reference %s, got %s", testFilePath, rule.Reference)
			}
			if len(rule.LocalFiles) != 1 || rule.LocalFiles[0] != "test-rule.mdc" {
				t.Errorf("Expected LocalFiles [test-rule.mdc], got %v", rule.LocalFiles)
			}
			break
		}
	}

	if !found {
		t.Errorf("Rule not found in lockfile")
	}

	// Check that the file was copied to the destination
	destPath := filepath.Join(tempDir, "test-rule.mdc")
	if _, err := os.Stat(destPath); os.IsNotExist(err) {
		t.Errorf("Rule file was not copied to destination")
	}

	// Test removing the rule
	err = RemoveRule(tempDir, "test-rule")
	if err != nil {
		t.Fatalf("RemoveRule failed: %v", err)
	}

	// Check the rule was removed from the lockfile
	lock, err = LoadLockFile(tempDir)
	if err != nil {
		t.Fatalf("Failed to load lockfile after removal: %v", err)
	}

	for _, rule := range lock.Rules {
		if rule.Key == "test-rule" {
			t.Errorf("Rule still exists in lockfile after removal")
		}
	}

	// Check the file was deleted
	if _, err := os.Stat(destPath); !os.IsNotExist(err) {
		t.Errorf("Rule file was not deleted")
	}
}
