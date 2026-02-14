#!/usr/bin/env bash
# init.sh - Plugin initialization helper
# Called by plugin.tmux to set up keybindings

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

enable_binding=$(tmux show-option -gqv "@linenumbers-enable-binding")
custom_key=$(tmux show-option -gqv "@linenumbers-custom-key")
custom_key="${custom_key:-N}"

if [[ "$enable_binding" == "on" ]]; then
    # Bind prefix + custom_key to launch line numbers view
    tmux bind-key "$custom_key" run-shell "$SCRIPT_DIR/line_numbers.sh"

    # Also provide a copy-mode-vi binding so it can be triggered from
    # within copy-mode (re-enter with line numbers)
    tmux bind-key -T copy-mode-vi "$custom_key" \
        send-keys -X cancel \; \
        run-shell "$SCRIPT_DIR/line_numbers.sh"
fi
