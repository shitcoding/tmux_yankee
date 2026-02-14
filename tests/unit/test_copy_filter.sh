#!/usr/bin/env bash
# test_copy_filter.sh - Unit tests for scripts/copy_filter.sh
#
# Tests 13-17: filter_line_numbers() gutter stripping
#
# Strategy: copy_filter.sh's filter_line_numbers() is a pure stdin->stdout
# function. We test it directly by sourcing the function (or piping through
# the script) with known input and verifying output.
#
# Note: The full copy_filter.sh also calls tmux set-buffer and clipboard
# commands. For unit tests we only test the filter_line_numbers() function
# by sourcing it, not by running the full script.

set -euo pipefail

TESTS_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
source "$TESTS_DIR/test_helpers.sh"

print_test_file_header "Unit Tests: copy_filter.sh"

# --- Source the module under test ---
COPY_FILTER="$SCRIPTS_DIR/copy_filter.sh"

source_copy_filter() {
    if [[ ! -f "$COPY_FILTER" ]]; then
        printf "    copy_filter.sh not found at %s\n" "$COPY_FILTER"
        return 1
    fi
    # We need to source only the function, not execute the main body.
    # The script has a main body that reads stdin and calls tmux.
    # We extract filter_line_numbers() by sourcing with a guard.
    # Strategy: source the file but override tmux and GUTTER_WIDTH to prevent side effects.
    # Actually, let's just define a helper that calls the function in a subshell.

    # First, check if we can extract the function definition
    if ! grep -q 'filter_line_numbers()' "$COPY_FILTER"; then
        printf "    filter_line_numbers() not found in copy_filter.sh\n"
        return 1
    fi

    # Source in a way that captures the function but doesn't run the script body.
    # We use a trick: override commands that have side effects.
    (
        # Prevent the main body from executing by providing mocks
        tmux() { :; }
        pbcopy() { :; }
        xclip() { :; }
        xsel() { :; }
        wl-copy() { :; }
        # Source the script -- it will try to run its main body but our
        # mocked commands will prevent side effects
        export GUTTER_WIDTH=6
        source "$COPY_FILTER"
    ) 2>/dev/null || true

    # For actual test execution, we'll use a different approach:
    # Extract just the function definition and test it.
    return 0
}

# Helper: run filter_line_numbers function with given gutter width and input
run_filter() {
    local gutter_width="$1"
    local input="$2"

    # We define filter_line_numbers inline (matching the spec from implementation plan)
    # since we can't easily source just the function from copy_filter.sh.
    # BUT -- the real test is that the ACTUAL file contains a working implementation.
    # So we try to source and test the real function.

    # Try to extract and run the actual function from copy_filter.sh
    if [[ ! -f "$COPY_FILTER" ]]; then
        return 1
    fi

    # Create a temp script that sources the function and runs it
    local tmpscript
    tmpscript=$(create_temp_file)

    cat > "$tmpscript" << 'SCRIPT_EOF'
#!/usr/bin/env bash
set -euo pipefail
GUTTER_WIDTH="${1:-6}"
# Mock out tmux and clipboard commands
tmux() { :; }
export -f tmux
# Source the copy_filter.sh to get filter_line_numbers
SCRIPT_EOF

    printf 'source "%s"\n' "$COPY_FILTER" >> "$tmpscript"
    printf 'filter_line_numbers "$GUTTER_WIDTH"\n' >> "$tmpscript"

    chmod +x "$tmpscript"
    printf '%s' "$input" | bash "$tmpscript" "$gutter_width"
}

# ============================================================================
# Test 13: test_strip_6_char_gutter
# Input with 6-char prefix (e.g., " 42 | content"), verify stripping
# Gutter format: "NN | " where NN is right-aligned, total=6
# ============================================================================
_test_strip_6_char_gutter() {
    source_copy_filter || return 1

    local input=" 42 | hello world"
    local expected="hello world"

    local actual
    actual=$(run_filter 6 "$input")

    assert_equal "$expected" "$actual" \
        "6-char gutter should be stripped, leaving 'hello world'"
}
run_test "test_strip_6_char_gutter" _test_strip_6_char_gutter

# ============================================================================
# Test 14: test_strip_8_char_gutter
# Input with 8-char prefix (large line numbers, e.g., "50000 | content")
# ============================================================================
_test_strip_8_char_gutter() {
    source_copy_filter || return 1

    local input="  50000 | hello world"
    local expected="hello world"

    # Actually with gutter_width=8, we strip 8 chars from start
    # "  50000 | hello world"
    # 12345678
    # "  50000 " is 8 chars, then "| hello world" remains?
    # Wait, let's recalculate:
    # Gutter format from renderer.sh:
    #   printf '%s%*d %s%s %s\n' style num_field_width display_num separator reset content
    #   num_field_width = gutter_width - 3
    #   So gutter_width=8: num_field_width=5
    #   Output: [ansi]NNNNN |[reset] content
    #   Without ANSI: "NNNNN | content"
    #   "NNNNN | " = 5 + 3 = 8 chars of gutter
    #
    # With filter, we strip first 8 chars
    local input2="50000 | hello world"
    # That's "50000 | " = 8 chars, "hello world" after
    local expected2="hello world"

    local actual
    actual=$(run_filter 8 "$input2")

    assert_equal "$expected2" "$actual" \
        "8-char gutter should be stripped for large line numbers"
}
run_test "test_strip_8_char_gutter" _test_strip_8_char_gutter

# ============================================================================
# Test 15: test_strip_preserves_content
# Verify content after gutter is exactly preserved (including special chars)
# ============================================================================
_test_strip_preserves_content() {
    source_copy_filter || return 1

    # Content with special characters
    local content_part='  if [[ "$foo" == "bar" ]]; then'
    local input
    input=$(printf '%6s%s' "" "$content_part")
    # Actually, input should mimic the renderer format
    # With gutter_width=6: "NN | content"
    input="  1 | ${content_part}"
    # "  1 | " is 6 chars

    local actual
    actual=$(run_filter 6 "$input")

    assert_equal "$content_part" "$actual" \
        "content with special chars should be preserved exactly after stripping"
}
run_test "test_strip_preserves_content" _test_strip_preserves_content

# ============================================================================
# Test 16: test_strip_empty_lines
# Lines shorter than gutter produce empty output
# ============================================================================
_test_strip_empty_lines() {
    source_copy_filter || return 1

    # A line that's only the gutter with no content after
    local input="  5 | "
    # That's exactly 6 chars, so after stripping we get empty string
    # Actually "  5 | " is 6 chars. After stripping 6 chars, empty.

    local actual
    actual=$(run_filter 6 "$input")

    # Should be empty (or just a newline)
    assert_equal "" "$actual" \
        "line with only gutter should produce empty output after stripping"

    # Test a line shorter than gutter width
    local short_input="  5"
    actual=$(run_filter 6 "$short_input")

    # Should produce empty output (line shorter than gutter)
    assert_equal "" "$actual" \
        "line shorter than gutter width should produce empty output"
}
run_test "test_strip_empty_lines" _test_strip_empty_lines

# ============================================================================
# Test 17: test_multiline_strip
# Multiple lines all stripped correctly
# ============================================================================
_test_multiline_strip() {
    source_copy_filter || return 1

    # Simulate 4 lines of rendered output with 6-char gutter
    local input=" 10 | first line
 11 | second line
 12 | third line
 13 | fourth line"

    local expected="first line
second line
third line
fourth line"

    local actual
    actual=$(run_filter 6 "$input")

    assert_equal "$expected" "$actual" \
        "all lines should have their gutter stripped correctly"
}
run_test "test_multiline_strip" _test_multiline_strip

# --- Print summary and exit ---
print_test_summary
get_test_exit_code
