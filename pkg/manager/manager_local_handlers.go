package manager

// handleLocalFile handles a local file reference.
func handleLocalFile(cursorDir, ref string, isAbs bool) (RuleSource, error) {
	rule, err := processLocalFile(cursorDir, ref, isAbs)
	if err != nil {
		return RuleSource{}, err
	}

	// Keep the original reference
	rule.Reference = ref

	Debugf("handleLocalFile completed with rule key: '%s'", rule.Key)
	return rule, nil
}
