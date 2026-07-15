#!/usr/bin/env bash
# run_all_tests.sh - Main test runner for tmux-yankee
#
# Runs the bash test suite: portable unit tests, then integration tests
# (integration requires tmux). Reports a pass/fail summary and exits with a
# proper code. (Go unit tests run separately via `go test ./...`.)
#
# Usage:
#   ./tests/run_all_tests.sh              # Run all tests
#   ./tests/run_all_tests.sh unit         # Run only unit tests
#   ./tests/run_all_tests.sh integration  # Run only integration tests

set -euo pipefail

TESTS_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$TESTS_DIR/.." && pwd)"

# Colors
if [[ -t 1 ]]; then
    CLR_GREEN=$'\033[32m'
    CLR_RED=$'\033[31m'
    CLR_YELLOW=$'\033[33m'
    CLR_CYAN=$'\033[36m'
    CLR_RESET=$'\033[0m'
    CLR_BOLD=$'\033[1m'
else
    CLR_GREEN=""
    CLR_RED=""
    CLR_YELLOW=""
    CLR_CYAN=""
    CLR_RESET=""
    CLR_BOLD=""
fi

# --- Configuration ---
FILTER="${1:-all}"  # "all", "unit", or "integration"
case "$FILTER" in
    all | unit | integration) ;;
    *)
        printf "Unknown filter: %s (use 'all', 'unit', or 'integration')\n" "$FILTER" >&2
        exit 2
        ;;
esac

TOTAL_PASS=0
TOTAL_FAIL=0
TOTAL_SKIP=0
TOTAL_FILES=0
FAILED_FILES=()
START_TIME=$(date +%s)

# --- Helper: run a test file and capture results ---
run_test_file() {
    local file="$1"
    local label="$2"
    local file_basename
    file_basename=$(basename "$file")

    if [[ ! -f "$file" ]]; then
        printf "${CLR_YELLOW}SKIP${CLR_RESET} %s (file not found: %s)\n" "$label" "$file"
        return 0
    fi

    if [[ ! -x "$file" ]]; then
        chmod +x "$file"
    fi

    TOTAL_FILES=$(( TOTAL_FILES + 1 ))

    local exit_code=0
    local output
    output=$(bash "$file" 2>&1) || exit_code=$?

    # Print the test output
    printf '%s\n' "$output"

    # Parse pass/fail/skip counts from the output
    # Our test_helpers.sh prints: "  Passed:  N", "  Failed:  N", "  Skipped: N"
    local file_pass file_fail file_skip
    file_pass=$(printf '%s\n' "$output" | grep -oE 'Passed:\s+[0-9]+' | grep -oE '[0-9]+' || echo "0")
    file_fail=$(printf '%s\n' "$output" | grep -oE 'Failed:\s+[0-9]+' | grep -oE '[0-9]+' || echo "0")
    file_skip=$(printf '%s\n' "$output" | grep -oE 'Skipped:\s+[0-9]+' | grep -oE '[0-9]+' || echo "0")

    TOTAL_PASS=$(( TOTAL_PASS + ${file_pass:-0} ))
    TOTAL_FAIL=$(( TOTAL_FAIL + ${file_fail:-0} ))
    TOTAL_SKIP=$(( TOTAL_SKIP + ${file_skip:-0} ))

    if [[ $exit_code -eq 77 ]]; then
        # 77 is the conventional "skip" exit code — not a failure.
        TOTAL_SKIP=$(( TOTAL_SKIP + 1 ))
    elif [[ $exit_code -ne 0 ]]; then
        FAILED_FILES+=("$file_basename")
    fi

    return 0
}

# --- Banner ---
printf "\n${CLR_BOLD}${CLR_CYAN}========================================${CLR_RESET}\n"
printf "${CLR_BOLD}${CLR_CYAN}  tmux-yankee test suite${CLR_RESET}\n"
printf "${CLR_BOLD}${CLR_CYAN}========================================${CLR_RESET}\n"
printf "  Filter: %s\n" "$FILTER"
printf "  Project: %s\n" "$PROJECT_ROOT"

# --- Run unit tests ---
if [[ "$FILTER" == "all" ]] || [[ "$FILTER" == "unit" ]]; then
    printf "\n${CLR_BOLD}${CLR_CYAN}--- Unit Tests ---${CLR_RESET}\n"

    unit_tests=(
        "$TESTS_DIR/unit/test_install_atomic.sh"
        "$TESTS_DIR/unit/test_install_version.sh"
        "$TESTS_DIR/unit/test_launch_yankee_flags.sh"
    )

    for test_file in "${unit_tests[@]}"; do
        run_test_file "$test_file" "$(basename "$test_file")"
    done
fi

# --- Run integration tests ---
if [[ "$FILTER" == "all" ]] || [[ "$FILTER" == "integration" ]]; then
    printf "\n${CLR_BOLD}${CLR_CYAN}--- Integration Tests ---${CLR_RESET}\n"

    # Check tmux availability
    if ! command -v tmux &>/dev/null; then
        printf "  ${CLR_YELLOW}SKIP${CLR_RESET} Integration tests (tmux not available)\n"
    else
        integration_tests=(
            "$TESTS_DIR/integration/test_clipboard_backends.sh"
            "$TESTS_DIR/integration/test_clipboard_errors.sh"
        )

        for test_file in "${integration_tests[@]}"; do
            run_test_file "$test_file" "$(basename "$test_file")"
        done
    fi
fi

# --- Overall summary ---
END_TIME=$(date +%s)
ELAPSED=$(( END_TIME - START_TIME ))
TOTAL_TESTS=$(( TOTAL_PASS + TOTAL_FAIL + TOTAL_SKIP ))

printf "\n${CLR_BOLD}${CLR_CYAN}========================================${CLR_RESET}\n"
printf "${CLR_BOLD}${CLR_CYAN}  Overall Results${CLR_RESET}\n"
printf "${CLR_BOLD}${CLR_CYAN}========================================${CLR_RESET}\n"
printf "  Test files run: %d\n" "$TOTAL_FILES"
printf "  Total tests:    %d\n" "$TOTAL_TESTS"
printf "  ${CLR_GREEN}Passed:${CLR_RESET}         %d\n" "$TOTAL_PASS"
printf "  ${CLR_RED}Failed:${CLR_RESET}         %d\n" "$TOTAL_FAIL"
if [[ $TOTAL_SKIP -gt 0 ]]; then
    printf "  ${CLR_YELLOW}Skipped:${CLR_RESET}        %d\n" "$TOTAL_SKIP"
fi
printf "  Time elapsed:   %ds\n" "$ELAPSED"

if [[ ${#FAILED_FILES[@]} -gt 0 ]]; then
    printf "\n  ${CLR_RED}Files with failures:${CLR_RESET}\n"
    for f in "${FAILED_FILES[@]}"; do
        printf "    - %s\n" "$f"
    done
fi

printf "\n"

# --- Exit code ---
# Fail on any counted failure OR any test file that errored (non-zero exit),
# including one that crashed before printing a parsable summary.
if [[ $TOTAL_FAIL -gt 0 || ${#FAILED_FILES[@]} -gt 0 ]]; then
    printf "${CLR_RED}${CLR_BOLD}RESULT: FAIL${CLR_RESET} (%d test(s) failed, %d file(s) errored)\n\n" "$TOTAL_FAIL" "${#FAILED_FILES[@]}"
    exit 1
else
    printf "${CLR_GREEN}${CLR_BOLD}RESULT: PASS${CLR_RESET} (all %d tests passed)\n\n" "$TOTAL_PASS"
    exit 0
fi
