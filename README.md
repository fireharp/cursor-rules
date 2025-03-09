# Cursor Rules Initializer

A CLI tool to help initialize and manage [Cursor Editor](https://cursor.sh/) rules for your projects.

## About

This tool makes it easy to set up and manage `.cursor/rules` configuration for Cursor editor. It provides:

- Pre-defined templates for various languages (Python, Go, etc.)
- Pre-defined templates for frameworks (React, etc.)
- Ability to create custom rule templates
- Interactive CLI interface

## Installation

### From source

```
go install github.com/fireharp/cursor-rules/cmd/cursor-rules@latest
```

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

## Usage

Navigate to your project directory and run:

```
cursor-rules
```

The tool will:

1. Create a `.cursor/rules` directory if it doesn't exist
2. Check for any existing templates
3. Show available template categories
4. Prompt you to select language templates
5. Prompt you to select framework templates
6. Offer to create a general rules template
7. Offer to create custom templates

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

## Development and Contributing

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