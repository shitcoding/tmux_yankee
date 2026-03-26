#!/usr/bin/env bash
# test_linux_lifecycle.sh - Full tmux lifecycle test for Linux
#
# Tests that tmux-yankee launches, renders the TUI with line numbers,
# exits cleanly, and leaves no orphaned state.
#
# Usage:
#   ./tests/integration/test_linux_lifecycle.sh [shell_path]
#   shell_path defaults to /bin/bash

set -euo pipefail

TESTS_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
source "$TESTS_DIR/test_helpers.sh"

SHELL_PATH="${1:-/bin/bash}"
SHELL_NAME="$(basename "$SHELL_PATH")"
TMUX_SOCKET="yankee-linux-test-$$"
PROJECT_ROOT="$(cd "$TESTS_DIR/.." && pwd)"

print_test_file_header "Linux Lifecycle Tests (shell=$SHELL_NAME)"

# --- Helpers ---

tmux_cmd() {
    tmux -f /dev/null -L "$TMUX_SOCKET" "$@"
}

cleanup() {
    tmux_cmd kill-server 2>/dev/null || true
}
trap cleanup EXIT

# ============================================================================
# Test: Binary starts on Linux
# ============================================================================
_test_binary_starts() {
    local output
    output=$("$PROJECT_ROOT/bin/tmux-yankee" --help 2>&1 || true)

    local has_usage=false
    if printf '%s' "$output" | grep -qiE 'usage|flag|help|tmux-yankee'; then
        has_usage=true
    fi

    assert_equal "true" "$has_usage" "binary should print usage info"
}
run_test "binary_starts_on_linux ($SHELL_NAME)" _test_binary_starts

# ============================================================================
# Test: Full lifecycle - launch, render, exit, cleanup
# ============================================================================
_test_full_lifecycle() {
    local session="lifecycle-$$"

    # Create tmux session with specified shell
    tmux_cmd new-session -d -s "$session" -x 120 -y 40 "$SHELL_PATH"
    sleep 0.5

    # Source the plugin (bash entry point, NOT tmux source-file)
    tmux_cmd send-keys -t "$session" "bash '$PROJECT_ROOT/yankee.tmux'" C-m
    sleep 0.5

    # Generate content
    tmux_cmd send-keys -t "$session" "seq 1 50" C-m
    sleep 0.5

    # Get pane ID for launch script
    local pane_id window_id pane_index
    pane_id=$(tmux_cmd display-message -t "$session" -p '#{pane_id}')
    window_id=$(tmux_cmd display-message -t "$session" -p '#{window_id}')
    pane_index=$(tmux_cmd display-message -t "$session" -p '#{pane_index}')

    # Launch yankee (same path as real usage via binding)
    tmux_cmd run-shell -t "$session" \
        "bash '$PROJECT_ROOT/scripts/launch_yankee.sh' '$pane_id' '$window_id' '$pane_index'" &
    local launch_pid=$!

    # Wait for TUI to appear (poll for Unicode separator)
    local tui_appeared=false
    local tries=0
    while [ "$tries" -lt 30 ]; do
        sleep 0.5
        local content
        content=$(tmux_cmd capture-pane -t "$session" -p 2>/dev/null || true)
        # Check for Unicode box-drawing separator (│ = U+2502)
        if printf '%s' "$content" | grep -q '│'; then
            tui_appeared=true
            break
        fi
        tries=$((tries + 1))
    done

    assert_equal "true" "$tui_appeared" \
        "TUI should render with Unicode separator (shell=$SHELL_NAME)"

    # Capture TUI content for inspection
    local tui_content
    tui_content=$(tmux_cmd capture-pane -t "$session" -p 2>/dev/null || true)

    # Verify line numbers are visible (digits before the separator)
    local has_line_numbers=false
    if printf '%s\n' "$tui_content" | grep -qE '[0-9]+\s*│'; then
        has_line_numbers=true
    fi
    assert_equal "true" "$has_line_numbers" \
        "TUI should display line numbers before separator (shell=$SHELL_NAME)"

    # Exit yankee by sending q
    tmux_cmd send-keys -t "$session" q
    sleep 1

    # Wait for launch script to finish
    wait "$launch_pid" 2>/dev/null || true

    # Verify cleanup: no helper sessions
    local helper_sessions
    helper_sessions=$(tmux_cmd list-sessions -F '#{session_name}' 2>/dev/null | grep -c 'tmux-yankee-ovl' || true)
    assert_equal "0" "$helper_sessions" \
        "no tmux-yankee-ovl helper sessions should remain (shell=$SHELL_NAME)"

    # Verify cleanup: no @yankee_busy flag
    local busy_flag
    busy_flag=$(tmux_cmd show-options -pqv -t "$session" @yankee_busy 2>/dev/null || true)
    assert_equal "" "$busy_flag" \
        "@yankee_busy should be cleared (shell=$SHELL_NAME)"

    # Verify cleanup: no state/lock files
    local state_files
    state_files=$(find /tmp/tmux-yankee -name "*.state" 2>/dev/null | wc -l | tr -d ' ')
    assert_equal "0" "$state_files" \
        "no state files should remain in /tmp/tmux-yankee (shell=$SHELL_NAME)"

    local lock_dirs
    lock_dirs=$(find /tmp/tmux-yankee -name "*.lock" -type d 2>/dev/null | wc -l | tr -d ' ')
    assert_equal "0" "$lock_dirs" \
        "no lock dirs should remain in /tmp/tmux-yankee (shell=$SHELL_NAME)"

    # Cleanup
    tmux_cmd kill-session -t "$session" 2>/dev/null || true
}
run_test "full_lifecycle ($SHELL_NAME)" _test_full_lifecycle

# ============================================================================
# Test: No orphaned panes after lifecycle
# ============================================================================
_test_no_orphaned_panes() {
    local session="orphan-$$"

    tmux_cmd new-session -d -s "$session" -x 80 -y 24 "$SHELL_PATH"
    sleep 0.3

    # Record pane count before
    local count_before
    count_before=$(tmux_cmd list-panes -t "$session" -F '#{pane_id}' | wc -l | tr -d ' ')

    # Generate content and source plugin
    tmux_cmd send-keys -t "$session" "seq 1 20" C-m
    sleep 0.3

    tmux_cmd send-keys -t "$session" "bash '$PROJECT_ROOT/yankee.tmux'" C-m
    sleep 0.3

    local pane_id window_id pane_index
    pane_id=$(tmux_cmd display-message -t "$session" -p '#{pane_id}')
    window_id=$(tmux_cmd display-message -t "$session" -p '#{window_id}')
    pane_index=$(tmux_cmd display-message -t "$session" -p '#{pane_index}')

    tmux_cmd run-shell -t "$session" \
        "bash '$PROJECT_ROOT/scripts/launch_yankee.sh' '$pane_id' '$window_id' '$pane_index'" &
    local launch_pid=$!

    # Wait for TUI
    sleep 2

    # Exit
    tmux_cmd send-keys -t "$session" q
    sleep 1
    wait "$launch_pid" 2>/dev/null || true

    # Check pane count
    local count_after
    count_after=$(tmux_cmd list-panes -t "$session" -F '#{pane_id}' | wc -l | tr -d ' ')

    assert_equal "$count_before" "$count_after" \
        "pane count should be same after lifecycle (no orphans, shell=$SHELL_NAME)"

    tmux_cmd kill-session -t "$session" 2>/dev/null || true
}
run_test "no_orphaned_panes ($SHELL_NAME)" _test_no_orphaned_panes

# --- Summary ---
print_test_summary
cleanup
get_test_exit_code
