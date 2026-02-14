# tmux-copymode-linenumbers

Line numbers for tmux copy-mode. Displays absolute, relative, or hybrid line numbers in a snapshot view, similar to Vim's `set number` / `set relativenumber`.

## Features

- **Three display modes:** absolute, relative, hybrid (like Vim)
- **Mode cycling:** press `L` (configurable) to toggle between modes while viewing
- **Customizable styles:** configure colors for absolute, relative, and cursor line numbers
- **Copy filtering:** line number gutter is automatically stripped when you yank text
- **Opt-in keybinding:** does not override native `[` -- uses a separate key (`prefix + N` by default)
- **Zoom-safe:** works correctly in zoomed panes
- **Clean lifecycle:** trap-based cleanup ensures no orphaned state

## Requirements

- tmux 3.1+ (copy-mode snapshot support)
- copy-mode-vi enabled (`set -g mode-keys vi`)
- Bash 4+

## Installation

### With TPM (Tmux Plugin Manager)

Add to your `~/.tmux.conf`:

```tmux
set -g @plugin 'your-username/tmux-copymode-linenumbers'
```

Then press `prefix + I` to install.

### Manual

```bash
git clone https://github.com/your-username/tmux-copymode-linenumbers.git ~/.tmux/plugins/tmux-copymode-linenumbers
```

Add to `~/.tmux.conf`:

```tmux
run-shell ~/.tmux/plugins/tmux-copymode-linenumbers/plugin.tmux
```

## Configuration

All options use the `@linenumbers-` prefix. Add these to `~/.tmux.conf` **before** the plugin is loaded.

### Enable the keybinding (required)

By default, the plugin does not bind any keys. You must opt in:

```tmux
set -g @linenumbers-enable-binding "on"
```

### Options

| Option | Default | Description |
|--------|---------|-------------|
| `@linenumbers-enable-binding` | `off` | Set to `on` to enable the custom keybinding |
| `@linenumbers-custom-key` | `N` | Key to trigger line numbers view (with prefix) |
| `@linenumbers-mode` | `hybrid` | Display mode: `absolute`, `relative`, or `hybrid` |
| `@linenumbers-toggle-key` | `L` | Key to cycle modes while in line numbers view |
| `@linenumbers-style-absolute` | `fg=white` | tmux style string for absolute line numbers |
| `@linenumbers-style-relative` | `fg=yellow` | tmux style string for relative line numbers |
| `@linenumbers-style-cursor` | `fg=green,bold` | tmux style string for the cursor line |

### Example configuration

```tmux
# Enable the plugin keybinding
set -g @linenumbers-enable-binding "on"

# Use prefix + n instead of prefix + N
set -g @linenumbers-custom-key "n"

# Start in absolute mode
set -g @linenumbers-mode "absolute"

# Customize colors
set -g @linenumbers-style-absolute "fg=cyan"
set -g @linenumbers-style-relative "fg=magenta"
set -g @linenumbers-style-cursor "fg=green,bold"

# Cycle modes with M instead of L
set -g @linenumbers-toggle-key "M"
```

## Usage

1. Press `prefix + N` (or your configured key) to enter line numbers view
2. Navigate using standard copy-mode-vi keys (`j`, `k`, `C-u`, `C-d`, etc.)
3. Press `L` to cycle between display modes (absolute -> relative -> hybrid)
4. Select and yank text normally (`v`, `y`, `Enter`) -- the gutter is stripped automatically
5. Press `q` or `Escape` to exit and return to your shell

## How It Works

The plugin uses a **capture-and-replace** pattern (same technique used by tmux-fingers, tmux-thumbs):

1. Captures the current viewport content and scroll position
2. Renders line numbers into the captured text
3. Replaces the pane content using `respawn-pane` (keeps the same pane ID)
4. Enters copy-mode for navigation
5. On exit, restores the original shell via `respawn-pane`

This is a **snapshot view** -- it shows the content as it was when you triggered the command. It does not scroll in real-time.

## Architecture

```
plugin.tmux              TPM entry point, option defaults, keybinding setup
scripts/
  config.sh              Read @linenumbers-* tmux options
  utils.sh               Shared helpers (pane queries, logging)
  renderer.sh            Pure line number renderer (no tmux calls)
  line_numbers.sh        Core orchestrator (capture -> render -> respawn -> wait)
  state_cleanup.sh       Trap-based cleanup and keybinding restoration
  toggle_and_rerender.sh Mode cycling (L key handler)
  copy_filter.sh         Strip gutter from yanked text
  init.sh                Keybinding setup helper
```

## Known Limitations

- **Snapshot view:** Content is static; you cannot scroll to see new content while line numbers are active
- **CWD:** After exiting, the shell restarts in `$HOME` (the working directory from before is not preserved). Use `cd -` or set up shell initialization to mitigate this
- **Shell history:** In-memory command history from the current session is lost when line numbers view is exited (inherent trade-off of the respawn-pane approach)
- **Minimum width:** Pane must be at least 15 columns wide

## License

MIT
