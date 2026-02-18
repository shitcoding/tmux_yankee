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

# _get_yankee_opt_from_dump extracts a @yankee_* option value from a pre-fetched
# tmux show-options -g dump. Prints the value, or empty string if the option is unset.
# Args: $1 = option name (e.g. @yankee_mode), $2 = raw dump string
_get_yankee_opt_from_dump() {
    local opt="$1" dump="$2" line
    # show-options -g format: "@opt_name value" one per line (value is unquoted for user options)
    # grep returns 1 when no match; use || true to avoid tripping set -e / pipefail.
    line=$(printf '%s\n' "$dump" | grep -m1 "^${opt} " || true)
    if [[ -n "$line" ]]; then
        printf '%s' "${line#"${opt} "}"
    fi
}

# Build the CLI argument array to pass to the tmux-yankee binary.
# Populates the global _YANKEE_ARGS array and also emits it as null-delimited output.
# Reads all @yankee_* tmux options in a single show-options call.
build_yankee_args() {
    # Fetch all global options in one subprocess call instead of one per option.
    local opts_dump
    opts_dump=$(tmux show-options -g 2>/dev/null || true)

    local mode
    mode=$(_get_yankee_opt_from_dump @yankee_mode "$opts_dump")
    mode="${mode:-hybrid}"

    _YANKEE_ARGS=("--pane" "$PANE_ID" "--mode" "$mode")

    # Pairs: tmux option name, CLI flag
    local opt_map
    opt_map=(
        "@yankee_scrollback_lines"    "--scrollback-lines"
        "@yankee_theme"               "--theme"
        "@yankee_cursor_fg"           "--cursor-fg"
        "@yankee_cursor_bg"           "--cursor-bg"
        "@yankee_selection_fg"        "--selection-fg"
        "@yankee_selection_bg"        "--selection-bg"
        "@yankee_gutter_fg"           "--gutter-fg"
        "@yankee_gutter_bg"           "--gutter-bg"
        "@yankee_gutter_separator_fg" "--gutter-separator-fg"
        "@yankee_linenum_absolute_fg" "--linenum-absolute-fg"
        "@yankee_linenum_relative_fg" "--linenum-relative-fg"
        "@yankee_linenum_cursor_fg"   "--linenum-cursor-fg"
        "@yankee_linenum_cursor_bold" "--linenum-cursor-bold"
        "@yankee_status_fg"           "--status-fg"
        "@yankee_status_bg"           "--status-bg"
        "@yankee_toggle_mode_key"     "--toggle-mode-key"
        "@yankee_copy_target"         "--copy-target"
        "@yankee_exit_on_yank"        "--exit-on-yank"
        "@yankee_start_position"      "--start-position"
    )

    local i
    for (( i=0; i<${#opt_map[@]}; i+=2 )); do
        local tmux_opt="${opt_map[i]}"
        local flag="${opt_map[i+1]}"
        local val
        val=$(_get_yankee_opt_from_dump "$tmux_opt" "$opts_dump")
        if [ -n "$val" ]; then
            _YANKEE_ARGS+=("$flag" "$val")
        fi
    done

    printf '%s\0' "${_YANKEE_ARGS[@]}"
}

# Launch in overlay mode: helper command performs inline swap-back + wait-for signal.
launch_overlay() {
    if ! yankee_lock_acquire; then return 0; fi
    trap 'yankee_lock_release; cleanup_overlay' EXIT INT TERM HUP

    local orig_pane_id orig_pane_path orig_zoom_state orig_pane_width orig_pane_height
    local helper_window_id helper_pane_id wait_signal helper_cmd

    # Capture original state
    orig_pane_id="$PANE_ID"
    orig_pane_path="$(tmux display-message -p -t "$orig_pane_id" '#{pane_current_path}')"
    orig_zoom_state="$(tmux display-message -p -t "$orig_pane_id" '#{window_zoomed_flag}')"
    orig_pane_width="$(tmux display-message -p -t "$orig_pane_id" '#{pane_width}')"
    orig_pane_height="$(tmux display-message -p -t "$orig_pane_id" '#{pane_height}')"

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
    # Match original pane dimensions to prevent SIGWINCH when the shell pane is
    # swapped into the helper session — SIGWINCH causes zsh to redraw its prompt,
    # leaving a duplicate prompt line visible after swap-back.
    # The session is hidden from the user's window list and killed on cleanup.
    if ! helper_window_id="$(tmux new-session -d -s "$helper_session_name" -x "$orig_pane_width" -y "$orig_pane_height" -P -F '#{window_id}' -c "$orig_pane_path" "$helper_cmd")"; then
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

    # Force the helper window to exactly match the original pane dimensions.
    # tmux may ignore -x/-y in new-session (e.g. in tmux 3.5a it uses default-size
    # instead), so we resize explicitly here — before the swap — to prevent SIGWINCH
    # on the original pane when it is moved into the helper session.
    tmux resize-window -t "$helper_window_id" -x "$orig_pane_width" -y "$orig_pane_height" 2>/dev/null || true

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
