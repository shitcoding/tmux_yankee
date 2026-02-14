#!/usr/bin/env bash
# state_cleanup.sh - Idempotent pane restoration and cleanup

# Global state for cleanup (set by setup_cleanup_trap)
_CLEANUP_SOURCE_PANE=""
_CLEANUP_TEMP_PANE=""
_CLEANUP_WAIT_CHANNEL=""
_CLEANUP_WAS_ZOOMED="0"
_CLEANUP_DONE=0

setup_cleanup_trap() {
    # Signature: setup_cleanup_trap(source_pane, temp_pane, wait_channel, was_zoomed)
    # Side effect: Registers trap handlers for EXIT, INT, TERM, HUP
    local source_pane="$1"
    local temp_pane="$2"
    local wait_channel="$3"
    local was_zoomed="${4:-0}"

    _CLEANUP_SOURCE_PANE="$source_pane"
    _CLEANUP_TEMP_PANE="$temp_pane"
    _CLEANUP_WAIT_CHANNEL="$wait_channel"
    _CLEANUP_WAS_ZOOMED="$was_zoomed"

    trap '_run_cleanup' EXIT INT TERM HUP
}

_run_cleanup() {
    cleanup "$_CLEANUP_SOURCE_PANE" "$_CLEANUP_TEMP_PANE"
}

cleanup() {
    # Signature: cleanup(source_pane, temp_pane)
    # Returns: 0 always (best-effort cleanup)
    # Side effects: Restores source pane (respawn with shell), removes bindings
    # IDEMPOTENT: safe to call multiple times

    local source_pane="$1"
    local temp_pane="$2"

    # Guard against double execution
    if [[ $_CLEANUP_DONE -eq 1 ]]; then
        return 0
    fi
    _CLEANUP_DONE=1

    # Step 1: Restore source pane by respawning with default shell
    # The source pane was respawned with numbered content; restore it to a shell
    if pane_exists "$source_pane"; then
        tmux respawn-pane -k -t "$source_pane" 2>/dev/null || true
    fi

    # Step 2: Kill separate temp pane if one exists (legacy swap-pane approach)
    if [[ -n "$temp_pane" ]] && pane_exists "$temp_pane"; then
        tmux kill-pane -t "$temp_pane" 2>/dev/null || true
    fi

    # Step 3: Restore zoom state if pane was zoomed before
    if [[ "$_CLEANUP_WAS_ZOOMED" == "1" ]] && pane_exists "$source_pane"; then
        tmux resize-pane -t "$source_pane" -Z 2>/dev/null || true
    fi

    # Step 4: Restore original copy-mode-vi key bindings
    restore_keybindings

    # Step 5: Signal wait-for channel (in case cleanup is triggered by signal
    # while main is still waiting)
    if [[ -n "$_CLEANUP_WAIT_CHANNEL" ]]; then
        tmux wait-for -S "$_CLEANUP_WAIT_CHANNEL" 2>/dev/null || true
    fi

    # Step 6: Clean up state directory
    local state_dir="/tmp/linenumbers-state-$$"
    if [[ -d "$state_dir" ]]; then
        rm -rf "$state_dir" 2>/dev/null || true
    fi
}

pane_exists() {
    # Signature: pane_exists(pane_id)
    # Returns: 0 if pane exists, 1 otherwise
    local pane_id="$1"
    [[ -n "$pane_id" ]] && tmux list-panes -a -F '#{pane_id}' 2>/dev/null | grep -qF "$pane_id"
}

restore_keybindings() {
    # Signature: restore_keybindings()
    # Side effect: Removes plugin-specific copy-mode-vi overrides
    # Returns: 0 always

    # Remove our custom bindings from copy-mode-vi table
    local toggle_key
    toggle_key=$(tmux show-option -gqv "@linenumbers-toggle-key" 2>/dev/null || echo "L")
    toggle_key="${toggle_key:-L}"

    tmux unbind-key -T copy-mode-vi "$toggle_key" 2>/dev/null || true
    # Restore standard copy-mode-vi key defaults
    tmux bind-key -T copy-mode-vi q send-keys -X cancel 2>/dev/null || true
    tmux bind-key -T copy-mode-vi Escape send-keys -X cancel 2>/dev/null || true
    tmux bind-key -T copy-mode-vi Enter send-keys -X copy-selection-and-cancel 2>/dev/null || true
    tmux bind-key -T copy-mode-vi y send-keys -X copy-selection-and-cancel 2>/dev/null || true
}
