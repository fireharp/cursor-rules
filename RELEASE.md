# Release Process for cursor-rules

This document outlines the release process for the cursor-rules CLI tool, including version management, tagging, and release procedures.

## Version Management

The version is defined in multiple places that need to be kept in sync:

1. **Taskfile.yml**: `VERSION: 0.1.0` (currently out of date)
2. **Git tags**: Used for triggering GitHub Actions workflows (latest: v0.1.3)
3. **Main package**: Version is set via `-ldflags` during build

## Pre-Release Checklist

Before creating a new release:

1. Update the version in `taskfile.yml`
2. Ensure all tests pass: `task test`
3. Verify GoReleaser configuration: `goreleaser check`
4. Update CHANGELOG.md with notable changes
5. Commit all changes to the main branch

## Release Process

### Local Testing

```bash
# Build and test locally
task all

# Verify the release configuration
goreleaser check

# Optional: Test a local release without publishing
goreleaser release --snapshot --clean
```

### Creating a Release

```bash
# 1. Update the version in taskfile.yml
# Edit taskfile.yml and update the VERSION variable

# 2. Commit version changes
git add taskfile.yml
git commit -m "Bump version to v0.x.x"
git push

# 3. Create and push a new tag
git tag v0.x.x
git push origin v0.x.x
```

This will trigger the GitHub Actions workflow defined in `.github/workflows/release.yml`, which:

1. Checks out the repository
2. Sets up Go with the correct version (1.23.4)
3. Runs the tests
4. Uses GoReleaser v2 to build and publish the release
5. Updates the Homebrew formula in the fireharp/homebrew-tap repository

## Homebrew Installation

After the release is complete, users can install cursor-rules using:

```bash
brew install fireharp/tap/cursor-rules
```

## Troubleshooting

### Common Issues

1. **GoReleaser configuration issues**:

   - Run `goreleaser check` to validate the configuration
   - Ensure version in .goreleaser.yaml matches the installed GoReleaser

2. **Homebrew tap update failures**:

   - Verify that the `HOMEBREW_TAP_TOKEN` secret has repository access permissions
   - Check GitHub Actions logs for specific errors

3. **Version mismatch**:
   - Ensure version is consistent across taskfile.yml, git tags, and main package

## Development Process

The complete workflow for developing and releasing a new version is:

1. Develop features and fix bugs
2. Update tests and documentation
3. Update version in taskfile.yml
4. Run `task all` to validate the build
5. Commit all changes and push to main
6. Create and push a new tag (v0.x.x)
7. Verify the GitHub Actions workflow completes successfully
8. Verify the Homebrew formula is updated

## Notes

- GoReleaser is configured to use version 2 of its configuration format
- The GitHub workflow uses a Personal Access Token (PAT) stored as `HOMEBREW_TAP_TOKEN` for cross-repository access
- Release artifacts are automatically uploaded to GitHub Releases
- Releases include binaries for multiple platforms (Windows, macOS, Linux) and architectures
