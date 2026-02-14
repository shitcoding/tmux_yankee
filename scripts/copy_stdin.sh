#!/usr/bin/env bash

# Clipboard adapter for numbered mode
# Reads text from stdin and delegates to tmux-yank helpers

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# shellcheck source=scripts/helpers.sh
source "${SCRIPT_DIR}/helpers.sh"

# Get clipboard copy command from tmux-yank helpers
copy_command=$(clipboard_copy_command)

# Copy stdin to clipboard
$copy_command
