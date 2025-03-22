package templates

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// CreateCustomTemplate interactively creates a custom template
func CreateCustomTemplate(targetDir string) error {
	reader := bufio.NewReader(os.Stdin)

	// Get template name
	fmt.Print("Enter a name for your custom template: ")
	name, _ := reader.ReadString('\n')
	name = strings.TrimSpace(name)

	if name == "" {
		return fmt.Errorf("template name cannot be empty")
	}

	// Get description
	fmt.Print("Enter a description: ")
	description, _ := reader.ReadString('\n')
	description = strings.TrimSpace(description)

	// Get glob patterns
	fmt.Print("Enter glob patterns (comma-separated, e.g., '**/*.py, src/*.js'): ")
	globsInput, _ := reader.ReadString('\n')
	globsInput = strings.TrimSpace(globsInput)

	var globs []string
	if globsInput != "" {
		// Split by commas
		for _, g := range strings.Split(globsInput, ",") {
			trimmed := strings.TrimSpace(g)
			if trimmed != "" {
				globs = append(globs, trimmed)
			}
		}
	}

	// Get alwaysApply
	fmt.Print("Always apply this template? (y/n): ")
	alwaysApplyInput, _ := reader.ReadString('\n')
	alwaysApplyInput = strings.TrimSpace(alwaysApplyInput)
	alwaysApply := strings.ToLower(alwaysApplyInput) == "y"

	// Get filename
	fmt.Print("Enter a filename (without extension): ")
	filename, _ := reader.ReadString('\n')
	filename = strings.TrimSpace(filename)

	if filename == "" {
		filename = strings.ToLower(strings.ReplaceAll(name, " ", "-"))
	}

	// Ensure it has .mdc extension
	if !strings.HasSuffix(filename, ".mdc") {
		filename += ".mdc"
	}

	fmt.Println("\nEnter template content (enter 'EOF' on a new line when finished):")
	var contentLines []string
	for {
		line, _ := reader.ReadString('\n')
		line = strings.TrimSuffix(line, "\n")

		if line == "EOF" {
			break
		}

		contentLines = append(contentLines, line)
	}

	if len(contentLines) == 0 {
		return fmt.Errorf("template content cannot be empty")
	}

	// Create the template
	customTemplate := Template{
		Name:        name,
		Description: description,
		Globs:       globs,
		AlwaysApply: alwaysApply,
		Filename:    filename,
		Content:     strings.Join(contentLines, "\n"),
		Category:    "custom",
	}

	return CreateTemplate(targetDir, customTemplate)
}

// ScanTemplatesDir scans a directory for existing templates
func ScanTemplatesDir(dir string) ([]string, error) {
	// Ensure directory exists
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return nil, nil // No templates yet
	}

	files, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory: %w", err)
	}

	var templates []string
	for _, file := range files {
		if !file.IsDir() && strings.HasSuffix(file.Name(), ".mdc") {
			templates = append(templates, file.Name())
		}
	}

	return templates, nil
}

// ListExistingTemplates lists templates in the target directory
func ListExistingTemplates(dir string) error {
	templates, err := ScanTemplatesDir(dir)
	if err != nil {
		return err
	}

	if len(templates) == 0 {
		fmt.Println("No existing templates found.")
		return nil
	}

	fmt.Println("Existing templates:")
	for _, tmpl := range templates {
		fmt.Printf("  - %s\n", tmpl)
	}

	return nil
}
