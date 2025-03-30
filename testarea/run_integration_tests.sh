#!/bin/bash

# -----------------------------------------------------------------------------
# Comprehensive Gum-Enhanced Integration Test Script for cursor-rules
#
# Tests local path resolution, various GitHub source formats, default username
# configuration, and rule counting. Uses Gum for enhanced output when available.
#
# Requires:
#   - Go (to build the binary)
#   - Gum (optional, https://github.com/charmbracelet/gum)
#   - Network access (for GitHub tests, unless skipped)
#   - A 'test-rules/' directory relative to script execution containing
#     test rule files (e.g., monorepo.mdc, go/go-rule.mdc, etc.)
#
# Usage:
#   ./run_integration_tests.sh                  # Run with Gum if available
#   GUM_AVAILABLE=false ./run_integration_tests.sh  # Force plain output mode
#   SKIP_GITHUB_TESTS=1 ./run_integration_tests.sh  # Skip network tests
# -----------------------------------------------------------------------------

# --- Configuration ---
set -e # Exit immediately if a command exits with a non-zero status.
set -o pipefail # Prevent errors in a pipeline from being hidden

# Trap SIGPIPE to avoid "broken pipe" errors when piping output
trap '' PIPE

# Determine paths relative to script location
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
BIN_DIR="${PROJECT_ROOT}/bin"
TEST_RULES_SOURCE_DIR="${PROJECT_ROOT}/test-rules" # Local dir containing raw *.mdc files
CMD_DIR="${PROJECT_ROOT}/cmd/cursor-rules" # Path to the cursor-rules command directory

# Test binary configuration
BINARY_NAME="cursor-rules-test-$(date +%s)" # Unique name to avoid clashes
BINARY_PATH="${BIN_DIR}/${BINARY_NAME}"

# GitHub test configurations
DEFAULT_GITHUB_USER="fireharp"
RULES_COLLECTION_REPO="cursor-rules-collection" # Default repo for username/rule format

# --- Gum Setup & Styling ---
if [ "$GUM_AVAILABLE" != "false" ]; then
  GUM_CMD=$(command -v gum)
  if [ -n "$GUM_CMD" ]; then
    GUM_AVAILABLE=true
  else
    GUM_AVAILABLE=false
    echo "Warning: Gum is not installed. Using simple logging instead." >&2
    echo "For enhanced output, install Gum: https://github.com/charmbracelet/gum#installation" >&2
  fi
else
  echo "Plain output mode enabled (GUM_AVAILABLE=false)" >&2
  GUM_AVAILABLE=false
fi

# --- Logging functions ---
log_header() { 
  if [ "$GUM_AVAILABLE" = true ]; then
    "$GUM_CMD" style --border normal --margin "1" --padding "0 1" --border-foreground 99 "$1"
  else
    echo -e "\n=== $1 ==="
  fi
}

log_success() { 
  if [ "$GUM_AVAILABLE" = true ]; then
    "$GUM_CMD" style --foreground 2 "✅ SUCCESS: $1"
  else
    echo -e "✅ SUCCESS: $1"
  fi
}

log_error() { 
  if [ "$GUM_AVAILABLE" = true ]; then
    "$GUM_CMD" style --foreground 1 "❌ ERROR: $1" >&2
  else
    echo -e "❌ ERROR: $1" >&2
  fi
  exit 1
}

log_warning() { 
  if [ "$GUM_AVAILABLE" = true ]; then
    "$GUM_CMD" style --foreground 3 "⚠️ WARNING: $1"
  else
    echo -e "⚠️ WARNING: $1"
  fi
}

log_info() { 
  if [ "$GUM_AVAILABLE" = true ]; then
    "$GUM_CMD" style --foreground 4 "ℹ️ INFO: $1"
  else
    echo -e "ℹ️ INFO: $1"
  fi
}

log_command() { 
  if [ "$GUM_AVAILABLE" = true ]; then
    "$GUM_CMD" style --foreground 6 "\$ $1"
  else
    echo -e "\$ $1"
  fi
}

log_bold() { 
  if [ "$GUM_AVAILABLE" = true ]; then
    # Quote the argument to prevent dashes from being interpreted as flags
    "$GUM_CMD" style --bold "$1"
  else
    echo -e "--- $1 ---"
  fi
}

# Gum Spin wrapper for showing progress during commands
# Usage: gum_spin "Title..." command arg1 arg2...
gum_spin() {
  local title="$1"; shift
  if [ "$GUM_AVAILABLE" = true ]; then
    "$GUM_CMD" spin --spinner dot --title "$title" --show-output -- "$@"
  else
    echo -e "Running: $title"
    "$@"
  fi
}

# --- Setup & Teardown ---
TEMP_DIR=$(mktemp -d -t cursor-rules-test-XXXXXX)
export CURSOR_RULES_DIR="$TEMP_DIR/rules"
export CURSOR_LOCK_PATH="$TEMP_DIR/lock.json"
export CURSOR_CONFIG_PATH="$TEMP_DIR/config.json" # Explicitly control config path

# Create necessary directories within the temp space
mkdir -p "$CURSOR_RULES_DIR"

# Ensure the bin directory exists
mkdir -p "$BIN_DIR"

# Ensure the local test rule directory exists
if [ ! -d "$TEST_RULES_SOURCE_DIR" ]; then
    log_error "Test rules source directory '$TEST_RULES_SOURCE_DIR' not found. Please create it and add test *.mdc files."
fi

# Ensure the cmd directory exists
if [ ! -d "$CMD_DIR" ]; then
    log_error "Command directory '$CMD_DIR' not found. Please check the path to the cursor-rules command."
fi

# Additional cleanup for testarea
clean_testarea() {
  log_info "Cleaning .cursor directory from testarea..."
  if [ -d "${SCRIPT_DIR}/.cursor" ]; then
    rm -rf "${SCRIPT_DIR}/.cursor"
    log_success "Removed .cursor directory from testarea"
  else
    log_info "No .cursor directory found in testarea"
  fi
}

# Cleanup function runs on script exit (normal or error) or interrupt
cleanup() {
  local exit_code=$? # Capture exit code
  log_header "Cleaning Up"
  # Use || true to prevent cleanup errors from masking the real test error code
  if [ -f "$BINARY_PATH" ]; then
    log_info "Attempting final rule cleanup..."
    # Remove each rule individually
    local rules=$("$BINARY_PATH" list | grep "^ *- " | sed 's/^ *- //')
    if [ -n "$rules" ]; then
      for rule in $rules; do
        log_info "Removing rule: $rule"
        "$BINARY_PATH" remove "$rule" > /dev/null 2>&1 || true
      done
    else
      log_info "No rules to remove."
    fi
    
    log_info "Removing test binary..."
    rm -f "$BINARY_PATH" || log_warning "Failed to remove test binary."
  else
    log_info "Test binary not found, skipping rule cleanup command."
  fi
  
  # Clean testarea directory
  clean_testarea
  
  log_info "Removing temporary directory: $TEMP_DIR"
  rm -rf "$TEMP_DIR" || log_warning "Failed to remove temporary directory."
  log_info "Cleanup finished."
  # Preserve original exit code
  exit $exit_code
}
trap cleanup EXIT INT TERM HUP

# --- Build Step ---
log_header "Building Test Binary"
cd "$PROJECT_ROOT"
if gum_spin "Compiling $CMD_DIR..." go build -o "$BINARY_PATH" "$CMD_DIR"; then
  log_success "Built test binary: '$BINARY_PATH'"
else
  log_error "Build failed."
fi
cd "$SCRIPT_DIR"

# --- Helper Functions ---

# Cleans all rules installed via the test binary in the temp dir
clean_all_rules() {
  log_info "Cleaning all rules in temporary directory..."
  # Run remove with each rule individually (no --all flag)
  local rules=$("$BINARY_PATH" list | grep "^ *- " | sed 's/^ *- //')
  if [ -n "$rules" ]; then
    for rule in $rules; do
      log_info "Removing rule: $rule"
      "$BINARY_PATH" remove "$rule" > /dev/null 2>&1 || true
    done
    log_info "Rules cleaned successfully."
  else
    log_info "No rules to remove."
  fi
  
  # Verify it's empty - grep output is captured more safely
  local rule_list=$("$BINARY_PATH" list)
  if echo "$rule_list" | grep -q "^ *- "; then
    local count_after_clean=$(echo "$rule_list" | grep -c "^ *- ")
    log_warning "Cleanup verification failed: $count_after_clean rules still present after cleanup."
    echo "$rule_list" | grep "^ *- " # Show remaining rules
  fi
}

# Executes a test case for the 'add' command
# Usage: test_case_add "Test Description" <expected_rule_count_after_add> command_args_for_add...
test_case_add() {
  local desc="$1"
  local expected_count="$2"
  shift 2
  local cmd_args=("$@")
  local full_cmd="$BINARY_PATH add ${cmd_args[*]}"

  # Print a sub-header for the test case - use simple echo
  log_bold "Testing: $desc"

  clean_all_rules

  log_info "Executing add command:"
  log_command "$full_cmd"

  # Run the add command manually to ensure arguments are passed correctly
  # Note we allow it to fail, and we'll check the results via 'list'
  "$BINARY_PATH" add "${cmd_args[@]}" || {
    log_warning "Command exited with non-zero status, but continuing to check results..."
  }
  log_info "Add command finished."

  log_info "Verifying installation via 'list'..."
  local list_output
  list_output=$("$BINARY_PATH" list) # Capture list output for inspection
  if [ $? -ne 0 ]; then
      log_error "Failed to list rules after adding for test: '$desc'"
  fi

  log_bold "Installed rules (from 'list'):"
  echo "$list_output" | grep --color=never "^ *- " || echo "(No rules listed)" # Use echo instead of log_info

  local installed_count
  installed_count=$(echo "$list_output" | grep -c "^ *- ")

  log_info "Verifying rule count..."
  if [ "$installed_count" -eq "$expected_count" ]; then
    log_success "Rule count matches expected ($installed_count)."
  else
    log_error "Rule count mismatch for '$desc'. Expected: $expected_count, Found: $installed_count."
    # echo "--- Full 'list' output on error ---"
    # echo "$list_output"
    # echo "------------------------------------"
  fi

  log_success "Test '$desc' completed."
  echo # Blank line for readability
}

# --- Test Execution ---
log_header "Starting Cursor Rules Integration Tests"
TIMESTAMP_START=$(date "+%Y-%m-%d %H:%M:%S %Z")
log_info "Start Time: $TIMESTAMP_START"
log_info "Using temporary rules directory: $CURSOR_RULES_DIR"
log_info "Using temporary lock file: $CURSOR_LOCK_PATH"
log_info "Using temporary config file: $CURSOR_CONFIG_PATH"

# --- Local Path Tests ---
log_header "Local Path Resolution Tests"
test_case_add "Local relative path (single file)" 1 "$TEST_RULES_SOURCE_DIR/monorepo.mdc"

# Use specific path instead of a glob
test_case_add "Local file in subdirectory" 1 "$TEST_RULES_SOURCE_DIR/go/go-rule.mdc"

# Use specific path instead of a deep glob
test_case_add "Local file in deep subdirectory" 1 "$TEST_RULES_SOURCE_DIR/go/important/important-rule.mdc"

# Adding multiple specific paths instead of globs
test_case_add "Multiple local files" 3 \
  "$TEST_RULES_SOURCE_DIR/go/go-rule.mdc" \
  "$TEST_RULES_SOURCE_DIR/mgmtn/tasks/task-rule.mdc" \
  "$TEST_RULES_SOURCE_DIR/monorepo.mdc"

# --- GitHub Path Tests ---
log_header "GitHub Source Resolution Tests"
if [ -n "$SKIP_GITHUB_TESTS" ]; then
  log_warning "Skipping GitHub Tests (SKIP_GITHUB_TESTS is set)"
else
  log_info "--- Section: GitHub Path Resolution (Requires Network) ---"
  
  # GitHub tests - Handle potential test failures gracefully
  run_github_test() {
    local desc=$1
    local expected_count=$2
    shift 2
    local args=("$@")
    
    if [ "$ALLOW_TEST_FAILURES" = "1" ]; then
      # Run the test but don't exit on failure
      if ! test_case_add "$desc" "$expected_count" "${args[@]}"; then
        log_warning "Test '$desc' failed, but continuing as ALLOW_TEST_FAILURES=1"
      fi
    else
      # Normal behavior - exit on test failure
      test_case_add "$desc" "$expected_count" "${args[@]}"
    fi
  }
  
  # This test should work with the fallback mechanism
  run_github_test "GitHub: username/rule (implies default collection)" 1 "$DEFAULT_GITHUB_USER/monorepo"
  
  # For repo path tests, we need to use simple directory names without file extensions
  run_github_test "GitHub: username/repo/directory" 1 "$DEFAULT_GITHUB_USER/$RULES_COLLECTION_REPO/monorepo"
  
  # Test with a general directory rule
  run_github_test "GitHub: username/repo/different-dir" 1 "$DEFAULT_GITHUB_USER/$RULES_COLLECTION_REPO/general"
  
  # Test with go directory
  run_github_test "GitHub: Go directory" 1 "$DEFAULT_GITHUB_USER/$RULES_COLLECTION_REPO/go"
  
  # Test Default Username configuration
  log_info "Setting up temporary default username config ($DEFAULT_GITHUB_USER)..."
  # Write config to the temp config path
  echo "{\"defaultUsername\":\"$DEFAULT_GITHUB_USER\"}" > "$CURSOR_CONFIG_PATH"
  if [ ! -f "$CURSOR_CONFIG_PATH" ]; then log_error "Failed to create temporary config file"; fi
  
  # This should work as it uses the fallback mechanism
  run_github_test "Rule name only (with default username configured)" 1 "monorepo"
  
  # Clean up temp config immediately after test
  rm -f "$CURSOR_CONFIG_PATH"
  log_info "Temporary default username config removed."
fi

# --- Completion ---
TIMESTAMP_END=$(date "+%Y-%m-%d %H:%M:%S %Z")
log_header "All Integration Tests Completed"
log_success "Finished successfully at $TIMESTAMP_END"

# Exit with 0, cleanup is handled by the trap
exit 0 