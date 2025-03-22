package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/fireharp/cursor-rules/pkg/templates"
)

// These variables will be set by goreleaser
var (
	version               = "dev"
	commit                = "none"
	date                  = "unknown"
	defaultCursorRulesDir = "cursor-rules"
)

func main() {
	// Define command line flags
	versionFlag := flag.Bool("version", false, "Print version information")
	initFlag := flag.Bool("init", false, "Initialize Cursor Rules with just the init template")
	setupFlag := flag.Bool("setup", false, "Run project type detection and setup appropriate rules")

	// Parse flags
	flag.Parse()

	// Check for command-style arguments (cursor-rules init, cursor-rules setup)
	args := flag.Args()
	command := ""
	if len(args) > 0 {
		command = args[0]
	}

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

	// Handle init command or flag
	if *initFlag || command == "init" {
		// Also create CR_SETUP as an alias for CursorRules.setup
		// Add only the init.mdc template
		initTemplate, ok := templates.Categories["general"].Templates["init"]
		if !ok {
			fmt.Println("Error: Init template not found")
			os.Exit(1)
		}

		// Modify the init template to include CR_SETUP as an alias
		initTemplate.Content = strings.Replace(
			initTemplate.Content,
			"Run CursorRules.setup in Cursor",
			"Run CursorRules.setup or CR_SETUP in Cursor",
			-1)

		if err := templates.CreateTemplate(cursorDir+"/"+defaultCursorRulesDir, initTemplate); err != nil {
			fmt.Printf("Error creating init template: %v\n", err)
			os.Exit(1)
		}

		fmt.Println("Added init template. Run CursorRules.setup or CR_SETUP in Cursor to continue setup.")
		return
	}

	// Handle setup command or flag
	if *setupFlag || command == "setup" {
		setupProject(cwd, cursorDir)
		return
	}

	// If no specific command, show interactive template selection
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

// setupProject detects project type and sets up appropriate rules
func setupProject(projectDir, cursorDir string) {
	fmt.Println("Detecting project type...")

	// Add setup template
	setupTemplate, ok := templates.Categories["general"].Templates["setup"]
	if !ok {
		fmt.Println("Error: Setup template not found")
		return
	}

	// Update the template to include CR_SETUP as an alias
	setupTemplate.Content = strings.Replace(
		setupTemplate.Content,
		"CursorRules.setup",
		"CursorRules.setup or CR_SETUP",
		-1)

	if err := templates.CreateTemplate(cursorDir, setupTemplate); err != nil {
		fmt.Printf("Error creating setup template: %v\n", err)
		return
	}

	fmt.Println("Added setup template.")

	// Check for npm project
	if fileExists(filepath.Join(projectDir, "package.json")) {
		fmt.Println("Detected npm/Node.js project.")

		// If it's a React project, add React template
		if hasReactDependency(filepath.Join(projectDir, "package.json")) {
			fmt.Println("Detected React dependency.")

			reactTemplate, ok := templates.Categories["frameworks"].Templates["react"]
			if ok {
				if err := templates.CreateTemplate(cursorDir, reactTemplate); err != nil {
					fmt.Printf("Error creating React template: %v\n", err)
				} else {
					fmt.Println("Added React template.")
				}
			}
		}
	}

	// Check for Python project
	if fileExists(filepath.Join(projectDir, "setup.py")) ||
		fileExists(filepath.Join(projectDir, "requirements.txt")) ||
		fileExists(filepath.Join(projectDir, "pyproject.toml")) {
		fmt.Println("Detected Python project.")

		pythonTemplate, ok := templates.Categories["languages"].Templates["python"]
		if ok {
			if err := templates.CreateTemplate(cursorDir, pythonTemplate); err != nil {
				fmt.Printf("Error creating Python template: %v\n", err)
			} else {
				fmt.Println("Added Python template.")
			}
		}
	}

	// Add general template for all projects
	generalTemplate, ok := templates.Categories["general"].Templates["general"]
	if ok {
		if err := templates.CreateTemplate(cursorDir, generalTemplate); err != nil {
			fmt.Printf("Error creating general template: %v\n", err)
		} else {
			fmt.Println("Added general template.")
		}
	}

	fmt.Println("\nCursor rules setup complete!")
}

// hasReactDependency checks if package.json contains React dependency
func hasReactDependency(packageJsonPath string) bool {
	data, err := os.ReadFile(packageJsonPath)
	if err != nil {
		return false
	}

	var packageJson map[string]interface{}
	if err := json.Unmarshal(data, &packageJson); err != nil {
		return false
	}

	// Check dependencies and devDependencies for React
	if deps, ok := packageJson["dependencies"].(map[string]interface{}); ok {
		if _, hasReact := deps["react"]; hasReact {
			return true
		}
	}

	if devDeps, ok := packageJson["devDependencies"].(map[string]interface{}); ok {
		if _, hasReact := devDeps["react"]; hasReact {
			return true
		}
	}

	return false
}

// fileExists checks if a file exists
func fileExists(filePath string) bool {
	info, err := os.Stat(filePath)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
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
