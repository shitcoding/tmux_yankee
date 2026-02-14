#!/usr/bin/env bash
# test_config.sh - Unit tests for scripts/config.sh
#
# Tests 9-12: get_mode(), cycle_mode(), default values
#
# Strategy: config.sh functions call `tmux show-option`, so we test
# by providing a mock tmux function or by using an isolated tmux server.
# For pure unit tests, we mock tmux. For integration, we use real tmux.

set -euo pipefail

TESTS_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
source "$TESTS_DIR/test_helpers.sh"

print_test_file_header "Unit Tests: config.sh"

# --- Source the module under test ---
CONFIG="$SCRIPTS_DIR/config.sh"

source_config() {
    if [[ ! -f "$CONFIG" ]]; then
        printf "    config.sh not found at %s\n" "$CONFIG"
        return 1
    fi
    source "$CONFIG"
}

# --- Mock tmux for unit testing ---
# These tests need an isolated tmux server to set/read options.
# We use the test socket from test_helpers.sh.

_MOCK_TMUX_OPTIONS=()

setup_config_tmux() {
    # Create a minimal tmux session for testing config reads
    require_tmux
    tmux -L "$TMUX_TEST_SOCKET" kill-server 2>/dev/null || true
    tmux -L "$TMUX_TEST_SOCKET" new-session -d -s config-test -x 80 -y 24
    sleep 0.1
}

teardown_config_tmux() {
    tmux -L "$TMUX_TEST_SOCKET" kill-server 2>/dev/null || true
}

set_tmux_option() {
    # Set a tmux option in the test server
    local option="$1"
    local value="$2"
    tmux -L "$TMUX_TEST_SOCKET" set-option -g "$option" "$value"
}

unset_tmux_option() {
    # Remove a tmux option from the test server
    local option="$1"
    tmux -L "$TMUX_TEST_SOCKET" set-option -gu "$option" 2>/dev/null || true
}

# Since config.sh calls `tmux` directly, we need to override it.
# We create a wrapper that redirects to our test socket.
setup_tmux_wrapper() {
    # Override tmux command to use our test socket
    tmux() {
        command tmux -L "$TMUX_TEST_SOCKET" "$@"
    }
    export -f tmux
}

restore_tmux() {
    unset -f tmux 2>/dev/null || true
}

# ============================================================================
# Test 9: test_default_mode
# With no @linenumbers-mode option set, get_mode() returns "hybrid"
# ============================================================================
_test_default_mode() {
    source_config || return 1

    setup_config_tmux || return 1
    setup_tmux_wrapper

    # Ensure the option is not set
    unset_tmux_option "@linenumbers-mode"

    local mode
    mode=$(get_mode)

    restore_tmux
    teardown_config_tmux

    assert_equal "hybrid" "$mode" \
        "default mode should be 'hybrid' when no option is set"
}
run_test "test_default_mode" _test_default_mode

# ============================================================================
# Test 10: test_invalid_mode_fallback
# With an invalid @linenumbers-mode option, get_mode() returns "hybrid"
# ============================================================================
_test_invalid_mode_fallback() {
    source_config || return 1

    setup_config_tmux || return 1
    setup_tmux_wrapper

    # Set an invalid mode
    set_tmux_option "@linenumbers-mode" "invalid_garbage"

    local mode
    mode=$(get_mode)

    restore_tmux
    teardown_config_tmux

    assert_equal "hybrid" "$mode" \
        "invalid mode value should fall back to 'hybrid'"
}
run_test "test_invalid_mode_fallback" _test_invalid_mode_fallback

# ============================================================================
# Test 11: test_cycle_mode_order
# Verify cycle order: hybrid -> absolute -> relative -> hybrid
# ============================================================================
_test_cycle_mode_order() {
    source_config || return 1

    setup_config_tmux || return 1
    setup_tmux_wrapper

    # Start with hybrid
    set_tmux_option "@linenumbers-mode" "hybrid"

    # Cycle: hybrid -> absolute
    local new_mode
    new_mode=$(cycle_mode)
    assert_equal "absolute" "$new_mode" \
        "cycling from hybrid should produce absolute"

    # Cycle: absolute -> relative
    new_mode=$(cycle_mode)
    assert_equal "relative" "$new_mode" \
        "cycling from absolute should produce relative"

    # Cycle: relative -> hybrid
    new_mode=$(cycle_mode)
    assert_equal "hybrid" "$new_mode" \
        "cycling from relative should produce hybrid"

    # Full cycle back to absolute
    new_mode=$(cycle_mode)
    assert_equal "absolute" "$new_mode" \
        "cycling from hybrid again should produce absolute"

    restore_tmux
    teardown_config_tmux
}
run_test "test_cycle_mode_order" _test_cycle_mode_order

# ============================================================================
# Test 12: test_all_defaults
# Verify all default values match the specification
# Default values:
#   @linenumbers-mode            -> "hybrid"
#   @linenumbers-style-absolute  -> "fg=white"
#   @linenumbers-style-relative  -> "fg=yellow"
#   @linenumbers-style-cursor    -> "fg=green,bold"
#   @linenumbers-toggle-key      -> "L"
#   @linenumbers-enable-binding  -> "off"
#   @linenumbers-custom-key      -> "N"
# ============================================================================
_test_all_defaults() {
    source_config || return 1

    setup_config_tmux || return 1
    setup_tmux_wrapper

    # Clear all linenumbers options
    for opt in "@linenumbers-mode" "@linenumbers-style-absolute" \
               "@linenumbers-style-relative" "@linenumbers-style-cursor" \
               "@linenumbers-toggle-key" "@linenumbers-enable-binding" \
               "@linenumbers-custom-key"; do
        unset_tmux_option "$opt"
    done

    # Test each default
    local val

    val=$(get_mode)
    assert_equal "hybrid" "$val" "default mode should be hybrid"

    val=$(get_style_absolute)
    assert_equal "fg=white" "$val" "default absolute style should be fg=white"

    val=$(get_style_relative)
    assert_equal "fg=yellow" "$val" "default relative style should be fg=yellow"

    val=$(get_style_cursor)
    assert_equal "fg=green,bold" "$val" "default cursor style should be fg=green,bold"

    val=$(get_toggle_key)
    assert_equal "L" "$val" "default toggle key should be L"

    val=$(get_enable_binding)
    assert_equal "off" "$val" "default enable-binding should be off"

    val=$(get_custom_key)
    assert_equal "N" "$val" "default custom key should be N"

    restore_tmux
    teardown_config_tmux
}
run_test "test_all_defaults" _test_all_defaults

# --- Print summary and exit ---
print_test_summary
get_test_exit_code
