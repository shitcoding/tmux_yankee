#!/usr/bin/env bash
# test_cleanup.sh - Integration tests for cleanup and state restoration
#
# Tests 25-26, 28: Keybinding restoration, copy stripping, cleanup on kill
#
# These tests require tmux to be available.

set -euo pipefail

TESTS_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
source "$TESTS_DIR/test_helpers.sh"

print_test_file_header "Integration Tests: Cleanup & Restoration"

# --- Test prerequisites ---
require_tmux

# ============================================================================
# Test 25: test_keybinding_restoration
# After exiting line-numbered view, verify copy-mode-vi bindings are standard
#
# Acceptance criteria:
#   1. Record standard copy-mode-vi bindings for q, Escape, Enter, y
#   2. Trigger line numbers (which overrides these bindings)
#   3. Exit line numbers view
#   4. Verify bindings are restored to standard values
# ============================================================================
_test_keybinding_restoration() {
    local session_name
    session_name=$(setup_tmux_test_session "keybind-$$")

    # Capture standard bindings before plugin activation
    local q_binding_before
    q_binding_before=$(tmux_test_cmd list-keys -T copy-mode-vi q 2>/dev/null || echo "")

    local esc_binding_before
    esc_binding_before=$(tmux_test_cmd list-keys -T copy-mode-vi Escape 2>/dev/null || echo "")

    # Source plugin and trigger
    if [[ -f "$PROJECT_ROOT/plugin.tmux" ]] && [[ -f "$SCRIPTS_DIR/line_numbers.sh" ]]; then
        tmux_test_cmd send-keys -t "$session_name" "seq 1 10" C-m
        wait_for_tmux_idle 0.3

        tmux_test_cmd send-keys -t "$session_name" \
            "tmux source-file '$PROJECT_ROOT/plugin.tmux'" C-m
        wait_for_tmux_idle 0.3

        tmux_test_cmd send-keys -t "$session_name" \
            "tmux run-shell '$SCRIPTS_DIR/line_numbers.sh'" C-m
        wait_for_tmux_idle 0.5

        # Exit
        tmux_test_cmd send-keys -t "$session_name" q
        wait_for_tmux_idle 0.5
    else
        printf "    plugin files not found, test will fail as expected\n"
        teardown_tmux_test_session "$session_name"
        return 1
    fi

    # Capture bindings after plugin cleanup
    local q_binding_after
    q_binding_after=$(tmux_test_cmd list-keys -T copy-mode-vi q 2>/dev/null || echo "")

    local esc_binding_after
    esc_binding_after=$(tmux_test_cmd list-keys -T copy-mode-vi Escape 2>/dev/null || echo "")

    # Standard q binding should contain "cancel"
    assert_contains "$q_binding_after" "cancel" \
        "q binding should be restored to 'cancel' after cleanup"

    # Standard Escape binding should contain "cancel"
    assert_contains "$esc_binding_after" "cancel" \
        "Escape binding should be restored to 'cancel' after cleanup"

    # Bindings should not contain "wait-for" (that's our plugin override)
    assert_not_contains "$q_binding_after" "wait-for" \
        "q binding should not contain 'wait-for' after cleanup"

    assert_not_contains "$esc_binding_after" "wait-for" \
        "Escape binding should not contain 'wait-for' after cleanup"

    # Cleanup
    teardown_tmux_test_session "$session_name"
}
run_test "test_keybinding_restoration" _test_keybinding_restoration

# ============================================================================
# Test 26: test_copy_strips_numbers
# In numbered view, select text, copy with y, verify paste has no numbers
#
# Acceptance criteria:
#   1. Trigger line numbers view
#   2. Enter copy-mode in the numbered pane
#   3. Select some text (using vi motions)
#   4. Copy with y
#   5. Read tmux paste buffer
#   6. Verify copied text does NOT contain line number prefixes
# ============================================================================
_test_copy_strips_numbers() {
    local session_name
    session_name=$(setup_tmux_test_session "copy-$$")

    # Create known content
    tmux_test_cmd send-keys -t "$session_name" "printf 'alpha\nbeta\ngamma\n'" C-m
    wait_for_tmux_idle 0.5

    # Source plugin and trigger
    if [[ -f "$PROJECT_ROOT/plugin.tmux" ]] && [[ -f "$SCRIPTS_DIR/line_numbers.sh" ]]; then
        tmux_test_cmd send-keys -t "$session_name" \
            "tmux source-file '$PROJECT_ROOT/plugin.tmux'" C-m
        wait_for_tmux_idle 0.3

        tmux_test_cmd send-keys -t "$session_name" \
            "tmux run-shell '$SCRIPTS_DIR/line_numbers.sh'" C-m
        wait_for_tmux_idle 0.5

        # Enter copy-mode if not already
        tmux_test_cmd copy-mode -t "$session_name" 2>/dev/null || true
        wait_for_tmux_idle 0.2

        # Move to start of line, select one line, yank
        # This uses vi copy-mode commands
        tmux_test_cmd send-keys -t "$session_name" -X begin-selection 2>/dev/null || true
        tmux_test_cmd send-keys -t "$session_name" -X end-of-line 2>/dev/null || true
        tmux_test_cmd send-keys -t "$session_name" -X copy-selection-and-cancel 2>/dev/null || true
        wait_for_tmux_idle 0.3

        # Read the paste buffer
        local buffer_content
        buffer_content=$(tmux_test_cmd show-buffer 2>/dev/null || echo "")

        # Buffer should NOT contain line number prefix pattern
        local has_numbers=false
        if printf '%s\n' "$buffer_content" | grep -qE '^\s*[0-9]+\s*\|'; then
            has_numbers=true
        fi

        assert_equal "false" "$has_numbers" \
            "copied text should not contain line number prefixes"

        # Exit line numbers view
        tmux_test_cmd send-keys -t "$session_name" q 2>/dev/null || true
        wait_for_tmux_idle 0.3
    else
        printf "    plugin files not found, test will fail as expected\n"
        teardown_tmux_test_session "$session_name"
        return 1
    fi

    # Cleanup
    teardown_tmux_test_session "$session_name"
}
run_test "test_copy_strips_numbers" _test_copy_strips_numbers

# ============================================================================
# Test 28: test_cleanup_on_kill
# Trigger line numbers, then kill the script process, verify cleanup happened
#
# Acceptance criteria:
#   1. Trigger line numbers view
#   2. Find and kill the line_numbers.sh process
#   3. Wait briefly for trap to fire
#   4. Verify: no orphaned temp panes remain
#   5. Verify: original pane is accessible
# ============================================================================
_test_cleanup_on_kill() {
    local session_name
    session_name=$(setup_tmux_test_session "kill-$$")

    # Create content
    tmux_test_cmd send-keys -t "$session_name" "seq 1 10" C-m
    wait_for_tmux_idle 0.5

    local pane_count_before
    pane_count_before=$(get_tmux_pane_count "$session_name")

    if [[ -f "$PROJECT_ROOT/plugin.tmux" ]] && [[ -f "$SCRIPTS_DIR/line_numbers.sh" ]]; then
        tmux_test_cmd send-keys -t "$session_name" \
            "tmux source-file '$PROJECT_ROOT/plugin.tmux'" C-m
        wait_for_tmux_idle 0.3

        # Run line_numbers.sh in background so we can kill it
        tmux_test_cmd send-keys -t "$session_name" \
            "'$SCRIPTS_DIR/line_numbers.sh' &" C-m
        wait_for_tmux_idle 0.5

        # Find the line_numbers.sh process
        local ln_pid
        ln_pid=$(pgrep -f "line_numbers.sh" 2>/dev/null | head -1 || echo "")

        if [[ -n "$ln_pid" ]]; then
            # Kill it (SIGTERM - should trigger trap)
            kill -TERM "$ln_pid" 2>/dev/null || true
            wait_for_tmux_idle 0.5
        fi

        # Verify no orphaned panes
        local pane_count_after
        pane_count_after=$(get_tmux_pane_count "$session_name")

        assert_equal "$pane_count_before" "$pane_count_after" \
            "no orphaned panes should remain after killing line_numbers.sh"
    else
        printf "    plugin files not found, test will fail as expected\n"
        teardown_tmux_test_session "$session_name"
        return 1
    fi

    # Cleanup
    teardown_tmux_test_session "$session_name"
}
run_test "test_cleanup_on_kill" _test_cleanup_on_kill

# --- Print summary and exit ---
print_test_summary
teardown_tmux_test_server
get_test_exit_code
