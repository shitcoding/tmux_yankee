#!/usr/bin/env bash
# test_linux_clipboard.sh - Clipboard integration test with Xvfb + xclip
#
# Tests that tmux-yankee correctly detects and uses xclip on Linux
# when an X display is available via Xvfb.
#
# Must be run on Linux with xvfb and xclip installed.

set -euo pipefail

TESTS_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
source "$TESTS_DIR/test_helpers.sh"

PROJECT_ROOT="$(cd "$TESTS_DIR/.." && pwd)"

print_test_file_header "Linux Clipboard Tests (Xvfb + xclip)"

# --- Prerequisites ---

_require_linux() {
    if [[ "$(uname -s)" != "Linux" ]]; then
        skip_test "not on Linux"
    fi
}

_require_xvfb() {
    if ! command -v Xvfb &>/dev/null; then
        skip_test "Xvfb not installed"
    fi
}

_require_xclip() {
    if ! command -v xclip &>/dev/null; then
        skip_test "xclip not installed"
    fi
}

# --- Xvfb lifecycle ---

XVFB_PID=""
XVFB_DISPLAY=":99"

start_xvfb() {
    Xvfb "$XVFB_DISPLAY" -screen 0 1024x768x24 &>/dev/null &
    XVFB_PID=$!

    # Poll for X socket readiness (no fixed sleep)
    local tries=0
    while [ "$tries" -lt 20 ]; do
        if [ -S "/tmp/.X11-unix/X99" ]; then
            export DISPLAY="$XVFB_DISPLAY"
            return 0
        fi
        sleep 0.25
        tries=$((tries + 1))
    done

    echo "ERROR: Xvfb failed to start within 5 seconds"
    return 1
}

stop_xvfb() {
    if [ -n "$XVFB_PID" ]; then
        kill "$XVFB_PID" 2>/dev/null || true
        wait "$XVFB_PID" 2>/dev/null || true
        XVFB_PID=""
    fi
    unset DISPLAY 2>/dev/null || true
}
trap stop_xvfb EXIT

# ============================================================================
# Test: xclip is detected as clipboard backend
# ============================================================================
_test_clipboard_detection() {
    _require_linux
    _require_xvfb
    _require_xclip

    start_xvfb

    # Source helpers.sh and call clipboard_copy_command
    local copy_cmd
    copy_cmd=$(DISPLAY="$XVFB_DISPLAY" bash -c "source '$PROJECT_ROOT/scripts/helpers.sh'; clipboard_copy_command" 2>/dev/null || true)

    # Should detect xclip (not pbcopy, not wl-copy)
    assert_contains "$copy_cmd" "xclip" \
        "clipboard_copy_command should detect xclip on Linux"

    stop_xvfb
}
run_test "clipboard_detection_xclip" _test_clipboard_detection

# ============================================================================
# Test: Actual clipboard copy via copy_stdin.sh
# ============================================================================
_test_clipboard_copy() {
    _require_linux
    _require_xvfb
    _require_xclip

    start_xvfb

    local test_text="Hello from tmux-yankee Linux test $(date +%s)"

    # copy_stdin.sh uses display_message which needs a tmux server
    local tmux_socket="clipboard-test-$$"
    tmux -f /dev/null -L "$tmux_socket" new-session -d -s cliptest -x 80 -y 24
    sleep 0.3

    # Pipe text through copy_stdin.sh
    # The script sources helpers.sh which calls tmux show-option, so it needs
    # the TMUX env to find the server. Set it via the tmux socket.
    printf '%s' "$test_text" | \
        DISPLAY="$XVFB_DISPLAY" \
        bash "$PROJECT_ROOT/scripts/copy_stdin.sh" 2>/dev/null || true

    # Verify xclip received the text
    local clip_content
    clip_content=$(DISPLAY="$XVFB_DISPLAY" xclip -o -selection clipboard 2>/dev/null || true)

    assert_equal "$test_text" "$clip_content" \
        "xclip clipboard should contain the copied text"

    tmux -f /dev/null -L "$tmux_socket" kill-server 2>/dev/null || true
    stop_xvfb
}
run_test "clipboard_copy_via_copy_stdin" _test_clipboard_copy

# --- Summary ---
print_test_summary
get_test_exit_code
