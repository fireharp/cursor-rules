package templates

import (
	"os"
	"path/filepath"
	"testing"
)

func TestScanTemplatesDir(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "cursor-rules-scan-test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create some test template files
	testFiles := []string{
		"test1.mdc",
		"test2.mdc",
		"test3.mdc",
	}

	for _, file := range testFiles {
		filePath := filepath.Join(tempDir, file)
		if err := os.WriteFile(filePath, []byte("test content"), 0644); err != nil {
			t.Fatalf("Failed to create test file %s: %v", file, err)
		}
	}

	// Also create a non-MDC file (should be ignored)
	nonMdcFile := filepath.Join(tempDir, "test.txt")
	if err := os.WriteFile(nonMdcFile, []byte("not a template"), 0644); err != nil {
		t.Fatalf("Failed to create non-MDC file: %v", err)
	}

	// Test scanning the directory
	templates, err := ScanTemplatesDir(tempDir)
	if err != nil {
		t.Fatalf("Failed to scan templates directory: %v", err)
	}

	// Check if all MDC files were found
	if len(templates) != len(testFiles) {
		t.Fatalf("Expected %d templates, got %d", len(testFiles), len(templates))
	}

	// Check if all expected files are in the result
	for _, file := range testFiles {
		found := false
		for _, template := range templates {
			if template == file {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected to find %s in templates, but it was not found", file)
		}
	}

	// Test scanning a non-existent directory
	nonExistentDir := filepath.Join(tempDir, "non-existent")
	nonExistentTemplates, err := ScanTemplatesDir(nonExistentDir)
	if err != nil {
		t.Fatalf("Expected nil error for non-existent directory, got: %v", err)
	}
	if nonExistentTemplates != nil {
		t.Fatalf("Expected nil templates for non-existent directory, got: %v", nonExistentTemplates)
	}
}