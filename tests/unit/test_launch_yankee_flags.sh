#!/usr/bin/env bash
# Unit tests for launch_yankee.sh build_yankee_args function
#
# Tests that build_yankee_args correctly reads @yankee_* tmux options and
# emits them as null-delimited CLI flag pairs.
#
# Strategy: define a mock tmux function, then source only the two functions
# we need (_append_yankee_opt and build_yankee_args) by extracting them into
# a temporary file via awk, bypassing the top-level side effects of the launcher.

set -euo pipefail

TESTS_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
# shellcheck source=../test_helpers.sh disable=SC1091
source "$TESTS_DIR/test_helpers.sh"

print_test_file_header "Unit Tests: launch_yankee.sh build_yankee_args"

LAUNCH_SCRIPT="$SCRIPTS_DIR/launch_yankee.sh"

if [[ ! -f "$LAUNCH_SCRIPT" ]]; then
    printf "    SKIP: launch_yankee.sh not found at %s\n" "$LAUNCH_SCRIPT"
    exit 77
fi

# ---- Extract testable functions from launcher ----
# We cannot source the full launcher because it executes tmux at top level.
# Extract _append_yankee_opt and build_yankee_args into a temp file.

_TMP_FUNCS="$(mktemp /tmp/test_launch_yankee_funcs.XXXXXX)"
# Register with the framework's temp-file cleanup (via _TEMP_FILES array).
_TEMP_FILES+=("$_TMP_FUNCS")

# awk collects function bodies by tracking brace depth.
awk '
    /^(_append_yankee_opt|build_yankee_args)\(\)/ { in_fn=1; depth=0 }
    in_fn {
        print
        for (i=1; i<=length($0); i++) {
            c = substr($0,i,1)
            if (c == "{") depth++
            else if (c == "}") {
                depth--
                if (depth == 0) { in_fn=0; print ""; next }
            }
        }
    }
' "$LAUNCH_SCRIPT" > "$_TMP_FUNCS"

# ---- Mock environment ----
# PANE_ID and _YANKEE_ARGS are globals referenced inside the extracted functions.
# shellcheck disable=SC2034  # used inside sourced build_yankee_args
PANE_ID="%42"
_YANKEE_ARGS=()

# Mock tmux: simulate set options; all others return empty.
# show-option -gqv @option_name -> $1=show-option $2=-gqv $3=@option_name
# shellcheck disable=SC2317
tmux() {
    if [[ "${1:-}" == "show-option" ]]; then
        local opt="${3:-}"
        case "$opt" in
            @yankee_mode)             printf '%s\n' "absolute" ;;
            @yankee_scrollback_lines) printf '%s\n' "5000" ;;
            @yankee_theme)            printf '%s\n' "dracula" ;;
            @yankee_status_indicator) printf '%s\n' "on" ;;
            @yankee_cursor_bg)        printf '%s\n' "#ff5555" ;;
            @yankee_toggle_mode_key)  printf '%s\n' "M" ;;
            *)                        printf '%s\n' "" ;;
        esac
    fi
}
export -f tmux

# Source the extracted functions.
# shellcheck disable=SC1090
source "$_TMP_FUNCS"

# ---- Collect output once ----
# Call build_yankee_args to populate _YANKEE_ARGS (global side-effect).
# Also capture its null-delimited stdout and convert to space-separated for grep.
build_yankee_args > /tmp/_test_yankee_args_raw.bin
OUTPUT=$(tr '\0' '\n' < /tmp/_test_yankee_args_raw.bin | tr '\n' ' ')
rm -f /tmp/_test_yankee_args_raw.bin

# ---- Tests ----

_test_pane_flag() {
    assert_contains "$OUTPUT" "--pane" "expected --pane flag in output"
}
run_test "pane flag present" _test_pane_flag

_test_pane_id() {
    assert_contains "$OUTPUT" "%42" "expected pane id %42 in output"
}
run_test "pane id forwarded" _test_pane_id

_test_mode() {
    assert_contains "$OUTPUT" "--mode absolute" "expected --mode absolute"
}
run_test "mode forwarded" _test_mode

_test_scrollback() {
    assert_contains "$OUTPUT" "--scrollback-lines 5000" "expected --scrollback-lines 5000"
}
run_test "scrollback-lines forwarded" _test_scrollback

_test_theme() {
    assert_contains "$OUTPUT" "--theme dracula" "expected --theme dracula"
}
run_test "theme forwarded" _test_theme

_test_status_indicator() {
    assert_contains "$OUTPUT" "--status-indicator on" "expected --status-indicator on"
}
run_test "status-indicator forwarded" _test_status_indicator

_test_cursor_bg() {
    assert_contains "$OUTPUT" "--cursor-bg #ff5555" "expected --cursor-bg #ff5555"
}
run_test "cursor-bg forwarded" _test_cursor_bg

_test_toggle_mode_key() {
    assert_contains "$OUTPUT" "--toggle-mode-key M" "expected --toggle-mode-key M"
}
run_test "toggle-mode-key forwarded" _test_toggle_mode_key

_test_cursor_fg_absent() {
    assert_not_contains "$OUTPUT" "--cursor-fg" \
        "cursor-fg should not appear when option is empty"
}
run_test "empty cursor-fg not forwarded" _test_cursor_fg_absent

_test_display_mode_absent() {
    assert_not_contains "$OUTPUT" "--display-mode" \
        "display-mode is a shell-routing option, must not be forwarded"
}
run_test "display-mode not forwarded" _test_display_mode_absent

_test_key_absent() {
    assert_not_contains "$OUTPUT" "--key " \
        "yankee_key is a shell-routing option, must not be forwarded"
}
run_test "yankee_key not forwarded" _test_key_absent

# _YANKEE_ARGS global array is populated as a side-effect of build_yankee_args.
# Verify it outside run_test (global array not visible inside run_test's subshell).
_args_len="${#_YANKEE_ARGS[@]}"
if [[ "$_args_len" -gt 0 ]] && \
   [[ "${_YANKEE_ARGS[0]}" == "--pane" ]] && \
   [[ "${_YANKEE_ARGS[1]}" == "%42" ]]; then
    # Count this as a passing test by incrementing counters directly.
    _TEST_TOTAL=$(( _TEST_TOTAL + 1 ))
    _TEST_PASS=$(( _TEST_PASS + 1 ))
    printf '  %sPASS%s _YANKEE_ARGS global array populated correctly\n' \
        "$_CLR_GREEN" "$_CLR_RESET"
else
    _TEST_TOTAL=$(( _TEST_TOTAL + 1 ))
    _TEST_FAIL=$(( _TEST_FAIL + 1 ))
    _TEST_FAILURES+=("_YANKEE_ARGS global array populated correctly")
    printf '  %sFAIL%s _YANKEE_ARGS global array populated correctly\n' \
        "$_CLR_RED" "$_CLR_RESET"
    printf "    len=%d, [0]=%s, [1]=%s\n" \
        "$_args_len" "${_YANKEE_ARGS[0]:-<empty>}" "${_YANKEE_ARGS[1]:-<empty>}"
fi

print_test_summary
get_test_exit_code
