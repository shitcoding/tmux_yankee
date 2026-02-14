#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/config.sh"
source "$SCRIPT_DIR/renderer.sh"
source "$SCRIPT_DIR/utils.sh"

# Arguments passed from the keybinding
# shellcheck disable=SC2034
SOURCE_PANE="$1"
TEMP_PANE="$2"
# shellcheck disable=SC2034
WAIT_CHANNEL="$3"

# Cycle mode
new_mode=$(cycle_mode)

# Display brief mode indicator
tmux display-message "Line numbers: $new_mode"

# Re-render using persisted state
# Find the state directory. The PID used when creating it was the PID of
# the line_numbers.sh process. We scan for any matching state dir since we
# don't have access to the original PID directly.
STATE_DIR=""
for d in /tmp/linenumbers-state-*; do
    if [[ -d "$d" ]] && [[ -f "$d/content" ]]; then
        STATE_DIR="$d"
        break
    fi
done

if [[ -z "$STATE_DIR" ]] || [[ ! -f "$STATE_DIR/content" ]]; then
    # Cannot re-render without state; just show the mode change message
    exit 0
fi

# Read persisted state
captured_content=$(cat "$STATE_DIR/content")
base_absolute=$(cat "$STATE_DIR/base_absolute")
cursor_absolute=$(cat "$STATE_DIR/cursor_absolute")
pane_width=$(cat "$STATE_DIR/pane_width")
gutter_width=$(cat "$STATE_DIR/gutter_width")
style_absolute=$(cat "$STATE_DIR/style_absolute")
style_relative=$(cat "$STATE_DIR/style_relative")
style_cursor=$(cat "$STATE_DIR/style_cursor")

# Re-render with new mode
rendered_content=$(render_line_numbers \
    "$captured_content" \
    "$base_absolute" \
    "$cursor_absolute" \
    "$pane_width" \
    "$gutter_width" \
    "$new_mode" \
    "$style_absolute" \
    "$style_relative" \
    "$style_cursor"
)

# Write to a temp file and send to the temp pane
tmpfile=$(mktemp /tmp/linenumbers-rerender.XXXXXX)
printf '%s\n' "$rendered_content" > "$tmpfile"

# Cancel copy-mode before sending content, then re-enter
tmux send-keys -t "$TEMP_PANE" -X cancel 2>/dev/null || true
sleep 0.05

# Send the content to the temp pane by respawning it
tmux respawn-pane -t "$TEMP_PANE" -k "cat '$tmpfile'; rm -f '$tmpfile'; while :; do sleep 86400; done" 2>/dev/null || true
sleep 0.1

# Re-enter copy-mode
tmux copy-mode -t "$TEMP_PANE" 2>/dev/null || true
