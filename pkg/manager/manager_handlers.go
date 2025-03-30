package manager

import (
	"context"
	"fmt"
)

// ReferenceHandler defines the interface for handling different reference types.
// This is part of the Strategy Pattern implementation that replaced the large
// if/else chain in the original addRuleByReferenceImpl function.
type ReferenceHandler interface {
	// CanHandle determines if this handler can process the given reference.
	CanHandle(ref string) bool

	// Process handles the reference and returns a RuleSource if successful.
	// For glob patterns, Process should handle updating the lockfile itself
	// and return a nil RuleSource with no error to indicate successful processing.
	Process(ctx context.Context, cursorDir, ref string) (RuleSource, error)
}

// ReferenceHandlerRegistry holds registered handlers in order of specificity.
// This allows for a clean way to add new reference types without modifying
// the core addRuleByReferenceImpl function.
type ReferenceHandlerRegistry struct {
	handlers []ReferenceHandler
}

// NewReferenceHandlerRegistry creates a new registry with default handlers.
// The order of handlers is important as they are tried in sequence.
func NewReferenceHandlerRegistry() *ReferenceHandlerRegistry {
	registry := &ReferenceHandlerRegistry{
		handlers: []ReferenceHandler{
			&GlobPatternHandler{},
			&GitHubBlobHandler{},
			&GitHubTreeHandler{},
			&AbsolutePathHandler{},
			&RelativePathHandler{},
			&UsernameRuleWithShaHandler{},
			&UsernameRuleWithTagHandler{},
			&UsernamePathRuleHandler{},
			&UsernameRuleHandler{},
			&DefaultUsernameHandler{},
		},
	}
	return registry
}

// FindHandler finds the first handler that can process the given reference.
func (r *ReferenceHandlerRegistry) FindHandler(ref string) ReferenceHandler {
	for _, handler := range r.handlers {
		if handler.CanHandle(ref) {
			return handler
		}
	}
	return nil
}

// Process processes the reference using the appropriate handler.
func (r *ReferenceHandlerRegistry) Process(ctx context.Context, cursorDir, ref string) (RuleSource, error) {
	handler := r.FindHandler(ref)
	if handler == nil {
		return RuleSource{}, fmt.Errorf("unsupported reference format or rule not found: %s", ref)
	}
	return handler.Process(ctx, cursorDir, ref)
}
