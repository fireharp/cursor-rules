# Setting Up Homebrew Tap for cursor-rules

This document explains how to set up a Homebrew tap repository for the cursor-rules CLI tool.

## 1. Create the Homebrew Tap Repository

1. Go to GitHub and create a new repository named `homebrew-tap`

   - The repository must be public
   - The repository name must start with `homebrew-`
   - Initialize with a README

2. Clone the repository locally:

   ```bash
   git clone https://github.com/fireharp/homebrew-tap.git
   cd homebrew-tap
   ```

3. Create a `Formula` directory:

   ```bash
   mkdir -p Formula
   ```

4. Commit and push the directory:

   ```bash
   git add Formula
   git commit -m "Add Formula directory"
   git push
   ```

## 2. Configure GitHub Token Permissions

The GitHub token used by GoReleaser needs permissions to push to the tap repository:

1. Go to GitHub -> Settings -> Developer settings -> Personal access tokens
2. Create a new token with the `repo` scope
3. Save this token securely

## 3. Set Up GitHub Actions for Release

In your release workflow, make sure the GitHub token has the right permissions:

```yaml
permissions:
  contents: write # For creating the release
  packages: write # If you're using GitHub Packages
```

## 4. Create a Release

To create a new release that will automatically update the Homebrew formula:

```bash
git tag v0.1.0
git push origin v0.1.0
```

The GitHub Actions workflow will run GoReleaser, which will:

1. Build the binaries
2. Create a GitHub release
3. Generate the Homebrew formula
4. Push the formula to your homebrew-tap repository

## 5. Install via Homebrew

Once the release is done, users can install cursor-rules with:

```bash
# Add the tap
brew tap fireharp/tap

# Install cursor-rules
brew install cursor-rules
# or
brew install fireharp/tap/cursor-rules
```

## Troubleshooting

- If the formula doesn't get updated, check the GitHub Actions logs
- Verify that your GitHub token has the right permissions
- Make sure the repository names in the GoReleaser config match your GitHub username
