package manager

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// LockFileName is the file that tracks installed rules (similar to a package-lock.json).
const LockFileName = "cursor-rules.lock"

// This can be set through an environment variable or config file.
var UseRootLockFile = false

// LockFile represents the structure of the lockfile on disk
type LockFile struct {
	// For backwards compatibility
	Installed []string `json:"installed,omitempty"`

	// Enhanced structure for tracking rule sources
	Rules []RuleSource `json:"rules"`
}

// getRootDirectory returns the project root directory from the cursor rules directory.
func getRootDirectory(cursorDir string) string {
	// cursorDir is typically /path/to/project/.cursor/rules
	// We need to go up two levels to get the project root
	return filepath.Dir(filepath.Dir(cursorDir))
}

// getLockFilePath returns the path to the lockfile based on the UseRootLockFile setting.
func getLockFilePath(cursorDir string) string {
	if UseRootLockFile {
		rootDir := getRootDirectory(cursorDir)
		return filepath.Join(rootDir, LockFileName)
	}
	return filepath.Join(cursorDir, LockFileName)
}

// LoadLockFile loads the lockfile from disk, or creates a new one if it doesn't exist.
func LoadLockFile(cursorDir string) (*LockFile, error) {
	lockPath := getLockFilePath(cursorDir)
	data, err := os.ReadFile(lockPath)

	if err != nil {
		if os.IsNotExist(err) {
			// Return an empty lockfile if it doesn't exist yet
			return &LockFile{
				Installed: []string{},
				Rules:     []RuleSource{},
			}, nil
		}
		return nil, fmt.Errorf("failed to read lockfile: %w", err)
	}

	var lock LockFile
	err = json.Unmarshal(data, &lock)
	if err != nil {
		return nil, fmt.Errorf("failed to parse lockfile: %w", err)
	}

	// Migrate older lockfiles which only have the Installed field
	if len(lock.Installed) > 0 && len(lock.Rules) == 0 {
		migrated := make([]RuleSource, 0, len(lock.Installed))
		for _, key := range lock.Installed {
			migrated = append(migrated, RuleSource{
				Key:        key,
				SourceType: SourceTypeBuiltIn,
				Reference:  key,
				Category:   "",
				LocalFiles: []string{filepath.Join(cursorDir, key+".mdc")},
			})
		}
		lock.Rules = migrated
	}

	return &lock, nil
}

// Save writes the lockfile to disk.
func (lock *LockFile) Save(cursorDir string) error {
	lockPath := getLockFilePath(cursorDir)
	data, err := json.MarshalIndent(lock, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to serialize lockfile: %w", err)
	}

	err = os.WriteFile(lockPath, data, 0o644)
	if err != nil {
		return fmt.Errorf("failed to write lockfile: %w", err)
	}
	return nil
}

// SetLockFileLocation changes the lockfile location setting and ensures the lockfile exists.
func SetLockFileLocation(cursorDir string, useRoot bool) (string, error) {
	if useRoot == UseRootLockFile {
		// Nothing to do if the setting isn't changing
		return getLockFilePath(cursorDir), nil
	}

	// Load the existing lockfile
	oldUseRoot := UseRootLockFile
	oldLockPath := getLockFilePath(cursorDir)

	// Change the setting temporarily to calculate the new path
	UseRootLockFile = useRoot
	newLockPath := getLockFilePath(cursorDir)

	// Check if the old lockfile exists
	oldLockExists := false
	if _, err := os.Stat(oldLockPath); err == nil {
		oldLockExists = true
	}

	// Check if the new lockfile already exists
	newLockExists := false
	if _, err := os.Stat(newLockPath); err == nil {
		newLockExists = true
	}

	// Handle potential conflicts
	if oldLockExists && newLockExists {
		// Both files exist, we'd need to merge them or make a decision
		UseRootLockFile = oldUseRoot // Revert setting temporarily
		return "", errors.New("cannot change lockfile location: both project and global lockfiles exist")
	}

	if oldLockExists {
		// Read the old lockfile
		data, err := os.ReadFile(oldLockPath)
		if err != nil {
			UseRootLockFile = oldUseRoot // Revert setting
			return "", fmt.Errorf("failed to read existing lockfile: %w", err)
		}

		// Ensure the directory exists for the new location
		err = os.MkdirAll(filepath.Dir(newLockPath), 0o755)
		if err != nil {
			UseRootLockFile = oldUseRoot // Revert setting
			return "", fmt.Errorf("failed to create directory for new lockfile: %w", err)
		}

		// Write to the new location
		err = os.WriteFile(newLockPath, data, 0o644)
		if err != nil {
			UseRootLockFile = oldUseRoot // Revert setting
			return "", fmt.Errorf("failed to write to new lockfile location: %w", err)
		}

		// Optionally, remove the old lockfile
		err = os.Remove(oldLockPath)
		if err != nil {
			// Non-critical error
			fmt.Printf("Warning: Failed to remove old lockfile at %s: %v\n", oldLockPath, err)
		}
	} else if !newLockExists {
		// Neither file exists, create a new empty lockfile
		newLock := &LockFile{
			Installed: []string{},
			Rules:     []RuleSource{},
		}

		// Ensure the directory exists
		err := os.MkdirAll(filepath.Dir(newLockPath), 0o755)
		if err != nil {
			UseRootLockFile = oldUseRoot // Revert setting
			return "", fmt.Errorf("failed to create directory for new lockfile: %w", err)
		}

		// Save the new empty lockfile
		data, err := json.MarshalIndent(newLock, "", "  ")
		if err != nil {
			UseRootLockFile = oldUseRoot // Revert setting
			return "", fmt.Errorf("failed to serialize new lockfile: %w", err)
		}

		err = os.WriteFile(newLockPath, data, 0o644)
		if err != nil {
			UseRootLockFile = oldUseRoot // Revert setting
			return "", fmt.Errorf("failed to write new lockfile: %w", err)
		}
	}

	// The setting change is now permanent
	UseRootLockFile = useRoot
	return newLockPath, nil
}

// loadOrCreateLockFile loads the lockfile or creates a new one if it doesn't exist.
func loadOrCreateLockFile(cursorDir string) (*LockFile, error) {
	lock, err := LoadLockFile(cursorDir)
	if err != nil {
		return nil, fmt.Errorf("failed to load lockfile: %w", err)
	}
	if lock == nil {
		lock = &LockFile{
			Installed: []string{},
			Rules:     []RuleSource{},
		}
	}
	return lock, nil
}
