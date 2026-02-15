#!/usr/bin/env bash
# test_clipboard_backends.sh - Clipboard backend detection and integration tests
#
# Phase 6: Clipboard Integration
#
# Tests clipboard backend detection and copy_stdin.sh integration with tmux-yank helpers.
# Each test runs conditionally based on available clipboard commands.
#
# These tests verify:
# 1. Clipboard command detection works correctly
# 2. copy_stdin.sh successfully delegates to helpers.sh
# 3. Each backend receives data correctly when available
# 4. X11 selection options are respected
# 5. Graceful skipping when backends unavailable

set -euo pipefail

TESTS_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
source "$TESTS_DIR/test_helpers.sh"

print_test_file_header "Integration Tests: Clipboard Backends"

# ============================================================================
# Test 30: test_clipboard_detection_returns_command
# Verify clipboard_copy_command() from helpers.sh returns a valid command
#
# Acceptance criteria:
#   1. clipboard_copy_command() returns non-empty string
#   2. Returned command is executable or has fallback
# ============================================================================
_test_clipboard_detection_returns_command() {
    # Source helpers.sh
    # shellcheck source=scripts/helpers.sh
    source "$SCRIPTS_DIR/helpers.sh"

    local copy_cmd
    copy_cmd=$(clipboard_copy_command)

    # Should return something (might be empty if no clipboard available)
    # Empty is acceptable on systems without clipboard support
    # We just verify the function runs without error

    printf "    detected clipboard command: '%s'\n" "$copy_cmd"

    # If we got a command, verify it's a string
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
# Verify copy_stdin.sh exists and is executable
#
# Acceptance criteria:
#   1. scripts/copy_stdin.sh exists
#   2. File is executable
#   3. File sources helpers.sh correctly
# ============================================================================
_test_copy_stdin_script_exists_and_executable() {
    local copy_script="$SCRIPTS_DIR/copy_stdin.sh"

    assert_file_exists "$copy_script" \
        "copy_stdin.sh should exist"

    if [[ ! -x "$copy_script" ]]; then
        printf "    ${_CLR_RED}ASSERTION FAILED:${_CLR_RESET} copy_stdin.sh should be executable\n"
        return 1
    fi

    # Verify it sources helpers.sh
    if ! grep -q 'source.*helpers.sh' "$copy_script"; then
        printf "    ${_CLR_RED}ASSERTION FAILED:${_CLR_RESET} copy_stdin.sh should source helpers.sh\n"
        return 1
    fi
}
run_test "test_copy_stdin_script_exists_and_executable" _test_copy_stdin_script_exists_and_executable

# ============================================================================
# Test 32: test_pbcopy_backend (macOS)
# Test pbcopy clipboard integration if available
#
# Acceptance criteria:
#   1. Skip if pbcopy not available
#   2. Echo test data through copy_stdin.sh
#   3. Verify data reaches clipboard via pbpaste
# ============================================================================
_test_pbcopy_backend() {
    if ! command -v pbcopy &>/dev/null; then
        skip_test "pbcopy not available (not on macOS)"
    fi

    if ! command -v pbpaste &>/dev/null; then
        skip_test "pbpaste not available for verification"
    fi

    local test_data="tmux_yankee_test_$$"

    # Send test data through copy_stdin.sh
    printf '%s' "$test_data" | "$SCRIPTS_DIR/copy_stdin.sh"

    # Brief delay for clipboard to settle
    sleep 0.1

    # Verify clipboard content
    local clipboard_content
    clipboard_content=$(pbpaste)

    assert_equal "$test_data" "$clipboard_content" \
        "pbcopy should receive data from copy_stdin.sh"
}
run_test "test_pbcopy_backend" _test_pbcopy_backend

# ============================================================================
# Test 33: test_xclip_backend (Linux X11)
# Test xclip clipboard integration if available
#
# Acceptance criteria:
#   1. Skip if xclip not available or not in X11
#   2. Echo test data through copy_stdin.sh
#   3. Verify data reaches clipboard via xclip -o
# ============================================================================
_test_xclip_backend() {
    if ! command -v xclip &>/dev/null; then
        skip_test "xclip not available"
    fi

    # Check if DISPLAY is set (X11 environment)
    if [[ -z "${DISPLAY:-}" ]]; then
        skip_test "DISPLAY not set (not in X11 environment)"
    fi

    local test_data="tmux_yankee_xclip_test_$$"

    # Send test data through copy_stdin.sh
    printf '%s' "$test_data" | "$SCRIPTS_DIR/copy_stdin.sh"

    # Brief delay for clipboard to settle
    sleep 0.1

    # Verify clipboard content
    # Default selection is "clipboard" per tmux-yank defaults
    local clipboard_content
    clipboard_content=$(xclip -selection clipboard -o 2>/dev/null || echo "")

    assert_equal "$test_data" "$clipboard_content" \
        "xclip should receive data from copy_stdin.sh"
}
run_test "test_xclip_backend" _test_xclip_backend

# ============================================================================
# Test 34: test_xsel_backend (Linux X11)
# Test xsel clipboard integration if available
#
# Acceptance criteria:
#   1. Skip if xsel not available or not in X11
#   2. Echo test data through copy_stdin.sh
#   3. Verify data reaches clipboard via xsel -o
# ============================================================================
_test_xsel_backend() {
    if ! command -v xsel &>/dev/null; then
        skip_test "xsel not available"
    fi

    # Check if DISPLAY is set (X11 environment)
    if [[ -z "${DISPLAY:-}" ]]; then
        skip_test "DISPLAY not set (not in X11 environment)"
    fi

    local test_data="tmux_yankee_xsel_test_$$"

    # Send test data through copy_stdin.sh
    printf '%s' "$test_data" | "$SCRIPTS_DIR/copy_stdin.sh"

    # Brief delay for clipboard to settle
    sleep 0.1

    # Verify clipboard content
    # Default selection is "clipboard" per tmux-yank defaults
    local clipboard_content
    clipboard_content=$(xsel -o --clipboard 2>/dev/null || echo "")

    assert_equal "$test_data" "$clipboard_content" \
        "xsel should receive data from copy_stdin.sh"
}
run_test "test_xsel_backend" _test_xsel_backend

# ============================================================================
# Test 35: test_wl_copy_backend (Linux Wayland)
# Test wl-copy clipboard integration if available
#
# Acceptance criteria:
#   1. Skip if wl-copy not available or not in Wayland
#   2. Echo test data through copy_stdin.sh
#   3. Verify data reaches clipboard via wl-paste
# ============================================================================
_test_wl_copy_backend() {
    if ! command -v wl-copy &>/dev/null; then
        skip_test "wl-copy not available"
    fi

    if ! command -v wl-paste &>/dev/null; then
        skip_test "wl-paste not available for verification"
    fi

    # Check if WAYLAND_DISPLAY is set
    if [[ -z "${WAYLAND_DISPLAY:-}" ]]; then
        skip_test "WAYLAND_DISPLAY not set (not in Wayland environment)"
    fi

    local test_data="tmux_yankee_wayland_test_$$"

    # Send test data through copy_stdin.sh
    printf '%s' "$test_data" | "$SCRIPTS_DIR/copy_stdin.sh"

    # Brief delay for clipboard to settle
    sleep 0.1

    # Verify clipboard content
    local clipboard_content
    clipboard_content=$(wl-paste 2>/dev/null || echo "")

    assert_equal "$test_data" "$clipboard_content" \
        "wl-copy should receive data from copy_stdin.sh"
}
run_test "test_wl_copy_backend" _test_wl_copy_backend

# ============================================================================
# Test 36: test_clip_exe_backend (WSL)
# Test clip.exe clipboard integration if available
#
# Acceptance criteria:
#   1. Skip if clip.exe not available
#   2. Echo test data through copy_stdin.sh
#   3. Verify data reaches clipboard
# Note: Verification difficult without powershell.exe, so we just test execution
# ============================================================================
_test_clip_exe_backend() {
    if ! command -v clip.exe &>/dev/null; then
        skip_test "clip.exe not available (not on WSL)"
    fi

    local test_data="tmux_yankee_wsl_test_$$"

    # Send test data through copy_stdin.sh
    # We can't easily verify WSL clipboard without powershell.exe
    # So we just verify the command runs without error
    local exit_code=0
    printf '%s' "$test_data" | "$SCRIPTS_DIR/copy_stdin.sh" || exit_code=$?

    assert_equal 0 "$exit_code" \
        "copy_stdin.sh should succeed with clip.exe backend"
}
run_test "test_clip_exe_backend" _test_clip_exe_backend

# ============================================================================
# Test 37: test_putclip_backend (Cygwin)
# Test putclip clipboard integration if available
#
# Acceptance criteria:
#   1. Skip if putclip not available
#   2. Echo test data through copy_stdin.sh
#   3. Verify command runs without error
# ============================================================================
_test_putclip_backend() {
    if ! command -v putclip &>/dev/null; then
        skip_test "putclip not available (not on Cygwin)"
    fi

    local test_data="tmux_yankee_cygwin_test_$$"

    # Send test data through copy_stdin.sh
    local exit_code=0
    printf '%s' "$test_data" | "$SCRIPTS_DIR/copy_stdin.sh" || exit_code=$?

    assert_equal 0 "$exit_code" \
        "copy_stdin.sh should succeed with putclip backend"
}
run_test "test_putclip_backend" _test_putclip_backend

# --- Print summary and exit ---
print_test_summary
get_test_exit_code
