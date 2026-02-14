#!/usr/bin/env bash
# test_math.sh - Unit tests for line number mathematics
#
# Tests 18-22: base_absolute, cursor_absolute, relative distances
#
# These tests verify the mathematical formulas used throughout the plugin.
# The formulas are implemented in line_numbers.sh and renderer.sh.
# We test them as pure arithmetic -- no tmux dependency.
#
# Formulas (from Phase 2 Implementation Plan, Section 4):
#   base_absolute   = history_size - scroll_position
#   line_absolute   = base_absolute + row_index
#   cursor_absolute = base_absolute + copy_cursor_y
#   line_relative   = |line_absolute - cursor_absolute|
#                   = |row_index - copy_cursor_y|

set -euo pipefail

TESTS_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
source "$TESTS_DIR/test_helpers.sh"

print_test_file_header "Unit Tests: Line Number Math"

# --- Source utils.sh and renderer.sh for any math helper functions ---
# The math is spread across line_numbers.sh (base/cursor computation)
# and renderer.sh (per-line computation). We test the formulas directly
# and also verify the renderer produces correct numbers.

RENDERER="$SCRIPTS_DIR/renderer.sh"
LINE_NUMBERS="$SCRIPTS_DIR/line_numbers.sh"

source_math_deps() {
    # Try to source renderer.sh for calculate_gutter_width and render_line_numbers
    if [[ ! -f "$RENDERER" ]]; then
        printf "    renderer.sh not found at %s\n" "$RENDERER"
        return 1
    fi
    source "$RENDERER"
}

# --- Pure arithmetic test helpers ---
# These replicate the formulas from the implementation plan
# and verify them against expected values.

compute_base_absolute() {
    local history_size="$1"
    local scroll_position="$2"
    printf '%d' $(( history_size - scroll_position ))
}

compute_cursor_absolute() {
    local base_absolute="$1"
    local copy_cursor_y="$2"
    printf '%d' $(( base_absolute + copy_cursor_y ))
}

compute_line_absolute() {
    local base_absolute="$1"
    local row_index="$2"
    printf '%d' $(( base_absolute + row_index ))
}

compute_line_relative() {
    local line_absolute="$1"
    local cursor_absolute="$2"
    local diff=$(( line_absolute - cursor_absolute ))
    if [[ $diff -lt 0 ]]; then
        diff=$(( -diff ))
    fi
    printf '%d' "$diff"
}

# ============================================================================
# Test 18: test_base_absolute_at_bottom
# scroll_position=0 (at bottom), history_size=500 -> base_absolute=500
# ============================================================================
_test_base_absolute_at_bottom() {
    local base
    base=$(compute_base_absolute 500 0)
    assert_equal "500" "$base" \
        "base_absolute with scroll_position=0 should equal history_size"

    # Additional check: when at the bottom of scrollback,
    # the first visible line is at line number = history_size
    # (lines 0..history_size-1 are above, visible starts at history_size)
    base=$(compute_base_absolute 1000 0)
    assert_equal "1000" "$base" \
        "base_absolute=1000 when history=1000, scroll=0"

    base=$(compute_base_absolute 0 0)
    assert_equal "0" "$base" \
        "base_absolute=0 when history=0, scroll=0"
}
run_test "test_base_absolute_at_bottom" _test_base_absolute_at_bottom

# ============================================================================
# Test 19: test_base_absolute_scrolled
# scroll_position=100, history_size=500 -> base_absolute=400
# ============================================================================
_test_base_absolute_scrolled() {
    local base
    base=$(compute_base_absolute 500 100)
    assert_equal "400" "$base" \
        "base_absolute=400 when history=500, scroll=100"

    # Scrolled to the very top
    base=$(compute_base_absolute 500 500)
    assert_equal "0" "$base" \
        "base_absolute=0 when scrolled to top (scroll=history)"

    # Partially scrolled
    base=$(compute_base_absolute 10000 3000)
    assert_equal "7000" "$base" \
        "base_absolute=7000 when history=10000, scroll=3000"
}
run_test "test_base_absolute_scrolled" _test_base_absolute_scrolled

# ============================================================================
# Test 20: test_cursor_absolute
# base_absolute=400, copy_cursor_y=5 -> cursor_absolute=405
# ============================================================================
_test_cursor_absolute() {
    local cursor
    cursor=$(compute_cursor_absolute 400 5)
    assert_equal "405" "$cursor" \
        "cursor_absolute=405 when base=400, cursor_y=5"

    # Cursor at top of viewport
    cursor=$(compute_cursor_absolute 400 0)
    assert_equal "400" "$cursor" \
        "cursor_absolute=400 when base=400, cursor_y=0"

    # Cursor at bottom of 24-line viewport
    cursor=$(compute_cursor_absolute 400 23)
    assert_equal "423" "$cursor" \
        "cursor_absolute=423 when base=400, cursor_y=23"

    # Edge case: zero base
    cursor=$(compute_cursor_absolute 0 10)
    assert_equal "10" "$cursor" \
        "cursor_absolute=10 when base=0, cursor_y=10"
}
run_test "test_cursor_absolute" _test_cursor_absolute

# ============================================================================
# Test 21: test_relative_distances
# Verify relative values for all rows in a viewport
# Using the example from Section 4.3 of the implementation plan:
#   history=500, scroll=100, cursor_y=5, pane_height=24
#   base=400, cursor=405
# ============================================================================
_test_relative_distances() {
    local base=400
    local cursor=405
    local pane_height=24

    # Expected relative values from implementation plan example:
    # Row 0: abs=400, rel=5
    # Row 1: abs=401, rel=4
    # Row 2: abs=402, rel=3
    # Row 3: abs=403, rel=2
    # Row 4: abs=404, rel=1
    # Row 5: abs=405, rel=0 (cursor line)
    # Row 6: abs=406, rel=1
    # Row 7: abs=407, rel=2
    # ...
    # Row 23: abs=423, rel=18

    local expected_relative=(5 4 3 2 1 0 1 2 3 4 5 6 7 8 9 10 11 12 13 14 15 16 17 18)

    local row_index
    for (( row_index=0; row_index<pane_height; row_index++ )); do
        local line_abs
        line_abs=$(compute_line_absolute "$base" "$row_index")
        local line_rel
        line_rel=$(compute_line_relative "$line_abs" "$cursor")

        assert_equal "${expected_relative[$row_index]}" "$line_rel" \
            "row $row_index: relative distance should be ${expected_relative[$row_index]}"
    done

    # Also verify the absolute values
    local abs_first
    abs_first=$(compute_line_absolute "$base" 0)
    assert_equal "400" "$abs_first" "first row absolute should be 400"

    local abs_cursor
    abs_cursor=$(compute_line_absolute "$base" 5)
    assert_equal "405" "$abs_cursor" "cursor row absolute should be 405"

    local abs_last
    abs_last=$(compute_line_absolute "$base" 23)
    assert_equal "423" "$abs_last" "last row absolute should be 423"
}
run_test "test_relative_distances" _test_relative_distances

# ============================================================================
# Test 22: test_zero_history
# history=0, scroll=0 -> base=0
# All lines start from 0, cursor at row 0 is absolute 0
# ============================================================================
_test_zero_history() {
    local base
    base=$(compute_base_absolute 0 0)
    assert_equal "0" "$base" \
        "base_absolute=0 when history=0"

    local cursor
    cursor=$(compute_cursor_absolute 0 0)
    assert_equal "0" "$cursor" \
        "cursor_absolute=0 when base=0, cursor_y=0"

    # 5 rows, all starting from 0
    local row_index
    for (( row_index=0; row_index<5; row_index++ )); do
        local line_abs
        line_abs=$(compute_line_absolute 0 "$row_index")
        assert_equal "$row_index" "$line_abs" \
            "row $row_index absolute should be $row_index"
    done

    # Relative distances from cursor at row 0
    local line_rel
    for (( row_index=0; row_index<5; row_index++ )); do
        local line_abs
        line_abs=$(compute_line_absolute 0 "$row_index")
        line_rel=$(compute_line_relative "$line_abs" 0)
        assert_equal "$row_index" "$line_rel" \
            "row $row_index relative distance should be $row_index"
    done

    # Now also verify via the renderer (if available)
    # This ensures the renderer correctly handles zero history
    if [[ -f "$RENDERER" ]]; then
        source "$RENDERER"

        local content
        content=$(printf 'line0\nline1\nline2\nline3\nline4\n')

        local output
        output=$(render_line_numbers \
            "$content" \
            0 \
            0 \
            80 \
            5 \
            "absolute" \
            "fg=white" \
            "fg=yellow" \
            "fg=green,bold"
        )

        # First line should show number 0
        local first_num
        first_num=$(printf '%s\n' "$output" | head -1 | sed 's/\x1b\[[0-9;]*m//g' | sed -n 's/^[[:space:]]*\([0-9]*\)[[:space:]]*|.*/\1/p')
        assert_equal "0" "$first_num" \
            "renderer should show 0 as first line number when history=0"
    fi
}
run_test "test_zero_history" _test_zero_history

# ============================================================================
# Bonus: test_math_with_renderer_integration
# End-to-end math check: feed known values into renderer, verify output numbers
# ============================================================================
_test_math_with_renderer_integration() {
    source_math_deps || return 1

    # Use the Section 4.3 example values
    local history_size=500
    local scroll_position=100
    local copy_cursor_y=5
    local pane_height=10  # Use 10 rows for simpler test

    local base_absolute
    base_absolute=$(compute_base_absolute "$history_size" "$scroll_position")
    assert_equal "400" "$base_absolute" "base should be 400"

    local cursor_absolute
    cursor_absolute=$(compute_cursor_absolute "$base_absolute" "$copy_cursor_y")
    assert_equal "405" "$cursor_absolute" "cursor should be 405"

    # Generate 10 lines of content
    local content=""
    local i
    for (( i=0; i<pane_height; i++ )); do
        content+="content line $i"
        if [[ $i -lt $(( pane_height - 1 )) ]]; then
            content+=$'\n'
        fi
    done

    local gutter_width
    gutter_width=$(calculate_gutter_width "$history_size")

    # Render in hybrid mode
    local output
    output=$(render_line_numbers \
        "$content" \
        "$base_absolute" \
        "$cursor_absolute" \
        80 \
        "$gutter_width" \
        "hybrid" \
        "fg=white" \
        "fg=yellow" \
        "fg=green,bold"
    )

    # Extract display numbers (strip ANSI)
    local numbers
    numbers=$(printf '%s\n' "$output" | sed 's/\x1b\[[0-9;]*m//g' | sed -n 's/^[[:space:]]*\([0-9]*\)[[:space:]]*|.*/\1/p')

    # Expected in hybrid mode:
    # Row 0: rel=5 -> show 5
    # Row 1: rel=4 -> show 4
    # Row 2: rel=3 -> show 3
    # Row 3: rel=2 -> show 2
    # Row 4: rel=1 -> show 1
    # Row 5: cursor -> show 405 (absolute)
    # Row 6: rel=1 -> show 1
    # Row 7: rel=2 -> show 2
    # Row 8: rel=3 -> show 3
    # Row 9: rel=4 -> show 4
    local expected_numbers="5
4
3
2
1
405
1
2
3
4"

    assert_equal "$expected_numbers" "$numbers" \
        "hybrid mode numbers should match expected relative+absolute pattern"
}
run_test "test_math_with_renderer_integration" _test_math_with_renderer_integration

# --- Print summary and exit ---
print_test_summary
get_test_exit_code
