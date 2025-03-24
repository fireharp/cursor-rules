package manager

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// SourceType represents the type of rule source.
type SourceType string

// Source types for rules.
const (
	SourceTypeBuiltIn    SourceType = "built-in"
	SourceTypeLocalAbs   SourceType = "local-abs"
	SourceTypeLocalRel   SourceType = "local-rel"
	SourceTypeGitHubFile SourceType = "github-file"
	SourceTypeGitHubDir  SourceType = "github-dir"
)

// These are constants for the GitHub action values in rule conflict resolution.
const (
	ActionSkip      string = "skip"
	ActionOverwrite string = "overwrite"
	ActionRename    string = "rename"
)

// RuleSource represents a source for a cursor rule.
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

	// SHA256 hash of the file content - used to detect local modifications
	ContentSHA256 string `json:"contentSHA256,omitempty"`
}

// Regular expressions for parsing GitHub URLs
var githubBlobPattern = regexp.MustCompile(`^https://github\.com/([^/]+)/([^/]+)/blob/([^/]+)/(.+)$`)
var githubTreePattern = regexp.MustCompile(`^https://github\.com/([^/]+)/([^/]+)/tree/([^/]+)/(.+)$`)

// containsRule checks if a rule key exists in a slice of rule sources.
func containsRule(rules []RuleSource, key string) bool {
	for _, r := range rules {
		if r.Key == key {
			return true
		}
	}
	return false
}

// IsInstalled checks if a rule is already installed.
func (lock *LockFile) IsInstalled(ruleKey string) bool {
	// First, check the Rules array (newer format)
	for _, rule := range lock.Rules {
		if rule.Key == ruleKey {
			return true
		}
	}

	// For backwards compatibility, also check the Installed array
	for _, key := range lock.Installed {
		if key == ruleKey {
			return true
		}
	}

	return false
}

// isAbsolutePath checks if a path is absolute.
func isAbsolutePath(path string) bool {
	return filepath.IsAbs(path)
}

// isRelativePath checks if a path is relative and not a URL.
func isRelativePath(path string) bool {
	return !filepath.IsAbs(path) && !strings.HasPrefix(path, "http://") && !strings.HasPrefix(path, "https://")
}

// isGitHubBlobURL checks if a reference is a GitHub blob URL.
func isGitHubBlobURL(ref string) bool {
	return githubBlobPattern.MatchString(ref)
}

// isGitHubTreeURL checks if a reference is a GitHub tree URL.
func isGitHubTreeURL(ref string) bool {
	return githubTreePattern.MatchString(ref)
}

// generateRuleKey creates a rule key from a reference.
func generateRuleKey(ref string) string {
	// For GitHub URLs, extract owner/repo/path
	if isGitHubBlobURL(ref) {
		matches := githubBlobPattern.FindStringSubmatch(ref)
		if len(matches) == 5 {
			// Use owner-repo-file format
			owner := matches[1]
			repo := matches[2]
			path := matches[4]

			// Take just the filename without extension as the base
			base := filepath.Base(path)
			ext := filepath.Ext(base)
			baseName := strings.TrimSuffix(base, ext)

			return owner + "-" + repo + "-" + baseName
		}
	}

	// For file paths, just use the basename without extension
	base := filepath.Base(ref)
	ext := filepath.Ext(base)
	return strings.TrimSuffix(base, ext)
}

// fileExists checks if a file exists.
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// calculateSHA256 calculates the SHA256 hash of a byte slice.
func calculateSHA256(data []byte) string {
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}

// fileContentSHA256 calculates the SHA256 hash of a file's content.
func fileContentSHA256(filePath string) (string, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}

	return hex.EncodeToString(h.Sum(nil)), nil
}

// isGitCommitHash checks if a string looks like a Git commit hash.
func isGitCommitHash(s string) bool {
	// Git commit hashes are 40 characters of hex digits
	if len(s) != 40 {
		return false
	}

	for _, c := range s {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
			return false
		}
	}

	return true
}

// shortCommit returns a shortened (7-character) version of a commit hash for display.
func shortCommit(commit string) string {
	if len(commit) > 7 {
		return commit[:7]
	}
	return commit
}
