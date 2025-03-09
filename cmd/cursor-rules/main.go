package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/fireharp/cursor-rules/pkg/templates"
)

// These variables will be set by goreleaser
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	// Define command line flags
	versionFlag := flag.Bool("version", false, "Print version information")
	flag.Parse()

	// Handle version flag
	if *versionFlag {
		fmt.Printf("cursor-rules version %s, commit %s, built at %s\n", version, commit, date)
		return
	}

	fmt.Println("Cursor Rules Initializer")

	// Get current working directory and executable path
	cwd, err := os.Getwd()
	if err != nil {
		fmt.Printf("Error getting current directory: %v\n", err)
		os.Exit(1)
	}

	// Get executable directory to find template files
	execPath, err := os.Executable()
	if err != nil {
		fmt.Printf("Error getting executable path: %v\n", err)
		os.Exit(1)
	}

	execDir := filepath.Dir(execPath)
	projectDir := findProjectRoot(execDir)

	// Load templates
	err = templates.LoadTemplates(projectDir)
	if err != nil {
		fmt.Printf("Error loading templates: %v\n", err)
		os.Exit(1)
	}

	// Create .cursor/rules directory if it doesn't exist
	cursorDir := filepath.Join(cwd, ".cursor", "rules")
	if err := os.MkdirAll(cursorDir, 0755); err != nil {
		fmt.Printf("Error creating directory: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Initialized .cursor/rules directory in %s\n", cursorDir)

	// Check for existing templates
	fmt.Println("\nChecking for existing templates...")
	if err := templates.ListExistingTemplates(cursorDir); err != nil {
		fmt.Printf("Error listing existing templates: %v\n", err)
	}

	// Interactive template selection
	reader := bufio.NewReader(os.Stdin)

	// Iterate through each category
	for _, categoryInfo := range templates.Categories {
		if len(categoryInfo.Templates) == 0 {
			continue
		}

		fmt.Printf("\nAvailable %s templates:\n", categoryInfo.Name)
		for key, tmpl := range categoryInfo.Templates {
			fmt.Printf("  - %s: %s", key, tmpl.Description)
			if len(tmpl.Globs) > 0 {
				fmt.Printf(" (globs: %s)", strings.Join(tmpl.Globs, ", "))
			}
			if tmpl.AlwaysApply {
				fmt.Printf(" (always apply)")
			}
			fmt.Println()
		}

		// Ask which templates to add
		fmt.Printf("\nWhich %s templates would you like to add? (comma-separated, or 'all', or press Enter to skip): ", strings.ToLower(categoryInfo.Name))
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)

		if input != "" {
			templateList := strings.Split(input, ",")

			for _, templateName := range templateList {
				templateName = strings.TrimSpace(templateName)

				if templateName == "all" {
					// Add all templates in this category
					for _, tmpl := range categoryInfo.Templates {
						if err := templates.CreateTemplate(cursorDir, tmpl); err != nil {
							fmt.Printf("Error creating template %s: %v\n", tmpl.Name, err)
							continue
						}
						fmt.Printf("Added %s template\n", tmpl.Name)
					}
					break
				}

				if tmpl, ok := categoryInfo.Templates[templateName]; ok {
					if err := templates.CreateTemplate(cursorDir, tmpl); err != nil {
						fmt.Printf("Error creating template %s: %v\n", tmpl.Name, err)
						continue
					}
					fmt.Printf("Added %s template\n", tmpl.Name)
				} else {
					fmt.Printf("Unknown template: %s\n", templateName)
				}
			}
		}
	}

	// Custom template
	fmt.Print("\nWould you like to create a custom template? (y/n): ")
	customInput, _ := reader.ReadString('\n')
	customInput = strings.TrimSpace(customInput)

	if strings.ToLower(customInput) == "y" {
		if err := templates.CreateCustomTemplate(cursorDir); err != nil {
			fmt.Printf("Error creating custom template: %v\n", err)
		} else {
			fmt.Println("Added custom template")
		}
	}

	fmt.Println("\nCursor rules initialization complete!")
}

// findProjectRoot tries to find the project root directory by
// looking for the templates directory
func findProjectRoot(startDir string) string {
	// First, check if we're running from the project directory
	if _, err := os.Stat(filepath.Join(startDir, "templates")); err == nil {
		return startDir
	}

	// Check one level up
	parentDir := filepath.Dir(startDir)
	if _, err := os.Stat(filepath.Join(parentDir, "templates")); err == nil {
		return parentDir
	}

	// Check commonly used paths
	candidatePaths := []string{
		filepath.Join(startDir, ".."),
		filepath.Join(startDir, "..", ".."),
		filepath.Join(os.Getenv("GOPATH"), "src", "github.com", "fireharp", "cursor-rules"),
	}

	for _, path := range candidatePaths {
		if _, err := os.Stat(filepath.Join(path, "templates")); err == nil {
			return path
		}
	}

	// Fallback to the current working directory
	cwd, err := os.Getwd()
	if err != nil {
		return startDir
	}

	return cwd
}
