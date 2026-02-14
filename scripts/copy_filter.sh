#!/usr/bin/env bash
set -euo pipefail

# copy_filter.sh - Strip line number prefixes from copied text
#
# Usage: Used as the pipe command in copy-pipe-and-cancel
#   send-keys -X copy-pipe-and-cancel "scripts/copy_filter.sh <gutter_width>"
#
# Input:  Text on stdin (from tmux copy-pipe) with line number prefixes
# Output: Clean text without line numbers, sent to tmux buffer and system clipboard

GUTTER_WIDTH="${1:-6}"

filter_line_numbers() {
    # Signature: filter_line_numbers(gutter_width)
    # Input:  stdin - text with line number prefixes
    # Output: stdout - text with prefixes stripped
    #
    # Line format: "  42 | content here"
    # After strip: "content here"
    #
    # The gutter is fixed-width, so we can use a simple cut:
    # Remove first $gutter_width characters from each line.

    local gutter_width="$1"

    while IFS= read -r line || [[ -n "$line" ]]; do
        # Strip the gutter prefix (number + " | ")
        if [[ ${#line} -gt $gutter_width ]]; then
            printf '%s\n' "${line:$gutter_width}"
        else
            # Line is shorter than or equal to gutter (empty line)
            printf '\n'
        fi
    done
}

# Only execute main body when run directly (not when sourced)
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    # Read from stdin, filter, and:
    # 1. Set tmux buffer (so paste works normally)
    # 2. Copy to system clipboard if available
    filtered=$(filter_line_numbers "$GUTTER_WIDTH")

    # Set tmux paste buffer
    tmux set-buffer -- "$filtered"

    # Attempt system clipboard copy (best effort)
    if command -v pbcopy &>/dev/null; then
        printf '%s' "$filtered" | pbcopy
    elif command -v xclip &>/dev/null; then
        printf '%s' "$filtered" | xclip -selection clipboard
    elif command -v xsel &>/dev/null; then
        printf '%s' "$filtered" | xsel --clipboard --input
    elif command -v wl-copy &>/dev/null; then
        printf '%s' "$filtered" | wl-copy
    fi
fi
