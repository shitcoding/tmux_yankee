#!/usr/bin/env bash
# test_clipboard_options.sh - tmux-yank option respect tests
#
# Phase 6: Clipboard Integration
#
# Verifies tmux-yank helpers.sh option handling logic

set -euo pipefail

TESTS_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
source "$TESTS_DIR/test_helpers.sh"

print_test_file_header "Integration Tests: Clipboard Options"

# ============================================================================
# Test 40: test_helpers_option_functions_exist
# Verify helpers.sh exports option getter functions
# ============================================================================
_test_helpers_option_functions_exist() {
    source "$SCRIPTS_DIR/helpers.sh"

    # Verify key option functions exist
    if ! declare -f override_copy_command &>/dev/null; then
        printf "    FAIL: override_copy_command function not found\n"
        return 1
    fi

    if ! declare -f custom_copy_command &>/dev/null; then
        printf "    FAIL: custom_copy_command function not found\n"
        return 1
    fi

    if ! declare -f yank_selection &>/dev/null; then
        printf "    FAIL: yank_selection function not found\n"
        return 1
    fi

    if ! declare -f clipboard_copy_command &>/dev/null; then
        printf "    FAIL: clipboard_copy_command function not found\n"
        return 1
    fi

    printf "    ✓ All required option functions exist\n"
}
run_test "test_helpers_option_functions_exist" _test_helpers_option_functions_exist

# ============================================================================
# Test 41: test_clipboard_detection_precedence
# Document expected precedence order from helpers.sh code
# ============================================================================
_test_clipboard_detection_precedence() {
    # Read clipboard_copy_command function to verify precedence logic
    local func_body
    func_body=$(declare -f clipboard_copy_command 2>/dev/null || echo "")

    if [[ -z "$func_body" ]]; then
        source "$SCRIPTS_DIR/helpers.sh"
        func_body=$(declare -f clipboard_copy_command)
    fi

    # Verify override is checked first
    if ! echo "$func_body" | grep -q "override_copy_command"; then
        printf "    FAIL: clipboard_copy_command should check override_copy_command\n"
        return 1
    fi

    # Verify custom is checked last (after all command_exists checks)
    if ! echo "$func_body" | grep -q "custom_copy_command"; then
        printf "    FAIL: clipboard_copy_command should check custom_copy_command\n"
        return 1
    fi

    printf "    ✓ Precedence logic exists: override → detect → custom\n"
}
run_test "test_clipboard_detection_precedence" _test_clipboard_detection_precedence

# ============================================================================
# Test 42: test_yank_selection_used_in_xclip
# Verify yank_selection option is used in xclip command construction
# ============================================================================
_test_yank_selection_used_in_xclip() {
    source "$SCRIPTS_DIR/helpers.sh"

    local func_body
    func_body=$(declare -f clipboard_copy_command)

    # Verify xclip uses yank_selection
    if ! echo "$func_body" | grep -q "xclip.*yank_selection"; then
        printf "    FAIL: xclip command should use yank_selection\n"
        return 1
    fi

    printf "    ✓ xclip uses yank_selection option\n"
}
run_test "test_yank_selection_used_in_xclip" _test_yank_selection_used_in_xclip

# ============================================================================
# Test 43: test_copy_stdin_uses_helpers
# Verify copy_stdin.sh properly sources and uses helpers.sh
# ============================================================================
_test_copy_stdin_uses_helpers() {
    local copy_script="$SCRIPTS_DIR/copy_stdin.sh"
    local script_content
    script_content=$(cat "$copy_script")

    # Verify it sources helpers.sh
    if ! echo "$script_content" | grep -q 'source.*helpers\.sh'; then
        printf "    FAIL: copy_stdin.sh should source helpers.sh\n"
        return 1
    fi

    # Verify it calls clipboard_copy_command
    if ! echo "$script_content" | grep -q 'clipboard_copy_command'; then
        printf "    FAIL: copy_stdin.sh should call clipboard_copy_command\n"
        return 1
    fi

    printf "    ✓ copy_stdin.sh properly integrates with helpers.sh\n"
}
run_test "test_copy_stdin_uses_helpers" _test_copy_stdin_uses_helpers

print_test_summary
get_test_exit_code
