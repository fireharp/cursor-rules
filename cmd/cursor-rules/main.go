package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/fireharp/cursor-rules/pkg/manager"
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

	// Define flag sets for new subcommands
	addCmd := flag.NewFlagSet("add", flag.ExitOnError)

	addRefCmd := flag.NewFlagSet("add-ref", flag.ExitOnError)

	removeCmd := flag.NewFlagSet("remove", flag.ExitOnError)
	upgradeCmd := flag.NewFlagSet("upgrade", flag.ExitOnError)

	listCmd := flag.NewFlagSet("list", flag.ExitOnError)
	listDetailedFlag := listCmd.Bool("detailed", false, "Show detailed information about installed rules")

	lockLocationCmd := flag.NewFlagSet("set-lock-location", flag.ExitOnError)
	useRootFlag := lockLocationCmd.Bool("root", false, "Use project root for lockfile location (if false, uses .cursor/rules)")

	// Add share and restore commands
	shareCmd := flag.NewFlagSet("share", flag.ExitOnError)
	shareOutputFlag := shareCmd.String("output", "cursor-rules-share.json", "Output file path for the shareable file")
	shareEmbedFlag := shareCmd.Bool("embed", false, "Embed .mdc content for local references")

	restoreCmd := flag.NewFlagSet("restore", flag.ExitOnError)
	restoreAutoResolveFlag := restoreCmd.String("auto-resolve", "", "Automatically resolve conflicts (options: skip, overwrite, rename)")

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

	// Handle subcommands
	if len(args) > 0 {
		switch args[0] {
		case "add":
			// Usage: cursor-rules add <ruleKey or reference>
			_ = addCmd.Parse(args[1:])
			if addCmd.NArg() < 1 {
				fmt.Println("Usage: cursor-rules add <reference>")
				fmt.Println("  where <reference> can be:")
				fmt.Println("  - Local file path: /path/to/rule.mdc or ./relative/path.mdc")
				fmt.Println("  - GitHub file: https://github.com/user/repo/blob/main/path/to/rule.mdc")
				fmt.Println("  - GitHub directory: https://github.com/user/repo/tree/main/rules/")
				return
			}
			reference := addCmd.Arg(0)

			// Always use AddRuleByReference for all add operations
			if err := manager.AddRuleByReference(cursorDir, reference); err != nil {
				fmt.Printf("Error adding rule from reference: %v\n", err)
			} else {
				fmt.Printf("Rule from %q added successfully\n", reference)
			}
			return

		case "add-ref":
			// Usage: cursor-rules add-ref <reference>
			// Reference can be:
			// - Local file path (absolute or relative)
			// - GitHub URL (blob or tree)
			_ = addRefCmd.Parse(args[1:])
			if addRefCmd.NArg() < 1 {
				fmt.Println("Usage: cursor-rules add-ref <reference>")
				fmt.Println("  where <reference> can be:")
				fmt.Println("  - Local file path: /path/to/rule.mdc or ./relative/path.mdc")
				fmt.Println("  - GitHub file: https://github.com/user/repo/blob/main/path/to/rule.mdc")
				fmt.Println("  - GitHub directory: https://github.com/user/repo/tree/main/rules/")
				return
			}
			reference := addRefCmd.Arg(0)
			if err := manager.AddRuleByReference(cursorDir, reference); err != nil {
				fmt.Printf("Error adding rule from reference: %v\n", err)
			} else {
				fmt.Printf("Rule from %q added successfully\n", reference)
			}
			return

		case "remove":
			// Usage: cursor-rules remove <ruleKey>
			_ = removeCmd.Parse(args[1:])
			if removeCmd.NArg() < 1 {
				fmt.Println("Usage: cursor-rules remove <ruleKey>")
				return
			}
			ruleKey := removeCmd.Arg(0)
			if err := manager.RemoveRule(cursorDir, ruleKey); err != nil {
				fmt.Printf("Error removing rule: %v\n", err)
			} else {
				fmt.Printf("Rule %q removed successfully.\n", ruleKey)
			}
			return

		case "upgrade":
			// Usage: cursor-rules upgrade <ruleKey>
			_ = upgradeCmd.Parse(args[1:])
			if upgradeCmd.NArg() < 1 {
				fmt.Println("Usage: cursor-rules upgrade <ruleKey>")
				return
			}
			ruleKey := upgradeCmd.Arg(0)
			if err := manager.UpgradeRule(cursorDir, ruleKey); err != nil {
				fmt.Printf("Error upgrading rule: %v\n", err)
			} else {
				fmt.Printf("Rule %q upgraded successfully.\n", ruleKey)
			}
			return

		case "list":
			// Usage: cursor-rules list [--detailed]
			_ = listCmd.Parse(args[1:])

			// First sync any local rules that aren't in the lockfile
			err := manager.SyncLocalRules(cursorDir)
			if err != nil {
				fmt.Printf("Error syncing local rules: %v\n", err)
				// Continue anyway to show what's in the lockfile
			}

			if *listDetailedFlag {
				// Detailed list
				rules, err := manager.GetInstalledRules(cursorDir)
				if err != nil {
					fmt.Printf("Error listing rules: %v\n", err)
					return
				}
				if len(rules) == 0 {
					fmt.Println("No rules installed.")
				} else {
					fmt.Println("Installed rules:")
					for _, r := range rules {
						fmt.Printf("  - %s\n", r.Key)
						fmt.Printf("    Type: %s\n", r.SourceType)
						fmt.Printf("    Reference: %s\n", r.Reference)
						if r.GitRef != "" {
							fmt.Printf("    Git Ref: %s\n", r.GitRef)
						}
						if len(r.LocalFiles) > 0 {
							fmt.Printf("    Files: %s\n", strings.Join(r.LocalFiles, ", "))
						}
						fmt.Println()
					}
				}
			} else {
				// Simple list
				installed, err := manager.ListInstalledRules(cursorDir)
				if err != nil {
					fmt.Printf("Error listing rules: %v\n", err)
					return
				}
				if len(installed) == 0 {
					fmt.Println("No rules installed.")
				} else {
					fmt.Println("Installed rules:")
					for _, r := range installed {
						fmt.Printf("  - %s\n", r)
					}
				}
			}
			return

		case "set-lock-location":
			// Usage: cursor-rules set-lock-location [--root]
			_ = lockLocationCmd.Parse(args[1:])

			// Set the lock file location
			newPath, err := manager.SetLockFileLocation(cursorDir, *useRootFlag)
			if err != nil {
				fmt.Printf("Error setting lockfile location: %v\n", err)
				return
			}

			location := "project root"
			if !*useRootFlag {
				location = ".cursor/rules directory"
			}

			fmt.Printf("Lock file location set to %s\n", location)
			fmt.Printf("Lock file path: %s\n", newPath)
			return

		case "share":
			// Usage: cursor-rules share [--output file] [--embed]
			_ = shareCmd.Parse(args[1:])
			outputPath := *shareOutputFlag
			embedContent := *shareEmbedFlag

			if err := manager.ShareRules(cursorDir, outputPath, embedContent); err != nil {
				fmt.Printf("Error sharing rules: %v\n", err)
				return
			}

			if embedContent {
				fmt.Printf("Rules shared with embedded content to %s\n", outputPath)
			} else {
				fmt.Printf("Rules shared to %s\n", outputPath)
			}
			return

		case "restore":
			// Usage: cursor-rules restore <file> [--auto-resolve (skip|overwrite|rename)]
			_ = restoreCmd.Parse(args[1:])
			if restoreCmd.NArg() < 1 {
				fmt.Println("Usage: cursor-rules restore <file> [--auto-resolve (skip|overwrite|rename)]")
				return
			}
			sharedFilePath := restoreCmd.Arg(0)
			autoResolve := *restoreAutoResolveFlag

			// Validate auto-resolve option
			if autoResolve != "" && autoResolve != "skip" && autoResolve != "overwrite" && autoResolve != "rename" {
				fmt.Println("Invalid auto-resolve option. Must be one of: skip, overwrite, rename")
				return
			}

			if err := manager.RestoreFromShared(cursorDir, sharedFilePath, autoResolve); err != nil {
				fmt.Printf("Error restoring rules: %v\n", err)
				return
			}

			fmt.Println("Rules successfully restored")
			return

		case "init":
			// Handle init command
			runInitCommand(cursorDir)
			return

		case "setup":
			// Handle setup command
			setupProject(cwd, cursorDir)
			return
		}
	}

	// Handle init flag
	if *initFlag || command == "init" {
		// Also create CR_SETUP as an alias for CursorRules.setup
		// Add only the init.mdc template
		runInitCommand(cursorDir)
		return
	}

	// Handle setup flag
	if *setupFlag || command == "setup" {
		setupProject(cwd, cursorDir)
		return
	}

	// If no specific command, show help
	showHelp()
}

// Show help information for the cursor-rules command
func showHelp() {
	fmt.Println("\nUsage:")
	fmt.Println("  cursor-rules [command] [flags]")
	fmt.Println("\nCommands:")
	fmt.Println("  init                Initialize Cursor Rules with the init template")
	fmt.Println("  setup               Auto-detect project type, then add rules")
	fmt.Println("  add <reference>     Add a rule from a reference (local file or GitHub URL)")
	fmt.Println("  add-ref <reference> Add a rule from a reference (alias for 'add')")
	fmt.Println("  remove <rule>       Remove an installed rule")
	fmt.Println("  upgrade <rule>      Reinstall / upgrade a rule")
	fmt.Println("  list [--detailed]   List installed rules (--detailed for more info)")
	fmt.Println("  set-lock-location   Set the location of the lockfile (--root for project root)")
	fmt.Println("  share [--output <file>] [--embed]  Share rules to a file (--embed for local rule content)")
	fmt.Println("  restore <file> [--auto-resolve <option>]  Restore rules from a shared file")
	fmt.Println("  --init              Same as the 'init' command")
	fmt.Println("  --setup             Same as the 'setup' command")
	fmt.Println("  --version           Print version information")
	fmt.Println("\nExamples:")
	fmt.Println("  cursor-rules init")
	fmt.Println("  cursor-rules setup")
	fmt.Println("  cursor-rules add ./custom-rules/my-rule.mdc")
	fmt.Println("  cursor-rules add https://github.com/user/repo/blob/main/rules/python.mdc")
	fmt.Println("  cursor-rules add-ref /Users/me/custom-rule.mdc")
	fmt.Println("  cursor-rules add-ref https://github.com/user/repo/blob/main/rules/python.mdc")
	fmt.Println("  cursor-rules remove python")
	fmt.Println("  cursor-rules list --detailed")
	fmt.Println("  cursor-rules set-lock-location --root")
	fmt.Println("  cursor-rules share --output my-rules.json --embed")
	fmt.Println("  cursor-rules restore shared-rules.json --auto-resolve overwrite")
}

// runInitCommand initializes cursor rules with just the init template
func runInitCommand(cursorDir string) {
	// Get the init template from the general category
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

	// Update the template in the global map with the modified content
	templates.Categories["general"].Templates["init"] = initTemplate

	// Write init template to filesystem and add it by reference
	initPath := filepath.Join(cursorDir, "init.mdc")
	err := os.WriteFile(initPath, []byte(initTemplate.Content), 0644)
	if err != nil {
		fmt.Printf("Error writing init template: %v\n", err)
		os.Exit(1)
	}

	// Add the init template using the reference-based approach
	if err := manager.AddRuleByReference(cursorDir, initPath); err != nil {
		fmt.Printf("Error creating init template: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Added init template. Run CursorRules.setup or CR_SETUP in Cursor to continue setup.")
}

// setupProject detects project type and sets up appropriate rules
func setupProject(projectDir, cursorDir string) {
	fmt.Println("Detecting project type...")

	// Get the setup template from the general category
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

	// Update the template in the global map with the modified content
	templates.Categories["general"].Templates["setup"] = setupTemplate

	// Write setup template to filesystem so we can add it by reference
	setupPath := filepath.Join(cursorDir, "setup.mdc")
	err := os.WriteFile(setupPath, []byte(setupTemplate.Content), 0644)
	if err != nil {
		fmt.Printf("Error writing setup template: %v\n", err)
		return
	}

	// Add setup template using reference
	if err := manager.AddRuleByReference(cursorDir, setupPath); err != nil {
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
			if !ok {
				fmt.Println("Error: React template not found")
				return
			}
			reactPath := filepath.Join(cursorDir, "react.mdc")
			err = os.WriteFile(reactPath, []byte(reactTemplate.Content), 0644)
			if err != nil {
				fmt.Printf("Error writing react template: %v\n", err)
				return
			}

			if err := manager.AddRuleByReference(cursorDir, reactPath); err != nil {
				fmt.Printf("Error creating React template: %v\n", err)
			} else {
				fmt.Println("Added React template.")
			}
		}
	}

	// Check for Python project
	if fileExists(filepath.Join(projectDir, "setup.py")) ||
		fileExists(filepath.Join(projectDir, "requirements.txt")) ||
		fileExists(filepath.Join(projectDir, "pyproject.toml")) {
		fmt.Println("Detected Python project.")

		pythonTemplate, ok := templates.Categories["languages"].Templates["python"]
		if !ok {
			fmt.Println("Error: Python template not found")
			return
		}
		pythonPath := filepath.Join(cursorDir, "python.mdc")
		err = os.WriteFile(pythonPath, []byte(pythonTemplate.Content), 0644)
		if err != nil {
			fmt.Printf("Error writing python template: %v\n", err)
			return
		}

		if err := manager.AddRuleByReference(cursorDir, pythonPath); err != nil {
			fmt.Printf("Error creating Python template: %v\n", err)
		} else {
			fmt.Println("Added Python template.")
		}
	}

	// Write general template
	generalTemplate, ok := templates.Categories["general"].Templates["general"]
	if !ok {
		fmt.Println("Error: General template not found")
		return
	}
	generalPath := filepath.Join(cursorDir, "general.mdc")
	err = os.WriteFile(generalPath, []byte(generalTemplate.Content), 0644)
	if err != nil {
		fmt.Printf("Error writing general template: %v\n", err)
		return
	}

	// Add general template
	if err := manager.AddRuleByReference(cursorDir, generalPath); err != nil {
		fmt.Printf("Error creating general template: %v\n", err)
	} else {
		fmt.Println("Added general template.")
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
