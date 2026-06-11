#!/usr/bin/env bash
# test_helpers.sh - Shared test framework for tmux-yankee
#
# Provides: assert functions, test lifecycle, pass/fail tracking, tmux session helpers
# Usage: source this file at the top of every test script

set -euo pipefail

# --- Test framework state ---
_TEST_PASS=0
_TEST_FAIL=0
_TEST_SKIP=0
_TEST_TOTAL=0
_TEST_FAILURES=()
_CURRENT_TEST=""
_TEST_FILE="${BASH_SOURCE[1]:-unknown}"

# Colors for output (disabled if not a terminal)
if [[ -t 1 ]]; then
    _CLR_GREEN=$'\033[32m'
    _CLR_RED=$'\033[31m'
    _CLR_YELLOW=$'\033[33m'
    _CLR_CYAN=$'\033[36m'
    _CLR_RESET=$'\033[0m'
    _CLR_BOLD=$'\033[1m'
else
    _CLR_GREEN=""
    _CLR_RED=""
    _CLR_YELLOW=""
    _CLR_CYAN=""
    _CLR_RESET=""
    _CLR_BOLD=""
fi

# --- Project paths ---
PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
SCRIPTS_DIR="$PROJECT_ROOT/scripts"
TESTS_DIR="$PROJECT_ROOT/tests"

# --- Test lifecycle ---

run_test() {
    # Usage: run_test "test_name" test_function
    local test_name="$1"
    local test_func="$2"

    _CURRENT_TEST="$test_name"
    _TEST_TOTAL=$(( _TEST_TOTAL + 1 ))

    # Run test function in a subshell so set -e works properly inside it.
    # Without a subshell, the || on the invocation line disables set -e
    # inside the function, causing assert failures to not propagate.
    local exit_code=0
    local output
    output=$(set -e; "$test_func" 2>&1) || exit_code=$?

    # Print captured output (assertion messages)
    if [[ -n "$output" ]]; then
        printf '%s\n' "$output"
    fi

    if [[ $exit_code -eq 0 ]]; then
        _TEST_PASS=$(( _TEST_PASS + 1 ))
        printf "  ${_CLR_GREEN}PASS${_CLR_RESET} %s\n" "$test_name"
    elif [[ $exit_code -eq 77 ]]; then
        # Convention: exit 77 = skip
        _TEST_SKIP=$(( _TEST_SKIP + 1 ))
        printf "  ${_CLR_YELLOW}SKIP${_CLR_RESET} %s\n" "$test_name"
    else
        _TEST_FAIL=$(( _TEST_FAIL + 1 ))
        _TEST_FAILURES+=("$test_name")
        printf "  ${_CLR_RED}FAIL${_CLR_RESET} %s\n" "$test_name"
    fi
}

skip_test() {
    # Call from within a test function to skip it
    printf "    skipped: %s\n" "${1:-no reason given}"
    return 77
}

# --- Assert functions ---

assert_equal() {
    # Usage: assert_equal "expected" "actual" "message"
    local expected="$1"
    local actual="$2"
    local message="${3:-values should be equal}"

    if [[ "$expected" != "$actual" ]]; then
        printf "    ${_CLR_RED}ASSERTION FAILED:${_CLR_RESET} %s\n" "$message"
        printf "    expected: '%s'\n" "$expected"
        printf "    actual:   '%s'\n" "$actual"
        return 1
    fi
}

assert_not_equal() {
    # Usage: assert_not_equal "unexpected" "actual" "message"
    local unexpected="$1"
    local actual="$2"
    local message="${3:-values should not be equal}"

    if [[ "$unexpected" == "$actual" ]]; then
        printf "    ${_CLR_RED}ASSERTION FAILED:${_CLR_RESET} %s\n" "$message"
        printf "    should not be: '%s'\n" "$unexpected"
        return 1
    fi
}

assert_contains() {
    # Usage: assert_contains "haystack" "needle" "message"
    local haystack="$1"
    local needle="$2"
    local message="${3:-string should contain substring}"

    if [[ "$haystack" != *"$needle"* ]]; then
        printf "    ${_CLR_RED}ASSERTION FAILED:${_CLR_RESET} %s\n" "$message"
        printf "    string:    '%s'\n" "$haystack"
        printf "    expected to contain: '%s'\n" "$needle"
        return 1
    fi
}

assert_not_contains() {
    # Usage: assert_not_contains "haystack" "needle" "message"
    local haystack="$1"
    local needle="$2"
    local message="${3:-string should not contain substring}"

    if [[ "$haystack" == *"$needle"* ]]; then
        printf "    ${_CLR_RED}ASSERTION FAILED:${_CLR_RESET} %s\n" "$message"
        printf "    string:    '%s'\n" "$haystack"
        printf "    should not contain: '%s'\n" "$needle"
        return 1
    fi
}

assert_matches() {
    # Usage: assert_matches "string" "regex_pattern" "message"
    local string="$1"
    local pattern="$2"
    local message="${3:-string should match pattern}"

    if ! [[ "$string" =~ $pattern ]]; then
        printf "    ${_CLR_RED}ASSERTION FAILED:${_CLR_RESET} %s\n" "$message"
        printf "    string:  '%s'\n" "$string"
        printf "    pattern: '%s'\n" "$pattern"
        return 1
    fi
}

assert_line_count() {
    # Usage: assert_line_count "text" expected_count "message"
    local text="$1"
    local expected="$2"
    local message="${3:-line count should match}"

    local actual
    if [[ -z "$text" ]]; then
        actual=0
    else
        actual=$(printf '%s\n' "$text" | wc -l | tr -d ' ')
    fi

    if [[ "$actual" -ne "$expected" ]]; then
        printf "    ${_CLR_RED}ASSERTION FAILED:${_CLR_RESET} %s\n" "$message"
        printf "    expected %d lines, got %d\n" "$expected" "$actual"
        return 1
    fi
}

assert_exit_code() {
    # Usage: assert_exit_code expected_code "message" command [args...]
    local expected="$1"
    local message="$2"
    shift 2

    local actual=0
    "$@" 2>/dev/null || actual=$?

    if [[ "$actual" -ne "$expected" ]]; then
        printf "    ${_CLR_RED}ASSERTION FAILED:${_CLR_RESET} %s\n" "$message"
        printf "    expected exit code %d, got %d\n" "$expected" "$actual"
        return 1
    fi
}

assert_file_exists() {
    # Usage: assert_file_exists "path" "message"
    local path="$1"
    local message="${2:-file should exist}"

    if [[ ! -f "$path" ]]; then
        printf "    ${_CLR_RED}ASSERTION FAILED:${_CLR_RESET} %s\n" "$message"
        printf "    file not found: '%s'\n" "$path"
        return 1
    fi
}

assert_numeric() {
    # Usage: assert_numeric "value" "message"
    local value="$1"
    local message="${2:-value should be numeric}"

    if ! [[ "$value" =~ ^-?[0-9]+$ ]]; then
        printf "    ${_CLR_RED}ASSERTION FAILED:${_CLR_RESET} %s\n" "$message"
        printf "    not numeric: '%s'\n" "$value"
        return 1
    fi
}

assert_greater_than() {
    # Usage: assert_greater_than actual minimum "message"
    local actual="$1"
    local minimum="$2"
    local message="${3:-value should be greater than minimum}"

    if [[ "$actual" -le "$minimum" ]]; then
        printf "    ${_CLR_RED}ASSERTION FAILED:${_CLR_RESET} %s\n" "$message"
        printf "    actual: %d, should be > %d\n" "$actual" "$minimum"
        return 1
    fi
}

# --- Tmux session helpers (for integration tests) ---

# Socket name for isolated test sessions
TMUX_TEST_SOCKET="linenumbers-test-$$"

tmux_test_cmd() {
    # Run a tmux command in the test socket
    tmux -L "$TMUX_TEST_SOCKET" "$@"
}

setup_tmux_test_session() {
    # Create an isolated tmux session for testing
    # Returns: session name on stdout
    local session_name="${1:-test-$$}"

    # Kill any leftover session with the same name
    tmux -L "$TMUX_TEST_SOCKET" kill-session -t "$session_name" 2>/dev/null || true

    # Create new detached session
    tmux -L "$TMUX_TEST_SOCKET" new-session -d -s "$session_name" -x 80 -y 24

    # Brief settle time for session creation
    sleep 0.1

    printf '%s' "$session_name"
}

teardown_tmux_test_session() {
    # Kill test session and clean up
    local session_name="${1:-test-$$}"
    tmux -L "$TMUX_TEST_SOCKET" kill-session -t "$session_name" 2>/dev/null || true
}

teardown_tmux_test_server() {
    # Kill the entire test tmux server (full cleanup)
    tmux -L "$TMUX_TEST_SOCKET" kill-server 2>/dev/null || true
}

get_tmux_pane_mode() {
    # Get the current pane mode (empty = normal, "copy-mode" = copy-mode)
    local target="${1:-%0}"
    tmux_test_cmd display-message -t "$target" -p '#{pane_mode}'
}

get_tmux_pane_count() {
    # Get the number of panes in current window
    local session="${1:-}"
    if [[ -n "$session" ]]; then
        tmux_test_cmd list-panes -t "$session" -F '#{pane_id}' | wc -l | tr -d ' '
    else
        tmux_test_cmd list-panes -F '#{pane_id}' | wc -l | tr -d ' '
    fi
}

capture_tmux_pane() {
    # Capture pane content as text
    local target="${1:-%0}"
    tmux_test_cmd capture-pane -t "$target" -p
}

wait_for_tmux_idle() {
    # Wait until tmux pane appears idle (no new output)
    # Simple approach: sleep a short time
    local wait_time="${1:-0.3}"
    sleep "$wait_time"
}

require_tmux() {
    # Skip test if tmux is not available
    if ! command -v tmux &>/dev/null; then
        skip_test "tmux not available"
    fi
}

# --- Report functions ---

print_test_file_header() {
    local file_name="$1"
    printf "\n${_CLR_BOLD}${_CLR_CYAN}=== %s ===${_CLR_RESET}\n" "$file_name"
}

print_test_summary() {
    printf "\n${_CLR_BOLD}--- Summary for %s ---${_CLR_RESET}\n" "$(basename "$_TEST_FILE")"
    printf "  Total:   %d\n" "$_TEST_TOTAL"
    printf "  ${_CLR_GREEN}Passed:  %d${_CLR_RESET}\n" "$_TEST_PASS"
    printf "  ${_CLR_RED}Failed:  %d${_CLR_RESET}\n" "$_TEST_FAIL"
    if [[ $_TEST_SKIP -gt 0 ]]; then
        printf "  ${_CLR_YELLOW}Skipped: %d${_CLR_RESET}\n" "$_TEST_SKIP"
    fi

    if [[ ${#_TEST_FAILURES[@]} -gt 0 ]]; then
        printf "\n  ${_CLR_RED}Failed tests:${_CLR_RESET}\n"
        for f in "${_TEST_FAILURES[@]}"; do
            printf "    - %s\n" "$f"
        done
    fi

    printf "\n"
}

get_test_exit_code() {
    # Return 0 if all tests passed, 1 if any failed
    if [[ $_TEST_FAIL -gt 0 ]]; then
        return 1
    fi
    return 0
}

# --- Utility: temp file management ---

_TEMP_FILES=()

create_temp_file() {
    # Create a temp file that will be cleaned up automatically
    local tmpfile
    tmpfile=$(mktemp /tmp/linenumbers-test.XXXXXX)
    _TEMP_FILES+=("$tmpfile")
    printf '%s' "$tmpfile"
}

_cleanup_temp_files() {
    if [[ ${#_TEMP_FILES[@]} -gt 0 ]]; then
        for f in "${_TEMP_FILES[@]}"; do
            rm -f "$f" 2>/dev/null || true
        done
    fi
}

trap '_cleanup_temp_files' EXIT
