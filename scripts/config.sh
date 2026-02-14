#!/usr/bin/env bash
# config.sh - Read and validate tmux @linenumbers-* options
#
# All functions read from tmux options with fallback defaults.
# These defaults match what plugin.tmux sets, providing defense-in-depth
# if the plugin is sourced without running plugin.tmux first.

get_mode() {
    # Signature: get_mode()
    # Returns: "absolute" | "relative" | "hybrid" (stdout)
    local mode
    mode=$(tmux show-option -gqv "@linenumbers-mode")
    case "$mode" in
        absolute|relative|hybrid) printf '%s' "$mode" ;;
        *) printf '%s' "hybrid" ;;
    esac
}

get_style_absolute() {
    # Signature: get_style_absolute()
    # Returns: tmux style string (stdout)
    local style
    style=$(tmux show-option -gqv "@linenumbers-style-absolute")
    printf '%s' "${style:-fg=white}"
}

get_style_relative() {
    # Signature: get_style_relative()
    # Returns: tmux style string (stdout)
    local style
    style=$(tmux show-option -gqv "@linenumbers-style-relative")
    printf '%s' "${style:-fg=yellow}"
}

get_style_cursor() {
    # Signature: get_style_cursor()
    # Returns: tmux style string (stdout)
    local style
    style=$(tmux show-option -gqv "@linenumbers-style-cursor")
    printf '%s' "${style:-fg=green,bold}"
}

get_toggle_key() {
    # Signature: get_toggle_key()
    # Returns: single key string (stdout)
    local key
    key=$(tmux show-option -gqv "@linenumbers-toggle-key")
    printf '%s' "${key:-L}"
}

get_enable_binding() {
    # Signature: get_enable_binding()
    # Returns: "on" | "off" (stdout)
    local val
    val=$(tmux show-option -gqv "@linenumbers-enable-binding")
    case "$val" in
        on) printf '%s' "on" ;;
        *) printf '%s' "off" ;;
    esac
}

get_custom_key() {
    # Signature: get_custom_key()
    # Returns: key string (stdout)
    local key
    key=$(tmux show-option -gqv "@linenumbers-custom-key")
    printf '%s' "${key:-N}"
}

cycle_mode() {
    # Signature: cycle_mode()
    # Side effect: Updates @linenumbers-mode in tmux
    # Returns: new mode on stdout
    local current_mode
    current_mode=$(get_mode)
    local new_mode
    case "$current_mode" in
        absolute) new_mode="relative" ;;
        relative) new_mode="hybrid" ;;
        hybrid)   new_mode="absolute" ;;
    esac
    tmux set-option -g "@linenumbers-mode" "$new_mode"
    printf '%s' "$new_mode"
}
