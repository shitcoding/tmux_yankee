#!/usr/bin/env bash

# Launcher for numbered copy mode
# Gathers tmux context and launches Go TUI binary with configurable display mode
# Supports multiple concurrent Yankee instances (one per pane) via per-pane locks.

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BIN_DIR="${SCRIPT_DIR}/../bin"

# Accept pane identity from the binding — avoids untargeted display-message -p
# which can return the wrong pane in run-shell -b context.
PANE_ID="${1:-}"
ORIG_WINDOW_ID="${2:-}"
ORIG_PANE_INDEX="${3:-}"

# Bail if arguments are missing (backward compat: fall back to display-message).
if [ -z "$PANE_ID" ]; then
    PANE_ID=$(tmux display-message -p '#{pane_id}' 2>/dev/null) || exit 0
fi
if [ -z "$ORIG_WINDOW_ID" ]; then
    ORIG_WINDOW_ID=$(tmux display-message -p -t "$PANE_ID" '#{window_id}' 2>/dev/null) || exit 0
fi
if [ -z "$ORIG_PANE_INDEX" ]; then
    ORIG_PANE_INDEX=$(tmux display-message -p -t "$PANE_ID" '#{pane_index}' 2>/dev/null) || exit 0
fi

# Helper: clear @yankee_busy on one or more panes (idempotent).
_yankee_clear_busy() {
    local pane
    for pane in "$@"; do
        [ -n "$pane" ] && tmux set-option -pu -t "$pane" @yankee_busy 2>/dev/null || true
    done
}

# Shell-routing decisions (not forwarded to binary)
DISPLAY_MODE=$(tmux show-option -gqv @yankee_display_mode 2>/dev/null || true)
DISPLAY_MODE="${DISPLAY_MODE:-overlay}"

# Check if binary exists
if [ ! -f "${BIN_DIR}/tmux-yankee" ]; then
    _yankee_clear_busy "$PANE_ID"
    tmux display-message "Error: tmux-yankee binary not found. Run 'make build' first." 2>/dev/null || true
    exit 0
fi

# ---------------------------------------------------------------------------
# Per-pane lock & state infrastructure
# ---------------------------------------------------------------------------
# State directory rooted under /tmp, keyed by tmux server identity so that
# a server restart invalidates all prior state.  Each pane gets its own lock
# dir and state file.

_yankee_runtime_base="${TMPDIR:-/tmp}"

# Build a server key that changes whenever the tmux server restarts.
# cksum is POSIX — no shasum dependency.
_yankee_server_key() {
    local socket_hash server_pid
    socket_hash=$(tmux display-message -p '#{socket_path}' | cksum | awk '{print $1}')
    server_pid=$(tmux display-message -p '#{pid}')
    printf 'u%s-s%s-p%s' "$UID" "$socket_hash" "$server_pid"
}

_YANKEE_SERVER_KEY="$(_yankee_server_key)"
_YANKEE_STATE_DIR="${_yankee_runtime_base}/tmux-yankee/${_YANKEE_SERVER_KEY}"
mkdir -p "$_YANKEE_STATE_DIR" 2>/dev/null || true

# Per-pane paths derived from the target pane id.
_yankee_pane_key() { printf '%s' "${1#%}"; }

_yankee_lock_dir()   { printf '%s/pane-%s.lock'  "$_YANKEE_STATE_DIR" "$(_yankee_pane_key "$1")"; }
_yankee_state_file() { printf '%s/pane-%s.state' "$_YANKEE_STATE_DIR" "$(_yankee_pane_key "$1")"; }

# --- Lock primitives (per-pane) -------------------------------------------

yankee_lock_acquire() {
    local lock_dir="$1"
    local pid_file="${lock_dir}/pid"
    if mkdir "$lock_dir" 2>/dev/null; then
        echo $$ > "$pid_file"
        return 0
    fi
    # Lock dir exists — check if holder is still alive.
    local lock_pid
    lock_pid=$(cat "$pid_file" 2>/dev/null || true)
    if [ -z "$lock_pid" ]; then
        # PID file missing or empty: another process just created the lock dir
        # but hasn't written its PID yet.  Treat as held (not stale).
        return 1
    fi
    if kill -0 "$lock_pid" 2>/dev/null; then
        return 1  # Holder alive: genuinely locked, skip this launch.
    fi
    # Holder dead: steal the lock.
    echo $$ > "$pid_file"
    return 0
}

yankee_lock_release() {
    local lock_dir="$1"
    rm -f "${lock_dir}/pid" 2>/dev/null || true
    rmdir "$lock_dir" 2>/dev/null || true
}

# --- State file helpers (key=value, atomic write) --------------------------

_yankee_state_write() {
    local state_file="$1"; shift
    local tmp_file="${state_file}.tmp.$$"
    # Write all key=value pairs passed as arguments
    printf '%s\n' "$@" > "$tmp_file"
    mv -f "$tmp_file" "$state_file"
}

_yankee_state_read_val() {
    local state_file="$1" key="$2" line
    [ -f "$state_file" ] || return 0
    line=$(grep -m1 "^${key}=" "$state_file" || true)
    if [ -n "$line" ]; then
        printf '%s' "${line#"${key}="}"
    fi
}

_yankee_state_remove() {
    rm -f "$1" 2>/dev/null || true
}

# --- Startup sweep: clean up stale state from crashes / server restarts ----

_yankee_startup_sweep() {
    local runtime_root="${_yankee_runtime_base}/tmux-yankee"
    [ -d "$runtime_root" ] || return 0

    local sweep_lock="${_YANKEE_STATE_DIR}/.sweep.lock"
    # Non-blocking: skip if another sweep is running.
    mkdir "$sweep_lock" 2>/dev/null || return 0

    # 1) Purge state dirs for other server keys (old/restarted tmux servers).
    local dir
    for dir in "$runtime_root"/u*; do
        [ -d "$dir" ] || continue
        [ "$dir" = "$_YANKEE_STATE_DIR" ] && continue
        rm -rf "$dir" 2>/dev/null || true
    done

    # 2) Recover stale overlays in current server dir.
    local state_file pane_lock
    for state_file in "$_YANKEE_STATE_DIR"/pane-*.state; do
        [ -f "$state_file" ] || continue

        # Derive lock dir from state file name.
        pane_lock="${state_file%.state}.lock"

        # Try non-blocking lock; skip if an active launcher owns this pane.
        if ! mkdir "$pane_lock" 2>/dev/null; then
            local existing_pid
            existing_pid=$(cat "${pane_lock}/pid" 2>/dev/null || true)
            if [ -z "$existing_pid" ]; then
                continue  # PID file not yet written — owner is starting up, skip.
            fi
            if kill -0 "$existing_pid" 2>/dev/null; then
                continue  # Active overlay, skip.
            fi
            # Dead owner — steal lock for recovery.
            echo $$ > "${pane_lock}/pid" 2>/dev/null || true
        else
            echo $$ > "${pane_lock}/pid"
        fi

        # Run cleanup for this orphaned state (also clears @yankee_busy).
        _yankee_cleanup_from_state "$state_file"
        yankee_lock_release "$pane_lock"
    done

    rmdir "$sweep_lock" 2>/dev/null || true
}

# --- tmux existence checks -------------------------------------------------

tmux_pane_exists() {
    local pane_id="${1:-}"
    [ -n "$pane_id" ] && tmux display-message -p -t "$pane_id" '#{pane_id}' >/dev/null 2>&1
}

tmux_window_exists() {
    local window_id="${1:-}"
    [ -n "$window_id" ] && tmux display-message -p -t "$window_id" '#{window_id}' >/dev/null 2>&1
}

tmux_session_exists() {
    local session_name="${1:-}"
    [ -n "$session_name" ] && tmux has-session -t "$session_name" 2>/dev/null
}

# --- Overlay cleanup (works from state file, not globals) ------------------

_yankee_cleanup_from_state() {
    local state_file="$1"
    [ -f "$state_file" ] || return 0

    local orig_pane_id orig_zoom_state helper_session_name helper_window_id helper_pane_id
    local swap_performed swapback_confirmed orig_session_name orig_window_id

    orig_pane_id=$(_yankee_state_read_val "$state_file" orig_pane_id)
    orig_zoom_state=$(_yankee_state_read_val "$state_file" orig_zoom_state)
    helper_session_name=$(_yankee_state_read_val "$state_file" helper_session_name)
    helper_window_id=$(_yankee_state_read_val "$state_file" helper_window_id)
    helper_pane_id=$(_yankee_state_read_val "$state_file" helper_pane_id)
    swap_performed=$(_yankee_state_read_val "$state_file" swap_performed)
    swap_performed="${swap_performed:-0}"
    swapback_confirmed=$(_yankee_state_read_val "$state_file" swapback_confirmed)
    swapback_confirmed="${swapback_confirmed:-0}"
    orig_session_name=$(_yankee_state_read_val "$state_file" orig_session_name)
    orig_window_id=$(_yankee_state_read_val "$state_file" orig_window_id)

    # Fallback swap-back: only attempt if the initial swap was performed but
    # the helper did not confirm a successful swap-back.
    if [ "$swap_performed" = "1" ] && [ "$swapback_confirmed" != "1" ] \
        && [ -n "$helper_pane_id" ] && [ -n "$orig_pane_id" ]; then
        if tmux_pane_exists "$helper_pane_id" && tmux_pane_exists "$orig_pane_id"; then
            tmux swap-pane -d -s "$helper_pane_id" -t "$orig_pane_id" -Z 2>/dev/null || true
        fi
    fi

    # SAFETY: Before killing the helper session, verify the user's original pane
    # is NOT inside it.  After a failed swap-back the original pane still sits in
    # the helper session — killing it would destroy the user's pane.
    local safe_to_kill=1
    if [ -n "$helper_session_name" ] && tmux_session_exists "$helper_session_name" \
        && [ -n "$orig_pane_id" ] && tmux_pane_exists "$orig_pane_id"; then
        local current_session
        current_session="$(tmux display-message -t "$orig_pane_id" -p '#{session_name}' 2>/dev/null || true)"
        if [ "$current_session" = "$helper_session_name" ]; then
            # Original pane is still trapped in the helper session.
            safe_to_kill=0
            # Rescue: move the pane back to the original window or session.
            if [ -n "$orig_window_id" ] && tmux_window_exists "$orig_window_id"; then
                if tmux move-pane -s "$orig_pane_id" -t "$orig_window_id" 2>/dev/null; then
                    safe_to_kill=1
                fi
            elif [ -n "$orig_session_name" ] && tmux_session_exists "$orig_session_name"; then
                if tmux move-pane -s "$orig_pane_id" -t "$orig_session_name" 2>/dev/null; then
                    safe_to_kill=1
                fi
            fi
        fi
    fi

    if [ "$safe_to_kill" = "1" ]; then
        if [ -n "$helper_session_name" ] && tmux_session_exists "$helper_session_name"; then
            tmux kill-session -t "$helper_session_name" 2>/dev/null || true
        elif [ -n "$helper_window_id" ] && tmux_window_exists "$helper_window_id"; then
            tmux kill-window -t "$helper_window_id" 2>/dev/null || true
        fi
    fi

    # Restore original zoom state.
    if [ -n "$orig_pane_id" ] && tmux_pane_exists "$orig_pane_id"; then
        local current_zoom
        current_zoom="$(tmux display-message -p -t "$orig_pane_id" '#{window_zoomed_flag}' 2>/dev/null || true)"
        if [ -n "$current_zoom" ] && [ "$current_zoom" != "${orig_zoom_state:-0}" ]; then
            tmux resize-pane -Z -t "$orig_pane_id" 2>/dev/null || true
        fi
    fi

    # Clear @yankee_busy on both panes so the binding gate reopens.
    _yankee_clear_busy "$orig_pane_id" "$helper_pane_id"

    _yankee_state_remove "$state_file"
}

# --- Option fetching (single show-options call) ----------------------------

# Global args array populated by build_yankee_args (reset on each call).
_YANKEE_ARGS=()

_get_yankee_opt_from_dump() {
    local opt="$1" dump="$2"
    local pattern=$'\n'"${opt} "
    if [[ "$dump" == *"$pattern"* ]]; then
        local after="${dump#*"$pattern"}"
        printf '%s' "${after%%$'\n'*}"
    elif [[ "$dump" == "${opt} "* ]]; then
        local after="${dump#"${opt} "}"
        printf '%s' "${after%%$'\n'*}"
    fi
}

build_yankee_args() {
    local opts_dump
    opts_dump=$(tmux show-options -g 2>/dev/null || true)

    local mode
    mode=$(_get_yankee_opt_from_dump @yankee_mode "$opts_dump")
    mode="${mode:-hybrid}"

    _YANKEE_ARGS=("--pane" "$PANE_ID" "--mode" "$mode")

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
        "@yankee_wrap_key"            "--wrap-key"
        "@yankee_copy_target"         "--copy-target"
        "@yankee_exit_on_yank"        "--exit-on-yank"
        "@yankee_start_position"      "--start-position"
        "@yankee_wrap_mode"           "--wrap-mode"
        "@yankee_status_bar"          "--status-bar"
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

    # Collect @yankee_bind_* and @yankee_unbind_* options into --bindings flag.
    # @yankee_bind_C-d half_page_down  →  "C-d=half_page_down"
    # @yankee_unbind_H ""              →  "!H"
    local bindings_str=""
    local line key_notation action
    while IFS= read -r line; do
        [ -z "$line" ] && continue
        # Match @yankee_bind_<key> <action>
        if [[ "$line" =~ ^@yankee_bind_(.+)" "(.+)$ ]]; then
            key_notation="${BASH_REMATCH[1]}"
            action="${BASH_REMATCH[2]}"
            if [ -n "$bindings_str" ]; then
                bindings_str="${bindings_str},${key_notation}=${action}"
            else
                bindings_str="${key_notation}=${action}"
            fi
        # Match @yankee_unbind_<key> (value is ignored)
        elif [[ "$line" =~ ^@yankee_unbind_(.+)" " ]]; then
            key_notation="${BASH_REMATCH[1]}"
            if [ -n "$bindings_str" ]; then
                bindings_str="${bindings_str},!${key_notation}"
            else
                bindings_str="!${key_notation}"
            fi
        fi
    done <<< "$(printf '%s' "$opts_dump" | grep -E '^@yankee_(bind|unbind)_' || true)"

    if [ -n "$bindings_str" ]; then
        _YANKEE_ARGS+=("--bindings" "$bindings_str")
    fi

    printf '%s\0' "${_YANKEE_ARGS[@]}"
}

# --- Wait for helper completion --------------------------------------------

wait_for_helper_completion() {
    local signal="$1"
    local helper_pane_id="$2"
    local timeout_ticks=1200   # 120 seconds at 0.1s/tick
    local ticks=0
    local waiter_pid

    tmux wait-for "$signal" &
    waiter_pid=$!

    while kill -0 "$waiter_pid" 2>/dev/null; do
        if ! tmux_pane_exists "$helper_pane_id"; then
            sleep 0.3
            if ! kill -0 "$waiter_pid" 2>/dev/null; then
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

# --- Check if display-popup is supported (tmux 3.2+) ----------------------

popup_supported() {
    tmux display-popup -E -B -w 1 -h 1 "true" >/dev/null 2>&1
}

# --- Launch modes ----------------------------------------------------------

launch_overlay() {
    local pane_lock_dir state_file
    pane_lock_dir="$(_yankee_lock_dir "$PANE_ID")"
    state_file="$(_yankee_state_file "$PANE_ID")"

    if ! yankee_lock_acquire "$pane_lock_dir"; then return 0; fi

    # Run stale-state sweep only after winning the lock (avoids sweep overhead
    # for the hundreds of blocked processes from rapid scroll events).
    _yankee_startup_sweep

    # CRITICAL: cleanup runs BEFORE lock release so the lock is held during restore.
    # Always clear @yankee_busy on the original pane (even if state file not yet written).
    trap '_yankee_cleanup_from_state "'"$state_file"'"; _yankee_clear_busy "'"$PANE_ID"'"; yankee_lock_release "'"$pane_lock_dir"'"' EXIT INT TERM HUP

    local orig_pane_id orig_pane_path orig_zoom_state orig_pane_width orig_pane_height
    local orig_session_name orig_window_id
    local helper_window_id helper_pane_id wait_signal helper_cmd

    orig_pane_id="$PANE_ID"
    # Batch all 6 tmux display-message calls into one (saves ~50-100ms).
    local _batch_out
    _batch_out="$(tmux display-message -p -t "$orig_pane_id" '#{pane_current_path}
#{window_zoomed_flag}
#{pane_width}
#{pane_height}
#{session_name}
#{window_id}' 2>/dev/null || true)"
    {
        IFS= read -r orig_pane_path
        IFS= read -r orig_zoom_state
        IFS= read -r orig_pane_width
        IFS= read -r orig_pane_height
        IFS= read -r orig_session_name
        IFS= read -r orig_window_id || true
    } <<< "$_batch_out"

    # Bail if pane disappeared between guard check and here.
    if [ -z "$orig_pane_width" ] || [ -z "$orig_pane_height" ]; then
        return 0
    fi

    wait_signal="yankee-finished-${$}-$(date +%s)-${RANDOM}"

    # Helper session: unique per pane key (not PID) so that the state file and
    # session name are deterministically recoverable.
    local pane_key
    pane_key="$(_yankee_pane_key "$PANE_ID")"
    local helper_session_name="tmux-yankee-ovl-${pane_key}"

    # Write initial state for crash recovery.
    _yankee_state_write "$state_file" \
        "version=2" \
        "owner_pid=$$" \
        "orig_pane_id=$orig_pane_id" \
        "orig_zoom_state=$orig_zoom_state" \
        "orig_session_name=$orig_session_name" \
        "orig_window_id=$orig_window_id" \
        "helper_session_name=$helper_session_name" \
        "helper_window_id=" \
        "helper_pane_id=" \
        "wait_signal=$wait_signal" \
        "swap_performed=0" \
        "swapback_confirmed=0"

    # Build yankee args
    local yankee_args=()
    while IFS= read -r -d '' arg; do yankee_args+=("$arg"); done < <(build_yankee_args)
    local yankee_args_quoted
    yankee_args_quoted=$(printf '%q ' "${yankee_args[@]}")

    # Helper command: run TUI → swap back → clear busy → signal launcher.
    printf -v helper_cmd '%q %s; tmux swap-pane -d -s "$TMUX_PANE" -t %q -Z 2>/dev/null || true; tmux set-option -pu -t "$TMUX_PANE" @yankee_busy 2>/dev/null; tmux set-option -pu -t %q @yankee_busy 2>/dev/null; tmux wait-for -S %q' \
        "${BIN_DIR}/tmux-yankee" "$yankee_args_quoted" "$orig_pane_id" "$orig_pane_id" "$wait_signal"

    # Kill any leftover helper session from a previous crash before creating a new one.
    if tmux_session_exists "$helper_session_name"; then
        tmux kill-session -t "$helper_session_name" 2>/dev/null || true
    fi

    # Create detached helper session with matching dimensions (prevents SIGWINCH on swap).
    if ! helper_window_id="$(tmux new-session -d -s "$helper_session_name" -x "$orig_pane_width" -y "$orig_pane_height" -P -F '#{window_id}' -c "$orig_pane_path" "$helper_cmd")"; then
        tmux display-message "Error: tmux-yankee failed to create helper session" 2>/dev/null || true
        return 0
    fi
    if [ -z "$helper_window_id" ]; then
        tmux display-message "Error: tmux-yankee failed to create helper session" 2>/dev/null || true
        return 0
    fi

    helper_pane_id="$(tmux list-panes -t "$helper_window_id" -F '#{pane_id}' 2>/dev/null | head -1)"
    if [ -z "$helper_pane_id" ]; then
        tmux display-message "Error: tmux-yankee failed to get helper pane ID" 2>/dev/null || true
        return 0
    fi

    # Update state with helper identifiers.
    _yankee_state_write "$state_file" \
        "version=2" \
        "owner_pid=$$" \
        "orig_pane_id=$orig_pane_id" \
        "orig_zoom_state=$orig_zoom_state" \
        "orig_session_name=$orig_session_name" \
        "orig_window_id=$orig_window_id" \
        "helper_session_name=$helper_session_name" \
        "helper_window_id=$helper_window_id" \
        "helper_pane_id=$helper_pane_id" \
        "wait_signal=$wait_signal" \
        "swap_performed=0" \
        "swapback_confirmed=0"

    # Keep helper pane alive even if the TUI process crashes.  Without this,
    # a crashed TUI destroys the helper pane (the one visible in the user's
    # window after swap), and the cleanup can no longer swap back — leaving the
    # user's original pane trapped in the helper session which then gets killed.
    tmux set-option -w -t "$helper_window_id" remain-on-exit on 2>/dev/null || true

    # Force helper window to match original pane dimensions (tmux 3.5a bug workaround).
    tmux resize-window -t "$helper_window_id" -x "$orig_pane_width" -y "$orig_pane_height" 2>/dev/null || true

    # Lock the HELPER pane BEFORE the swap.  After swap-pane the helper pane
    # sits in the user's window with a different pane ID than the one we
    # originally locked.  Without this second lock, WheelUpPane events on the
    # (now visible) helper pane find no lock and launch a second overlay.
    local helper_pane_lock_dir
    helper_pane_lock_dir="$(_yankee_lock_dir "$helper_pane_id")"
    yankee_lock_acquire "$helper_pane_lock_dir" 2>/dev/null || true

    # Set @yankee_busy on the helper pane too.  After swap, this pane is visible
    # in the user's window.  The atomic gate in the binding will reject scroll
    # events targeting it because @yankee_busy already exists.
    tmux set-option -pq -t "$helper_pane_id" @yankee_busy 1 2>/dev/null || true

    # Update trap to also release the helper pane lock and clear busy flags.
    trap '_yankee_cleanup_from_state "'"$state_file"'"; _yankee_clear_busy "'"$PANE_ID"'" "'"$helper_pane_id"'"; yankee_lock_release "'"$helper_pane_lock_dir"'"; yankee_lock_release "'"$pane_lock_dir"'"' EXIT INT TERM HUP

    # Swap helper into original pane position.
    # No pre-swap wait needed: the atomic @yankee_busy gate in the tmux binding
    # prevents reentry before the TUI sets alternate_on and mouse_any_flag.
    if ! tmux swap-pane -d -s "$orig_pane_id" -t "$helper_pane_id" -Z; then
        yankee_lock_release "$helper_pane_lock_dir"
        tmux display-message "Error: tmux-yankee swap-pane failed" 2>/dev/null || true
        return 0
    fi

    # Record that the swap was performed — cleanup needs this to decide
    # whether a fallback swap-back is appropriate.
    _yankee_state_write "$state_file" \
        "version=2" \
        "owner_pid=$$" \
        "orig_pane_id=$orig_pane_id" \
        "orig_zoom_state=$orig_zoom_state" \
        "orig_session_name=$orig_session_name" \
        "orig_window_id=$orig_window_id" \
        "helper_session_name=$helper_session_name" \
        "helper_window_id=$helper_window_id" \
        "helper_pane_id=$helper_pane_id" \
        "wait_signal=$wait_signal" \
        "swap_performed=1" \
        "swapback_confirmed=0"

    # Wait for helper's "swap-back done" signal.
    if wait_for_helper_completion "$wait_signal" "$helper_pane_id"; then
        # Update state: swap-back confirmed so cleanup won't force swap-back again.
        _yankee_state_write "$state_file" \
            "version=2" \
            "owner_pid=$$" \
            "orig_pane_id=$orig_pane_id" \
            "orig_zoom_state=$orig_zoom_state" \
            "orig_session_name=$orig_session_name" \
            "orig_window_id=$orig_window_id" \
            "helper_session_name=$helper_session_name" \
            "helper_window_id=$helper_window_id" \
            "helper_pane_id=$helper_pane_id" \
            "wait_signal=$wait_signal" \
            "swap_performed=1" \
            "swapback_confirmed=1"
    else
        tmux display-message "tmux-yankee: helper completion signal missing; forcing fallback cleanup" 2>/dev/null || true
    fi

    _yankee_cleanup_from_state "$state_file"
    _yankee_clear_busy "$PANE_ID" "$helper_pane_id"
    yankee_lock_release "$helper_pane_lock_dir"
    yankee_lock_release "$pane_lock_dir"
    trap - EXIT INT TERM HUP
}

launch_popup() {
    # Optional args: --borderless  → seamless, matches zoomed pane exactly
    #                (default)     → -w 90% -h 90% (standard popup with border)
    local popup_flags="-w 90% -h 90%"
    if [ "${1:-}" = "--borderless" ]; then
        # Use the actual pane dimensions (excludes status bar) and anchor at
        # top-left so the popup covers exactly the zoomed pane area without
        # hiding the tmux status bar.
        local _pw _ph
        _pw="$(tmux display-message -p -t "$PANE_ID" '#{pane_width}' 2>/dev/null)"
        _ph="$(tmux display-message -p -t "$PANE_ID" '#{pane_height}' 2>/dev/null)"
        popup_flags="-B -x 0 -y 0 -w ${_pw} -h ${_ph}"
    fi
    # -K (tmux 3.3+): enable tmux key binding processing inside the popup,
    # so prefix bindings (pane switching) and root-table bindings (Alt+hjkl
    # window switching, Alt+z zoom, etc.) still work while yankee is open.
    if tmux display-popup -K -E -w 1 -h 1 "true" >/dev/null 2>&1; then
        popup_flags="-K $popup_flags"
    fi
    local pane_lock_dir
    pane_lock_dir="$(_yankee_lock_dir "$PANE_ID")"
    if ! yankee_lock_acquire "$pane_lock_dir"; then return 0; fi
    trap '_yankee_clear_busy "'"$PANE_ID"'"; yankee_lock_release "'"$pane_lock_dir"'"' EXIT INT TERM HUP
    local yankee_args=()
    while IFS= read -r -d '' arg; do yankee_args+=("$arg"); done < <(build_yankee_args)
    # shellcheck disable=SC2086
    tmux display-popup -E $popup_flags \
        "${BIN_DIR}/tmux-yankee" "${yankee_args[@]}"
    _yankee_clear_busy "$PANE_ID"
    yankee_lock_release "$pane_lock_dir"
    trap - EXIT INT TERM HUP
}

launch_split() {
    local pane_lock_dir
    pane_lock_dir="$(_yankee_lock_dir "$PANE_ID")"
    if ! yankee_lock_acquire "$pane_lock_dir"; then return 0; fi
    trap '_yankee_clear_busy "'"$PANE_ID"'"; yankee_lock_release "'"$pane_lock_dir"'"' EXIT INT TERM HUP
    local yankee_args=()
    while IFS= read -r -d '' arg; do yankee_args+=("$arg"); done < <(build_yankee_args)
    tmux split-window -h \
        "${BIN_DIR}/tmux-yankee" "${yankee_args[@]}"
    _yankee_clear_busy "$PANE_ID"
    yankee_lock_release "$pane_lock_dir"
    trap - EXIT INT TERM HUP
}

# --- Dispatch to display mode (sweep runs inside overlay after lock) -------

# When the pane is zoomed, swap-pane (used by overlay mode) triggers an internal
# unzoom→swap→rezoom cycle that causes tmux to reflow the entire scrollback
# buffer twice.  On panes with long history (50K+ lines) this produces a visible
# rapid-scroll artifact.  Popup mode avoids this entirely because it overlays a
# temporary window without moving or resizing any existing panes.
_yankee_zoom_flag="$(tmux display-message -p -t "$PANE_ID" '#{window_zoomed_flag}' 2>/dev/null || true)"

case "$DISPLAY_MODE" in
    overlay)
        if [ "$_yankee_zoom_flag" = "1" ] && popup_supported; then
            launch_popup --borderless
        else
            launch_overlay
        fi
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
