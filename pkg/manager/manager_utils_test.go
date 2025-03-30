package manager

import (
	"testing"
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

// Test the resolution and fallback logic
func TestResolveRuleFallback(t *testing.T) {
	// Skip if it's just a unit test that can't test network dependencies
	t.Skip("This is an integration test that requires network access")

	tests := []struct {
		name          string
		input         string
		expectFound   bool
		expectRepoURL string // Expected repository URL after resolution
		expectPath    string // Expected path within repository
	}{
		{
			name:          "First check default collection",
			input:         "rule-name",
			expectFound:   true,
			expectRepoURL: "https://github.com/username/cursor-rules-collection",
			expectPath:    "rule-name.mdc",
		},
		{
			name:          "Then check specific repo",
			input:         "username/repo/rule-name",
			expectFound:   true,
			expectRepoURL: "https://github.com/username/repo",
			expectPath:    "rule-name.mdc",
		},
		{
			name:          "Check with SHA",
			input:         "username/rule-name:abc123",
			expectFound:   true,
			expectRepoURL: "https://github.com/username/cursor-rules-collection@abc123",
			expectPath:    "rule-name.mdc",
		},
		{
			name:          "Check with tag",
			input:         "username/rule-name@v1.0",
			expectFound:   true,
			expectRepoURL: "https://github.com/username/cursor-rules-collection@v1.0",
			expectPath:    "rule-name.mdc",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Assuming there's a resolveRule function that implements the fallback logic
			// repoURL, path, found := resolveRule(tt.input)

			// if found != tt.expectFound {
			//     t.Errorf("resolveRule(%q).found = %v, want %v", tt.input, found, tt.expectFound)
			// }

			// if repoURL != tt.expectRepoURL {
			//     t.Errorf("resolveRule(%q).repoURL = %v, want %v", tt.input, repoURL, tt.expectRepoURL)
			// }

			// if path != tt.expectPath {
			//     t.Errorf("resolveRule(%q).path = %v, want %v", tt.input, path, tt.expectPath)
			// }

			// This test is stubbed for now until the resolveRule function exists
			t.Log("This test is stubbed and will pass until the resolveRule function is implemented")
		})
	}
}
