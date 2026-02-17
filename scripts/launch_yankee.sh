#!/usr/bin/env bash

# Launcher for numbered copy mode
# Gathers tmux context and launches Go TUI binary with configurable display mode

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BIN_DIR="${SCRIPT_DIR}/../bin"

# Get current pane ID
PANE_ID=$(tmux display-message -p '#{pane_id}')

# Get user configuration (or use defaults)
MODE=$(tmux show-option -gqv @yankee_mode)
MODE="${MODE:-hybrid}"

DISPLAY_MODE=$(tmux show-option -gqv @yankee_display_mode)
DISPLAY_MODE="${DISPLAY_MODE:-overlay}"

# Check if binary exists
if [ ! -f "${BIN_DIR}/tmux-yankee" ]; then
    tmux display-message "Error: tmux-yankee binary not found. Run 'make build' first."
    exit 1
fi

# Overlay lifecycle state shared with traps.
# Keep these global so they are still defined when EXIT trap runs under `set -u`.
YANKEE_OVERLAY_ACTIVE=0
YANKEE_OVERLAY_SWAPBACK_CONFIRMED=0
YANKEE_OVERLAY_ORIG_PANE_ID=""
YANKEE_OVERLAY_ORIG_ZOOM_STATE=""
YANKEE_OVERLAY_HELPER_WINDOW_ID=""
YANKEE_OVERLAY_HELPER_PANE_ID=""
YANKEE_OVERLAY_WAIT_SIGNAL=""

tmux_pane_exists() {
    local pane_id="${1:-}"
    [ -n "$pane_id" ] && tmux list-panes -a -F '#{pane_id}' 2>/dev/null | grep -Fxq -- "$pane_id"
}

tmux_window_exists() {
    local window_id="${1:-}"
    [ -n "$window_id" ] && tmux list-windows -a -F '#{window_id}' 2>/dev/null | grep -Fxq -- "$window_id"
}

cleanup_overlay() {
    if [ "${YANKEE_OVERLAY_ACTIVE:-0}" -ne 1 ]; then
        return 0
    fi

    local orig_pane_id="${YANKEE_OVERLAY_ORIG_PANE_ID:-}"
    local orig_zoom_state="${YANKEE_OVERLAY_ORIG_ZOOM_STATE:-0}"
    local helper_window_id="${YANKEE_OVERLAY_HELPER_WINDOW_ID:-}"
    local helper_pane_id="${YANKEE_OVERLAY_HELPER_PANE_ID:-}"
    local swapback_confirmed="${YANKEE_OVERLAY_SWAPBACK_CONFIRMED:-0}"
    local current_zoom_state=""

    YANKEE_OVERLAY_ACTIVE=0

    # Fallback swap-back only if helper did not confirm completion.
    if [ "$swapback_confirmed" -ne 1 ] && tmux_pane_exists "$helper_pane_id" && tmux_pane_exists "$orig_pane_id"; then
        tmux swap-pane -d -s "$helper_pane_id" -t "$orig_pane_id" -Z 2>/dev/null || true
    fi

    # Always kill helper window to avoid dead panes when remain-on-exit is enabled.
    if tmux_window_exists "$helper_window_id"; then
        tmux kill-window -t "$helper_window_id" 2>/dev/null || true
    fi

    # Enforce original zoom state if needed.
    if tmux_pane_exists "$orig_pane_id"; then
        current_zoom_state="$(tmux display-message -p -t "$orig_pane_id" '#{window_zoomed_flag}' 2>/dev/null || true)"
        if [ -n "$current_zoom_state" ] && [ "$current_zoom_state" != "$orig_zoom_state" ]; then
            tmux resize-pane -Z -t "$orig_pane_id" 2>/dev/null || true
        fi
    fi

    YANKEE_OVERLAY_SWAPBACK_CONFIRMED=0
    YANKEE_OVERLAY_ORIG_PANE_ID=""
    YANKEE_OVERLAY_ORIG_ZOOM_STATE=""
    YANKEE_OVERLAY_HELPER_WINDOW_ID=""
    YANKEE_OVERLAY_HELPER_PANE_ID=""
    YANKEE_OVERLAY_WAIT_SIGNAL=""
}

wait_for_helper_completion() {
    # Waits for helper's wait-for signal, but won't block forever:
    # returns non-zero if helper pane disappears first or timeout is reached.
    local signal="$1"
    local helper_pane_id="$2"
    local timeout_ticks=1200   # 120 seconds at 0.1s/tick
    local ticks=0
    local waiter_pid

    tmux wait-for "$signal" &
    waiter_pid=$!

    while kill -0 "$waiter_pid" 2>/dev/null; do
        if ! tmux_pane_exists "$helper_pane_id"; then
            kill "$waiter_pid" 2>/dev/null || true
            wait "$waiter_pid" 2>/dev/null || true
            return 1
        fi

        sleep 0.1
        ticks=$((ticks + 1))
        if [ "$ticks" -ge "$timeout_ticks" ]; then
            kill "$waiter_pid" 2>/dev/null || true
            wait "$waiter_pid" 2>/dev/null || true
            return 1
        fi
    done

    wait "$waiter_pid"
}

# Check if display-popup is supported (tmux 3.2+)
popup_supported() {
    tmux display-popup -E -B -w 1 -h 1 "true" >/dev/null 2>&1
}

# Launch in overlay mode: helper command performs inline swap-back + wait-for signal.
launch_overlay() {
    local orig_pane_id orig_pane_path orig_zoom_state
    local helper_window_id helper_pane_id wait_signal helper_cmd

    # Capture original state
    orig_pane_id="$PANE_ID"
    orig_pane_path="$(tmux display-message -p -t "$orig_pane_id" '#{pane_current_path}')"
    orig_zoom_state="$(tmux display-message -p -t "$orig_pane_id" '#{window_zoomed_flag}')"

    # Arm trap and initialize global overlay state.
    trap cleanup_overlay EXIT INT TERM HUP
    YANKEE_OVERLAY_ACTIVE=1
    YANKEE_OVERLAY_SWAPBACK_CONFIRMED=0
    YANKEE_OVERLAY_ORIG_PANE_ID="$orig_pane_id"
    YANKEE_OVERLAY_ORIG_ZOOM_STATE="$orig_zoom_state"
    YANKEE_OVERLAY_HELPER_WINDOW_ID=""
    YANKEE_OVERLAY_HELPER_PANE_ID=""

    wait_signal="numcopy-finished-${$}-$(date +%s)-${RANDOM}"
    YANKEE_OVERLAY_WAIT_SIGNAL="$wait_signal"

    # Helper command:
    # 1) run Go TUI
    # 2) swap back from helper pane to original pane position
    # 3) signal launcher that swap-back is complete
    #
    # Use "$TMUX_PANE" inside helper shell so source pane is always the helper pane.
    printf -v helper_cmd '%q --pane %q --mode %q; tmux swap-pane -d -s "$TMUX_PANE" -t %q -Z 2>/dev/null || true; tmux wait-for -S %q' \
        "${BIN_DIR}/tmux-yankee" "$orig_pane_id" "$MODE" "$orig_pane_id" "$wait_signal"

    # Create detached helper window in original CWD
    if ! helper_window_id="$(tmux new-window -d -P -F '#{window_id}' -c "$orig_pane_path" "$helper_cmd")"; then
        tmux display-message "Error: tmux-yankee failed to create helper window"
        return 1
    fi
    if [ -z "$helper_window_id" ]; then
        tmux display-message "Error: tmux-yankee failed to create helper window"
        return 1
    fi

    YANKEE_OVERLAY_HELPER_WINDOW_ID="$helper_window_id"

    # Resolve helper pane
    helper_pane_id="$(tmux list-panes -t "$helper_window_id" -F '#{pane_id}' | head -1)"
    if [ -z "$helper_pane_id" ]; then
        tmux display-message "Error: tmux-yankee failed to get helper pane ID"
        return 1
    fi

    YANKEE_OVERLAY_HELPER_PANE_ID="$helper_pane_id"

    # Swap helper into original pane position (overlay visible to user)
    if ! tmux swap-pane -d -s "$orig_pane_id" -t "$helper_pane_id" -Z; then
        tmux display-message "Error: tmux-yankee swap-pane failed"
        return 1
    fi

    # Wait for helper's "swap-back done" signal.
    # If this fails, trap cleanup performs fallback swap-back + window cleanup.
    if wait_for_helper_completion "$wait_signal" "$helper_pane_id"; then
        YANKEE_OVERLAY_SWAPBACK_CONFIRMED=1
    else
        tmux display-message "tmux-yankee: helper completion signal missing; forcing fallback cleanup"
    fi

    cleanup_overlay
    trap - EXIT INT TERM HUP
}

# Launch in centered popup mode (90% width/height)
launch_popup() {
    tmux display-popup -E -w 90% -h 90% \
        "${BIN_DIR}/tmux-yankee" --pane "$PANE_ID" --mode "$MODE"
}

# Launch in split window mode (horizontal split)
launch_split() {
    tmux split-window -h \
        "${BIN_DIR}/tmux-yankee" --pane "$PANE_ID" --mode "$MODE"
}

# Dispatch to appropriate display mode
case "$DISPLAY_MODE" in
    overlay)
        launch_overlay
        ;;
    popup)
        if popup_supported; then
            launch_popup
        else
            tmux display-message "tmux-yankee: popup requires tmux 3.2+ (display-popup); falling back to split"
            launch_split
        fi
        ;;
    split)
        launch_split
        ;;
    *)
        tmux display-message "tmux-yankee: invalid @yankee_display_mode='$DISPLAY_MODE'; using overlay"
        launch_overlay
        ;;
esac
