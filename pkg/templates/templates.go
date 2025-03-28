package templates

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Template represents a Cursor rule template
type Template struct {
	Name        string
	Description string
	Globs       []string
	AlwaysApply bool
	Content     string
	Filename    string // Will end with .mdc extension
	Category    string
}

// TemplateCategory represents a category of templates (e.g., languages, frameworks)
type TemplateCategory struct {
	Name        string
	Description string
	Templates   map[string]Template
}

var (
	// Categories of templates
	Categories = map[string]*TemplateCategory{}
)

// LoadTemplates loads all templates from the template directories
func LoadTemplates(baseDir string) error {
	// Define categories
	Categories["languages"] = &TemplateCategory{
		Name:        "Languages",
		Description: "Templates for programming languages",
		Templates:   make(map[string]Template),
	}

	Categories["frameworks"] = &TemplateCategory{
		Name:        "Frameworks",
		Description: "Templates for frameworks and libraries",
		Templates:   make(map[string]Template),
	}

	Categories["general"] = &TemplateCategory{
		Name:        "General",
		Description: "General templates for all projects",
		Templates:   make(map[string]Template),
	}

	// Walk through the template directories
	for category, _ := range Categories {
		categoryDir := filepath.Join(baseDir, "templates", category)
		if _, err := os.Stat(categoryDir); os.IsNotExist(err) {
			fmt.Printf("Template directory not found: %s\n", categoryDir)
			continue
		}

		files, err := os.ReadDir(categoryDir)
		if err != nil {
			return fmt.Errorf("failed to read directory %s: %w", categoryDir, err)
		}

		for _, file := range files {
			if file.IsDir() || !strings.HasSuffix(file.Name(), ".mdc") {
				continue
			}

			filePath := filepath.Join(categoryDir, file.Name())
			template, err := parseTemplateFile(filePath, category)
			if err != nil {
				fmt.Printf("Warning: Failed to parse template %s: %v\n", filePath, err)
				continue
			}

			keyName := strings.TrimSuffix(file.Name(), ".mdc")
			Categories[category].Templates[keyName] = template
		}
	}

	return nil
}

// parseTemplateFile parses a template markdown file with frontmatter
func parseTemplateFile(filePath, category string) (Template, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return Template{}, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	var (
		frontMatter     = false
		frontMatterDone = false
		title           = ""
		description     = ""
		globs           []string
		alwaysApply     = false
		content         = []string{}
	)

	scanner := bufio.NewScanner(file)
	lineNum := 0

	for scanner.Scan() {
		line := scanner.Text()
		lineNum++

		// Check for frontmatter start/end
		if lineNum == 1 && line == "---" {
			frontMatter = true
			continue
		}

		if frontMatter && line == "---" {
			frontMatter = false
			frontMatterDone = true
			continue
		}

		// Parse frontmatter
		if frontMatter {
			if strings.HasPrefix(line, "title:") {
				title = strings.TrimSpace(strings.TrimPrefix(line, "title:"))
			} else if strings.HasPrefix(line, "description:") {
				description = strings.TrimSpace(strings.TrimPrefix(line, "description:"))
			} else if strings.HasPrefix(line, "glob:") {
				// For backward compatibility with single glob
				glob := strings.TrimSpace(strings.TrimPrefix(line, "glob:"))
				// Remove quotes if present
				glob = strings.Trim(glob, "\"'")
				if glob != "" {
					globs = append(globs, glob)
				}
			} else if strings.HasPrefix(line, "globs:") {
				// New plural globs field
				globsStr := strings.TrimSpace(strings.TrimPrefix(line, "globs:"))
				// Remove quotes if present
				globsStr = strings.Trim(globsStr, "\"'")
				if globsStr != "" {
					// Split by commas if multiple globs are provided
					if strings.Contains(globsStr, ",") {
						for _, g := range strings.Split(globsStr, ",") {
							globs = append(globs, strings.TrimSpace(g))
						}
					} else {
						globs = append(globs, globsStr)
					}
				}
			} else if strings.HasPrefix(line, "alwaysApply:") {
				alwaysApplyStr := strings.TrimSpace(strings.TrimPrefix(line, "alwaysApply:"))
				alwaysApply = strings.ToLower(alwaysApplyStr) == "true"
			}
		} else if frontMatterDone {
			// Content starts after frontmatter
			content = append(content, line)
		}
	}

	if err := scanner.Err(); err != nil {
		return Template{}, fmt.Errorf("error reading file: %w", err)
	}

	if title == "" {
		// Use filename as title if not specified
		title = filepath.Base(filePath)
		title = strings.TrimSuffix(title, ".mdc")
		title = strings.ReplaceAll(title, "-", " ")
		title = strings.Title(title)
	}

	filename := filepath.Base(filePath)

	return Template{
		Name:        title,
		Description: description,
		Globs:       globs,
		AlwaysApply: alwaysApply,
		Content:     strings.Join(content, "\n"),
		Filename:    filename,
		Category:    category,
	}, nil
}

// CreateTemplate writes a template to the specified directory
func CreateTemplate(targetDir string, tmpl Template) error {
	filePath := filepath.Join(targetDir, tmpl.Filename)

	// Create or truncate the file
	file, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	// Start with frontmatter
	var builder strings.Builder
	builder.WriteString("---\n")
	builder.WriteString(fmt.Sprintf("description: %s\n", tmpl.Description))

	// Write globs
	if len(tmpl.Globs) == 1 {
		builder.WriteString(fmt.Sprintf("globs: %s\n", tmpl.Globs[0]))
	} else if len(tmpl.Globs) > 1 {
		builder.WriteString("globs: ")
		for i, glob := range tmpl.Globs {
			if i > 0 {
				builder.WriteString(", ")
			}
			builder.WriteString(glob)
		}
		builder.WriteString("\n")
	}

	// Write alwaysApply
	builder.WriteString(fmt.Sprintf("alwaysApply: %t\n", tmpl.AlwaysApply))
	builder.WriteString("---\n\n")

	// Write content
	builder.WriteString(tmpl.Content)

	// Write template content
	_, err = file.WriteString(builder.String())
	if err != nil {
		return fmt.Errorf("failed to write template: %w", err)
	}

	return nil
}

// ListAvailableTemplates prints all available templates by category
func ListAvailableTemplates() {
	for _, category := range Categories {
		if len(category.Templates) > 0 {
			fmt.Printf("Available %s Templates:\n", category.Name)

			for key, tmpl := range category.Templates {
				fmt.Printf("  - %s: %s", key, tmpl.Description)

				if len(tmpl.Globs) > 0 {
					fmt.Printf(" (globs: %s", strings.Join(tmpl.Globs, ", "))
					if tmpl.AlwaysApply {
						fmt.Printf(", always apply")
					}
					fmt.Printf(")")
				} else if tmpl.AlwaysApply {
					fmt.Printf(" (always apply)")
				}

				fmt.Println()
			}
			fmt.Println()
		}
	}
}
