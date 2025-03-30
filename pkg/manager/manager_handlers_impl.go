package manager

import (
	"context"
	"fmt"

	"github.com/fireharp/cursor-rules/pkg/templates"
)

// updateLockfileWithRule updates the lockfile with a new rule.
// This function was extracted from the original addRuleByReferenceImpl function
// to remove the goto statement and improve code clarity.
func updateLockfileWithRule(cursorDir string, rule RuleSource) error {
	lock, err := LoadLockFile(cursorDir)
	if err != nil {
		return fmt.Errorf("failed to load lockfile: %w", err)
	}

	// Check if rule is already installed and handle conflicts/updates
	if lock.IsInstalled(rule.Key) {
		// Find the existing rule
		var existingRule RuleSource
		for _, r := range lock.Rules {
			if r.Key == rule.Key {
				existingRule = r
				break
			}
		}

		// If this is a GitHub rule, check if commits differ
		if rule.ResolvedCommit != "" && existingRule.ResolvedCommit != "" &&
			rule.ResolvedCommit != existingRule.ResolvedCommit {
			fmt.Printf("Rule '%s' is already installed but has a different version.\n", rule.Key)
			fmt.Printf("  Current commit: %s\n", existingRule.ResolvedCommit)
			fmt.Printf("  New commit: %s\n", rule.ResolvedCommit)
			fmt.Printf("To update, use: cursor-rules upgrade %s\n", rule.Key)
			return nil
		}

		// If content hash is available, compare that
		if rule.ContentSHA256 != "" && existingRule.ContentSHA256 != "" &&
			rule.ContentSHA256 != existingRule.ContentSHA256 {
			fmt.Printf("Rule '%s' is already installed but has different content.\n", rule.Key)
			fmt.Printf("To update, use: cursor-rules upgrade %s\n", rule.Key)
			return nil
		}

		// If we get here, the rule is the same or we can't determine differences
		fmt.Printf("Rule '%s' is already installed and up-to-date.\n", rule.Key)
		return nil
	}

	// Update lockfile with the new rule
	lock.Rules = append(lock.Rules, rule)
	// For backwards compatibility
	lock.Installed = append(lock.Installed, rule.Key)

	err = lock.Save(cursorDir)
	if err != nil {
		return fmt.Errorf("failed to update lockfile: %w", err)
	}

	return nil
}

// GlobPatternHandler handles glob pattern references
// Implementation of the ReferenceHandler interface for glob patterns
type GlobPatternHandler struct{}

func (h *GlobPatternHandler) CanHandle(ref string) bool {
	return isGlobPattern(ref)
}

func (h *GlobPatternHandler) Process(ctx context.Context, cursorDir, ref string) (RuleSource, error) {
	// Handle glob pattern - this updates the lockfile directly
	err := handleGlobPattern(ctx, cursorDir, ref)
	// Return empty RuleSource with no error if successful
	// This special case is handled differently than others
	return RuleSource{}, err
}

// GitHubBlobHandler handles GitHub blob URL references
// Implementation of the ReferenceHandler interface for GitHub blob URLs
type GitHubBlobHandler struct{}

func (h *GitHubBlobHandler) CanHandle(ref string) bool {
	return isGitHubBlobURL(ref)
}

func (h *GitHubBlobHandler) Process(ctx context.Context, cursorDir, ref string) (RuleSource, error) {
	return handleGitHubBlob(ctx, cursorDir, ref)
}

// GitHubTreeHandler handles GitHub tree URL references
// Implementation of the ReferenceHandler interface for GitHub tree URLs
type GitHubTreeHandler struct{}

func (h *GitHubTreeHandler) CanHandle(ref string) bool {
	return isGitHubTreeURL(ref)
}

func (h *GitHubTreeHandler) Process(ctx context.Context, cursorDir, ref string) (RuleSource, error) {
	return handleGitHubDir(cursorDir, ref)
}

// AbsolutePathHandler handles absolute path references
// Implementation of the ReferenceHandler interface for absolute file paths
type AbsolutePathHandler struct{}

func (h *AbsolutePathHandler) CanHandle(ref string) bool {
	return isAbsolutePath(ref)
}

func (h *AbsolutePathHandler) Process(ctx context.Context, cursorDir, ref string) (RuleSource, error) {
	return handleLocalFile(cursorDir, ref, true)
}

// RelativePathHandler handles relative path references
// Implementation of the ReferenceHandler interface for relative file paths
type RelativePathHandler struct{}

func (h *RelativePathHandler) CanHandle(ref string) bool {
	return isRelativePath(ref)
}

func (h *RelativePathHandler) Process(ctx context.Context, cursorDir, ref string) (RuleSource, error) {
	return handleLocalFile(cursorDir, ref, false)
}

// UsernameRuleWithShaHandler handles username/rule:sha references
// Implementation of the ReferenceHandler interface for username/rule:sha pattern
type UsernameRuleWithShaHandler struct{}

func (h *UsernameRuleWithShaHandler) CanHandle(ref string) bool {
	return isUsernameRuleWithSha(ref)
}

func (h *UsernameRuleWithShaHandler) Process(ctx context.Context, cursorDir, ref string) (RuleSource, error) {
	rule, err := handleUsernameRuleWithSha(ctx, cursorDir, ref)
	if err != nil {
		return RuleSource{}, err
	}
	rule.SourceType = SourceTypeGitHubShorthand
	rule.Reference = ref
	return rule, nil
}

// UsernameRuleWithTagHandler handles username/rule@tag references
// Implementation of the ReferenceHandler interface for username/rule@tag pattern
type UsernameRuleWithTagHandler struct{}

func (h *UsernameRuleWithTagHandler) CanHandle(ref string) bool {
	return isUsernameRuleWithTag(ref)
}

func (h *UsernameRuleWithTagHandler) Process(ctx context.Context, cursorDir, ref string) (RuleSource, error) {
	rule, err := handleUsernameRuleWithTag(ctx, cursorDir, ref)
	if err != nil {
		return RuleSource{}, err
	}
	rule.SourceType = SourceTypeGitHubShorthand
	rule.Reference = ref
	return rule, nil
}

// UsernamePathRuleHandler handles username/path/rule references
// Implementation of the ReferenceHandler interface for username/path/rule pattern
type UsernamePathRuleHandler struct{}

func (h *UsernamePathRuleHandler) CanHandle(ref string) bool {
	return isUsernamePathRule(ref)
}

func (h *UsernamePathRuleHandler) Process(ctx context.Context, cursorDir, ref string) (RuleSource, error) {
	rule, err := handleUsernamePathRule(ctx, cursorDir, ref)
	if err != nil {
		return RuleSource{}, err
	}
	rule.SourceType = SourceTypeGitHubShorthand
	rule.Reference = ref
	return rule, nil
}

// UsernameRuleHandler handles username/rule references
// Implementation of the ReferenceHandler interface for username/rule pattern
type UsernameRuleHandler struct{}

func (h *UsernameRuleHandler) CanHandle(ref string) bool {
	return isUsernameRule(ref)
}

func (h *UsernameRuleHandler) Process(ctx context.Context, cursorDir, ref string) (RuleSource, error) {
	rule, err := handleUsernameRule(ctx, cursorDir, ref)
	if err != nil {
		return RuleSource{}, err
	}
	rule.SourceType = SourceTypeGitHubShorthand
	rule.Reference = ref
	return rule, nil
}

// DefaultUsernameHandler handles references with a default username
// Implementation of the ReferenceHandler interface for fallback to default username
type DefaultUsernameHandler struct{}

func (h *DefaultUsernameHandler) CanHandle(ref string) bool {
	// Only handle if we have a default username
	defaultUsername := getDefaultUsername()
	return defaultUsername != ""
}

func (h *DefaultUsernameHandler) Process(ctx context.Context, cursorDir, ref string) (RuleSource, error) {
	defaultUsername := getDefaultUsername()

	// Use the default username for resolution
	defaultRef := defaultUsername + "/" + ref

	// Directly construct GitHub URL
	githubURL := fmt.Sprintf("https://github.com/%s/cursor-rules-collection/blob/main/%s.mdc",
		defaultUsername, ref)

	rule, err := handleGitHubBlob(ctx, cursorDir, githubURL)
	if err == nil {
		// Found in the cursor-rules-collection repo
		rule.SourceType = SourceTypeGitHubShorthand
		rule.Reference = defaultRef // Store the resolved reference
		return rule, nil
	}

	// If not found, try with potential paths (could be nested)
	githubURL = fmt.Sprintf("https://github.com/%s/cursor-rules-collection/blob/main/%s/%s.mdc",
		defaultUsername, ref, ref)

	rule, err = handleGitHubBlob(ctx, cursorDir, githubURL)
	if err == nil {
		// Found in the cursor-rules-collection repo in a subdirectory
		rule.SourceType = SourceTypeGitHubShorthand
		rule.Reference = defaultRef // Store the resolved reference
		return rule, nil
	}

	// As a last resort, try to find a template with this name
	tmpl, err := templates.FindTemplateByName(ref)
	if err == nil && tmpl.Category != "" {
		// Found a template, but we need to add it differently
		// Return a special error so the calling code knows to use AddRule instead
		return RuleSource{}, fmt.Errorf("template_found:%s:%s", tmpl.Category, ref)
	}

	return RuleSource{}, fmt.Errorf("rule not found with default username: %s", ref)
}
