# Cursor Rules Initializer

A CLI tool to help initialize and manage [Cursor Editor](https://cursor.sh/) rules for your projects.

## Quick Start

```bash
# Install with Homebrew
brew install fireharp/tap/cursor-rules

# Initialize with just the init template
cursor-rules init

# Auto-detect project type and setup appropriate rules
cursor-rules setup
```

## About

This tool makes it easy to set up and manage `.cursor/rules` configuration for Cursor editor. It provides:

- Pre-defined templates for various languages (Python, Go, etc.)
- Pre-defined templates for frameworks (React, etc.)
- Ability to create custom rule templates
- Interactive CLI interface

## Usage

### Command Reference

```bash
# Initialize Cursor Rules with just the init template
cursor-rules init

# Auto-detect project type, then add rules
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

# List installed rules
cursor-rules list

# List installed rules with detailed information
cursor-rules list --detailed

# Set lockfile location (project root or .cursor/rules)
cursor-rules set-lock-location --root  # Store in project root
cursor-rules set-lock-location         # Store in .cursor/rules
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

### Language templates

Templates tailored for specific programming languages, including:

- Python
- Go
- (more coming soon)

### Framework templates

Templates tailored for specific frameworks, including:

- React
- (more coming soon)

### General template

A general template with common coding rules that apply to most projects.

### Custom templates

You can create your own templates with custom rules that fit your specific needs.

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

### Adding new templates

To add more templates:

1. Fork the repository
2. Add your templates to the `pkg/templates/templates.go` file
3. Make sure tests pass with `task test`
4. Submit a pull request

## License

MIT License
