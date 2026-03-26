#!/usr/bin/env bash

CURRENT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCRIPTS_DIR="${CURRENT_DIR}/scripts"

# Read a tmux option with a default fallback.
get_tmux_option() {
    local option="$1"
    local default_value="$2"
    local value
    value=$(tmux show-option -gqv "$option" 2>/dev/null) || true
    if [[ -n "$value" ]]; then
        echo "$value"
    else
        echo "$default_value"
    fi
}

set_yankee_binding() {
    local yankee_key yankee_table
    yankee_key=$(get_tmux_option "@yankee_key" "N")
    yankee_table=$(get_tmux_option "@yankee_key_table" "prefix")
    if [[ -z "$yankee_key" ]]; then
        return
    fi
    # Atomic busy gate: set-option -o fails if @yankee_busy already exists,
    # preventing concurrent launches on the same pane.
    local gate_cond="tmux set-option -opq -t #{pane_id} @yankee_busy 1"
    local gate_then="run-shell -b \"${SCRIPTS_DIR}/launch_yankee.sh #{pane_id} #{window_id} #{pane_index}\""
    if [[ "$yankee_table" == "root" ]]; then
        tmux bind-key -n "$yankee_key" if-shell "$gate_cond" "$gate_then"
    else
        tmux bind-key "$yankee_key" if-shell "$gate_cond" "$gate_then"
    fi
}

set_scroll_bindings() {
    # Override WheelUpPane to launch yankee instead of copy-mode.
    # Pass through if pane is already in a mode (copy-mode, etc.) or if the
    # pane has mouse_any_flag set (e.g., vim, less, or our own yankee TUI).
    # Also skip if the pane is showing an alternate screen (full-screen apps).
    #
    # Two-level guard:
    # 1) Format guard: pane_in_mode || mouse_any_flag || alternate_on → send-keys -M
    # 2) Atomic busy gate: set-option -o fails if @yankee_busy exists → send-keys -M
    #    This prevents concurrent launches and is non-reentrant: the flag is set
    #    synchronously in the tmux command queue BEFORE run-shell -b forks.
    #
    # The script receives #{pane_id} #{window_id} #{pane_index} so it doesn't need
    # untargeted display-message -p (which can return wrong pane in run-shell -b).
    #
    # Note: tmux's #{||:A,B,C} with 3 operands is broken (always truthy); use
    # nested #{||:A,#{||:B,C}} instead.
    local gate_cond="tmux set-option -opq -t #{pane_id} @yankee_busy 1"
    local gate_then="run-shell -b \"${SCRIPTS_DIR}/launch_yankee.sh #{pane_id} #{window_id} #{pane_index}\""
    local inner_gate
    inner_gate="if-shell \"${gate_cond}\" \"${gate_then}\" \"send-keys -M\""

    tmux bind-key -n WheelUpPane \
        if-shell -F '#{||:#{pane_in_mode},#{||:#{mouse_any_flag},#{alternate_on}}}' \
            'send-keys -M' \
            "$inner_gate"

    # WheelDownPane: pass through when in mode/mouse-aware; no-op otherwise.
    tmux bind-key -n WheelDownPane \
        if-shell -F '#{||:#{pane_in_mode},#{||:#{mouse_any_flag},#{alternate_on}}}' \
            'send-keys -M' \
            ''
}

# Flash navigation defaults
tmux set-option -gq @yankee_flash "on"
tmux set-option -gq @yankee_flash_min_chars "1"
tmux set-option -gq @yankee_flash_ft "off"
tmux set-option -gq @yankee_flash_jump_pos "match_end"
tmux set-option -gq @yankee_flash_alt_jump_pos "match_start"

ensure_binary() {
    if [[ ! -x "${CURRENT_DIR}/bin/tmux-yankee" ]]; then
        "${SCRIPTS_DIR}/install.sh"
    fi
}

main() {
    ensure_binary
    set_yankee_binding
    set_scroll_bindings
}
main
