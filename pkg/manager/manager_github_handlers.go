package manager

import (
	"context"
	"fmt"
	"strings"
)

// handleUsernameRule handles a reference in the username/rule format.
// This will look for the rule in the username/cursor-rules-collection repo.
func handleUsernameRule(ctx context.Context, cursorDir, ref string) (RuleSource, error) {
	username, ruleName, ok := parseUsernameRule(ref)
	if !ok {
		return RuleSource{}, fmt.Errorf("invalid username/rule format: %s", ref)
	}

	Debugf("handleUsernameRule: username='%s', ruleName='%s'\n", username, ruleName)

	// Try to find it in username/cursor-rules-collection repo at root level only
	githubURL := fmt.Sprintf("https://github.com/%s/cursor-rules-collection/blob/main/%s.mdc", username, ruleName)
	Debugf("handleUsernameRule: trying URL: %s\n", githubURL)

	rule, err := handleGitHubBlob(ctx, cursorDir, githubURL)

	if err == nil {
		// Found in the cursor-rules-collection repo
		rule.SourceType = SourceTypeGitHubShorthand
		rule.Reference = ref // Store the original reference
		Debugf("handleUsernameRule: Found rule at primary URL\n")
		return rule, nil
	}

	Debugf("handleUsernameRule: Primary URL failed with error: %v\n", err)

	// No subfolder fallback for username/rule format
	// This is intentional - username/rule should only look for the rule at the root level
	// If users want a nested rule, they should use username/path/rule format

	return RuleSource{}, fmt.Errorf("rule not found in username/cursor-rules-collection: %s", ref)
}

// handleUsernamePathRule handles a reference in the username/path/rule format.
// This will first try in username/cursor-rules-collection, then fallback to username/repo.
func handleUsernamePathRule(ctx context.Context, cursorDir, ref string) (RuleSource, error) {
	username, pathParts, ok := parseUsernamePathRule(ref)
	if !ok || len(pathParts) < 1 {
		return RuleSource{}, fmt.Errorf("invalid username/path/rule format: %s", ref)
	}

	Debugf("handleUsernamePathRule: username='%s', pathParts=%v\n", username, pathParts)

	// First, try to interpret it as username/path/to/rule in cursor-rules-collection
	if len(pathParts) >= 1 {
		// Extract the last part as the rule name and construct the path
		ruleName := pathParts[len(pathParts)-1]
		pathToRule := strings.Join(pathParts[:len(pathParts)-1], "/")

		// If the rule already has .mdc extension, don't add it again
		ruleFile := ruleName
		if !strings.HasSuffix(ruleName, ".mdc") {
			ruleFile = ruleName + ".mdc"
		}

		var githubURL string
		if pathToRule == "" {
			githubURL = fmt.Sprintf("https://github.com/%s/cursor-rules-collection/blob/main/%s",
				username, ruleFile)
		} else {
			githubURL = fmt.Sprintf("https://github.com/%s/cursor-rules-collection/blob/main/%s/%s",
				username, pathToRule, ruleFile)
		}

		Debugf("handleUsernamePathRule: trying URL: %s\n", githubURL)

		rule, err := handleGitHubBlob(ctx, cursorDir, githubURL)
		if err == nil {
			// Found in the cursor-rules-collection repo
			rule.SourceType = SourceTypeGitHubShorthand
			rule.Reference = ref // Store the original reference
			Debugf("handleUsernamePathRule: Found rule in cursor-rules-collection\n")
			return rule, nil
		}

		Debugf("handleUsernamePathRule: cursor-rules-collection URL failed with error: %v\n", err)
	}

	// As a fallback, attempt to interpret it as username/repo/path/to/rule.mdc
	if len(pathParts) >= 2 {
		repoName := pathParts[0]
		remainingPath := strings.Join(pathParts[1:], "/")

		// If the last part already has .mdc extension, don't add it again
		if !strings.HasSuffix(remainingPath, ".mdc") {
			remainingPath += ".mdc"
		}

		githubURL := fmt.Sprintf("https://github.com/%s/%s/blob/main/%s",
			username, repoName, remainingPath)

		Debugf("handleUsernamePathRule: trying fallback URL (any repo): %s\n", githubURL)

		rule, err := handleGitHubBlob(ctx, cursorDir, githubURL)
		if err == nil {
			// Found in the other repo
			rule.SourceType = SourceTypeGitHubRepoPath
			rule.Reference = ref // Store the original reference
			Debugf("handleUsernamePathRule: Found rule in other repo\n")
			return rule, nil
		}

		Debugf("handleUsernamePathRule: fallback URL failed with error: %v\n", err)
	}

	// Extra fallback: Try to handle the special case of "username/path/rule"
	// This is specifically for fireharp/monorepo/monorepo type paths
	if len(pathParts) >= 1 {
		// Try to look for the file at "path/rule.mdc" in cursor-rules-collection
		// This creates a URL like: https://github.com/fireharp/cursor-rules-collection/blob/main/monorepo/monorepo.mdc

		// Join all path parts with /
		fullPath := strings.Join(pathParts, "/")

		// If the last part already has .mdc extension, don't add it again
		if !strings.HasSuffix(fullPath, ".mdc") {
			fullPath += ".mdc"
		}

		githubURL := fmt.Sprintf("https://github.com/%s/cursor-rules-collection/blob/main/%s",
			username, fullPath)

		Debugf("handleUsernamePathRule: trying second fallback URL (nested structure): %s\n", githubURL)

		rule, err := handleGitHubBlob(ctx, cursorDir, githubURL)
		if err == nil {
			// Found in the cursor-rules-collection repo with the nested structure
			rule.SourceType = SourceTypeGitHubShorthand
			rule.Reference = ref // Store the original reference
			Debugf("handleUsernamePathRule: Found rule in cursor-rules-collection nested structure\n")
			return rule, nil
		}

		Debugf("handleUsernamePathRule: second fallback URL failed with error: %v\n", err)
	}

	return RuleSource{}, fmt.Errorf("rule not found in any repo: %s", ref)
}

// handleUsernameRuleWithSha handles a reference in the username/rule:sha format.
func handleUsernameRuleWithSha(ctx context.Context, cursorDir, ref string) (RuleSource, error) {
	username, ruleName, sha, ok := parseUsernameRuleWithSha(ref)
	if !ok {
		return RuleSource{}, fmt.Errorf("invalid username/rule:sha format: %s", ref)
	}

	// Build GitHub URL with specific commit
	githubURL := fmt.Sprintf("https://github.com/%s/cursor-rules-collection/blob/%s/%s.mdc",
		username, sha, ruleName)

	rule, err := handleGitHubBlob(ctx, cursorDir, githubURL)
	if err == nil {
		// Found in the cursor-rules-collection repo at specific commit
		rule.SourceType = SourceTypeGitHubShorthand
		rule.Reference = ref // Store the original reference
		rule.GitRef = "commit=" + sha
		return rule, nil
	}

	// Try in a subdirectory as well
	githubURL = fmt.Sprintf("https://github.com/%s/cursor-rules-collection/blob/%s/%s/%s.mdc",
		username, sha, ruleName, ruleName)

	rule, err = handleGitHubBlob(ctx, cursorDir, githubURL)
	if err == nil {
		// Found in the cursor-rules-collection repo in a subdirectory at specific commit
		rule.SourceType = SourceTypeGitHubShorthand
		rule.Reference = ref // Store the original reference
		rule.GitRef = "commit=" + sha
		return rule, nil
	}

	return RuleSource{}, fmt.Errorf("rule not found in username/cursor-rules-collection at commit %s: %s", sha, ref)
}

// handleUsernameRuleWithTag handles a reference in the username/rule@tag format.
func handleUsernameRuleWithTag(ctx context.Context, cursorDir, ref string) (RuleSource, error) {
	username, ruleName, tag, ok := parseUsernameRuleWithTag(ref)
	if !ok {
		return RuleSource{}, fmt.Errorf("invalid username/rule@tag format: %s", ref)
	}

	// Build GitHub URL with specific tag
	githubURL := fmt.Sprintf("https://github.com/%s/cursor-rules-collection/blob/%s/%s.mdc",
		username, tag, ruleName)

	rule, err := handleGitHubBlob(ctx, cursorDir, githubURL)
	if err == nil {
		// Found in the cursor-rules-collection repo at specific tag
		rule.SourceType = SourceTypeGitHubShorthand
		rule.Reference = ref // Store the original reference
		rule.GitRef = "tag=" + tag
		return rule, nil
	}

	// Try in a subdirectory as well
	githubURL = fmt.Sprintf("https://github.com/%s/cursor-rules-collection/blob/%s/%s/%s.mdc",
		username, tag, ruleName, ruleName)

	rule, err = handleGitHubBlob(ctx, cursorDir, githubURL)
	if err == nil {
		// Found in the cursor-rules-collection repo in a subdirectory at specific tag
		rule.SourceType = SourceTypeGitHubShorthand
		rule.Reference = ref // Store the original reference
		rule.GitRef = "tag=" + tag
		return rule, nil
	}

	return RuleSource{}, fmt.Errorf("rule not found in username/cursor-rules-collection at tag %s: %s", tag, ref)
}
