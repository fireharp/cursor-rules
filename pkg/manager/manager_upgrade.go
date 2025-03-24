package manager

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/fireharp/cursor-rules/pkg/templates"
)

// findRuleToUpgrade finds a rule in the lockfile by key.
func findRuleToUpgrade(lock *LockFile, ruleKey string) (*RuleSource, error) {
	for i := range lock.Rules {
		if lock.Rules[i].Key == ruleKey {
			return &lock.Rules[i], nil
		}
	}
	return nil, fmt.Errorf("rule not found: %s", ruleKey)
}

// upgradeBuiltInRule upgrades a built-in rule.
func upgradeBuiltInRule(cursorDir string, rule *RuleSource) error {
	// Get the template content
	content, err := templates.GetTemplate(rule.Category, rule.Key)
	if err != nil {
		return fmt.Errorf("failed to get template: %w", err)
	}

	// Write to each local file
	for _, filePath := range rule.LocalFiles {
		// Ensure we're using the full path, accounting for test files
		fullPath := filePath
		if !filepath.IsAbs(filePath) {
			fullPath = filepath.Join(cursorDir, filePath)
		}

		err = os.WriteFile(fullPath, []byte(content), 0o644)
		if err != nil {
			return fmt.Errorf("failed to write rule file: %w", err)
		}
	}

	return nil
}

// checkLocalModifications checks if a rule file has been modified locally.
func checkLocalModifications(rule *RuleSource, cursorDir string) (bool, error) {
	// Skip if no content hash
	if rule.ContentSHA256 == "" {
		return false, nil
	}

	// Check each file
	for _, filePath := range rule.LocalFiles {
		currentHash, err := fileContentSHA256(filePath)
		if err != nil {
			return false, fmt.Errorf("failed to calculate file hash: %w", err)
		}

		if currentHash != rule.ContentSHA256 {
			return true, nil
		}
	}

	return false, nil
}

// promptForLocalModifications prompts the user about local modifications.
func promptForLocalModifications(filePath string) error {
	fmt.Printf("Warning: Local modifications detected in %s\n", filePath)
	fmt.Print("Do you want to overwrite your changes? (y/N): ")

	var response string
	_, err := fmt.Scanln(&response)
	if err != nil {
		// If there's an error (e.g. empty input), treat as "no"
		return fmt.Errorf("upgrade cancelled")
	}

	response = strings.ToLower(strings.TrimSpace(response))
	if response != "y" && response != "yes" {
		return fmt.Errorf("upgrade cancelled")
	}

	return nil
}

// upgradeGitHubBranchRule upgrades a GitHub rule that references a branch.
func upgradeGitHubBranchRule(cursorDir string, rule *RuleSource, gitRef, owner, repo string) error {
	// Extract the branch name
	parts := strings.Split(gitRef, "=")
	if len(parts) != 2 || parts[0] != "branch" {
		return fmt.Errorf("invalid Git reference: %s", gitRef)
	}

	branch := parts[1]

	// Get the latest commit hash
	latestCommit, err := getHeadCommitForBranch(context.Background(), owner, repo, branch)
	if err != nil {
		return fmt.Errorf("failed to get latest commit: %w", err)
	}

	// If the commit hasn't changed, nothing to do
	if rule.ResolvedCommit == latestCommit {
		fmt.Printf("Rule is already at the latest commit (%s) for branch %s\n", shortCommit(latestCommit), branch)
		return nil
	}

	// Handle local modifications if any
	hasLocalMods, err := checkLocalModifications(rule, cursorDir)
	if err != nil {
		return err
	}

	if hasLocalMods {
		err = promptForLocalModifications(rule.LocalFiles[0])
		if err != nil {
			return err
		}
	}

	// Create reference with the new commit
	oldCommit := rule.ResolvedCommit
	rule.ResolvedCommit = latestCommit

	// Re-download the file
	newURL := strings.Replace(rule.Reference, oldCommit, latestCommit, 1)

	// For each local file, fetch and save the updated content
	for _, filePath := range rule.LocalFiles {
		resp, err := http.Get(newURL)
		if err != nil {
			return fmt.Errorf("failed to download file: %w", err)
		}
		defer resp.Body.Close()

		content, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("failed to read response: %w", err)
		}

		err = os.WriteFile(filePath, content, 0o644)
		if err != nil {
			return fmt.Errorf("failed to write file: %w", err)
		}

		// Update the content hash
		rule.ContentSHA256 = calculateSHA256(content)
	}

	fmt.Printf("Updated from %s to %s on branch %s\n", shortCommit(oldCommit), shortCommit(latestCommit), branch)
	return nil
}

// upgradeGitHubPinnedRule upgrades a GitHub rule that is pinned to a specific commit.
func upgradeGitHubPinnedRule(cursorDir string, rule *RuleSource, gitRef string) error {
	parts := strings.Split(gitRef, "=")
	if len(parts) != 2 || parts[0] != "commit" {
		return fmt.Errorf("invalid Git reference: %s", gitRef)
	}

	commitHash := parts[1]
	fmt.Printf("Rule is pinned to commit %s\n", shortCommit(commitHash))
	fmt.Print("Do you want to unpin and use the latest version? (y/N): ")

	var response string
	_, err := fmt.Scanln(&response)
	if err != nil {
		// If there's an error (e.g. empty input), treat as "no"
		return fmt.Errorf("upgrade cancelled")
	}

	response = strings.ToLower(strings.TrimSpace(response))
	if response != "y" && response != "yes" {
		return fmt.Errorf("upgrade cancelled - rule remains pinned to %s", shortCommit(commitHash))
	}

	// Extract owner, repo, and path from the reference
	matches := githubBlobPattern.FindStringSubmatch(rule.Reference)
	if len(matches) != 5 {
		return fmt.Errorf("invalid GitHub URL: %s", rule.Reference)
	}

	owner := matches[1]
	repo := matches[2]

	// Change from commit to branch reference
	rule.GitRef = "branch=main"
	return upgradeGitHubBranchRule(cursorDir, rule, "branch=main", owner, repo)
}

// upgradeLocalRule upgrades a local rule.
func upgradeLocalRule(cursorDir string, rule *RuleSource) error {
	// Local rules can't be upgraded automatically
	fmt.Printf("Rule %s is from a local file source and cannot be upgraded automatically.\n", rule.Key)
	fmt.Printf("Source: %s\n", rule.Reference)
	return nil
}

// UpgradeRule upgrades a rule to the latest version.
func UpgradeRule(cursorDir, ruleKey string) error {
	// Load the lockfile
	lock, err := LoadLockFile(cursorDir)
	if err != nil {
		return fmt.Errorf("failed to load lockfile: %w", err)
	}

	// Find the rule
	rule, err := findRuleToUpgrade(lock, ruleKey)
	if err != nil {
		return err
	}

	// Handle different source types
	switch rule.SourceType {
	case SourceTypeBuiltIn:
		// For built-in rules, just reinstall from the template
		fmt.Printf("Upgrading built-in rule: %s\n", rule.Key)
		err = upgradeBuiltInRule(cursorDir, rule)

	case SourceTypeGitHubFile:
		// For GitHub rules, check if it's a branch or pinned commit
		if strings.HasPrefix(rule.GitRef, "branch=") {
			// Get owner and repo from reference
			matches := githubBlobPattern.FindStringSubmatch(rule.Reference)
			if len(matches) != 5 {
				return fmt.Errorf("invalid GitHub URL: %s", rule.Reference)
			}

			owner := matches[1]
			repo := matches[2]

			fmt.Printf("Upgrading GitHub rule from branch: %s\n", strings.Split(rule.GitRef, "=")[1])
			err = upgradeGitHubBranchRule(cursorDir, rule, rule.GitRef, owner, repo)
		} else if strings.HasPrefix(rule.GitRef, "commit=") {
			fmt.Printf("Upgrading GitHub rule from pinned commit\n")
			err = upgradeGitHubPinnedRule(cursorDir, rule, rule.GitRef)
		} else {
			return fmt.Errorf("unknown Git reference type: %s", rule.GitRef)
		}

	case SourceTypeLocalAbs, SourceTypeLocalRel:
		// Local files can't be auto-upgraded
		err = upgradeLocalRule(cursorDir, rule)

	default:
		return fmt.Errorf("unsupported source type for upgrade: %s", rule.SourceType)
	}

	if err != nil {
		return fmt.Errorf("upgrade failed: %w", err)
	}

	// Save the lockfile with any changes (like updated commit hashes)
	err = lock.Save(cursorDir)
	if err != nil {
		return fmt.Errorf("failed to save lockfile: %w", err)
	}

	fmt.Printf("Rule %s upgraded successfully\n", rule.Key)
	return nil
}
