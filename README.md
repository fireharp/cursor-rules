# Cursor Rules Initializer

A CLI tool to help initialize and manage [Cursor Editor](https://cursor.sh/) rules for your projects.

## Quick Start

```bash
# Install with Homebrew
brew install fireharp/tap/cursor-rules

# Initialize with just the init template
cursor-rules init

# Auto-detect project type and setup appropriate rules
# DEPRECATED: The templates system will be removed in a future version
cursor-rules setup

# Share your rules with others
cursor-rules share shared-rules.json --embed-content

# Restore rules from a shared file
cursor-rules restore shared-rules.json --auto-resolve rename

# Restore rules from a URL
cursor-rules restore https://example.com/shared-rules.json --auto-resolve rename
```

## About

This tool makes it easy to set up and manage `.cursor/rules` configuration for Cursor editor. It provides:

- File-based rules management
- Interactive CLI interface
- Sharing and restoring rules between projects or users

## Usage

### Command Reference

```bash
# Initialize Cursor Rules with just the init template
# DEPRECATED: The templates system will be removed in a future version
cursor-rules init

# Auto-detect project type, then add rules
# DEPRECATED: The templates system will be removed in a future version
cursor-rules setup

# Add a rule from a local file path
cursor-rules add ./custom-rules/my-rule.mdc

# Add a rule from a GitHub URL
cursor-rules add https://github.com/username/repo/blob/main/rules/myrule.mdc

# Alternative method for adding from references (alias for 'add')
cursor-rules add-ref /Users/me/custom-rule.mdc

# Remove an installed rule
cursor-rules remove python

# Reinstall / upgrade a rule
cursor-rules upgrade python
# 'update' is an alias for 'upgrade'
cursor-rules update python

# List installed rules
cursor-rules list

# List installed rules with detailed information
cursor-rules list --detailed

# Set lockfile location (project root or .cursor/rules)
cursor-rules set-lock-location --root  # Store in project root
cursor-rules set-lock-location         # Store in .cursor/rules

# Share your rules with others
cursor-rules share shared-rules.json --embed-content

# Restore rules from a shared file
cursor-rules restore shared-rules.json --auto-resolve rename
```

### Rule References

The cursor-rules tool supports adding rules from various references, similar to how npm or pip handle dependencies:

```bash
# Add a rule from a local file (absolute path)
cursor-rules add /Users/username/rules/python-style.mdc

# Add a rule from a local file (relative path)
cursor-rules add ./custom-rules/go-style.mdc

# Add a rule from a GitHub file
cursor-rules add https://github.com/username/repo/blob/main/rules/react-style.mdc

# Add a rule from a GitHub file with specific commit
cursor-rules add https://github.com/username/repo/blob/a1b2c3d/rules/python-style.mdc
```

When rules are added from references, they can be managed just like built-in rules:

```bash
# Upgrade a rule (will re-fetch from the original source)
cursor-rules upgrade python-style

# Remove a rule added from a reference
cursor-rules remove python-style

# View detailed information about installed rules
cursor-rules list --detailed
```

### Sharing and Restoring Rules

You can easily share your rules with others or transfer them between projects:

```bash
# Share your rules to a JSON file
cursor-rules share shared-rules.json

# Share with embedded content (includes the actual .mdc file contents)
cursor-rules share shared-rules.json --embed-content

# Share to a specific output location
cursor-rules share /path/to/shared-rules.json --embed-content
```

When restoring rules from a shared file, you can specify how to handle conflicts:

```bash
# Restore rules from a shared file (will prompt for conflict resolution)
cursor-rules restore shared-rules.json

# Restore rules directly from a URL
cursor-rules restore https://example.com/shared-rules.json

# Automatically skip rules that would conflict
cursor-rules restore shared-rules.json --auto-resolve skip

# Automatically rename rules that would conflict
cursor-rules restore shared-rules.json --auto-resolve rename

# Automatically overwrite rules that would conflict
cursor-rules restore shared-rules.json --auto-resolve overwrite
```

The shared rule format is a JSON file that contains:

- Rule metadata (key, source type, reference)
- Optional embedded content of the actual .mdc files
- No sensitive information (paths are normalized)

This makes it easy to share rule configurations between team members or across different projects while maintaining privacy.

### Lockfile Location

By default, the lockfile (`cursor-rules.lock`) is stored in the `.cursor/rules` directory. However, you can configure it to be stored in your project root directory instead:

```bash
# Set lockfile location to project root
cursor-rules set-lock-location --root

# Set lockfile location back to .cursor/rules
cursor-rules set-lock-location
```

This can be useful for tracking the lockfile in version control or for better visibility of installed rules.

### Help

```bash
# Show help and usage information
cursor-rules help
```

## How It Works

Cursor Rules are stored in the `.cursor/rules` directory in your project root. Each rule is a file with a `.mdc` extension containing instructions for Cursor AI. When you use the package manager commands, a `cursor-rules.lock` file is created to track installed rules.

## Template Examples

**DEPRECATED**: The templates system is deprecated and will be removed in a future version. Instead, use direct file references:

```bash
# Add a rule from a local file
cursor-rules add ./my-rule.mdc

# Add a rule from a GitHub repository
cursor-rules add https://github.com/username/repo/blob/main/rules/my-rule.mdc
```

### File-based Rules

With Cursor Rules, you can now directly reference files:

- Local file paths (absolute or relative)
- GitHub repository references
- URLs to rule files

This approach is more flexible than the template system and allows for better customization and sharing of rules.

## Installation

### Using Homebrew

```bash
# Install directly
brew install fireharp/tap/cursor-rules

# Or, tap first and then install
brew tap fireharp/tap
brew install cursor-rules
```

### From source

```
go install github.com/fireharp/cursor-rules/cmd/cursor-rules@latest
```

## Development and Contributing

### Using Task

This project uses [Task](https://taskfile.dev/) as a task runner. First, [install Task](https://taskfile.dev/installation/).

Then you can run:

```bash
# Build the binary
task build

# Run the application
task run

# Run tests
task test

# Install the binary
task install

# Clean build artifacts
task clean
```

### Manual build

```
git clone https://github.com/fireharp/cursor-rules.git
cd cursor-rules
go build -o cursor-rules ./cmd/cursor-rules
```

### Development workflow

1. Clone the repository
2. Make your changes
3. Run tests with `task test`
4. Build and test locally with `task run`

### Adding custom rules

To add custom rules:

1. Create a Markdown document (`.mdc`) file with your rule content
2. Add it using the `add` command:

```bash
cursor-rules add path/to/your-rule.mdc
```

You can also reference rules directly from GitHub repositories:

```bash
cursor-rules add https://github.com/username/repo/blob/main/rules/my-rule.mdc
```

## License

MIT License
