package manager

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
)

// handleGitHubBlob handles a GitHub blob URL reference.
func handleGitHubBlob(ctx context.Context, cursorDir, ref string) (RuleSource, error) {
	// Parse the URL to extract owner, repo, commit/branch, and path
	matches := githubBlobPattern.FindStringSubmatch(ref)
	if len(matches) != 5 {
		return RuleSource{}, fmt.Errorf("invalid GitHub URL format: %s", ref)
	}

	owner := matches[1]
	repo := matches[2]
	gitRef := matches[3]
	path := matches[4]

	// Generate the rule key (owner-repo-filename)
	key := generateRuleKey(ref)

	// Create the raw URL for downloading the file
	rawURL := fmt.Sprintf("https://raw.githubusercontent.com/%s/%s/%s/%s", owner, repo, gitRef, path)

	// Create request with context
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return RuleSource{}, fmt.Errorf("failed to create request for GitHub file: %w", err)
	}

	// Download the file
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return RuleSource{}, fmt.Errorf("failed to download GitHub file: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return RuleSource{}, fmt.Errorf("HTTP error %d: %s", resp.StatusCode, resp.Status)
	}

	// Read the content
	content, err := io.ReadAll(resp.Body)
	if err != nil {
		return RuleSource{}, fmt.Errorf("failed to read GitHub file content: %w", err)
	}

	// Write to .cursor/rules/key.mdc
	targetPath := filepath.Join(cursorDir, key+".mdc")

	// Ensure parent directories exist for hierarchical keys
	if err := ensureRuleDirectory(cursorDir, key); err != nil {
		return RuleSource{}, fmt.Errorf("failed preparing directory for rule '%s': %w", key, err)
	}

	err = os.WriteFile(targetPath, content, 0o644)
	if err != nil {
		return RuleSource{}, fmt.Errorf("failed to write rule file: %w", err)
	}

	// Determine if this is a branch or a commit
	resolvedCommit := ""
	gitRefType := "commit="
	if isGitCommitHash(gitRef) {
		gitRefType = "commit="
	} else {
		gitRefType = "branch="
		// For branches, we should resolve the commit hash for reproducibility
		resolvedCommit, err = getHeadCommitForBranch(ctx, owner, repo, gitRef)
		if err != nil {
			// Not a fatal error, but log it
			fmt.Printf("Warning: Could not resolve commit hash for branch %s: %v\n", gitRef, err)
		}
	}

	// Create the rule source
	result := RuleSource{
		Key:        key,
		SourceType: SourceTypeGitHubFile,
		Reference:  ref, // The original GitHub URL
		GitRef:     gitRefType + gitRef,
		LocalFiles: []string{targetPath},
	}

	// Add resolved commit if available
	if resolvedCommit != "" {
		result.ResolvedCommit = resolvedCommit
	}

	// Calculate and store the content hash for future upgrade checks
	result.ContentSHA256 = calculateSHA256(content)

	return result, nil
}

// getHeadCommitForBranch fetches the latest commit hash for a branch.
func getHeadCommitForBranch(ctx context.Context, owner, repo, branch string) (string, error) {
	// Use the GitHub API to get the branch info
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/branches/%s", owner, repo, branch)

	// Create request with context
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request for GitHub API: %w", err)
	}

	// Add headers
	req.Header.Add("Accept", "application/vnd.github.v3+json")

	// Send the request
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to get branch information: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HTTP error %d: %s", resp.StatusCode, resp.Status)
	}

	// Parse the response
	var response struct {
		Commit struct {
			SHA string `json:"sha"`
		} `json:"commit"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return "", fmt.Errorf("failed to parse GitHub API response: %w", err)
	}

	return response.Commit.SHA, nil
}

// handleGitHubDir handles a GitHub directory URL reference.
func handleGitHubDir(cursorDir, ref string) (RuleSource, error) {
	// We don't support directories yet
	return RuleSource{}, fmt.Errorf("GitHub directory references are not yet supported: %s", ref)
}

// handleLocalFile handles a local file reference.
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
		Reference:  ref,
		LocalFiles: []string{destFilename},
		// Calculate and store content hash for future modification checks
		ContentSHA256: calculateSHA256(data),
	}

	return result, nil
}
