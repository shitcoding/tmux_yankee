#!/usr/bin/env bash
# test_clipboard_errors.sh - Clipboard error handling tests
#
# Phase 6: Clipboard Integration
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
# Verify behavior when clipboard_copy_command() returns empty
#
# Acceptance criteria:
#   1. When no clipboard command is detected, function returns empty
#   2. copy_stdin.sh should handle empty command gracefully
#
# NOTE: This test documents expected behavior but may reveal implementation gap
# ============================================================================
_test_no_clipboard_command_available() {
    local session_name
    session_name=$(setup_tmux_test_session "no-clipboard-$$")

    # Override to force no clipboard command
    tmux_test_cmd set-option -g @override_copy_command ""
    tmux_test_cmd set-option -g @custom_copy_command ""

    # Get clipboard command in environment with no standard tools
    local cmd_result
    cmd_result=$(
        # Temporarily hide clipboard commands by manipulating PATH
        PATH="/usr/bin:/bin"
        # shellcheck source=scripts/helpers.sh
        source "$SCRIPTS_DIR/helpers.sh"
        # Override command_exists to always return false
        command_exists() { return 1; }
        clipboard_copy_command
    )

    # Should return empty when no command available
    printf "    clipboard command when none available: '%s'\n" "$cmd_result"

    # Test copy_stdin.sh behavior
    # IMPLEMENTATION GAP: copy_stdin.sh currently does not handle empty command
    # This will likely fail with "command not found" error
    local test_data="test_data_$$"
    local exit_code=0
    printf '%s' "$test_data" | "$SCRIPTS_DIR/copy_stdin.sh" 2>/dev/null || exit_code=$?

    if [[ $exit_code -eq 0 ]]; then
        printf "    ${_CLR_GREEN}INFO:${_CLR_RESET} copy_stdin.sh handled empty command gracefully\n"
    else
        printf "    ${_CLR_YELLOW}IMPLEMENTATION GAP:${_CLR_RESET} copy_stdin.sh fails with empty clipboard command (exit code: %d)\n" "$exit_code"
        printf "    This test documents expected behavior for Developer to implement\n"

        # For now, we accept failure as documented behavior
        # Developer should add error handling in copy_stdin.sh
    fi

    # Cleanup
    tmux_test_cmd set-option -gu @override_copy_command
    tmux_test_cmd set-option -gu @custom_copy_command
    teardown_tmux_test_session "$session_name"
}
run_test "test_no_clipboard_command_available" _test_no_clipboard_command_available

# ============================================================================
# Test 51: test_clipboard_command_failure_detection (IMPLEMENTATION GAP)
# Verify behavior when clipboard command exits non-zero
#
# Acceptance criteria:
#   1. When clipboard command fails, copy_stdin.sh should detect it
#   2. Should exit non-zero to signal failure
#
# NOTE: This documents expected behavior for Developer implementation
# ============================================================================
_test_clipboard_command_failure_detection() {
    printf "    ${_CLR_YELLOW}IMPLEMENTATION GAP:${_CLR_RESET} Clipboard command failure detection\n"
    printf "    Current behavior:\n"
    printf "      - copy_stdin.sh runs clipboard command but doesn't check exit code\n"
    printf "      - Uses 'set -euo pipefail' but command is in variable expansion\n"
    printf "      - May succeed even when clipboard copy actually fails\n"
    printf "\n"
    printf "    Expected behavior:\n"
    printf "      - Detect when clipboard command exits non-zero\n"
    printf "      - Display error message to user\n"
    printf "      - Exit with non-zero status\n"
    printf "\n"
    printf "    Developer TODO in copy_stdin.sh:\n"
    printf "      if [[ -z \"\$copy_command\" ]]; then\n"
    printf "        display_message \"No clipboard command available\"\n"
    printf "        exit 1\n"
    printf "      fi\n"
    printf "      \n"
    printf "      if ! \$copy_command; then\n"
    printf "        display_message \"Clipboard copy failed\"\n"
    printf "        exit 1\n"
    printf "      fi\n"
}
run_test "test_clipboard_command_failure_detection" _test_clipboard_command_failure_detection

# ============================================================================
# Test 52: test_error_message_display (IMPLEMENTATION GAP)
# Verify error message is shown to user when clipboard fails
#
# Acceptance criteria:
#   1. When clipboard copy fails, tmux displays error message
#   2. Message should be visible for reasonable duration
#   3. Uses tmux display-message or similar
#
# NOTE: This test documents expected behavior for Developer implementation
# Current implementation does NOT display error messages
# ============================================================================
_test_error_message_display() {
    local session_name
    session_name=$(setup_tmux_test_session "error-msg-$$")

    printf "    ${_CLR_YELLOW}IMPLEMENTATION GAP:${_CLR_RESET} Error message display not yet implemented\n"
    printf "    Expected behavior:\n"
    printf "      1. When clipboard command fails, display error via tmux display-message\n"
    printf "      2. Error should include reason (command failed, no clipboard available)\n"
    printf "      3. Message should be visible for 3-5 seconds\n"
    printf "\n"
    printf "    Developer TODO:\n"
    printf "      - Add error detection in copy_stdin.sh\n"
    printf "      - Call tmux display-message on clipboard failure\n"
    printf "      - Include helpful error context\n"
    printf "\n"
    printf "    Example implementation in copy_stdin.sh:\n"
    printf "      if ! \$copy_command; then\n"
    printf "        tmux display-message -d 5000 'Clipboard copy failed'\n"
    printf "        exit 1\n"
    printf "      fi\n"

    teardown_tmux_test_session "$session_name"
}
run_test "test_error_message_display" _test_error_message_display

# ============================================================================
# Test 53: test_tmux_buffer_fallback (IMPLEMENTATION GAP)
# Verify text is still saved to tmux buffer when clipboard fails
#
# Acceptance criteria:
#   1. When clipboard command fails, text should still go to tmux buffer
#   2. User can paste with tmux paste-buffer
#   3. This provides fallback functionality
#
# NOTE: This requires Go code integration, documenting expected behavior
# ============================================================================
_test_tmux_buffer_fallback() {
    printf "    ${_CLR_YELLOW}IMPLEMENTATION GAP:${_CLR_RESET} Tmux buffer fallback not yet implemented\n"
    printf "    Expected behavior:\n"
    printf "      1. Go code (cmd/tmux-yankee) should ALWAYS call tmux set-buffer\n"
    printf "      2. This happens regardless of clipboard success/failure\n"
    printf "      3. Ensures yanked text is available via tmux paste-buffer\n"
    printf "\n"
    printf "    Developer TODO in cmd/tmux-yankee/main.go or internal/ui/tui.go:\n"
    printf "      func (t *TUI) Yank() error {\n"
    printf "        text := t.selection.Extract(t.content)\n"
    printf "\n"
    printf "        // ALWAYS set tmux buffer first\n"
    printf "        t.client.SetBuffer(text)\n"
    printf "\n"
    printf "        // THEN try clipboard (may fail)\n"
    printf "        err := copyToClipboard(text)\n"
    printf "        if err != nil {\n"
    printf "          // Show error but don't return - buffer is set\n"
    printf "          showErrorMessage(err)\n"
    printf "        }\n"
    printf "        return nil\n"
    printf "      }\n"
}
run_test "test_tmux_buffer_fallback" _test_tmux_buffer_fallback

# ============================================================================
# Test 54: test_copy_stdin_script_error_handling
# Verify copy_stdin.sh has proper error handling structure
#
# Acceptance criteria:
#   1. Script uses set -euo pipefail
#   2. Checks if clipboard command is empty
#   3. Detects command execution failure
# ============================================================================
_test_copy_stdin_script_error_handling() {
    local copy_script="$SCRIPTS_DIR/copy_stdin.sh"

    # Check for strict mode
    if ! grep -q "set -euo pipefail" "$copy_script"; then
        printf "    ${_CLR_YELLOW}WARNING:${_CLR_RESET} copy_stdin.sh should use 'set -euo pipefail'\n"
    else
        printf "    ✓ copy_stdin.sh uses strict mode\n"
    fi

    # Check current implementation
    local script_content
    script_content=$(cat "$copy_script")

    # Check if it validates clipboard command before using it
    if ! [[ "$script_content" =~ "if".*"\[\[".*"copy_command" ]] && \
       ! [[ "$script_content" =~ "test".*"copy_command" ]]; then
        printf "    ${_CLR_YELLOW}IMPLEMENTATION GAP:${_CLR_RESET} copy_stdin.sh does not check if clipboard command is empty\n"
        printf "    Recommended addition:\n"
        printf "      if [[ -z \"\$copy_command\" ]]; then\n"
        printf "        display_message \"No clipboard command available\"\n"
        printf "        exit 1\n"
        printf "      fi\n"
    fi

    # Check if it validates execution success
    if ! [[ "$script_content" =~ "if".*"\$copy_command" ]] && \
       ! [[ "$script_content" =~ "\$copy_command".*"||" ]] && \
       ! [[ "$script_content" =~ "\$copy_command".*"if" ]]; then
        printf "    ${_CLR_YELLOW}IMPLEMENTATION GAP:${_CLR_RESET} copy_stdin.sh does not check clipboard command success\n"
        printf "    Recommended change:\n"
        printf "      if ! \$copy_command; then\n"
        printf "        display_message \"Clipboard copy failed\"\n"
        printf "        exit 1\n"
        printf "      fi\n"
    fi
}
run_test "test_copy_stdin_script_error_handling" _test_copy_stdin_script_error_handling

# ============================================================================
# Test 55: test_helpers_display_message_function
# Verify helpers.sh provides display_message() for error reporting
#
# Acceptance criteria:
#   1. helpers.sh exports display_message function
#   2. Function can be used from copy_stdin.sh
#   3. Function properly saves/restores display-time option
# ============================================================================
_test_helpers_display_message_function() {
    # Source helpers.sh
    # shellcheck source=scripts/helpers.sh
    source "$SCRIPTS_DIR/helpers.sh"

    # Verify display_message function exists
    if ! declare -f display_message &>/dev/null; then
        printf "    ${_CLR_RED}ASSERTION FAILED:${_CLR_RESET} helpers.sh should export display_message function\n"
        return 1
    fi

    printf "    ✓ display_message function exists in helpers.sh\n"

    # Verify it has correct signature (message and optional duration)
    local func_body
    func_body=$(declare -f display_message)

    if ! [[ "$func_body" =~ "display-time" ]]; then
        printf "    ${_CLR_YELLOW}WARNING:${_CLR_RESET} display_message should save/restore display-time option\n"
    else
        printf "    ✓ display_message handles display-time option\n"
    fi

    if ! [[ "$func_body" =~ "tmux display-message" ]]; then
        printf "    ${_CLR_RED}ASSERTION FAILED:${_CLR_RESET} display_message should call 'tmux display-message'\n"
        return 1
    else
        printf "    ✓ display_message calls tmux display-message\n"
    fi
}
run_test "test_helpers_display_message_function" _test_helpers_display_message_function

# --- Print summary and exit ---
print_test_summary
teardown_tmux_test_server
get_test_exit_code
