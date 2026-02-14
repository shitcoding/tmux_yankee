#!/usr/bin/env bash
# utils.sh - Shared helper functions for tmux-copymode-linenumbers

get_active_pane_id() {
    # Signature: get_active_pane_id()
    # Returns: pane_id string (stdout), e.g., "%0"
    # Works from both interactive and run-shell contexts
    tmux list-panes -F '#{pane_id} #{pane_active}' | grep ' 1$' | awk '{print $1}'
}

get_pane_format() {
    # Signature: get_pane_format(pane_id, format_string)
    # Returns: format value (stdout)
    # Works from run-shell context using list-panes instead of display-message
    local pane_id="$1"
    local format="$2"
    tmux list-panes -a -F "#{pane_id} $format" | grep "^${pane_id} " | sed "s/^${pane_id} //"
}

check_tmux_version() {
    # Signature: check_tmux_version(minimum_version)
    # Returns: 0 if current tmux >= minimum, 1 otherwise
    local min_version="$1"
    local current_version
    current_version=$(tmux -V | sed 's/tmux //')

    # Compare major.minor
    local current_major current_minor min_major min_minor
    current_major=$(echo "$current_version" | cut -d. -f1)
    current_minor=$(echo "$current_version" | cut -d. -f2 | sed 's/[^0-9]//g')
    min_major=$(echo "$min_version" | cut -d. -f1)
    min_minor=$(echo "$min_version" | cut -d. -f2)

    if [[ $current_major -gt $min_major ]]; then
        return 0
    elif [[ $current_major -eq $min_major && $current_minor -ge $min_minor ]]; then
        return 0
    else
        return 1
    fi
}

log_error() {
    # Signature: log_error(message)
    # Side effect: Writes to stderr and tmux display-message
    local message="$1"
    echo "linenumbers: $message" >&2
    tmux display-message "linenumbers: $message" 2>/dev/null || true
}

log_debug() {
    # Signature: log_debug(message)
    # Side effect: Writes to /tmp/linenumbers-debug.log if DEBUG is set
    if [[ "${LINENUMBERS_DEBUG:-}" == "1" ]]; then
        echo "[$(date '+%H:%M:%S')] $1" >> /tmp/linenumbers-debug.log
    fi
}
