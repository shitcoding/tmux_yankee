#!/usr/bin/env bash
# run_linux_tests.sh - Linux-specific test runner for Docker
#
# Orchestrates 5 test phases:
#   1. Binary smoke test
#   2. Portable shell unit tests
#   3. Full tmux lifecycle (bash)
#   4. Full tmux lifecycle (zsh)
#   5. Clipboard with Xvfb + xclip
#
# This is NOT the same as run_all_tests.sh (which targets legacy paths).
# This runner is purpose-built for Linux/Docker testing.

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
    CLR_GREEN="" CLR_RED="" CLR_YELLOW="" CLR_CYAN="" CLR_RESET="" CLR_BOLD=""
fi

TOTAL_PHASES=0
PASSED_PHASES=0
FAILED_PHASES=()
START_TIME=$(date +%s)

run_phase() {
    local name="$1"
    shift
    TOTAL_PHASES=$((TOTAL_PHASES + 1))

    printf "\n${CLR_BOLD}${CLR_CYAN}=== Phase %d: %s ===${CLR_RESET}\n" "$TOTAL_PHASES" "$name"

    local exit_code=0
    "$@" || exit_code=$?

    if [ "$exit_code" -eq 0 ]; then
        printf "  ${CLR_GREEN}PHASE PASSED${CLR_RESET}: %s\n" "$name"
        PASSED_PHASES=$((PASSED_PHASES + 1))
    else
        printf "  ${CLR_RED}PHASE FAILED${CLR_RESET}: %s (exit code %d)\n" "$name" "$exit_code"
        FAILED_PHASES+=("$name")
    fi

    return 0  # Don't abort on phase failure
}

# --- Banner ---
printf "\n${CLR_BOLD}${CLR_CYAN}========================================${CLR_RESET}\n"
printf "${CLR_BOLD}${CLR_CYAN}  tmux-yankee Linux Test Suite${CLR_RESET}\n"
printf "${CLR_BOLD}${CLR_CYAN}========================================${CLR_RESET}\n"
printf "  Platform: %s %s\n" "$(uname -s)" "$(uname -m)"
printf "  tmux:     %s\n" "$(tmux -V 2>/dev/null || echo 'not found')"
printf "  bash:     %s\n" "$(bash --version | head -1)"
printf "  zsh:      %s\n" "$(zsh --version 2>/dev/null || echo 'not found')"
printf "  Go binary: %s\n" "$PROJECT_ROOT/bin/tmux-yankee"

# --- Phase 1: Binary smoke test ---
phase_binary_smoke() {
    local output
    output=$("$PROJECT_ROOT/bin/tmux-yankee" --help 2>&1 || true)
    if printf '%s' "$output" | grep -qiE 'usage|flag|help|tmux-yankee'; then
        printf "  ${CLR_GREEN}OK${CLR_RESET}: binary runs and shows help\n"
        return 0
    else
        printf "  ${CLR_RED}FAIL${CLR_RESET}: binary did not produce help output\n"
        printf "  output: %s\n" "$output"
        return 1
    fi
}
run_phase "Binary smoke test" phase_binary_smoke

# --- Phase 2: Portable shell unit tests ---
# test_launch_yankee_flags.sh runs here as a Linux smoke check (it also runs in
# run_all_tests.sh). test_install_atomic.sh is intentionally NOT run here: the
# Docker image runs as root, and its readonly-dir case relies on DAC write
# denial, which root bypasses. It runs in run_all_tests.sh (non-root).
phase_shell_unit_tests() {
    local exit_code=0
    bash "$TESTS_DIR/unit/test_launch_yankee_flags.sh" || exit_code=1
    return "$exit_code"
}
run_phase "Shell unit tests" phase_shell_unit_tests

# --- Phase 3: Tmux lifecycle (bash) ---
phase_lifecycle_bash() {
    if [ ! -f "$TESTS_DIR/integration/test_linux_lifecycle.sh" ]; then
        printf "  ${CLR_RED}FAIL${CLR_RESET}: test_linux_lifecycle.sh not found\n"
        return 1
    fi
    timeout 60 bash "$TESTS_DIR/integration/test_linux_lifecycle.sh" /bin/bash
}
run_phase "Tmux lifecycle (bash)" phase_lifecycle_bash

# --- Phase 4: Tmux lifecycle (zsh) ---
phase_lifecycle_zsh() {
    if ! command -v zsh &>/dev/null; then
        printf "  ${CLR_YELLOW}SKIP${CLR_RESET}: zsh not available\n"
        return 0
    fi
    if [ ! -f "$TESTS_DIR/integration/test_linux_lifecycle.sh" ]; then
        printf "  ${CLR_RED}FAIL${CLR_RESET}: test_linux_lifecycle.sh not found\n"
        return 1
    fi
    timeout 60 bash "$TESTS_DIR/integration/test_linux_lifecycle.sh" /usr/bin/zsh
}
run_phase "Tmux lifecycle (zsh)" phase_lifecycle_zsh

# --- Phase 5: Clipboard (Xvfb + xclip) ---
phase_clipboard() {
    if [ ! -f "$TESTS_DIR/integration/test_linux_clipboard.sh" ]; then
        printf "  ${CLR_RED}FAIL${CLR_RESET}: test_linux_clipboard.sh not found\n"
        return 1
    fi
    timeout 30 bash "$TESTS_DIR/integration/test_linux_clipboard.sh"
}
run_phase "Clipboard (Xvfb + xclip)" phase_clipboard

# --- Summary ---
END_TIME=$(date +%s)
ELAPSED=$((END_TIME - START_TIME))
FAILED_COUNT=${#FAILED_PHASES[@]}

printf "\n${CLR_BOLD}${CLR_CYAN}========================================${CLR_RESET}\n"
printf "${CLR_BOLD}${CLR_CYAN}  Linux Test Results${CLR_RESET}\n"
printf "${CLR_BOLD}${CLR_CYAN}========================================${CLR_RESET}\n"
printf "  Phases run:    %d\n" "$TOTAL_PHASES"
printf "  ${CLR_GREEN}Passed:${CLR_RESET}        %d\n" "$PASSED_PHASES"
printf "  ${CLR_RED}Failed:${CLR_RESET}        %d\n" "$FAILED_COUNT"
printf "  Time elapsed:  %ds\n" "$ELAPSED"

if [ "$FAILED_COUNT" -gt 0 ]; then
    printf "\n  ${CLR_RED}Failed phases:${CLR_RESET}\n"
    for phase in "${FAILED_PHASES[@]}"; do
        printf "    - %s\n" "$phase"
    done
    printf "\n${CLR_RED}${CLR_BOLD}RESULT: FAIL${CLR_RESET}\n\n"
    exit 1
else
    printf "\n${CLR_GREEN}${CLR_BOLD}RESULT: PASS${CLR_RESET} (all phases passed)\n\n"
    exit 0
fi
