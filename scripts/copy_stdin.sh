#!/usr/bin/env bash

# Clipboard adapter for yankee
# Reads text from stdin and copies to system clipboard.
# Self-contained — no external dependencies.

set -euo pipefail

# Detect the appropriate system clipboard command.
# Returns the command string on stdout; empty if none found.
detect_clipboard_command() {
    if command -v pbcopy >/dev/null 2>&1; then
        echo "pbcopy"
    elif command -v wl-copy >/dev/null 2>&1; then
        echo "wl-copy"
    elif command -v xsel >/dev/null 2>&1; then
        echo "xsel -i --clipboard"
    elif command -v xclip >/dev/null 2>&1; then
        echo "xclip -selection clipboard"
    elif command -v clip.exe >/dev/null 2>&1; then
        # WSL
        echo "cat | clip.exe"
    elif command -v putclip >/dev/null 2>&1; then
        # Cygwin
        echo "putclip"
    fi
}

copy_command=$(detect_clipboard_command)

if [ -z "$copy_command" ]; then
    tmux display-message "tmux-yankee: No clipboard command found. Text saved to tmux buffer only." 2>/dev/null || true
    exit 1
fi

# eval is needed because some commands contain shell operators (e.g., "cat | clip.exe" on WSL).
if ! eval "$copy_command"; then
    tmux display-message "tmux-yankee: Clipboard copy failed. Text saved to tmux buffer only." 2>/dev/null || true
    exit 1
fi
