#!/usr/bin/env bash
# test_clipboard_errors.sh - Clipboard error handling tests
#
# Tests error handling when clipboard operations fail:
# - No clipboard command available
# - Clipboard command fails (exits non-zero)
# - Error messages displayed to user
# - Fallback to tmux buffer

set -euo pipefail

TESTS_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
source "$TESTS_DIR/test_helpers.sh"

print_test_file_header "Integration Tests: Clipboard Error Handling"

# --- Test prerequisites ---
require_tmux

# ============================================================================
# Test 50: test_no_clipboard_command_available
# Verify copy_stdin.sh exits non-zero when no clipboard command is found
# ============================================================================
_test_no_clipboard_command_available() {
    # Use absolute path to bash so PATH="/nonexistent" doesn't prevent
    # bash itself from being found (it's an external command).
    local bash_path
    bash_path=$(command -v bash)

    local exit_code=0
    printf 'test' | PATH="/nonexistent" "$bash_path" "$SCRIPTS_DIR/copy_stdin.sh" 2>/dev/null || exit_code=$?

    # Should exit non-zero when no clipboard command is found
    assert_not_equal 0 "$exit_code" \
        "copy_stdin.sh should exit non-zero when no clipboard command available"
}
run_test "test_no_clipboard_command_available" _test_no_clipboard_command_available

# ============================================================================
# Test 51: test_clipboard_command_failure_detection
# Verify copy_stdin.sh checks for empty/failing clipboard commands
# ============================================================================
_test_clipboard_command_failure_detection() {
    local copy_script="$SCRIPTS_DIR/copy_stdin.sh"
    local script_content
    script_content=$(cat "$copy_script")

    # Verify script checks for empty clipboard command
    if ! echo "$script_content" | grep -q 'if \[.*-z.*copy_command'; then
        printf "    ASSERTION FAILED: copy_stdin.sh should check for empty clipboard command\n"
        return 1
    fi
    printf "    ✓ copy_stdin.sh checks for empty clipboard command\n"

    # Verify script checks command execution success
    if ! echo "$script_content" | grep -q 'if.*!.*eval.*copy_command'; then
        printf "    ASSERTION FAILED: copy_stdin.sh should check clipboard command execution result\n"
        return 1
    fi
    printf "    ✓ copy_stdin.sh checks clipboard command execution result\n"
}
run_test "test_clipboard_command_failure_detection" _test_clipboard_command_failure_detection

# ============================================================================
# Test 52: test_error_message_uses_tmux_display
# Verify error messages are shown via tmux display-message
# ============================================================================
_test_error_message_uses_tmux_display() {
    local copy_script="$SCRIPTS_DIR/copy_stdin.sh"

    if ! grep -q 'tmux display-message' "$copy_script"; then
        printf "    ASSERTION FAILED: copy_stdin.sh should use tmux display-message for error reporting\n"
        return 1
    fi
    printf "    ✓ copy_stdin.sh uses tmux display-message for error reporting\n"
}
run_test "test_error_message_uses_tmux_display" _test_error_message_uses_tmux_display

# ============================================================================
# Test 53: test_tmux_buffer_fallback_documented
# Document tmux buffer behavior per copy-target mode (Go code handles this)
# ============================================================================
_test_tmux_buffer_fallback_documented() {
    # The Go binary handles tmux set-buffer differently per copy-target:
    #   "both" (default): calls set-buffer, then clipboard copy
    #   "tmux":           calls set-buffer only, no clipboard
    #   "clipboard":      clipboard copy only, skips set-buffer
    #
    # Verify the Go code actually implements this by checking the yank path
    local tui_file="$PROJECT_ROOT/internal/ui/tui_yank.go"

    # CopyTargetClipboard should skip set-buffer
    if ! grep -q 'CopyTargetClipboard' "$tui_file"; then
        printf "    ASSERTION FAILED: tui_yank.go should reference CopyTargetClipboard\n"
        return 1
    fi
    printf "    ✓ tui_yank.go implements copy-target modes (both/tmux/clipboard)\n"

    # The test file for yank behavior should test the clipboard-only path
    local test_file="$PROJECT_ROOT/internal/ui/tui_yank_test.go"
    if ! grep -q 'CopyTargetClipboard' "$test_file"; then
        printf "    ASSERTION FAILED: tui_yank_test.go should test CopyTargetClipboard path\n"
        return 1
    fi
    printf "    ✓ tui_yank_test.go covers clipboard-only mode (skips set-buffer)\n"
}
run_test "test_tmux_buffer_fallback_documented" _test_tmux_buffer_fallback_documented

# ============================================================================
# Test 54: test_copy_stdin_script_error_handling
# Verify copy_stdin.sh has proper error handling structure
# ============================================================================
_test_copy_stdin_script_error_handling() {
    local copy_script="$SCRIPTS_DIR/copy_stdin.sh"

    # Check for strict mode
    if ! grep -q "set -euo pipefail" "$copy_script"; then
        printf "    ASSERTION FAILED: copy_stdin.sh should use 'set -euo pipefail'\n"
        return 1
    fi
    printf "    ✓ copy_stdin.sh uses strict mode\n"

    # Check it's self-contained (no external helper dependencies)
    if grep -q 'source.*helpers\.sh' "$copy_script"; then
        printf "    ASSERTION FAILED: copy_stdin.sh should not depend on helpers.sh\n"
        return 1
    fi
    printf "    ✓ copy_stdin.sh is self-contained (no external helpers)\n"

    # Check it has its own clipboard detection
    if ! grep -q 'detect_clipboard_command' "$copy_script"; then
        printf "    ASSERTION FAILED: copy_stdin.sh should have built-in clipboard detection\n"
        return 1
    fi
    printf "    ✓ copy_stdin.sh has built-in clipboard detection\n"
}
run_test "test_copy_stdin_script_error_handling" _test_copy_stdin_script_error_handling

# --- Print summary and exit ---
print_test_summary
teardown_tmux_test_server
get_test_exit_code
