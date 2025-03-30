package manager

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/fireharp/cursor-rules/pkg/templates"
)

func TestPathDetection(t *testing.T) {
	tests := []struct {
		name                  string
		path                  string
		expectedIsRelative    bool
		expectedIsAbsolute    bool
		expectedIsUsername    bool
		expectedIsUserPath    bool
		expectedIsGlobPattern bool
	}{
		{
			name:                  "Relative path with ./",
			path:                  "./path/to/file.mdc",
			expectedIsRelative:    true,
			expectedIsAbsolute:    false,
			expectedIsUsername:    false,
			expectedIsUserPath:    false,
			expectedIsGlobPattern: false,
		},
		{
			name:                  "Relative path with ../",
			path:                  "../path/to/file.mdc",
			expectedIsRelative:    true,
			expectedIsAbsolute:    false,
			expectedIsUsername:    false,
			expectedIsUserPath:    false,
			expectedIsGlobPattern: false,
		},
		{
			name:                  "Simple relative path without ./",
			path:                  "path/to/file.mdc",
			expectedIsRelative:    true,
			expectedIsAbsolute:    false,
			expectedIsUsername:    false,
			expectedIsUserPath:    false,
			expectedIsGlobPattern: false,
		},
		{
			name:                  "Absolute path",
			path:                  "/Users/username/path/to/file.mdc",
			expectedIsRelative:    false,
			expectedIsAbsolute:    true,
			expectedIsUsername:    false,
			expectedIsUserPath:    false,
			expectedIsGlobPattern: false,
		},
		{
			name:                  "GitHub URL",
			path:                  "https://github.com/username/repo/blob/main/file.mdc",
			expectedIsRelative:    false,
			expectedIsAbsolute:    false,
			expectedIsUsername:    false,
			expectedIsUserPath:    false,
			expectedIsGlobPattern: false,
		},
		{
			name:                  "Username/rule pattern",
			path:                  "username/rule-name",
			expectedIsRelative:    false,
			expectedIsAbsolute:    false,
			expectedIsUsername:    true,
			expectedIsUserPath:    false,
			expectedIsGlobPattern: false,
		},
		{
			name:                  "Username/path/rule pattern",
			path:                  "username/path/rule-name",
			expectedIsRelative:    false,
			expectedIsAbsolute:    false,
			expectedIsUsername:    false,
			expectedIsUserPath:    true,
			expectedIsGlobPattern: false,
		},
		{
			name:                  "Glob pattern with *",
			path:                  "path/to/*.mdc",
			expectedIsRelative:    true,
			expectedIsAbsolute:    false,
			expectedIsUsername:    false,
			expectedIsUserPath:    false,
			expectedIsGlobPattern: true,
		},
		{
			name:                  "Username with glob pattern",
			path:                  "username/*.mdc",
			expectedIsRelative:    false,
			expectedIsAbsolute:    false,
			expectedIsUsername:    false,
			expectedIsUserPath:    false,
			expectedIsGlobPattern: true,
		},
		{
			name:                  "Relative path that looks like username/rule but has file extension",
			path:                  "test-rules/go/file.mdc",
			expectedIsRelative:    true,
			expectedIsAbsolute:    false,
			expectedIsUsername:    false,
			expectedIsUserPath:    false,
			expectedIsGlobPattern: false,
		},
		// New test cases for SHA and tag references
		{
			name:                  "Username/rule with full SHA",
			path:                  "username/rule:abcdef1234567890",
			expectedIsRelative:    false,
			expectedIsAbsolute:    false,
			expectedIsUsername:    true, // Should be true if SHA is handled in isUsernameRule
			expectedIsUserPath:    false,
			expectedIsGlobPattern: false,
		},
		{
			name:                  "Username/rule with short SHA",
			path:                  "username/rule:abc123",
			expectedIsRelative:    false,
			expectedIsAbsolute:    false,
			expectedIsUsername:    true,
			expectedIsUserPath:    false,
			expectedIsGlobPattern: false,
		},
		{
			name:                  "Username/rule with tag",
			path:                  "username/rule@v1.2",
			expectedIsRelative:    false,
			expectedIsAbsolute:    false,
			expectedIsUsername:    true,
			expectedIsUserPath:    false,
			expectedIsGlobPattern: false,
		},
		{
			name:                  "Full GitHub-style path with tag",
			path:                  "username/repo/path/to/rule@v1",
			expectedIsRelative:    false,
			expectedIsAbsolute:    false,
			expectedIsUsername:    false,
			expectedIsUserPath:    true,
			expectedIsGlobPattern: false,
		},
		// Various glob pattern scenarios
		{
			name:                  "Local directory glob",
			path:                  "go/*",
			expectedIsRelative:    true,
			expectedIsAbsolute:    false,
			expectedIsUsername:    false,
			expectedIsUserPath:    false,
			expectedIsGlobPattern: true,
		},
		{
			name:                  "Double star glob",
			path:                  "go/**/important/*",
			expectedIsRelative:    true,
			expectedIsAbsolute:    false,
			expectedIsUsername:    false,
			expectedIsUserPath:    false,
			expectedIsGlobPattern: true,
		},
		// Potential collisions
		{
			name:                  "Local folder named 'username'",
			path:                  "username/path",
			expectedIsRelative:    true, // This will likely fail with current implementation
			expectedIsAbsolute:    false,
			expectedIsUsername:    false,
			expectedIsUserPath:    false,
			expectedIsGlobPattern: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isRelativePath(tt.path); got != tt.expectedIsRelative {
				t.Errorf("isRelativePath(%q) = %v, want %v", tt.path, got, tt.expectedIsRelative)
			}

			if got := isAbsolutePath(tt.path); got != tt.expectedIsAbsolute {
				t.Errorf("isAbsolutePath(%q) = %v, want %v", tt.path, got, tt.expectedIsAbsolute)
			}

			if got := isUsernameRule(tt.path); got != tt.expectedIsUsername {
				t.Errorf("isUsernameRule(%q) = %v, want %v", tt.path, got, tt.expectedIsUsername)
			}

			if got := isUsernamePathRule(tt.path); got != tt.expectedIsUserPath {
				t.Errorf("isUsernamePathRule(%q) = %v, want %v", tt.path, got, tt.expectedIsUserPath)
			}

			if got := isGlobPattern(tt.path); got != tt.expectedIsGlobPattern {
				t.Errorf("isGlobPattern(%q) = %v, want %v", tt.path, got, tt.expectedIsGlobPattern)
			}

			// Debug information for glob patterns
			if tt.expectedIsGlobPattern {
				username, pattern, ok := parseGlobPattern(tt.path)
				t.Logf("parseGlobPattern(%q) = %q, %q, %v", tt.path, username, pattern, ok)
			}
		})
	}
}

func TestGenerateRuleKey(t *testing.T) {
	tests := []struct {
		name        string
		reference   string
		expectedKey string
	}{
		{
			name:        "Relative path with ./",
			reference:   "./path/to/file.mdc",
			expectedKey: "local/rel/path-to-file",
		},
		{
			name:        "Relative path with ../",
			reference:   "../path/to/file.mdc",
			expectedKey: "local/rel/path-to-file",
		},
		{
			name:        "Relative path without ./",
			reference:   "path/to/file.mdc",
			expectedKey: "local/rel/path-to-file",
		},
		{
			name:        "Username/rule pattern",
			reference:   "username/rule-name",
			expectedKey: "username/rule-name",
		},
		{
			name:        "Username/path/rule pattern with 3+ parts",
			reference:   "username/path/to/rule-name",
			expectedKey: "username/path/to/rule-name",
		},
		// New test cases for generateRuleKey
		{
			name:        "Username/rule with full SHA",
			reference:   "username/rule:abcdef1234567890",
			expectedKey: "username/rule-abcdef1234567890", // Adjust expected based on implementation
		},
		{
			name:        "Username/rule with short SHA",
			reference:   "username/rule:abc123",
			expectedKey: "username/rule-abc123", // Adjust expected based on implementation
		},
		{
			name:        "Username/rule with tag",
			reference:   "username/rule@v1.2",
			expectedKey: "username/rule-v1.2", // Adjust expected based on implementation
		},
		{
			name:        "Full path with tag",
			reference:   "username/repo/path/to/rule@v1",
			expectedKey: "username/repo/path/to/rule-v1", // Adjust expected based on implementation
		},
		{
			name:        "Local glob pattern",
			reference:   "path/to/*.mdc",
			expectedKey: "local/rel/path-to-glob", // Adjust expected based on implementation
		},
		{
			name:        "Double star glob pattern",
			reference:   "path/to/**/file.mdc",
			expectedKey: "local/rel/path-to-deep-glob", // Adjust expected based on implementation
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := generateRuleKey(tt.reference); got != tt.expectedKey {
				t.Errorf("generateRuleKey(%q) = %v, want %v", tt.reference, got, tt.expectedKey)
			}
		})
	}
}

// Test the DefaultUsernameHandler which implements fallback logic
func TestResolveRuleFallback(t *testing.T) {
	// This test verifies that DefaultUsernameHandler properly implements
	// fallback resolution for rules using the default username

	// Create a mock implementation of getDefaultUsername to avoid
	// dependencies on external files or environment variables
	origGetDefaultUsername := getDefaultUsername
	defer func() {
		getDefaultUsername = origGetDefaultUsername
	}()

	getDefaultUsername = func() string {
		return "testuser"
	}

	// Create a temporary directory
	tempDir, err := os.MkdirTemp("", "cursor-rules-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a cursor rules directory
	cursorDir := filepath.Join(tempDir, ".cursor", "rules")
	if err := os.MkdirAll(cursorDir, 0o755); err != nil {
		t.Fatalf("Failed to create cursor rules dir: %v", err)
	}

	// Initialize a handler to test
	handler := &DefaultUsernameHandler{}

	// Test that it will properly handle refs when a default username is set
	if !handler.CanHandle("rule-name") {
		t.Error("DefaultUsernameHandler.CanHandle() returned false when default username is set")
	}

	// Test the fallback strategy for templates
	getDefaultUsername = func() string {
		return "testuser"
	}

	// Add a mock template to test the fallback to templates
	if templates.Categories["test"] == nil {
		templates.Categories["test"] = &templates.TemplateCategory{
			Name:        "Test",
			Description: "Test templates",
			Templates:   make(map[string]templates.Template),
		}
	}
	templates.Categories["test"].Templates["test-template"] = templates.Template{
		Name:        "Test Template",
		Description: "A test template",
		Content:     "# Test template content",
		Category:    "test",
	}

	// Process should normally call GitHub, but we don't want to actually do that in tests
	// So we'll test the template fallback path
	_, err = handler.Process(context.Background(), cursorDir, "test-template")

	// Verify we got the appropriate ErrTemplateFound error
	var templateErr *ErrTemplateFound
	if !errors.As(err, &templateErr) {
		t.Errorf("Expected ErrTemplateFound, got: %v", err)
	} else {
		if templateErr.Category != "test" || templateErr.Name != "test-template" {
			t.Errorf("Expected template category 'test' and name 'test-template', got category '%s' and name '%s'",
				templateErr.Category, templateErr.Name)
		}
	}

	// Clean up test template
	delete(templates.Categories["test"].Templates, "test-template")
}
