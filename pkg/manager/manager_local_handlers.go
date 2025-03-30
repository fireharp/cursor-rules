package manager

import (
	"fmt"
	"os"
	"path/filepath"
)

// handleLocalFile handles a local file reference.
func handleLocalFile(cursorDir, ref string, isAbs bool) (RuleSource, error) {
	rule, err := processLocalFile(cursorDir, ref, isAbs)
	if err != nil {
		// Pass through any custom errors
		if IsLocalFileAccessError(err) || IsReferenceTypeError(err) {
			return RuleSource{}, err
		}

		// Wrap other errors in our custom error type
		return RuleSource{}, &ErrLocalFileAccess{
			Path:  ref,
			Cause: err,
		}
	}

	// Keep the original reference
	rule.Reference = ref

	Debugf("handleLocalFile completed with rule key: '%s'", rule.Key)
	return rule, nil
}

// processLocalFile processes a local file and returns a RuleSource without updating the lockfile.
// This is a helper function extracted from handleLocalFile to be used with glob patterns.
func processLocalFile(cursorDir, filePath string, isAbs bool) (RuleSource, error) {
	// 1. Validate path and ensure it's readable
	var fullPath string
	if isAbs {
		fullPath = filePath
	} else {
		var err error
		fullPath, err = filepath.Abs(filePath)
		if err != nil {
			return RuleSource{}, &ErrLocalFileAccess{
				Path:  filePath,
				Cause: err,
			}
		}
	}

	// Check if the file exists and is readable
	info, err := os.Stat(fullPath)
	if err != nil {
		return RuleSource{}, &ErrLocalFileAccess{
			Path:  fullPath,
			Cause: err,
		}
	}

	if info.IsDir() {
		return RuleSource{}, &ErrReferenceType{
			Reference: fullPath,
			Message:   "is a directory, not a file",
		}
	}

	// 2. Read the file
	data, err := os.ReadFile(fullPath)
	if err != nil {
		return RuleSource{}, &ErrLocalFileAccess{
			Path:  fullPath,
			Cause: err,
		}
	}

	// 3. Generate rule key and determine destination filename
	ruleKey := generateRuleKey(filePath)
	destFilename := ruleKey + ".mdc"
	destPath := filepath.Join(cursorDir, destFilename)

	// Ensure parent directories exist for hierarchical keys
	if err := ensureRuleDirectory(cursorDir, ruleKey); err != nil {
		return RuleSource{}, &ErrLocalFileAccess{
			Path:  cursorDir,
			Cause: fmt.Errorf("failed preparing directory for rule '%s': %w", ruleKey, err),
		}
	}

	// 4. Write to .cursor/rules
	err = os.WriteFile(destPath, data, 0o600)
	if err != nil {
		return RuleSource{}, &ErrLocalFileAccess{
			Path:  destPath,
			Cause: err,
		}
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
		Reference:  filePath,
		LocalFiles: []string{destFilename},
		// Calculate and store content hash for future modification checks
		ContentSHA256: calculateSHA256(data),
		// Store the original glob pattern
		GlobPattern: filePath,
	}

	return result, nil
}
