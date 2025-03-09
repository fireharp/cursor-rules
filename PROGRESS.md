# Project Progress

## TS: 2025-03-09 20:43:57 CET

## PROBLEM: Need a CLI tool for Cursor editor rules initialization with template support

WHAT WAS DONE:

- Created a CLI tool for initializing Cursor editor rules
- Implemented multiple template categories (languages, frameworks)
- Added custom template creation functionality
- Built interactive template selection
- Established project structure with main application entry point, templates package, and configuration
- Set up documentation in README.md

---

MEMO:

- Tool creates .cursor/rules directory
- Provides pre-defined templates selection
- Supports custom template creation
- Run with: `go run cmd/cursor-rules/main.go`
- Or build with: `mkdir -p bin && go build -o bin/cursor-rules ./cmd/cursor-rules`

## TS: 2025-03-09 21:10:51 CET

## PROBLEM: Need to update Template structure to support multiple globs and always-apply functionality

WHAT WAS DONE:

- Updated Template structure to include Globs array and AlwaysApply flag
- Enhanced template parsing logic for backward compatibility and new fields
- Improved template creation to support the new format
- Updated all example templates with new format including preamble text
- Created comprehensive taskfile.yml with build, test, coverage, and distribution options
- Added test cases for multiple globs parsing

---

MEMO:

- Templates now use globs (array) instead of glob (string)
- AlwaysApply flag controls whether rules always apply regardless of file type
- Format now includes frontmatter (description, globs, alwaysApply) followed by preamble and main content
- Enhanced development workflow with watch mode

## TS: 2025-03-09 21:16:23 CET

## PROBLEM: Need automated release process for the cursor-rules CLI tool

WHAT WAS DONE:

- Configured GitHub Actions workflow for GoReleaser in .github/workflows/release.yml
- Set up automatic building and releasing when new tags are pushed
- Ensured proper Go version specification (1.21)
- Added tests execution as part of the release process

---

MEMO:

- Release process triggers on tags starting with 'v' (e.g., v0.1.0)
- GoReleaser configuration already existed in .goreleaser.yaml
- To create a new release: `git tag v0.1.0 && git push origin v0.1.0`
- Releases will be available on the GitHub Releases page

## TS: 2025-03-09 21:19:45 CET

## PROBLEM: GoReleaser configuration has deprecation warnings that need to be fixed

WHAT WAS DONE:

- Fixed deprecated fields in `.goreleaser.yaml` configuration
- Simplified archive configuration to remove format_overrides and format fields
- Specified files to include in archives (LICENSE, README, CHANGELOG)
- Verified configuration with `goreleaser check`

---

MEMO:

- GoReleaser archives configuration now uses a simplified approach
- Explicit file patterns included in archives
- No more deprecation warnings when checking configuration
- Workflow files correctly configured for future releases

## TS: 2025-03-09 21:22:54 CET

## PROBLEM: Need to make cursor-rules available via Homebrew

WHAT WAS DONE:

- Added Homebrew tap configuration to GoReleaser
- Added --version flag to the CLI tool for proper Homebrew formula testing
- Configured brews section in .goreleaser.yaml with repository details
- Set up proper license and documentation for Homebrew distribution

---

MEMO:

- A separate "homebrew-tap" repository needs to be created in GitHub
- Homebrew users can install using: `brew install fireharp/tap/cursor-rules`
- GoReleaser will automatically update the Homebrew formula with each release
- Make sure the GITHUB_TOKEN has correct permissions for the tap repository

## TS: 2025-03-09 21:30:05 CET

## PROBLEM: Need to create the first official release of cursor-rules

WHAT WAS DONE:

- Set up homebrew-tap repository with proper structure
- Created placeholder formula in homebrew-tap repository
- Tagged and pushed v0.1.0 to trigger the release workflow
- Verified release workflow execution

---

MEMO:

- First official release tagged as v0.1.0
- Release workflow will build binaries for all supported platforms
- The Homebrew formula will be automatically updated
- Once the workflow completes, users can install with: `brew install fireharp/tap/cursor-rules`
- Need to verify the release artifacts and Homebrew formula after workflow completion

## TS: 2025-03-09 21:36:27 CET

## PROBLEM: GitHub Actions workflows failing due to directory structure mismatch

WHAT WAS DONE:

- Fixed GitHub Actions workflow files by removing `cd go-claude` commands
- Removed workdir parameter from GoReleaser actions
- Created and pushed a new tag (v0.1.1) to trigger the fixed workflow
- Pushed workflow fixes to the main branch

---

MEMO:

- Local repository structure had files in go-claude directory
- GitHub repository structure has files in the root
- GitHub Actions needs to run commands from the repository root
- New release v0.1.1 should build successfully with the fixed workflows

## TS: 2025-03-09 21:45:12 CET

## PROBLEM: GitHub Actions workflow failing due to GoReleaser version incompatibility

WHAT WAS DONE:

- Changed GoReleaser configuration to use version 1 instead of version 2
- Updated GitHub Actions workflow to use goreleaser-action@v6
- Changed GoReleaser version specification to use "~> v1"
- Updated Go version in workflows to match go.mod (1.23.4)
- Created a new tag (v0.1.2) to trigger the updated workflow

---

MEMO:

- GoReleaser in GitHub Actions was using a version that doesn't support config version 2
- Explicit Go version ensures consistent builds matching development environment
- Using v1.x.x of GoReleaser which is compatible with our configuration
- Upgraded GitHub Actions components to latest versions
- Maintained the same directory structure fixes from previous attempts

## TS: 2025-03-09 21:52:48 CET

## PROBLEM: GoReleaser unable to update Homebrew tap due to permission issues

WHAT WAS DONE:

- Created a Personal Access Token (PAT) with repository access permissions
- Added the token as a repository secret named HOMEBREW_TAP_TOKEN
- Updated the GitHub Actions workflow to use the PAT instead of the default GITHUB_TOKEN
- Recreated and pushed the v0.1.2 tag to trigger the updated workflow

---

MEMO:

- Default GITHUB_TOKEN only has permissions for the repository where the workflow runs
- A Personal Access Token (PAT) is required for cross-repository operations
- The PAT needs repo permissions to push to the homebrew-tap repository
- Used same version number (v0.1.2) but with fixed permissions
- The Homebrew formula should now be correctly updated in the tap repository

## TS: 2025-03-09 22:05:32 CET

## PROBLEM: GoReleaser still unable to update Homebrew tap despite using PAT

WHAT WAS DONE:

- Modified `.goreleaser.yaml` to explicitly specify `token_env: HOMEBREW_TAP_TOKEN` in the brews section
- Updated the GitHub Actions workflow to use both tokens separately:
  - `GITHUB_TOKEN` for main repository operations
  - `HOMEBREW_TAP_TOKEN` for Homebrew tap repository updates
- Prepared for another release attempt with these changes

---

MEMO:

- GoReleaser needs to use a different token for cross-repository operations
- The token must be explicitly specified in both the workflow environment variables and the GoReleaser config
- Without the explicit token_env in GoReleaser config, it defaults to using GITHUB_TOKEN
- Using separate tokens for different operations ensures proper permissions
- This should allow GoReleaser to update the Homebrew formula in the tap repository
