#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Source dependencies
source "$SCRIPT_DIR/utils.sh"
source "$SCRIPT_DIR/config.sh"
source "$SCRIPT_DIR/renderer.sh"
source "$SCRIPT_DIR/state_cleanup.sh"

# --- Public entry point ---
# Called by plugin.tmux via run-shell. No arguments.
main() {
    # Signature: main()
    # Returns: 0 on success, 1 on error
    # Side effects: Respawns pane with numbered content, waits, restores

    local source_pane
    local pane_width pane_height
    local history_size scroll_position copy_cursor_y
    local base_absolute cursor_absolute
    local mode toggle_key
    local style_absolute style_relative style_cursor
    local captured_content rendered_content
    local wait_channel
    local gutter_width

    # Step 1: Identify source pane
    source_pane=$(get_active_pane_id)

    # Step 2: Read configuration
    mode=$(get_mode)
    toggle_key=$(get_toggle_key)
    style_absolute=$(get_style_absolute)
    style_relative=$(get_style_relative)
    style_cursor=$(get_style_cursor)

    # Step 3: Capture pane state (format variables)
    pane_width=$(get_pane_format "$source_pane" "#{pane_width}")
    pane_height=$(get_pane_format "$source_pane" "#{pane_height}")

    # Narrow pane guard
    local min_width=15
    if [[ $pane_width -lt $min_width ]]; then
        log_error "Pane too narrow for line numbers (min: ${min_width} cols)"
        # Attempt to resize the pane to minimum width so it remains functional
        tmux resize-pane -t "$source_pane" -x "$min_width" 2>/dev/null || true
        return 0
    fi

    # If in copy-mode, read scroll position and cursor, then exit
    local pane_mode
    pane_mode=$(get_pane_format "$source_pane" "#{pane_mode}")
    if [[ "$pane_mode" == "copy-mode" ]]; then
        history_size=$(get_pane_format "$source_pane" "#{history_size}")
        scroll_position=$(get_pane_format "$source_pane" "#{scroll_position}")
        copy_cursor_y=$(get_pane_format "$source_pane" "#{copy_cursor_y}")
        # Exit copy-mode before capture
        tmux send-keys -t "$source_pane" -X cancel 2>/dev/null || true
        sleep 0.05
    else
        # Enter copy-mode to read format variables, then exit
        tmux copy-mode -t "$source_pane"
        sleep 0.05
        history_size=$(get_pane_format "$source_pane" "#{history_size}")
        scroll_position=$(get_pane_format "$source_pane" "#{scroll_position}")
        copy_cursor_y=$(get_pane_format "$source_pane" "#{copy_cursor_y}")
        tmux send-keys -t "$source_pane" -X cancel 2>/dev/null || true
        sleep 0.05
    fi

    # Default to 0 if empty
    history_size="${history_size:-0}"
    scroll_position="${scroll_position:-0}"
    copy_cursor_y="${copy_cursor_y:-0}"

    # Step 4: Compute line number bases
    base_absolute=$(( history_size - scroll_position ))
    cursor_absolute=$(( base_absolute + copy_cursor_y ))

    # Step 5: Capture pane content (viewport only for performance)
    captured_content=$(tmux capture-pane -p -t "$source_pane" -S 0 -E "$((pane_height - 1))")

    # Step 6: Render with line numbers
    gutter_width=$(calculate_gutter_width "$history_size")
    rendered_content=$(render_line_numbers \
        "$captured_content" \
        "$base_absolute" \
        "$cursor_absolute" \
        "$pane_width" \
        "$gutter_width" \
        "$mode" \
        "$style_absolute" \
        "$style_relative" \
        "$style_cursor"
    )

    # Step 7: Persist state for toggle re-render
    local state_dir="/tmp/linenumbers-state-$$"
    mkdir -p "$state_dir"
    printf '%s\n' "$captured_content" > "$state_dir/content"
    printf '%d\n' "$base_absolute"    > "$state_dir/base_absolute"
    printf '%d\n' "$cursor_absolute"  > "$state_dir/cursor_absolute"
    printf '%d\n' "$pane_width"       > "$state_dir/pane_width"
    printf '%d\n' "$gutter_width"     > "$state_dir/gutter_width"
    printf '%d\n' "$pane_height"      > "$state_dir/pane_height"
    printf '%s\n' "$style_absolute"   > "$state_dir/style_absolute"
    printf '%s\n' "$style_relative"   > "$state_dir/style_relative"
    printf '%s\n' "$style_cursor"     > "$state_dir/style_cursor"

    # Step 8: Write rendered content to temp file and respawn source pane
    # This replaces the pane's content with numbered view while keeping the same pane ID
    wait_channel="linenumbers-$$"
    local tmpfile
    tmpfile=$(mktemp /tmp/linenumbers.XXXXXX)
    printf '%s\n' "$rendered_content" > "$tmpfile"

    tmux respawn-pane -k -t "$source_pane" \
        "cat '$tmpfile'; rm -f '$tmpfile'; while :; do sleep 86400; done"
    sleep 0.1

    # Step 9: Set up cleanup trap
    # With respawn approach, temp_pane is the same as source_pane
    setup_cleanup_trap "$source_pane" "" "$wait_channel" "0"

    # Step 10: Bind toggle key
    bind_toggle_key "$toggle_key" "$source_pane" "$source_pane" "$wait_channel"

    # Step 11: Bind exit keys (q, Escape send wait-for signal)
    bind_exit_keys "$source_pane" "$wait_channel"

    # Step 12: Set up copy filtering
    bind_copy_filter "$source_pane" "$gutter_width"

    # Step 13: Enter copy-mode for navigation
    tmux copy-mode -t "$source_pane"

    # Step 14: Wait for user to exit
    # Use background + wait so bash can handle SIGTERM for cleanup
    tmux wait-for "$wait_channel" &
    wait $!

    # Step 15: Cleanup (also happens via trap)
    cleanup "$source_pane" ""
}

# --- Helper: bind toggle key ---
bind_toggle_key() {
    # Signature: bind_toggle_key(key, source_pane, temp_pane, wait_channel)
    local key="$1"
    local source_pane="$2"
    local temp_pane="$3"
    local wait_channel="$4"

    # When L is pressed in copy-mode-vi on the temp pane,
    # cycle @linenumbers-mode and re-render
    tmux bind-key -T copy-mode-vi "$key" \
        run-shell "$SCRIPT_DIR/toggle_and_rerender.sh '$source_pane' '$temp_pane' '$wait_channel'"
}

# --- Helper: bind exit keys ---
bind_exit_keys() {
    # Signature: bind_exit_keys(temp_pane, wait_channel)
    local temp_pane="$1"
    local wait_channel="$2"

    # q and Escape signal the wait-for channel to unblock
    tmux bind-key -T copy-mode-vi q \
        send-keys -X cancel '\;' \
        run-shell "tmux wait-for -S '$wait_channel'"

    tmux bind-key -T copy-mode-vi Escape \
        send-keys -X cancel '\;' \
        run-shell "tmux wait-for -S '$wait_channel'"
}

# --- Helper: bind copy filter ---
bind_copy_filter() {
    # Signature: bind_copy_filter(temp_pane, gutter_width)
    local temp_pane="$1"
    local gutter_width="$2"

    # Override copy-selection to filter through copy_filter.sh
    tmux bind-key -T copy-mode-vi Enter \
        send-keys -X copy-pipe-and-cancel \
        "$SCRIPT_DIR/copy_filter.sh $gutter_width"

    tmux bind-key -T copy-mode-vi y \
        send-keys -X copy-pipe-and-cancel \
        "$SCRIPT_DIR/copy_filter.sh $gutter_width"
}

main "$@"
