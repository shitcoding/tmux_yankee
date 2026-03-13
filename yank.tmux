#!/usr/bin/env bash

CURRENT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCRIPTS_DIR="${CURRENT_DIR}/scripts"
HELPERS_DIR="${CURRENT_DIR}/scripts"

# shellcheck source=scripts/helpers.sh
source "${HELPERS_DIR}/helpers.sh"

clipboard_copy_without_newline_command() {
    local copy_command="$1"
    printf "tr -d '\\n' | %s" "$copy_command"
}

set_error_bindings() {
    local key_bindings key
    key_bindings="$(yank_key) $(put_key) $(yank_put_key)"
    for key in $key_bindings; do
        if tmux_is_at_least 2.4; then
            tmux bind-key -T copy-mode-vi "$key" send-keys -X copy-pipe-and-cancel "tmux display-message 'Error! tmux-yank dependencies not installed!'"
            tmux bind-key -T copy-mode "$key" send-keys -X copy-pipe-and-cancel "tmux display-message 'Error! tmux-yank dependencies not installed!'"
        else
            tmux bind-key -t vi-copy "$key" copy-pipe "tmux display-message 'Error! tmux-yank dependencies not installed!'"
            tmux bind-key -t emacs-copy "$key" copy-pipe "tmux display-message 'Error! tmux-yank dependencies not installed!'"
        fi
    done
}

error_handling_if_command_not_present() {
    local copy_command="$1"
    if [ -z "$copy_command" ]; then
        set_error_bindings
        exit 0
    fi
}

# `yank_without_newline` binding isn't intended to be used by the user. It is
# a helper for `copy_line` command.
set_copy_mode_bindings() {
    local copy_command="$1"
    local copy_wo_newline_command
    copy_wo_newline_command="$(clipboard_copy_without_newline_command "$copy_command")"
    local copy_command_mouse
    copy_command_mouse="$(clipboard_copy_command "true")"
    if tmux_is_at_least 2.4; then
        tmux bind-key -T copy-mode-vi "$(yank_key)" send-keys -X "$(yank_action)" "$copy_command"
        tmux bind-key -T copy-mode-vi "$(put_key)" send-keys -X copy-pipe-and-cancel "tmux paste-buffer -p"
        tmux bind-key -T copy-mode-vi "$(yank_put_key)" send-keys -X copy-pipe-and-cancel "$copy_command; tmux paste-buffer -p"
        tmux bind-key -T copy-mode-vi "$(yank_wo_newline_key)" send-keys -X "$(yank_action)" "$copy_wo_newline_command"
        if [[ "$(yank_with_mouse)" == "on" ]]; then
            tmux bind-key -T copy-mode-vi MouseDragEnd1Pane send-keys -X "$(yank_action)" "$copy_command_mouse"
        fi

        tmux bind-key -T copy-mode "$(yank_key)" send-keys -X "$(yank_action)" "$copy_command"
        tmux bind-key -T copy-mode "$(put_key)" send-keys -X copy-pipe-and-cancel "tmux paste-buffer -p"
        tmux bind-key -T copy-mode "$(yank_put_key)" send-keys -X copy-pipe-and-cancel "$copy_command; tmux paste-buffer -p"
        tmux bind-key -T copy-mode "$(yank_wo_newline_key)" send-keys -X "$(yank_action)" "$copy_wo_newline_command"
        if [[ "$(yank_with_mouse)" == "on" ]]; then
            tmux bind-key -T copy-mode MouseDragEnd1Pane send-keys -X "$(yank_action)" "$copy_command_mouse"
        fi
    else
        tmux bind-key -t vi-copy "$(yank_key)" copy-pipe "$copy_command"
        tmux bind-key -t vi-copy "$(put_key)" copy-pipe "tmux paste-buffer -p"
        tmux bind-key -t vi-copy "$(yank_put_key)" copy-pipe "$copy_command; tmux paste-buffer -p"
        tmux bind-key -t vi-copy "$(yank_wo_newline_key)" copy-pipe "$copy_wo_newline_command"
        if [[ "$(yank_with_mouse)" == "on" ]]; then
            tmux bind-key -t vi-copy MouseDragEnd1Pane copy-pipe "$copy_command_mouse"
        fi

        tmux bind-key -t emacs-copy "$(yank_key)" copy-pipe "$copy_command"
        tmux bind-key -t emacs-copy "$(put_key)" copy-pipe "tmux paste-buffer -p"
        tmux bind-key -t emacs-copy "$(yank_put_key)" copy-pipe "$copy_command; tmux paste-buffer -p"
        tmux bind-key -t emacs-copy "$(yank_wo_newline_key)" copy-pipe "$copy_wo_newline_command"
        if [[ "$(yank_with_mouse)" == "on" ]]; then
            tmux bind-key -t emacs-copy MouseDragEnd1Pane copy-pipe "$copy_command_mouse"
        fi
    fi
}

set_normal_bindings() {
    tmux bind-key "$(yank_line_key)" run-shell -b "$SCRIPTS_DIR/copy_line.sh"
    tmux bind-key "$(yank_pane_pwd_key)" run-shell -b "$SCRIPTS_DIR/copy_pane_pwd.sh"
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
    local gate_then="run-shell -b '${SCRIPTS_DIR}/launch_yankee.sh #{pane_id} #{window_id} #{pane_index}'"
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
    local gate_then="run-shell -b '${SCRIPTS_DIR}/launch_yankee.sh #{pane_id} #{window_id} #{pane_index}'"
    # Build nested if-shell as the else branch of the outer format guard.
    # When the outer guard passes (no mode/mouse/alt), the inner gate atomically
    # sets @yankee_busy — if it already exists, the launch is rejected.
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

    # Register yankee bindings first — they don't need a clipboard command
    # and must not be blocked by the early exit in error_handling_if_command_not_present.
    set_yankee_binding
    set_scroll_bindings

    local copy_command
    copy_command="$(clipboard_copy_command)"
    error_handling_if_command_not_present "$copy_command"
    set_copy_mode_bindings "$copy_command"
    set_normal_bindings
}
main
