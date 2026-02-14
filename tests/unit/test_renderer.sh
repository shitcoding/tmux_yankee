#!/usr/bin/env bash
# test_renderer.sh - Unit tests for scripts/renderer.sh
#
# Tests 1-8: render_line_numbers(), calculate_gutter_width(), tmux_style_to_ansi()
# These are pure function tests -- no tmux dependency.

set -euo pipefail

TESTS_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
source "$TESTS_DIR/test_helpers.sh"

print_test_file_header "Unit Tests: renderer.sh"

# --- Source the module under test ---
# renderer.sh must exist and be sourceable without side effects
RENDERER="$SCRIPTS_DIR/renderer.sh"

source_renderer() {
    if [[ ! -f "$RENDERER" ]]; then
        printf "    renderer.sh not found at %s\n" "$RENDERER"
        return 1
    fi
    source "$RENDERER"
}

# --- Helper: generate mock content ---
make_content() {
    # Generate N lines of content: "line 0", "line 1", ...
    local count="$1"
    local i
    for (( i=0; i<count; i++ )); do
        printf 'line %d\n' "$i"
    done
}

# --- Helper: extract display numbers from rendered output ---
extract_display_numbers() {
    # From rendered output, extract the number before the "|" separator
    # Expected format: "  NNN | content"
    # Strips ANSI escapes first, then extracts the number
    local output="$1"
    printf '%s\n' "$output" | sed 's/\x1b\[[0-9;]*m//g' | sed -n 's/^[[:space:]]*\([0-9]*\)[[:space:]]*|.*/\1/p'
}

# ============================================================================
# Test 1: test_absolute_mode_basic
# Feed 5 lines with base_absolute=100, verify output has "100" through "104"
# ============================================================================
_test_absolute_mode_basic() {
    source_renderer || return 1

    local content
    content=$(make_content 5)

    local output
    output=$(render_line_numbers \
        "$content" \
        100 \
        102 \
        80 \
        6 \
        "absolute" \
        "fg=white" \
        "fg=yellow" \
        "fg=green,bold"
    )

    local numbers
    numbers=$(extract_display_numbers "$output")

    # Should show absolute line numbers 100, 101, 102, 103, 104
    local expected_numbers="100
101
102
103
104"
    assert_equal "$expected_numbers" "$numbers" \
        "absolute mode should show numbers 100-104"
}
run_test "test_absolute_mode_basic" _test_absolute_mode_basic

# ============================================================================
# Test 2: test_relative_mode_basic
# Feed 5 lines with cursor at row 2 (absolute=102),
# verify output shows distances 2, 1, (absolute at cursor), 1, 2
# ============================================================================
_test_relative_mode_basic() {
    source_renderer || return 1

    local content
    content=$(make_content 5)

    # base_absolute=100, cursor_absolute=102 (cursor at row 2)
    local output
    output=$(render_line_numbers \
        "$content" \
        100 \
        102 \
        80 \
        6 \
        "relative" \
        "fg=white" \
        "fg=yellow" \
        "fg=green,bold"
    )

    local numbers
    numbers=$(extract_display_numbers "$output")

    # In relative mode:
    # Row 0: abs=100, rel=|100-102|=2 -> show 2
    # Row 1: abs=101, rel=|101-102|=1 -> show 1
    # Row 2: abs=102, rel=0 -> show absolute 102 (cursor line)
    # Row 3: abs=103, rel=|103-102|=1 -> show 1
    # Row 4: abs=104, rel=|104-102|=2 -> show 2
    local expected_numbers="2
1
102
1
2"
    assert_equal "$expected_numbers" "$numbers" \
        "relative mode should show distances and absolute at cursor"
}
run_test "test_relative_mode_basic" _test_relative_mode_basic

# ============================================================================
# Test 3: test_hybrid_mode_basic
# Feed 5 lines with cursor at row 2 (absolute=102),
# cursor line shows absolute, others show relative
# ============================================================================
_test_hybrid_mode_basic() {
    source_renderer || return 1

    local content
    content=$(make_content 5)

    local output
    output=$(render_line_numbers \
        "$content" \
        100 \
        102 \
        80 \
        6 \
        "hybrid" \
        "fg=white" \
        "fg=yellow" \
        "fg=green,bold"
    )

    local numbers
    numbers=$(extract_display_numbers "$output")

    # Hybrid mode behaves same as relative for display numbers:
    # cursor gets absolute, others get relative distance
    local expected_numbers="2
1
102
1
2"
    assert_equal "$expected_numbers" "$numbers" \
        "hybrid mode should show absolute at cursor, relative elsewhere"
}
run_test "test_hybrid_mode_basic" _test_hybrid_mode_basic

# ============================================================================
# Test 4: test_cursor_line_style
# Verify cursor line uses cursor style, other lines use appropriate style
# ============================================================================
_test_cursor_line_style() {
    source_renderer || return 1

    local content
    content=$(make_content 3)

    local output
    output=$(render_line_numbers \
        "$content" \
        100 \
        101 \
        80 \
        6 \
        "absolute" \
        "fg=white" \
        "fg=yellow" \
        "fg=green,bold"
    )

    # Cursor line is row 1 (abs=101, cursor_absolute=101)
    # Cursor style "fg=green,bold" -> ANSI: \033[1;32m
    # Non-cursor absolute style "fg=white" -> ANSI: \033[37m

    local cursor_line
    cursor_line=$(printf '%s\n' "$output" | sed -n '2p')

    local non_cursor_line
    non_cursor_line=$(printf '%s\n' "$output" | sed -n '1p')

    # Cursor line should contain the green+bold ANSI code
    local green_bold_ansi=$'\033[1;32m'
    assert_contains "$cursor_line" "$green_bold_ansi" \
        "cursor line should use green bold ANSI style"

    # Non-cursor line should contain the white ANSI code
    local white_ansi=$'\033[37m'
    assert_contains "$non_cursor_line" "$white_ansi" \
        "non-cursor line should use white ANSI style"
}
run_test "test_cursor_line_style" _test_cursor_line_style

# ============================================================================
# Test 5: test_gutter_width_calculation
# Verify gutter width for various history_size values
# Gutter = digits + 3 (" | "), minimum 2 digits
# ============================================================================
_test_gutter_width_calculation() {
    source_renderer || return 1

    # history_size=9 -> 1 digit, but min 2 -> gutter=5
    local gw
    gw=$(calculate_gutter_width 9)
    assert_equal "5" "$gw" "gutter for 9 should be 5 (2 digits min + 3)"

    # history_size=99 -> 2 digits -> gutter=5
    gw=$(calculate_gutter_width 99)
    assert_equal "5" "$gw" "gutter for 99 should be 5 (2+3)"

    # history_size=999 -> 3 digits -> gutter=6
    gw=$(calculate_gutter_width 999)
    assert_equal "6" "$gw" "gutter for 999 should be 6 (3+3)"

    # history_size=9999 -> 4 digits -> gutter=7
    gw=$(calculate_gutter_width 9999)
    assert_equal "7" "$gw" "gutter for 9999 should be 7 (4+3)"

    # history_size=99999 -> 5 digits -> gutter=8
    gw=$(calculate_gutter_width 99999)
    assert_equal "8" "$gw" "gutter for 99999 should be 8 (5+3)"
}
run_test "test_gutter_width_calculation" _test_gutter_width_calculation

# ============================================================================
# Test 6: test_content_truncation
# Verify long lines are truncated to fit within pane_width minus gutter
# ============================================================================
_test_content_truncation() {
    source_renderer || return 1

    # Create one line that is 100 chars long
    local long_line
    long_line=$(printf '%0.s' $(seq 1 100) | tr '0' 'A')
    # Actually: 100 'A' characters
    long_line=$(printf 'A%.0s' $(seq 1 100))

    local output
    output=$(render_line_numbers \
        "$long_line" \
        1 \
        1 \
        40 \
        6 \
        "absolute" \
        "fg=white" \
        "fg=yellow" \
        "fg=green,bold"
    )

    # pane_width=40, gutter_width=6, so content_width=34
    # Strip ANSI escapes and get the content after "|"
    local clean_output
    clean_output=$(printf '%s' "$output" | sed 's/\x1b\[[0-9;]*m//g')

    # The content portion (after "| ") should be at most 34 chars
    local content_part
    content_part=$(printf '%s' "$clean_output" | sed 's/^[^|]*| //')

    local content_len=${#content_part}
    if [[ $content_len -gt 34 ]]; then
        printf "    ASSERTION FAILED: content truncation\n"
        printf "    content length %d exceeds max %d\n" "$content_len" 34
        return 1
    fi
}
run_test "test_content_truncation" _test_content_truncation

# ============================================================================
# Test 7: test_empty_lines
# Verify empty content lines get line numbers too
# ============================================================================
_test_empty_lines() {
    source_renderer || return 1

    # 3 lines: non-empty, empty, non-empty
    local content=$'hello\n\nworld'

    local output
    output=$(render_line_numbers \
        "$content" \
        10 \
        11 \
        80 \
        6 \
        "absolute" \
        "fg=white" \
        "fg=yellow" \
        "fg=green,bold"
    )

    local numbers
    numbers=$(extract_display_numbers "$output")

    # All three lines should have numbers
    local expected_numbers="10
11
12"
    assert_equal "$expected_numbers" "$numbers" \
        "empty lines should still get line numbers"

    # Verify we get 3 output lines
    local line_count
    line_count=$(printf '%s\n' "$output" | wc -l | tr -d ' ')
    assert_equal "3" "$line_count" \
        "should produce exactly 3 output lines for 3 input lines"
}
run_test "test_empty_lines" _test_empty_lines

# ============================================================================
# Test 8: test_ansi_conversion
# Verify tmux_style_to_ansi for "fg=green,bold", "fg=yellow", "fg=white"
# ============================================================================
_test_ansi_conversion() {
    source_renderer || return 1

    local result

    # "fg=green,bold" -> \033[1;32m
    result=$(tmux_style_to_ansi "fg=green,bold")
    assert_equal $'\033[1;32m' "$result" \
        "fg=green,bold should produce ANSI bold+green"

    # "fg=yellow" -> \033[33m
    result=$(tmux_style_to_ansi "fg=yellow")
    assert_equal $'\033[33m' "$result" \
        "fg=yellow should produce ANSI yellow"

    # "fg=white" -> \033[37m
    result=$(tmux_style_to_ansi "fg=white")
    assert_equal $'\033[37m' "$result" \
        "fg=white should produce ANSI white"

    # "fg=red" -> \033[31m
    result=$(tmux_style_to_ansi "fg=red")
    assert_equal $'\033[31m' "$result" \
        "fg=red should produce ANSI red"

    # "bold" (no fg) -> \033[1m
    result=$(tmux_style_to_ansi "bold")
    assert_equal $'\033[1m' "$result" \
        "bold alone should produce ANSI bold"

    # Empty style -> empty string
    result=$(tmux_style_to_ansi "")
    assert_equal "" "$result" \
        "empty style should produce empty ANSI"
}
run_test "test_ansi_conversion" _test_ansi_conversion

# --- Print summary and exit ---
print_test_summary
get_test_exit_code
