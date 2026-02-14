#!/usr/bin/env bash
# test_toggle.sh - Integration tests for mode toggle functionality
#
# Test 27: Toggle cycles mode (hybrid -> absolute -> relative -> hybrid)
#
# These tests require tmux to be available.

set -euo pipefail

TESTS_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
source "$TESTS_DIR/test_helpers.sh"

print_test_file_header "Integration Tests: Toggle Mode"

# --- Test prerequisites ---
require_tmux

# ============================================================================
# Test 27: test_toggle_cycles_mode
# Enter numbered view, press L three times, verify mode cycle
#
# Acceptance criteria:
#   1. Set initial mode to "hybrid"
#   2. Trigger line numbers view
#   3. Press L -> mode should become "absolute"
#   4. Press L -> mode should become "relative"
#   5. Press L -> mode should become "hybrid"
#   6. Verify via @linenumbers-mode option value after each toggle
# ============================================================================
_test_toggle_cycles_mode() {
    local session_name
    session_name=$(setup_tmux_test_session "toggle-$$")

    # Create content
    tmux_test_cmd send-keys -t "$session_name" "seq 1 20" C-m
    wait_for_tmux_idle 0.5

    if [[ ! -f "$PROJECT_ROOT/plugin.tmux" ]] || [[ ! -f "$SCRIPTS_DIR/line_numbers.sh" ]]; then
        printf "    plugin files not found, test will fail as expected\n"
        teardown_tmux_test_session "$session_name"
        return 1
    fi

    # Source plugin
    tmux_test_cmd send-keys -t "$session_name" \
        "tmux source-file '$PROJECT_ROOT/plugin.tmux'" C-m
    wait_for_tmux_idle 0.3

    # Set initial mode to hybrid
    tmux_test_cmd set-option -g "@linenumbers-mode" "hybrid"

    # Trigger line numbers view
    tmux_test_cmd send-keys -t "$session_name" \
        "tmux run-shell '$SCRIPTS_DIR/line_numbers.sh'" C-m
    wait_for_tmux_idle 0.5

    # Verify initial mode is hybrid
    local mode
    mode=$(tmux_test_cmd show-option -gqv "@linenumbers-mode")
    assert_equal "hybrid" "$mode" "initial mode should be hybrid"

    # Press L (toggle key) -> should cycle to absolute
    tmux_test_cmd send-keys -t "$session_name" L
    wait_for_tmux_idle 0.3
    mode=$(tmux_test_cmd show-option -gqv "@linenumbers-mode")
    assert_equal "absolute" "$mode" \
        "after first L press, mode should be 'absolute'"

    # Press L again -> should cycle to relative
    tmux_test_cmd send-keys -t "$session_name" L
    wait_for_tmux_idle 0.3
    mode=$(tmux_test_cmd show-option -gqv "@linenumbers-mode")
    assert_equal "relative" "$mode" \
        "after second L press, mode should be 'relative'"

    # Press L again -> should cycle back to hybrid
    tmux_test_cmd send-keys -t "$session_name" L
    wait_for_tmux_idle 0.3
    mode=$(tmux_test_cmd show-option -gqv "@linenumbers-mode")
    assert_equal "hybrid" "$mode" \
        "after third L press, mode should be 'hybrid'"

    # Exit
    tmux_test_cmd send-keys -t "$session_name" q
    wait_for_tmux_idle 0.3

    # Cleanup
    teardown_tmux_test_session "$session_name"
}
run_test "test_toggle_cycles_mode" _test_toggle_cycles_mode

# --- Print summary and exit ---
print_test_summary
teardown_tmux_test_server
get_test_exit_code
