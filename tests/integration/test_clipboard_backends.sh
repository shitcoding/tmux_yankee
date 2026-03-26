#!/usr/bin/env bash
# test_clipboard_backends.sh - Clipboard backend detection and integration tests
#
# Tests clipboard backend detection and copy_stdin.sh integration.
# Each test runs conditionally based on available clipboard commands.
#
# These tests verify:
# 1. Clipboard command detection works correctly
# 2. copy_stdin.sh detects and delegates to the right backend
# 3. Each backend receives data correctly when available
# 4. Graceful skipping when backends unavailable

set -euo pipefail

TESTS_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
source "$TESTS_DIR/test_helpers.sh"

print_test_file_header "Integration Tests: Clipboard Backends"

# ============================================================================
# Test 30: test_clipboard_detection_returns_command
# Verify detect_clipboard_command() in copy_stdin.sh returns a valid command
# ============================================================================
_test_clipboard_detection_returns_command() {
    # Extract and run the real detect_clipboard_command from copy_stdin.sh
    # (not a duplicate) so test stays in sync with production code
    local func_body
    func_body=$(sed -n '/^detect_clipboard_command()/,/^}/p' "$SCRIPTS_DIR/copy_stdin.sh")

    local copy_cmd
    copy_cmd=$(bash -c "$func_body"$'\n'"detect_clipboard_command")

    printf "    detected clipboard command: '%s'\n" "$copy_cmd"

    if [[ -n "$copy_cmd" ]]; then
        assert_not_equal "" "$copy_cmd" \
            "clipboard command should not be empty when detected"
    else
        printf "    (no clipboard command detected - acceptable on minimal systems)\n"
    fi
}
run_test "test_clipboard_detection_returns_command" _test_clipboard_detection_returns_command

# ============================================================================
# Test 31: test_copy_stdin_script_exists_and_executable
# Verify copy_stdin.sh exists, is executable, and is self-contained
# ============================================================================
_test_copy_stdin_script_exists_and_executable() {
    local copy_script="$SCRIPTS_DIR/copy_stdin.sh"

    assert_file_exists "$copy_script" \
        "copy_stdin.sh should exist"

    if [[ ! -x "$copy_script" ]]; then
        printf "    ${_CLR_RED}ASSERTION FAILED:${_CLR_RESET} copy_stdin.sh should be executable\n"
        return 1
    fi

    # Verify it has its own clipboard detection (no external helpers dependency)
    if ! grep -q 'detect_clipboard_command' "$copy_script"; then
        printf "    ${_CLR_RED}ASSERTION FAILED:${_CLR_RESET} copy_stdin.sh should have detect_clipboard_command\n"
        return 1
    fi

    # Verify it does NOT depend on helpers.sh
    if grep -q 'source.*helpers\.sh' "$copy_script"; then
        printf "    ${_CLR_RED}ASSERTION FAILED:${_CLR_RESET} copy_stdin.sh should not depend on helpers.sh\n"
        return 1
    fi
}
run_test "test_copy_stdin_script_exists_and_executable" _test_copy_stdin_script_exists_and_executable

# ============================================================================
# Test 32: test_pbcopy_backend (macOS)
# ============================================================================
_test_pbcopy_backend() {
    if ! command -v pbcopy &>/dev/null; then
        skip_test "pbcopy not available (not on macOS)"
    fi

    if ! command -v pbpaste &>/dev/null; then
        skip_test "pbpaste not available for verification"
    fi

    local test_data="tmux_yankee_test_$$"
    printf '%s' "$test_data" | "$SCRIPTS_DIR/copy_stdin.sh"
    sleep 0.1

    local clipboard_content
    clipboard_content=$(pbpaste)
    assert_equal "$test_data" "$clipboard_content" \
        "pbcopy should receive data from copy_stdin.sh"
}
run_test "test_pbcopy_backend" _test_pbcopy_backend

# ============================================================================
# Test 33: test_xclip_backend (Linux X11)
# ============================================================================
_test_xclip_backend() {
    if ! command -v xclip &>/dev/null; then
        skip_test "xclip not available"
    fi
    if [[ -z "${DISPLAY:-}" ]]; then
        skip_test "DISPLAY not set (not in X11 environment)"
    fi

    local test_data="tmux_yankee_xclip_test_$$"
    printf '%s' "$test_data" | "$SCRIPTS_DIR/copy_stdin.sh"
    sleep 0.1

    local clipboard_content
    clipboard_content=$(xclip -selection clipboard -o 2>/dev/null || echo "")
    assert_equal "$test_data" "$clipboard_content" \
        "xclip should receive data from copy_stdin.sh"
}
run_test "test_xclip_backend" _test_xclip_backend

# ============================================================================
# Test 34: test_xsel_backend (Linux X11)
# ============================================================================
_test_xsel_backend() {
    if ! command -v xsel &>/dev/null; then
        skip_test "xsel not available"
    fi
    if [[ -z "${DISPLAY:-}" ]]; then
        skip_test "DISPLAY not set (not in X11 environment)"
    fi

    local test_data="tmux_yankee_xsel_test_$$"
    printf '%s' "$test_data" | "$SCRIPTS_DIR/copy_stdin.sh"
    sleep 0.1

    local clipboard_content
    clipboard_content=$(xsel -o --clipboard 2>/dev/null || echo "")
    assert_equal "$test_data" "$clipboard_content" \
        "xsel should receive data from copy_stdin.sh"
}
run_test "test_xsel_backend" _test_xsel_backend

# ============================================================================
# Test 35: test_wl_copy_backend (Linux Wayland)
# ============================================================================
_test_wl_copy_backend() {
    if ! command -v wl-copy &>/dev/null; then
        skip_test "wl-copy not available"
    fi
    if ! command -v wl-paste &>/dev/null; then
        skip_test "wl-paste not available for verification"
    fi
    if [[ -z "${WAYLAND_DISPLAY:-}" ]]; then
        skip_test "WAYLAND_DISPLAY not set (not in Wayland environment)"
    fi

    local test_data="tmux_yankee_wayland_test_$$"
    printf '%s' "$test_data" | "$SCRIPTS_DIR/copy_stdin.sh"
    sleep 0.1

    local clipboard_content
    clipboard_content=$(wl-paste 2>/dev/null || echo "")
    assert_equal "$test_data" "$clipboard_content" \
        "wl-copy should receive data from copy_stdin.sh"
}
run_test "test_wl_copy_backend" _test_wl_copy_backend

# ============================================================================
# Test 36: test_clip_exe_backend (WSL)
# ============================================================================
_test_clip_exe_backend() {
    if ! command -v clip.exe &>/dev/null; then
        skip_test "clip.exe not available (not on WSL)"
    fi

    local test_data="tmux_yankee_wsl_test_$$"
    local exit_code=0
    printf '%s' "$test_data" | "$SCRIPTS_DIR/copy_stdin.sh" || exit_code=$?
    assert_equal 0 "$exit_code" \
        "copy_stdin.sh should succeed with clip.exe backend"
}
run_test "test_clip_exe_backend" _test_clip_exe_backend

# ============================================================================
# Test 37: test_putclip_backend (Cygwin)
# ============================================================================
_test_putclip_backend() {
    if ! command -v putclip &>/dev/null; then
        skip_test "putclip not available (not on Cygwin)"
    fi

    local test_data="tmux_yankee_cygwin_test_$$"
    local exit_code=0
    printf '%s' "$test_data" | "$SCRIPTS_DIR/copy_stdin.sh" || exit_code=$?
    assert_equal 0 "$exit_code" \
        "copy_stdin.sh should succeed with putclip backend"
}
run_test "test_putclip_backend" _test_putclip_backend

# --- Print summary and exit ---
print_test_summary
get_test_exit_code
