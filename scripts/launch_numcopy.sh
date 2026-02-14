#!/usr/bin/env bash

# Launcher for numbered copy mode
# Gathers tmux context and launches Go TUI binary

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BIN_DIR="${SCRIPT_DIR}/../bin"

# Get current pane ID
PANE_ID=$(tmux display-message -p '#{pane_id}')

# Get user configuration (or use defaults)
MODE=$(tmux show-option -gqv @yankee_mode)
MODE="${MODE:-hybrid}"

# Check if binary exists
if [ ! -f "${BIN_DIR}/tmux-yankee" ]; then
    tmux display-message "Error: tmux-yankee binary not found. Run 'make build' first."
    exit 1
fi

# Use popup on tmux 3.2+, fallback to split on 3.1
if tmux display-popup -h 1 -w 1 "true" 2>/dev/null; then
    tmux display-popup -E -w 90% -h 90% \
        "${BIN_DIR}/tmux-yankee" --pane "$PANE_ID" --mode "$MODE"
else
    tmux split-window -h \
        "${BIN_DIR}/tmux-yankee" --pane "$PANE_ID" --mode "$MODE"
fi
