package manager

import (
	"errors"
	"fmt"
)

// ErrReferenceType is returned when there's an issue with the reference type.
type ErrReferenceType struct {
	Reference string
	Message   string
}

func (e *ErrReferenceType) Error() string {
	return fmt.Sprintf("reference type error for '%s': %s", e.Reference, e.Message)
}

// ErrGitHubAccess is returned when there's an issue accessing GitHub.
type ErrGitHubAccess struct {
	Reference string
	Cause     error
}

func (e *ErrGitHubAccess) Error() string {
	return fmt.Sprintf("GitHub access error for '%s': %v", e.Reference, e.Cause)
}

// ErrGitHubRateLimit is returned when GitHub rate limit is exceeded.
type ErrGitHubRateLimit struct {
	Reference string
	ResetTime string
}

func (e *ErrGitHubRateLimit) Error() string {
	return fmt.Sprintf("GitHub rate limit exceeded for '%s', reset at %s", e.Reference, e.ResetTime)
}

// ErrLocalFileAccess is returned when there's an issue accessing a local file.
type ErrLocalFileAccess struct {
	Path  string
	Cause error
}

func (e *ErrLocalFileAccess) Error() string {
	return fmt.Sprintf("failed to access local file '%s': %v", e.Path, e.Cause)
}

// ErrRuleNotFound is returned when a requested rule cannot be found.
type ErrRuleNotFound struct {
	RuleKey string
}

func (e *ErrRuleNotFound) Error() string {
	return fmt.Sprintf("rule not found: %s", e.RuleKey)
}

// ErrTemplateFound is a special error indicating a template was found.
// This replaces the string-based "template_found:" error pattern.
type ErrTemplateFound struct {
	Category string
	Name     string
}

func (e *ErrTemplateFound) Error() string {
	return fmt.Sprintf("template found: %s/%s", e.Category, e.Name)
}

// IsRuleNotFoundError checks if an error is an ErrRuleNotFound.
func IsRuleNotFoundError(err error) bool {
	var notFoundErr *ErrRuleNotFound
	return errors.As(err, &notFoundErr)
}

// IsTemplateFoundError checks if an error is an ErrTemplateFound.
func IsTemplateFoundError(err error) bool {
	var templateFoundErr *ErrTemplateFound
	return errors.As(err, &templateFoundErr)
}

// IsGitHubRateLimitError checks if an error is an ErrGitHubRateLimit.
func IsGitHubRateLimitError(err error) bool {
	var rateLimitErr *ErrGitHubRateLimit
	return errors.As(err, &rateLimitErr)
}

// IsGitHubAccessError checks if an error is an ErrGitHubAccess.
func IsGitHubAccessError(err error) bool {
	var accessErr *ErrGitHubAccess
	return errors.As(err, &accessErr)
}

// IsLocalFileAccessError checks if an error is an ErrLocalFileAccess.
func IsLocalFileAccessError(err error) bool {
	var accessErr *ErrLocalFileAccess
	return errors.As(err, &accessErr)
}

// IsReferenceTypeError checks if an error is an ErrReferenceType.
func IsReferenceTypeError(err error) bool {
	var typeErr *ErrReferenceType
	return errors.As(err, &typeErr)
}
