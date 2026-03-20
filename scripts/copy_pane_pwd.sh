#!/usr/bin/env bash
set -euo pipefail

CURRENT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
HELPERS_DIR="$CURRENT_DIR"

# shellcheck source=scripts/helpers.sh
source "${HELPERS_DIR}/helpers.sh"

pane_current_path() {
    tmux display -p -F "#{pane_current_path}"
}

display_notice() {
    display_message 'PWD copied to clipboard!'
}

main() {
    local copy_command
    local payload
    # shellcheck disable=SC2119
    copy_command="$(clipboard_copy_command)"
    if [[ -z "$copy_command" ]]; then
        echo "copy_pane_pwd: no clipboard command found" >&2
        return 1
    fi
    payload="$(pane_current_path | tr -d '\n')"
    # eval is needed because $copy_command may contain shell operators (e.g., pipelines on WSL).
    if ! echo "$payload" | eval "$copy_command"; then
        echo "copy_pane_pwd: clipboard copy failed" >&2
        return 1
    fi
    tmux set-buffer -- "$payload"
    display_notice
}
main
