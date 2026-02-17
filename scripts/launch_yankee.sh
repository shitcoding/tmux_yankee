#!/usr/bin/env bash

# Launcher for numbered copy mode
# Gathers tmux context and launches Go TUI binary with configurable display mode

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BIN_DIR="${SCRIPT_DIR}/../bin"

# Get current pane ID
PANE_ID=$(tmux display-message -p '#{pane_id}')

# Shell-routing decisions (not forwarded to binary)
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
YANKEE_OVERLAY_HELPER_SESSION_NAME=""
YANKEE_OVERLAY_HELPER_WINDOW_ID=""
YANKEE_OVERLAY_HELPER_PANE_ID=""
YANKEE_OVERLAY_WAIT_SIGNAL=""

# Launch lock: prevent concurrent launches (rapid inertial scroll firing multiple instances).
# mkdir is atomic on all POSIX filesystems — the OS kernel guarantees only one caller succeeds.
# A PID file inside the lock dir enables stale-lock recovery after SIGKILL.
_YANKEE_LOCK_DIR="/tmp/tmux-yankee-launch-${UID}.lock"
_YANKEE_LOCK_PID_FILE="${_YANKEE_LOCK_DIR}/pid"

yankee_lock_acquire() {
    if mkdir "$_YANKEE_LOCK_DIR" 2>/dev/null; then
        echo $$ > "$_YANKEE_LOCK_PID_FILE"
        return 0
    fi
    # Lock dir exists — check if the holder is still alive.
    local lock_pid
    lock_pid=$(cat "$_YANKEE_LOCK_PID_FILE" 2>/dev/null || true)
    if [ -n "$lock_pid" ] && kill -0 "$lock_pid" 2>/dev/null; then
        return 1  # Holder is alive: genuinely locked, skip this launch.
    fi
    # Holder is dead (SIGKILL): steal the lock.
    echo $$ > "$_YANKEE_LOCK_PID_FILE"
    return 0
}

yankee_lock_release() {
    rm -f "$_YANKEE_LOCK_PID_FILE" 2>/dev/null || true
    rmdir "$_YANKEE_LOCK_DIR" 2>/dev/null || true
}

# Global args array populated by build_yankee_args (reset on each call).
_YANKEE_ARGS=()

tmux_pane_exists() {
    local pane_id="${1:-}"
    [ -n "$pane_id" ] && tmux list-panes -a -F '#{pane_id}' 2>/dev/null | grep -Fxq -- "$pane_id"
}

tmux_window_exists() {
    local window_id="${1:-}"
    [ -n "$window_id" ] && tmux list-windows -a -F '#{window_id}' 2>/dev/null | grep -Fxq -- "$window_id"
}

tmux_session_exists() {
    local session_name="${1:-}"
    [ -n "$session_name" ] && tmux has-session -t "$session_name" 2>/dev/null
}

cleanup_overlay() {
    if [ "${YANKEE_OVERLAY_ACTIVE:-0}" -ne 1 ]; then
        return 0
    fi

    local orig_pane_id="${YANKEE_OVERLAY_ORIG_PANE_ID:-}"
    local orig_zoom_state="${YANKEE_OVERLAY_ORIG_ZOOM_STATE:-0}"
    local helper_session_name="${YANKEE_OVERLAY_HELPER_SESSION_NAME:-}"
    local helper_window_id="${YANKEE_OVERLAY_HELPER_WINDOW_ID:-}"
    local helper_pane_id="${YANKEE_OVERLAY_HELPER_PANE_ID:-}"
    local swapback_confirmed="${YANKEE_OVERLAY_SWAPBACK_CONFIRMED:-0}"
    local current_zoom_state=""

    YANKEE_OVERLAY_ACTIVE=0

    # Fallback swap-back only if helper did not confirm completion.
    if [ "$swapback_confirmed" -ne 1 ] && tmux_pane_exists "$helper_pane_id" && tmux_pane_exists "$orig_pane_id"; then
        tmux swap-pane -d -s "$helper_pane_id" -t "$orig_pane_id" -Z 2>/dev/null || true
    fi

    # Kill the temporary helper session (takes the window and pane with it).
    # Fall back to killing just the window if session tracking is unavailable.
    if tmux_session_exists "$helper_session_name"; then
        tmux kill-session -t "$helper_session_name" 2>/dev/null || true
    elif tmux_window_exists "$helper_window_id"; then
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
    YANKEE_OVERLAY_HELPER_SESSION_NAME=""
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
            # Pane exited — give the in-flight wait-for signal a moment to arrive
            # before concluding it was lost (the signal is sent just before pane exit).
            sleep 0.3
            if ! kill -0 "$waiter_pid" 2>/dev/null; then
                # Waiter already exited: signal was received successfully.
                wait "$waiter_pid"
                return $?
            fi
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

# Append a flag+value pair to _YANKEE_ARGS if the tmux option is non-empty.
_append_yankee_opt() {
    local tmux_opt="$1" flag="$2" val
    val=$(tmux show-option -gqv "$tmux_opt")
    if [ -n "$val" ]; then
        _YANKEE_ARGS+=("$flag" "$val")
    fi
}

# Build the CLI argument array to pass to the tmux-yankee binary.
# Populates the global _YANKEE_ARGS array and also emits it as null-delimited output.
# Reads all @yankee_* tmux options and forwards non-empty values as flags.
build_yankee_args() {
    local mode
    mode=$(tmux show-option -gqv @yankee_mode)
    mode="${mode:-hybrid}"

    _YANKEE_ARGS=("--pane" "$PANE_ID" "--mode" "$mode")

    _append_yankee_opt @yankee_scrollback_lines    --scrollback-lines
    _append_yankee_opt @yankee_theme               --theme
    _append_yankee_opt @yankee_cursor_fg           --cursor-fg
    _append_yankee_opt @yankee_cursor_bg           --cursor-bg
    _append_yankee_opt @yankee_selection_fg        --selection-fg
    _append_yankee_opt @yankee_selection_bg        --selection-bg
    _append_yankee_opt @yankee_gutter_fg           --gutter-fg
    _append_yankee_opt @yankee_gutter_bg           --gutter-bg
    _append_yankee_opt @yankee_gutter_separator_fg --gutter-separator-fg
    _append_yankee_opt @yankee_linenum_absolute_fg --linenum-absolute-fg
    _append_yankee_opt @yankee_linenum_relative_fg --linenum-relative-fg
    _append_yankee_opt @yankee_linenum_cursor_fg   --linenum-cursor-fg
    _append_yankee_opt @yankee_linenum_cursor_bold --linenum-cursor-bold
    _append_yankee_opt @yankee_status_fg           --status-fg
    _append_yankee_opt @yankee_status_bg           --status-bg
    _append_yankee_opt @yankee_toggle_mode_key     --toggle-mode-key
    _append_yankee_opt @yankee_copy_target         --copy-target
    _append_yankee_opt @yankee_exit_on_yank        --exit-on-yank
    _append_yankee_opt @yankee_start_position      --start-position

    printf '%s\0' "${_YANKEE_ARGS[@]}"
}

# Launch in overlay mode: helper command performs inline swap-back + wait-for signal.
launch_overlay() {
    if ! yankee_lock_acquire; then return 0; fi
    trap 'yankee_lock_release; cleanup_overlay' EXIT INT TERM HUP

    local orig_pane_id orig_pane_path orig_zoom_state
    local helper_window_id helper_pane_id wait_signal helper_cmd

    # Capture original state
    orig_pane_id="$PANE_ID"
    orig_pane_path="$(tmux display-message -p -t "$orig_pane_id" '#{pane_current_path}')"
    orig_zoom_state="$(tmux display-message -p -t "$orig_pane_id" '#{window_zoomed_flag}')"

    # Initialize global overlay state.
    YANKEE_OVERLAY_ACTIVE=1
    YANKEE_OVERLAY_SWAPBACK_CONFIRMED=0
    YANKEE_OVERLAY_ORIG_PANE_ID="$orig_pane_id"
    YANKEE_OVERLAY_ORIG_ZOOM_STATE="$orig_zoom_state"
    YANKEE_OVERLAY_HELPER_SESSION_NAME=""
    YANKEE_OVERLAY_HELPER_WINDOW_ID=""
    YANKEE_OVERLAY_HELPER_PANE_ID=""

    wait_signal="yankee-finished-${$}-$(date +%s)-${RANDOM}"
    YANKEE_OVERLAY_WAIT_SIGNAL="$wait_signal"

    # Helper session name: unique per launcher PID so concurrent invocations don't collide.
    local helper_session_name="tmux-yankee-tmp-$$"

    # Build yankee args as a null-delimited byte stream, decode to array (bash 3.2 compatible)
    local yankee_args=()
    while IFS= read -r -d '' arg; do yankee_args+=("$arg"); done < <(build_yankee_args)

    # Encode as shell-safe string for embedding in helper_cmd
    local yankee_args_quoted
    yankee_args_quoted=$(printf '%q ' "${yankee_args[@]}")

    # Helper command:
    # 1) run Go TUI
    # 2) swap back from helper pane to original pane position
    # 3) signal launcher that swap-back is complete
    #
    # Use "$TMUX_PANE" inside helper shell so source pane is always the helper pane.
    printf -v helper_cmd '%q %s; tmux swap-pane -d -s "$TMUX_PANE" -t %q -Z 2>/dev/null || true; tmux wait-for -S %q' \
        "${BIN_DIR}/tmux-yankee" "$yankee_args_quoted" "$orig_pane_id" "$wait_signal"

    # Create a detached temporary session to host the helper pane.
    # The session is hidden from the user's window list and killed on cleanup.
    if ! helper_window_id="$(tmux new-session -d -s "$helper_session_name" -P -F '#{window_id}' -c "$orig_pane_path" "$helper_cmd")"; then
        tmux display-message "Error: tmux-yankee failed to create helper session"
        return 1
    fi
    if [ -z "$helper_window_id" ]; then
        tmux display-message "Error: tmux-yankee failed to create helper session"
        return 1
    fi

    YANKEE_OVERLAY_HELPER_SESSION_NAME="$helper_session_name"
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
    if ! yankee_lock_acquire; then return 0; fi
    trap 'yankee_lock_release' EXIT INT TERM HUP
    local yankee_args=()
    while IFS= read -r -d '' arg; do yankee_args+=("$arg"); done < <(build_yankee_args)
    tmux display-popup -E -w 90% -h 90% \
        "${BIN_DIR}/tmux-yankee" "${yankee_args[@]}"
    trap - EXIT INT TERM HUP
    yankee_lock_release
}

# Launch in split window mode (horizontal split)
launch_split() {
    if ! yankee_lock_acquire; then return 0; fi
    trap 'yankee_lock_release' EXIT INT TERM HUP
    local yankee_args=()
    while IFS= read -r -d '' arg; do yankee_args+=("$arg"); done < <(build_yankee_args)
    tmux split-window -h \
        "${BIN_DIR}/tmux-yankee" "${yankee_args[@]}"
    trap - EXIT INT TERM HUP
    yankee_lock_release
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
