package templates

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCreateTemplate(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "cursor-rules-test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Test creating a template
	testTemplate := Template{
		Name:        "Test",
		Description: "Test template",
		Globs:       []string{"**/*.test", "src/*.test"},
		AlwaysApply: true,
		Filename:    "test.mdc",
		Content:     "# Test Template\n\n- Test rule 1\n- Test rule 2\n",
		Category:    "test",
	}

	// Create the template
	err = CreateTemplate(tempDir, testTemplate)
	if err != nil {
		t.Fatalf("Failed to create template: %v", err)
	}

	// Check if the file was created
	filePath := filepath.Join(tempDir, testTemplate.Filename)
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		t.Fatalf("Template file was not created at %s", filePath)
	}

	// Read the file content
	content, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("Failed to read template file: %v", err)
	}

	// Check if the content contains our test content
	if !strings.Contains(string(content), testTemplate.Content) {
		t.Fatalf("Template content doesn't include expected content. Content: %s", string(content))
	}

	// Check if the content contains frontmatter
	if !strings.Contains(string(content), "---") ||
		!strings.Contains(string(content), "description:") ||
		!strings.Contains(string(content), "globs:") ||
		!strings.Contains(string(content), "alwaysApply:") {
		t.Fatalf("Template content doesn't include frontmatter. Content: %s", string(content))
	}

	// Check if specific globs are in the content
	if !strings.Contains(string(content), "**/*.test") || !strings.Contains(string(content), "src/*.test") {
		t.Fatalf("Template content doesn't include expected globs. Content: %s", string(content))
	}

	// Check if alwaysApply is true
	if !strings.Contains(string(content), "alwaysApply: true") {
		t.Fatalf("Template content doesn't include expected alwaysApply. Content: %s", string(content))
	}
}

func TestParseTemplateFile(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "cursor-rules-test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a test template file
	templateContent := `---
description: UI.Vision
globs: macros/*.json
alwaysApply: false
---
when we work with UI.Vision JSON macros we should use docs from docs/ folder

# How to Use the Ui.Vision Documentation
`
	testFilePath := filepath.Join(tempDir, "test-template.mdc")
	err = os.WriteFile(testFilePath, []byte(templateContent), 0o644)
	if err != nil {
		t.Fatalf("Failed to create test template file: %v", err)
	}

	// Parse the template
	template, err := parseTemplateFile(testFilePath, "test")
	if err != nil {
		t.Fatalf("Failed to parse template file: %v", err)
	}

	// Check template fields
	if template.Description != "UI.Vision" {
		t.Errorf("Expected description 'UI.Vision', got '%s'", template.Description)
	}

	if len(template.Globs) != 1 || template.Globs[0] != "macros/*.json" {
		t.Errorf("Expected globs ['macros/*.json'], got %v", template.Globs)
	}

	if template.AlwaysApply != false {
		t.Errorf("Expected alwaysApply 'false', got '%t'", template.AlwaysApply)
	}

	if !strings.Contains(template.Content, "# How to Use the Ui.Vision Documentation") {
		t.Errorf("Expected content to contain '# How to Use the Ui.Vision Documentation', got '%s'", template.Content)
	}

	if template.Category != "test" {
		t.Errorf("Expected category 'test', got '%s'", template.Category)
	}
}

func TestParseTemplateWithMultipleGlobs(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "cursor-rules-test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create a test template file with multiple globs
	templateContent := `---
description: Multiple Globs Test
globs: *.js, src/*.jsx, components/**/*.tsx
alwaysApply: true
---
# Multiple Globs Test
`
	testFilePath := filepath.Join(tempDir, "test-multiple-globs.mdc")
	err = os.WriteFile(testFilePath, []byte(templateContent), 0o644)
	if err != nil {
		t.Fatalf("Failed to create test template file: %v", err)
	}

	// Parse the template
	template, err := parseTemplateFile(testFilePath, "test")
	if err != nil {
		t.Fatalf("Failed to parse template file: %v", err)
	}

	// Check template fields
	if len(template.Globs) != 3 {
		t.Errorf("Expected 3 globs, got %d", len(template.Globs))
	}

	expectedGlobs := []string{"*.js", "src/*.jsx", "components/**/*.tsx"}
	for i, expectedGlob := range expectedGlobs {
		if i >= len(template.Globs) || template.Globs[i] != expectedGlob {
			t.Errorf("Expected glob %s at position %d, got %s", expectedGlob, i, template.Globs[i])
		}
	}

	if template.AlwaysApply != true {
		t.Errorf("Expected alwaysApply 'true', got '%t'", template.AlwaysApply)
	}
}
