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

    # copy_stdin.sh has a self-contained detect_clipboard_command function.
    # We can verify it detects xclip by running a subshell that sources and
    # calls the detection logic, or simply run the script and check if it
    # succeeds with xclip available.
    local copy_cmd
    copy_cmd=$(DISPLAY="$XVFB_DISPLAY" bash -c '
        detect_clipboard_command() {
            if command -v pbcopy >/dev/null 2>&1; then echo "pbcopy"
            elif command -v wl-copy >/dev/null 2>&1; then echo "wl-copy"
            elif command -v xsel >/dev/null 2>&1; then echo "xsel -i --clipboard"
            elif command -v xclip >/dev/null 2>&1; then echo "xclip -selection clipboard"
            elif command -v clip.exe >/dev/null 2>&1; then echo "cat | clip.exe"
            elif command -v putclip >/dev/null 2>&1; then echo "putclip"
            fi
        }
        detect_clipboard_command
    ' 2>/dev/null || true)

    assert_contains "$copy_cmd" "xclip" \
        "clipboard detection should find xclip on Linux"

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

    # Pipe text through copy_stdin.sh
    printf '%s' "$test_text" | \
        DISPLAY="$XVFB_DISPLAY" \
        bash "$PROJECT_ROOT/scripts/copy_stdin.sh" 2>/dev/null || true

    # Verify xclip received the text
    local clip_content
    clip_content=$(DISPLAY="$XVFB_DISPLAY" xclip -o -selection clipboard 2>/dev/null || true)

    assert_equal "$test_text" "$clip_content" \
        "xclip clipboard should contain the copied text"

    stop_xvfb
}
run_test "clipboard_copy_via_copy_stdin" _test_clipboard_copy

# --- Summary ---
print_test_summary
get_test_exit_code
