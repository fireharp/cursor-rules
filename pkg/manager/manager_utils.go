package manager

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/gobwas/glob"
)

// SourceType represents the type of rule source.
type SourceType string

// Source types for rules.
const (
	SourceTypeBuiltIn         SourceType = "built-in"
	SourceTypeLocalAbs        SourceType = "local-abs"
	SourceTypeLocalRel        SourceType = "local-rel"
	SourceTypeGitHubFile      SourceType = "github-file"
	SourceTypeGitHubDir       SourceType = "github-dir"
	SourceTypeGitHubShorthand SourceType = "github-shorthand" // New source type for username/rule pattern
	SourceTypeGitHubRepoPath  SourceType = "github-repo-path" // New source type for username/repo/path/rule pattern
	SourceTypeGitHubGlob      SourceType = "github-glob"      // New source type for glob patterns
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

	// Original glob pattern used (only for glob patterns)
	GlobPattern string `json:"globPattern,omitempty"`
}

// Regular expressions for parsing GitHub URLs
var githubBlobPattern = regexp.MustCompile(`^https://github\.com/([^/]+)/([^/]+)/blob/([^/]+)/(.+)$`)
var githubTreePattern = regexp.MustCompile(`^https://github\.com/([^/]+)/([^/]+)/tree/([^/]+)/(.+)$`)

// Regular expressions for parsing shorthand formats
var usernameRulePattern = regexp.MustCompile(`^([^/]+)/([^/:@]+)$`)
var usernamePathRulePattern = regexp.MustCompile(`^([^/]+)/([^/]+)/(.+)$`)
var usernameRuleShaPattern = regexp.MustCompile(`^([^/]+)/([^/:@]+):([0-9a-fA-F]+)$`)
var usernameRuleTagPattern = regexp.MustCompile(`^([^/]+)/([^/:@]+)@([^/]+)$`)

// Regular expressions for glob patterns
var globPattern = regexp.MustCompile(`[*?[\]]`)

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

// isUsernameRule checks if a reference matches the username/rule pattern.
func isUsernameRule(ref string) bool {
	return usernameRulePattern.MatchString(ref)
}

// isUsernamePathRule checks if a reference matches the username/path/rule pattern.
func isUsernamePathRule(ref string) bool {
	return usernamePathRulePattern.MatchString(ref)
}

// isUsernameRuleWithSha checks if a reference matches the username/rule:sha pattern.
func isUsernameRuleWithSha(ref string) bool {
	return usernameRuleShaPattern.MatchString(ref)
}

// isUsernameRuleWithTag checks if a reference matches the username/rule@tag pattern.
func isUsernameRuleWithTag(ref string) bool {
	return usernameRuleTagPattern.MatchString(ref)
}

// isGlobPattern checks if a reference contains glob pattern characters.
func isGlobPattern(ref string) bool {
	return globPattern.MatchString(ref)
}

// parseUsernameRule parses a reference in the username/rule format.
// Returns username, rule name, and whether it's a direct match.
func parseUsernameRule(ref string) (string, string, bool) {
	matches := usernameRulePattern.FindStringSubmatch(ref)
	if len(matches) != 3 {
		return "", "", false
	}
	return matches[1], matches[2], true
}

// parseUsernamePathRule parses a reference in the username/path/rule format.
// Returns username, path parts, and whether it's a direct match.
func parseUsernamePathRule(ref string) (string, []string, bool) {
	matches := usernamePathRulePattern.FindStringSubmatch(ref)
	if len(matches) != 4 {
		return "", nil, false
	}

	username := matches[1]
	pathParts := strings.Split(matches[3], "/")

	return username, pathParts, true
}

// parseUsernameRuleWithSha parses a reference in the username/rule:sha format.
// Returns username, rule name, sha, and whether it's a direct match.
func parseUsernameRuleWithSha(ref string) (string, string, string, bool) {
	matches := usernameRuleShaPattern.FindStringSubmatch(ref)
	if len(matches) != 4 {
		return "", "", "", false
	}
	return matches[1], matches[2], matches[3], true
}

// parseUsernameRuleWithTag parses a reference in the username/rule@tag format.
// Returns username, rule name, tag, and whether it's a direct match.
func parseUsernameRuleWithTag(ref string) (string, string, string, bool) {
	matches := usernameRuleTagPattern.FindStringSubmatch(ref)
	if len(matches) != 4 {
		return "", "", "", false
	}
	return matches[1], matches[2], matches[3], true
}

// parseGlobPattern parses a reference with a glob pattern.
// Returns username (if any), pattern, and whether it's valid.
func parseGlobPattern(ref string) (string, string, bool) {
	// If there's no glob pattern characters, it's not a glob
	if !isGlobPattern(ref) {
		return "", "", false
	}

	// Check if it's a username/glob pattern
	if strings.Contains(ref, "/") {
		parts := strings.SplitN(ref, "/", 2)
		if len(parts) == 2 {
			username := parts[0]
			pattern := parts[1]

			// Make sure the username doesn't contain glob characters
			if !isGlobPattern(username) && isGlobPattern(pattern) {
				return username, pattern, true
			}
		}
	}

	// It's a pattern without username
	return "", ref, true
}

// getDefaultUsername gets the default username from configuration.
// Returns empty string if not configured.
func getDefaultUsername() string {
	// TODO: Implement by reading from ~/.cursor-rules/config.json
	return ""
}

// compileGlob compiles a glob pattern to use for matching.
func compileGlob(pattern string) (glob.Glob, error) {
	return glob.Compile(pattern, '/')
}

// matchGlob matches a path against a compiled glob pattern.
func matchGlob(g glob.Glob, path string) bool {
	return g.Match(path)
}

// generateRuleKey creates a rule key from a reference.
func generateRuleKey(ref string) string {
	// For GitHub URLs, extract owner/repo/path
	if isGitHubBlobURL(ref) {
		matches := githubBlobPattern.FindStringSubmatch(ref)
		if len(matches) == 5 {
			owner := matches[1]
			repo := matches[2]
			path := matches[4]

			// Generate a more contextual key with path structure preserved
			// Take the filename without extension
			base := filepath.Base(path)
			ext := filepath.Ext(base)
			baseName := strings.TrimSuffix(base, ext)

			// If it's from the cursor-rules-collection repo, use a special format
			if repo == "cursor-rules-collection" {
				// For cursor-rules-collection, use username/path format
				// Remove the .mdc extension for cleaner keys
				pathKey := strings.TrimSuffix(path, filepath.Ext(path))
				return owner + "/" + pathKey
			} else {
				// For other repos, include the repo name in the path
				// owner/repo/path format
				return owner + "/" + repo + "/" + baseName
			}
		}
	}

	// For username/rule format
	if isUsernameRule(ref) {
		username, rule, _ := parseUsernameRule(ref)
		// Use namespace format: username/rule
		return username + "/" + rule
	}

	// For username/path/rule format with 3+ parts
	if isUsernamePathRule(ref) {
		username, pathParts, _ := parseUsernamePathRule(ref)

		// We need to determine if this is a repo/path format or a path within cursor-rules-collection
		if len(pathParts) >= 2 {
			pathWithoutExt := strings.Join(pathParts, "/")
			// Remove .mdc extension if present on the last part
			lastPart := pathParts[len(pathParts)-1]
			if strings.HasSuffix(lastPart, ".mdc") {
				pathParts[len(pathParts)-1] = strings.TrimSuffix(lastPart, ".mdc")
				pathWithoutExt = strings.Join(pathParts, "/")
			}
			return username + "/" + pathWithoutExt
		}
	}

	// For username/rule:sha or username/rule@tag format
	if isUsernameRuleWithSha(ref) {
		username, rule, _, _ := parseUsernameRuleWithSha(ref)
		return username + "/" + rule
	}

	if isUsernameRuleWithTag(ref) {
		username, rule, _, _ := parseUsernameRuleWithTag(ref)
		return username + "/" + rule
	}

	// For file paths, use a source-prefixed format to avoid conflicts
	base := filepath.Base(ref)
	ext := filepath.Ext(base)
	baseWithoutExt := strings.TrimSuffix(base, ext)

	if isAbsolutePath(ref) {
		return "local/abs/" + baseWithoutExt
	} else if isRelativePath(ref) {
		return "local/rel/" + baseWithoutExt
	}

	// For built-in templates, prefix with built-in to avoid conflicts
	return "built-in/" + baseWithoutExt
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

// listGitHubRepoFiles lists files in a GitHub repository that match a pattern.
// Returns a list of paths that match the pattern.
func listGitHubRepoFiles(ctx context.Context, owner, repo, ref, pattern string) ([]string, error) {
	// Compile the glob pattern for matching
	g, err := compileGlob(pattern)
	if err != nil {
		return nil, fmt.Errorf("invalid glob pattern: %w", err)
	}

	// Start with the root directory
	return recursivelyListGitHubFiles(ctx, owner, repo, ref, "", g, pattern)
}

// recursivelyListGitHubFiles recursively traverses the GitHub repository structure
// and finds all files that match the glob pattern.
func recursivelyListGitHubFiles(ctx context.Context, owner, repo, ref, path string, g glob.Glob, pattern string) ([]string, error) {
	// Construct the API URL for the repository contents
	apiURL := fmt.Sprintf("https://api.github.com/repos/%s/%s/contents/%s", owner, repo, path)
	if ref != "" && ref != "main" {
		apiURL += fmt.Sprintf("?ref=%s", ref)
	}

	// Create request with context
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request for GitHub API: %w", err)
	}

	// Add headers
	req.Header.Add("Accept", "application/vnd.github.v3+json")

	// Send the request
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get repository contents: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP error %d: %s", resp.StatusCode, resp.Status)
	}

	// Parse the response
	var contents []struct {
		Name        string `json:"name"`
		Path        string `json:"path"`
		Type        string `json:"type"`
		DownloadURL string `json:"download_url"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&contents); err != nil {
		return nil, fmt.Errorf("failed to parse GitHub API response: %w", err)
	}

	var matches []string
	var subDirs []string

	// Process each file/directory
	for _, item := range contents {
		// Check if it's a directory or a file
		if item.Type == "dir" {
			// Save directories for recursive processing
			subDirs = append(subDirs, item.Path)
		} else if item.Type == "file" {
			// Check if the file matches our pattern and ends with .mdc
			if (g == nil || matchGlob(g, item.Path)) && strings.HasSuffix(item.Name, ".mdc") {
				matches = append(matches, item.Path)
			}
		}
	}

	// Recursively process subdirectories if glob pattern indicates we should
	// (check the original pattern string for wildcard characters)
	if strings.Contains(pattern, "**") || strings.Contains(pattern, "/") {
		for _, subDir := range subDirs {
			subMatches, err := recursivelyListGitHubFiles(ctx, owner, repo, ref, subDir, g, pattern)
			if err != nil {
				// Just log errors for subdirectories but don't fail the entire operation
				fmt.Printf("Warning: Error listing files in %s: %v\n", subDir, err)
				continue
			}
			matches = append(matches, subMatches...)
		}
	}

	return matches, nil
}
