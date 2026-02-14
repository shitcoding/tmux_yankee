# plugin.tmux - TPM entry point for tmux-copymode-linenumbers
#
# Loaded via: tmux source-file plugin.tmux
#           or: run-shell plugin.tmux (TPM)

# --- Option defaults ---
# Conditionally set each default only if not already configured
run-shell 'val=$(tmux show-option -gqv "@linenumbers-mode"); [ -z "$val" ] && tmux set-option -g "@linenumbers-mode" "hybrid"; true'
run-shell 'val=$(tmux show-option -gqv "@linenumbers-style-absolute"); [ -z "$val" ] && tmux set-option -g "@linenumbers-style-absolute" "fg=white"; true'
run-shell 'val=$(tmux show-option -gqv "@linenumbers-style-relative"); [ -z "$val" ] && tmux set-option -g "@linenumbers-style-relative" "fg=yellow"; true'
run-shell 'val=$(tmux show-option -gqv "@linenumbers-style-cursor"); [ -z "$val" ] && tmux set-option -g "@linenumbers-style-cursor" "fg=green,bold"; true'
run-shell 'val=$(tmux show-option -gqv "@linenumbers-toggle-key"); [ -z "$val" ] && tmux set-option -g "@linenumbers-toggle-key" "L"; true'
run-shell 'val=$(tmux show-option -gqv "@linenumbers-enable-binding"); [ -z "$val" ] && tmux set-option -g "@linenumbers-enable-binding" "off"; true'
run-shell 'val=$(tmux show-option -gqv "@linenumbers-custom-key"); [ -z "$val" ] && tmux set-option -g "@linenumbers-custom-key" "N"; true'

# --- Discover and store plugin directory ---
run-shell 'plugin_dir=$(tmux show-option -gqv "@linenumbers-plugin-dir"); if [ -z "$plugin_dir" ] || [ ! -f "$plugin_dir/scripts/line_numbers.sh" ]; then plugin_dir=""; for d in "$HOME/.tmux/plugins/tmux-copymode-linenumbers" "$HOME/.tmux/plugins/tmux-yankee" "$HOME/coding/pet_projects/tmux_yankee"; do [ -f "$d/scripts/line_numbers.sh" ] && plugin_dir="$d" && break; done; [ -n "$plugin_dir" ] && tmux set-option -g "@linenumbers-plugin-dir" "$plugin_dir"; fi; true'

# --- Ensure standard copy-mode bindings exist ---
# Guarantee prefix+[ enters copy-mode (standard tmux convention)
run-shell 'tmux bind-key -T prefix "[" copy-mode; true'

# --- Keybindings ---
# Only bind if user has opted in
run-shell 'enable=$(tmux show-option -gqv "@linenumbers-enable-binding"); custom_key=$(tmux show-option -gqv "@linenumbers-custom-key"); custom_key="${custom_key:-N}"; if [ "$enable" = "on" ]; then plugin_dir=$(tmux show-option -gqv "@linenumbers-plugin-dir"); if [ -n "$plugin_dir" ] && [ -f "$plugin_dir/scripts/line_numbers.sh" ]; then tmux bind-key "$custom_key" run-shell "$plugin_dir/scripts/line_numbers.sh"; fi; else tmux unbind-key "$custom_key" 2>/dev/null; tmux unbind-key -T copy-mode-vi "$custom_key" 2>/dev/null; fi; true'
