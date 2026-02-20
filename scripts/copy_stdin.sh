#!/usr/bin/env bash

# Clipboard adapter for numbered mode
# Reads text from stdin and delegates to tmux-yank helpers

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# shellcheck source=scripts/helpers.sh
source "${SCRIPT_DIR}/helpers.sh"

# Get clipboard copy command from tmux-yank helpers
copy_command=$(clipboard_copy_command)

# Validate command exists
if [ -z "$copy_command" ]; then
    display_message "tmux-yankee: No clipboard command available. Text saved to tmux buffer only."
    exit 1
fi

# Copy stdin to clipboard with error handling.
# eval is needed because $copy_command may contain shell operators (e.g., "cat | clip.exe" on WSL).
if ! eval "$copy_command"; then
    display_message "tmux-yankee: Clipboard copy failed. Text saved to tmux buffer only."
    exit 1
fi
