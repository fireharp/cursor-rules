package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/fireharp/cursor-rules/pkg/manager"
	"github.com/fireharp/cursor-rules/pkg/templates"
)

// These variables will be set by goreleaser.
var (
	version = "0.1.5"
	commit  = "none"
	date    = "unknown"
)

// AppFlags contains the parsed top-level command flags.
type AppFlags struct {
	versionFlag bool
	initFlag    bool
	setupFlag   bool
}

// AppFlagSets contains all the flag sets for subcommands.
type AppFlagSets struct {
	addCmd                 *flag.FlagSet
	addRefCmd              *flag.FlagSet
	removeCmd              *flag.FlagSet
	upgradeCmd             *flag.FlagSet
	updateCmd              *flag.FlagSet
	listCmd                *flag.FlagSet
	listDetailedFlag       *bool
	lockLocationCmd        *flag.FlagSet
	useRootFlag            *bool
	shareCmd               *flag.FlagSet
	shareOutputFlag        *string
	shareEmbedFlag         *bool
	restoreCmd             *flag.FlagSet
	restoreAutoResolveFlag *string
}

func main() {
	// Define flags and flag sets
	flags, flagSets := defineFlags()

	// Parse flags and get arguments
	flag.Parse()
	args := flag.Args()

	// Get command if present
	command := ""
	if len(args) > 0 {
		command = args[0]
	}

	// Handle version flag early - guard clause
	if flags.versionFlag {
		printVersion()
		return
	}

	fmt.Println("Cursor Rules Initializer")

	// Initialize environment (directories, templates)
	cwd, cursorDir, _, err := initializeEnvironment()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Initialization error: %v\n", err)
		os.Exit(1)
	}

	// Handle command-line style commands
	if len(args) > 0 {
		// Handle subcommands
		handled, err := handleCommand(cursorDir, args[0], args[1:], flagSets)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Command error: %v\n", err)
			os.Exit(1)
		}
		if handled {
			return
		}
	}

	// Handle flag-style commands
	if flags.initFlag || command == "init" {
		runInitCommand(cursorDir)
		return
	}

	if flags.setupFlag || command == "setup" {
		setupProject(cwd, cursorDir)
		return
	}

	// If no command was handled, show help
	showHelp()
}

// defineFlags sets up all command-line flags and returns the parsed flags.
func defineFlags() (AppFlags, AppFlagSets) {
	// Define top-level flags
	versionFlag := flag.Bool("version", false, "Print version information")
	initFlag := flag.Bool("init", false, "Initialize Cursor Rules with just the init template")
	setupFlag := flag.Bool("setup", false, "Run project type detection and setup appropriate rules")

	// Define flag sets for subcommands
	addCmd := flag.NewFlagSet("add", flag.ExitOnError)
	addRefCmd := flag.NewFlagSet("add-ref", flag.ExitOnError)
	removeCmd := flag.NewFlagSet("remove", flag.ExitOnError)
	upgradeCmd := flag.NewFlagSet("upgrade", flag.ExitOnError)
	updateCmd := flag.NewFlagSet("update", flag.ExitOnError)

	listCmd := flag.NewFlagSet("list", flag.ExitOnError)
	listDetailedFlag := listCmd.Bool("detailed", false, "Show detailed information about installed rules")

	lockLocationCmd := flag.NewFlagSet("set-lock-location", flag.ExitOnError)
	useRootFlag := lockLocationCmd.Bool("root", false,
		"Use project root for lockfile location (if false, uses .cursor/rules)")

	// Add share and restore commands
	shareCmd := flag.NewFlagSet("share", flag.ExitOnError)
	shareOutputFlag := shareCmd.String("output", "cursor-rules-share.json",
		"Output file path for the shareable file")
	shareEmbedFlag := shareCmd.Bool("embed", false, "Embed .mdc content for local references")

	restoreCmd := flag.NewFlagSet("restore", flag.ExitOnError)
	restoreAutoResolveFlag := restoreCmd.String("auto-resolve", "",
		"Automatically resolve conflicts (options: skip, overwrite, rename)")

	return AppFlags{
			versionFlag: *versionFlag,
			initFlag:    *initFlag,
			setupFlag:   *setupFlag,
		}, AppFlagSets{
			addCmd:                 addCmd,
			addRefCmd:              addRefCmd,
			removeCmd:              removeCmd,
			upgradeCmd:             upgradeCmd,
			updateCmd:              updateCmd,
			listCmd:                listCmd,
			listDetailedFlag:       listDetailedFlag,
			lockLocationCmd:        lockLocationCmd,
			useRootFlag:            useRootFlag,
			shareCmd:               shareCmd,
			shareOutputFlag:        shareOutputFlag,
			shareEmbedFlag:         shareEmbedFlag,
			restoreCmd:             restoreCmd,
			restoreAutoResolveFlag: restoreAutoResolveFlag,
		}
}

// initializeEnvironment sets up the environment (directories, templates).
func initializeEnvironment() (cwd, cursorDir, projectDir string, err error) {
	// Get current working directory
	cwd, err = os.Getwd()
	if err != nil {
		return "", "", "", fmt.Errorf("error getting current directory: %w", err)
	}

	// Get executable directory to find template files
	execPath, err := os.Executable()
	if err != nil {
		return "", "", "", fmt.Errorf("error getting executable path: %w", err)
	}

	execDir := filepath.Dir(execPath)
	projectDir = findProjectRoot(execDir)

	// Load templates
	err = templates.LoadTemplates(projectDir)
	if err != nil {
		return "", "", "", fmt.Errorf("error loading templates: %w", err)
	}

	// Create .cursor/rules directory if it doesn't exist
	cursorDir = filepath.Join(cwd, ".cursor", "rules")
	if err := os.MkdirAll(cursorDir, 0o755); err != nil {
		return "", "", "", fmt.Errorf("error creating directory: %w", err)
	}

	fmt.Printf("Initialized .cursor/rules directory in %s\n", cursorDir)
	return cwd, cursorDir, projectDir, nil
}

// printVersion prints the version information.
func printVersion() {
	fmt.Printf("cursor-rules version %s, commit %s, built at %s\n", version, commit, date)
}

// handleCommand processes the given command and its arguments.
func handleCommand(cursorDir, command string, args []string, flagSets AppFlagSets) (bool, error) {
	switch command {
	case "add":
		return true, handleAddCommand(cursorDir, args, flagSets.addCmd)
	case "add-ref":
		return true, handleAddRefCommand(cursorDir, args, flagSets.addRefCmd)
	case "remove":
		return true, handleRemoveCommand(cursorDir, args, flagSets.removeCmd)
	case "upgrade":
		return true, handleUpgradeCommand(cursorDir, args, flagSets.upgradeCmd)
	case "update":
		return true, handleUpdateCommand(cursorDir, args, flagSets.updateCmd)
	case "list":
		return true, handleListCommand(cursorDir, args, flagSets.listCmd, flagSets.listDetailedFlag)
	case "set-lock-location":
		return true, handleSetLockLocationCommand(cursorDir, args, flagSets.lockLocationCmd, flagSets.useRootFlag)
	case "share":
		return true, handleShareCommand(cursorDir, args, flagSets.shareCmd, flagSets.shareOutputFlag, flagSets.shareEmbedFlag)
	case "restore":
		return true, handleRestoreCommand(cursorDir, args, flagSets.restoreCmd, flagSets.restoreAutoResolveFlag)
	case "init":
		runInitCommand(cursorDir)
		return true, nil
	case "setup":
		setupProject(filepath.Dir(cursorDir), cursorDir)
		return true, nil
	}
	return false, nil
}

// Handler for the 'add' command.
func handleAddCommand(cursorDir string, args []string, cmd *flag.FlagSet) error {
	if err := cmd.Parse(args); err != nil {
		return fmt.Errorf("error parsing add command: %w", err)
	}

	if cmd.NArg() < 1 {
		fmt.Println("Usage: cursor-rules add <reference> [<reference2> ...]")
		fmt.Println("  where <reference> can be:")
		fmt.Println("  - Local file path: /path/to/rule.mdc or ./relative/path.mdc")
		fmt.Println("  - GitHub file: https://github.com/user/repo/blob/main/path/to/rule.mdc")
		fmt.Println("  - GitHub directory: https://github.com/user/repo/tree/main/rules/")
		return nil
	}

	// Process all references provided
	for i := 0; i < cmd.NArg(); i++ {
		reference := cmd.Arg(i)
		if err := manager.AddRuleByReference(cursorDir, reference); err != nil {
			return fmt.Errorf("error adding rule from reference %q: %w", reference, err)
		}
		fmt.Printf("Rule from %q added successfully\n", reference)
	}

	return nil
}

// Handler for the 'add-ref' command.
func handleAddRefCommand(cursorDir string, args []string, cmd *flag.FlagSet) error {
	if err := cmd.Parse(args); err != nil {
		return fmt.Errorf("error parsing add-ref command: %w", err)
	}

	if cmd.NArg() < 1 {
		fmt.Println("Usage: cursor-rules add-ref <reference> [<reference2> ...]")
		fmt.Println("  where <reference> can be:")
		fmt.Println("  - Local file path: /path/to/rule.mdc or ./relative/path.mdc")
		fmt.Println("  - GitHub file: https://github.com/user/repo/blob/main/path/to/rule.mdc")
		fmt.Println("  - GitHub directory: https://github.com/user/repo/tree/main/rules/")
		return nil
	}

	// Process all references provided
	for i := 0; i < cmd.NArg(); i++ {
		reference := cmd.Arg(i)
		if err := manager.AddRuleByReference(cursorDir, reference); err != nil {
			return fmt.Errorf("error adding rule from reference %q: %w", reference, err)
		}
		fmt.Printf("Rule from %q added successfully\n", reference)
	}

	return nil
}

// Handler for the 'remove' command.
func handleRemoveCommand(cursorDir string, args []string, cmd *flag.FlagSet) error {
	if err := cmd.Parse(args); err != nil {
		return fmt.Errorf("error parsing remove command: %w", err)
	}

	if cmd.NArg() < 1 {
		fmt.Println("Usage: cursor-rules remove <ruleKey>")
		return nil
	}

	ruleKey := cmd.Arg(0)
	if err := manager.RemoveRule(cursorDir, ruleKey); err != nil {
		return fmt.Errorf("error removing rule: %w", err)
	}

	fmt.Printf("Rule %q removed successfully.\n", ruleKey)
	return nil
}

// Handler for the 'upgrade' command.
func handleUpgradeCommand(cursorDir string, args []string, cmd *flag.FlagSet) error {
	if err := cmd.Parse(args); err != nil {
		return fmt.Errorf("error parsing upgrade command: %w", err)
	}

	if cmd.NArg() < 1 {
		fmt.Println("Usage: cursor-rules upgrade <ruleKey>")
		return nil
	}

	ruleKey := cmd.Arg(0)
	if err := manager.UpgradeRule(cursorDir, ruleKey); err != nil {
		return fmt.Errorf("error upgrading rule: %w", err)
	}

	fmt.Printf("Rule %q upgraded successfully.\n", ruleKey)
	return nil
}

// Handler for the 'update' command (alias for upgrade).
func handleUpdateCommand(cursorDir string, args []string, cmd *flag.FlagSet) error {
	if err := cmd.Parse(args); err != nil {
		return fmt.Errorf("error parsing update command: %w", err)
	}

	if cmd.NArg() < 1 {
		fmt.Println("Usage: cursor-rules update <ruleKey>")
		fmt.Println("  (This is an alias for 'upgrade')")
		return nil
	}

	ruleKey := cmd.Arg(0)
	if err := manager.UpgradeRule(cursorDir, ruleKey); err != nil {
		return fmt.Errorf("error updating rule: %w", err)
	}

	fmt.Printf("Rule %q updated successfully.\n", ruleKey)
	return nil
}

// Handler for the 'list' command.
func handleListCommand(cursorDir string, args []string, cmd *flag.FlagSet, detailedFlag *bool) error {
	if err := cmd.Parse(args); err != nil {
		return fmt.Errorf("error parsing list command: %w", err)
	}

	// First sync any local rules that aren't in the lockfile
	err := manager.SyncLocalRules(cursorDir)
	if err != nil {
		fmt.Printf("Error syncing local rules: %v\n", err)
		// Continue anyway to show what's in the lockfile
	}

	if *detailedFlag {
		return showDetailedList(cursorDir)
	}
	return showSimpleList(cursorDir)
}

// Shows a detailed list of installed rules.
func showDetailedList(cursorDir string) error {
	rules, err := manager.GetInstalledRules(cursorDir)
	if err != nil {
		return fmt.Errorf("error listing rules: %w", err)
	}

	if len(rules) == 0 {
		fmt.Println("No rules installed.")
		return nil
	}

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
	return nil
}

// Shows a simple list of installed rules.
func showSimpleList(cursorDir string) error {
	installed, err := manager.ListInstalledRules(cursorDir)
	if err != nil {
		return fmt.Errorf("error listing rules: %w", err)
	}

	if len(installed) == 0 {
		fmt.Println("No rules installed.")
		return nil
	}

	fmt.Println("Installed rules:")
	for _, r := range installed {
		fmt.Printf("  - %s\n", r)
	}
	return nil
}

// Handler for the 'set-lock-location' command.
func handleSetLockLocationCommand(cursorDir string, args []string, cmd *flag.FlagSet, useRootFlag *bool) error {
	if err := cmd.Parse(args); err != nil {
		return fmt.Errorf("error parsing set-lock-location command: %w", err)
	}

	// Set the lock file location
	newPath, err := manager.SetLockFileLocation(cursorDir, *useRootFlag)
	if err != nil {
		return fmt.Errorf("error setting lockfile location: %w", err)
	}

	location := "project root"
	if !*useRootFlag {
		location = ".cursor/rules directory"
	}

	fmt.Printf("Lock file location set to %s\n", location)
	fmt.Printf("Lock file path: %s\n", newPath)
	return nil
}

// Handler for the 'share' command.
func handleShareCommand(cursorDir string, args []string, cmd *flag.FlagSet, outputFlag *string, embedFlag *bool) error {
	if err := cmd.Parse(args); err != nil {
		return fmt.Errorf("error parsing share command: %w", err)
	}

	outputPath := *outputFlag
	embedContent := *embedFlag

	if err := manager.ShareRules(cursorDir, outputPath, embedContent); err != nil {
		return fmt.Errorf("error sharing rules: %w", err)
	}

	if embedContent {
		fmt.Printf("Rules shared with embedded content to %s\n", outputPath)
	} else {
		fmt.Printf("Rules shared to %s\n", outputPath)
	}
	return nil
}

// Handler for the 'restore' command.
func handleRestoreCommand(cursorDir string, args []string, cmd *flag.FlagSet, autoResolveFlag *string) error {
	if err := cmd.Parse(args); err != nil {
		return fmt.Errorf("error parsing restore command: %w", err)
	}

	if cmd.NArg() < 1 {
		fmt.Println("Usage: cursor-rules restore <file|url> [--auto-resolve=OPTION]")
		fmt.Println("  where auto-resolve can be 'skip', 'overwrite', or 'rename'")
		return nil
	}

	sharedFilePath := cmd.Arg(0)
	autoResolve := *autoResolveFlag

	// Validate auto-resolve option
	if autoResolve != "" && autoResolve != "skip" && autoResolve != "overwrite" && autoResolve != "rename" {
		fmt.Println("Invalid auto-resolve option. Must be one of: skip, overwrite, rename")
		return nil
	}

	if err := manager.RestoreFromShared(context.Background(), cursorDir, sharedFilePath, autoResolve); err != nil {
		return fmt.Errorf("error restoring rules: %w", err)
	}

	fmt.Println("Rules successfully restored")
	return nil
}

// Show help information for the cursor-rules command.
func showHelp() {
	fmt.Println("Usage: cursor-rules [command]")
	fmt.Println("\nCommands:")
	fmt.Println("  init                           Initialize Cursor Rules with just the init template")
	fmt.Println("  setup                          Run project type detection and setup appropriate rules")
	fmt.Println("  add <reference> [<ref2> ...]   Add rule(s) from reference(s) (local file or GitHub URL)")
	fmt.Println("  add-ref <reference> [<ref2> ...] (Alias for 'add') Add rule(s) using direct reference(s)")
	fmt.Println("  remove <ruleKey>               Remove an installed rule")
	fmt.Println("  upgrade <ruleKey>              Upgrade a rule to the latest version")
	fmt.Println("  update <ruleKey>               (Alias for 'upgrade') Update a rule to the latest version")
	fmt.Println("  list [--detailed]              List installed rules, optionally with details")
	fmt.Println("  set-lock-location [--root]     Set lockfile location (default is .cursor/rules)")
	fmt.Println("  share [--output=FILE] [--embed] Generate shareable rule definitions")
	fmt.Println("  restore <file|url> [--auto-resolve=OPTION] Restore rules from shareable definitions")
	fmt.Println("                                              (file|url can be a local file path or a URL)")
	fmt.Println("\nFlags:")
	fmt.Println("  --version                      Show version information")
	fmt.Println("  --init                         Initialize Cursor Rules with just the init template")
	fmt.Println("  --setup                        Run project type detection and setup appropriate rules")
	fmt.Println("\nExamples:")
	fmt.Println("  cursor-rules add https://github.com/user/repo/blob/main/path/to/rule.mdc")
	fmt.Println("  cursor-rules add ./local/path/to/rule.mdc")
	fmt.Println("  cursor-rules add rule1.mdc rule2.mdc rule3.mdc")
	fmt.Println("  cursor-rules upgrade my-rule")
	fmt.Println("  cursor-rules list --detailed")
}

// runInitCommand initializes cursor rules with just the init template.
func runInitCommand(cursorDir string) {
	// Get the init template from the general category
	initTemplate, ok := templates.Categories["general"].Templates["init"]
	if !ok {
		fmt.Println("Error: Init template not found")
		os.Exit(1)
	}

	// Modify the init template to include CR_SETUP as an alias
	initTemplate.Content = strings.ReplaceAll(
		initTemplate.Content,
		"Run CursorRules.setup in Cursor",
		"Run CursorRules.setup or CR_SETUP in Cursor")

	// Update the template in the global map with the modified content
	templates.Categories["general"].Templates["init"] = initTemplate

	// Write init template to filesystem and add it by reference
	initPath := filepath.Join(cursorDir, "init.mdc")
	err := os.WriteFile(initPath, []byte(initTemplate.Content), 0o600)
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

// setupProject detects project type and sets up appropriate rules.
func setupProject(projectDir, cursorDir string) {
	fmt.Println("Detecting project type...")

	// Get the setup template from the general category
	setupTemplate, ok := templates.Categories["general"].Templates["setup"]
	if !ok {
		fmt.Println("Error: Setup template not found")
		return
	}

	// Update the template to include CR_SETUP as an alias
	setupTemplate.Content = strings.ReplaceAll(
		setupTemplate.Content,
		"CursorRules.setup",
		"CursorRules.setup or CR_SETUP")

	// Update the template in the global map with the modified content
	templates.Categories["general"].Templates["setup"] = setupTemplate

	// Write setup template to filesystem so we can add it by reference
	setupPath := filepath.Join(cursorDir, "setup.mdc")
	err := os.WriteFile(setupPath, []byte(setupTemplate.Content), 0o600)
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
			// Write React template to filesystem and add it by reference
			reactPath := filepath.Join(cursorDir, "react.mdc")
			err = os.WriteFile(reactPath, []byte(reactTemplate.Content), 0o600)
			if err != nil {
				fmt.Printf("Error writing React template: %v\n", err)
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
		// Write Python template to filesystem and add it by reference
		pythonPath := filepath.Join(cursorDir, "python.mdc")
		err = os.WriteFile(pythonPath, []byte(pythonTemplate.Content), 0o600)
		if err != nil {
			fmt.Printf("Error writing Python template: %v\n", err)
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
	// Add the general template for all project types
	generalPath := filepath.Join(cursorDir, "general.mdc")
	err = os.WriteFile(generalPath, []byte(generalTemplate.Content), 0o600)
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

// hasReactDependency checks if package.json contains React dependency.
func hasReactDependency(packageJSONPath string) bool {
	data, err := os.ReadFile(packageJSONPath)
	if err != nil {
		return false
	}

	var packageJSON map[string]interface{}
	if err := json.Unmarshal(data, &packageJSON); err != nil {
		return false
	}

	// Check dependencies and devDependencies for React
	if deps, ok := packageJSON["dependencies"].(map[string]interface{}); ok {
		if _, hasReact := deps["react"]; hasReact {
			return true
		}
	}

	if devDeps, ok := packageJSON["devDependencies"].(map[string]interface{}); ok {
		if _, hasReact := devDeps["react"]; hasReact {
			return true
		}
	}

	return false
}

// fileExists checks if a file exists.
func fileExists(filePath string) bool {
	info, err := os.Stat(filePath)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}

// looking for the templates directory.
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
