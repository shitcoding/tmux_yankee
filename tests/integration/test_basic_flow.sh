#!/usr/bin/env bash
# test_basic_flow.sh - Integration tests for basic capture-swap lifecycle
#
# Tests 23-24: Full lifecycle enter/exit, no orphaned panes
#
# These tests require tmux to be available.
# They use an isolated tmux socket to avoid interfering with the user's session.

set -euo pipefail

TESTS_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
source "$TESTS_DIR/test_helpers.sh"

print_test_file_header "Integration Tests: Basic Flow"

# --- Test prerequisites ---
require_tmux

# ============================================================================
# Test 23: test_full_lifecycle_enter_exit
# Trigger line numbers, verify numbered pane visible, press q, verify restored
#
# Acceptance criteria:
#   1. Source plugin.tmux successfully
#   2. Trigger line_numbers.sh via run-shell
#   3. Verify the pane shows line numbers (regex: "^\s*\d+\s*\|")
#   4. Exit via the wait-for channel (simulating q key)
#   5. Verify original pane content is restored
# ============================================================================
_test_full_lifecycle_enter_exit() {
    local session_name
    session_name=$(setup_tmux_test_session "lifecycle-$$")

    # Create some content in the pane
    tmux_test_cmd send-keys -t "$session_name" "seq 1 20" C-m
    wait_for_tmux_idle 0.5

    # Verify plugin.tmux exists
    assert_file_exists "$PROJECT_ROOT/plugin.tmux" \
        "plugin.tmux should exist"

    # Source the plugin
    tmux_test_cmd send-keys -t "$session_name" \
        "tmux source-file '$PROJECT_ROOT/plugin.tmux'" C-m
    wait_for_tmux_idle 0.3

    # Verify line_numbers.sh exists
    assert_file_exists "$SCRIPTS_DIR/line_numbers.sh" \
        "scripts/line_numbers.sh should exist"

    # Trigger line numbers view
    tmux_test_cmd send-keys -t "$session_name" \
        "tmux run-shell '$SCRIPTS_DIR/line_numbers.sh'" C-m
    wait_for_tmux_idle 0.5

    # Capture the pane content and verify line numbers are present
    local pane_content
    pane_content=$(tmux_test_cmd capture-pane -t "$session_name" -p)

    # Line numbers should match pattern: optional spaces, digits, space, pipe, space
    local has_line_numbers=false
    if printf '%s\n' "$pane_content" | grep -qE '^\s*[0-9]+\s*\|'; then
        has_line_numbers=true
    fi

    assert_equal "true" "$has_line_numbers" \
        "pane should display line numbers after triggering"

    # Exit the line-numbered view by signaling the wait-for channel
    # (In real usage, q/Escape would do this. We simulate via direct signal.)
    # The channel name is "linenumbers-<PID>", but we don't know the PID.
    # Alternative: send 'q' key to the pane
    tmux_test_cmd send-keys -t "$session_name" q
    wait_for_tmux_idle 0.5

    # Verify original content is restored
    local restored_content
    restored_content=$(tmux_test_cmd capture-pane -t "$session_name" -p)

    # Original content should NOT have line number prefixes
    local still_has_numbers=false
    if printf '%s\n' "$restored_content" | grep -qE '^\s*[0-9]+\s*\|'; then
        still_has_numbers=true
    fi

    assert_equal "false" "$still_has_numbers" \
        "pane should be restored to original content after exit"

    # Cleanup
    teardown_tmux_test_session "$session_name"
}
run_test "test_full_lifecycle_enter_exit" _test_full_lifecycle_enter_exit

# ============================================================================
# Test 24: test_no_orphaned_panes
# Record pane count before, trigger and exit, verify pane count same after
#
# Acceptance criteria:
#   1. Count panes before triggering
#   2. Trigger line numbers, then exit
#   3. Count panes after
#   4. Pane count should be identical
# ============================================================================
_test_no_orphaned_panes() {
    local session_name
    session_name=$(setup_tmux_test_session "orphan-$$")

    # Create content
    tmux_test_cmd send-keys -t "$session_name" "seq 1 10" C-m
    wait_for_tmux_idle 0.5

    # Record pane count before
    local pane_count_before
    pane_count_before=$(get_tmux_pane_count "$session_name")

    # Source plugin and trigger
    if [[ -f "$PROJECT_ROOT/plugin.tmux" ]]; then
        tmux_test_cmd send-keys -t "$session_name" \
            "tmux source-file '$PROJECT_ROOT/plugin.tmux'" C-m
        wait_for_tmux_idle 0.3
    fi

    if [[ -f "$SCRIPTS_DIR/line_numbers.sh" ]]; then
        tmux_test_cmd send-keys -t "$session_name" \
            "tmux run-shell '$SCRIPTS_DIR/line_numbers.sh'" C-m
        wait_for_tmux_idle 0.5

        # Exit
        tmux_test_cmd send-keys -t "$session_name" q
        wait_for_tmux_idle 0.5
    fi

    # Record pane count after
    local pane_count_after
    pane_count_after=$(get_tmux_pane_count "$session_name")

    assert_equal "$pane_count_before" "$pane_count_after" \
        "pane count should be same after enter+exit (no orphaned panes)"

    # Cleanup
    teardown_tmux_test_session "$session_name"
}
run_test "test_no_orphaned_panes" _test_no_orphaned_panes

# --- Print summary and exit ---
print_test_summary
teardown_tmux_test_server
get_test_exit_code
