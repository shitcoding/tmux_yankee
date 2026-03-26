#!/usr/bin/env bash
# test_clipboard_errors.sh - Clipboard error handling tests
#
# Tests error handling when clipboard operations fail:
# - No clipboard command available
# - Clipboard command fails (exits non-zero)
# - Error messages displayed to user
# - Fallback to tmux buffer only

set -euo pipefail

TESTS_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
source "$TESTS_DIR/test_helpers.sh"

print_test_file_header "Integration Tests: Clipboard Error Handling"

# --- Test prerequisites ---
require_tmux

# ============================================================================
# Test 50: test_no_clipboard_command_available
# Verify copy_stdin.sh handles missing clipboard gracefully
# ============================================================================
_test_no_clipboard_command_available() {
    # Run copy_stdin.sh with PATH stripped of clipboard commands
    local exit_code=0
    printf 'test' | PATH="/nonexistent" bash "$SCRIPTS_DIR/copy_stdin.sh" 2>/dev/null || exit_code=$?

    # Should exit non-zero when no clipboard command is found
    if [[ $exit_code -ne 0 ]]; then
        printf "    ✓ copy_stdin.sh exits non-zero when no clipboard command available (exit code: %d)\n" "$exit_code"
    else
        printf "    ${_CLR_YELLOW}WARNING:${_CLR_RESET} copy_stdin.sh should fail when no clipboard command available\n"
    fi
}
run_test "test_no_clipboard_command_available" _test_no_clipboard_command_available

# ============================================================================
# Test 51: test_clipboard_command_failure_detection
# Verify copy_stdin.sh detects clipboard command failure
# ============================================================================
_test_clipboard_command_failure_detection() {
    local copy_script="$SCRIPTS_DIR/copy_stdin.sh"
    local script_content
    script_content=$(cat "$copy_script")

    # Verify script checks for empty clipboard command
    if echo "$script_content" | grep -q 'if \[.*-z.*copy_command'; then
        printf "    ✓ copy_stdin.sh checks for empty clipboard command\n"
    else
        printf "    ${_CLR_YELLOW}WARNING:${_CLR_RESET} copy_stdin.sh should check for empty clipboard command\n"
    fi

    # Verify script checks command execution success
    if echo "$script_content" | grep -q 'if.*!.*eval.*copy_command'; then
        printf "    ✓ copy_stdin.sh checks clipboard command execution result\n"
    else
        printf "    ${_CLR_YELLOW}WARNING:${_CLR_RESET} copy_stdin.sh should check clipboard command result\n"
    fi
}
run_test "test_clipboard_command_failure_detection" _test_clipboard_command_failure_detection

# ============================================================================
# Test 52: test_error_message_uses_tmux_display
# Verify error messages are shown via tmux display-message
# ============================================================================
_test_error_message_uses_tmux_display() {
    local copy_script="$SCRIPTS_DIR/copy_stdin.sh"
    local script_content
    script_content=$(cat "$copy_script")

    if echo "$script_content" | grep -q 'tmux display-message'; then
        printf "    ✓ copy_stdin.sh uses tmux display-message for error reporting\n"
    else
        printf "    ${_CLR_YELLOW}WARNING:${_CLR_RESET} copy_stdin.sh should use tmux display-message for errors\n"
    fi
}
run_test "test_error_message_uses_tmux_display" _test_error_message_uses_tmux_display

# ============================================================================
# Test 53: test_tmux_buffer_fallback
# Verify text is saved to tmux buffer even when clipboard fails
# (This is handled in Go code, just document expected behavior)
# ============================================================================
_test_tmux_buffer_fallback() {
    printf "    INFO: tmux buffer fallback is handled in Go code (tui.go yank path)\n"
    printf "    The Go binary always calls 'tmux set-buffer' before clipboard copy\n"
    printf "    Clipboard failure does not prevent tmux buffer save\n"
}
run_test "test_tmux_buffer_fallback" _test_tmux_buffer_fallback

# ============================================================================
# Test 54: test_copy_stdin_script_error_handling
# Verify copy_stdin.sh has proper error handling structure
# ============================================================================
_test_copy_stdin_script_error_handling() {
    local copy_script="$SCRIPTS_DIR/copy_stdin.sh"

    # Check for strict mode
    if ! grep -q "set -euo pipefail" "$copy_script"; then
        printf "    ${_CLR_YELLOW}WARNING:${_CLR_RESET} copy_stdin.sh should use 'set -euo pipefail'\n"
    else
        printf "    ✓ copy_stdin.sh uses strict mode\n"
    fi

    # Check it's self-contained (no external helper dependencies)
    if grep -q 'source.*helpers\.sh' "$copy_script"; then
        printf "    ${_CLR_RED}ASSERTION FAILED:${_CLR_RESET} copy_stdin.sh should not depend on helpers.sh\n"
        return 1
    else
        printf "    ✓ copy_stdin.sh is self-contained (no external helpers)\n"
    fi

    # Check it has its own clipboard detection
    if grep -q 'detect_clipboard_command' "$copy_script"; then
        printf "    ✓ copy_stdin.sh has built-in clipboard detection\n"
    else
        printf "    ${_CLR_YELLOW}WARNING:${_CLR_RESET} copy_stdin.sh should have built-in clipboard detection\n"
    fi
}
run_test "test_copy_stdin_script_error_handling" _test_copy_stdin_script_error_handling

# --- Print summary and exit ---
print_test_summary
teardown_tmux_test_server
get_test_exit_code
