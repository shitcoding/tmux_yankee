# tmux-copymode-linenumbers

Line numbers for tmux copy-mode. Displays absolute, relative, or hybrid line numbers in a snapshot view, similar to Vim's `set number` / `set relativenumber`.

## Features

- **Three line number modes:** absolute, relative, hybrid (like Vim)
- **Three display modes:** overlay (default), popup, or split window
- **Overlay mode:** Full-pane coverage like tmux copy-mode (requires tmux 3.2+)
- **Vim motions:** Full vim-style navigation (hjkl, w/b/e, gg/G, ^/$, Ctrl-u/d, zt/zz/zb)
- **Visual selection:** Character-wise (v) and line-wise (V) visual modes
- **Mode cycling:** Press `L` to toggle line number modes while viewing
- **Color preservation:** Original terminal colors are maintained
- **Copy filtering:** Line number gutter is automatically stripped when you yank text
- **Clean keybinding:** Uses separate key (`prefix + N` by default)
- **Zoom-safe:** Works correctly in zoomed panes

## Requirements

- tmux 3.2+ (recommended for overlay/popup modes)
- tmux 3.1+ (minimum, split mode only)
- Bash 4+
- Go 1.19+ (for building the binary)

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

All options use the `@yankee_` prefix. Add these to `~/.tmux.conf` **before** the plugin is loaded.

### Display Mode

Control how tmux-yankee appears:

```tmux
set -g @yankee_display_mode "overlay"  # overlay (default), popup, or split
```

**Display modes:**
- **overlay** (default): Covers only the active pane (like tmux-fingers/tmux-thumbs). Uses swap-pane to preserve shell history and pane contents. Works on tmux 3.1+
- **popup**: Centered popup window (90% width/height). Requires tmux 3.2+
- **split**: Horizontal split window. Works on tmux 3.1+ (automatic fallback for older versions)

### Line Number Mode

```tmux
set -g @yankee_mode "hybrid"  # absolute, relative, or hybrid
```

### Custom Keybinding

```tmux
set -g @yankee_key "N"  # Key to trigger (with prefix)
```

### Options Reference

| Option | Default | Description |
|--------|---------|-------------|
| `@yankee_display_mode` | `overlay` | How to display: `overlay`, `popup`, or `split` |
| `@yankee_mode` | `hybrid` | Line number mode: `absolute`, `relative`, or `hybrid` |
| `@yankee_key` | `N` | Key to trigger line numbers view (with prefix) |

### Example configuration

```tmux
# Display mode (overlay covers pane like tmux copy-mode)
set -g @yankee_display_mode "overlay"

# Line number mode (hybrid shows relative numbers with absolute on cursor line)
set -g @yankee_mode "hybrid"

# Use prefix + n instead of prefix + N
set -g @yankee_key "n"
```

### Version Requirements

| Display Mode | Minimum tmux Version | Notes |
|--------------|---------------------|-------|
| `overlay` | 3.1+ | Uses swap-pane strategy, preserves shell history |
| `popup` | 3.2+ | Uses centered popup window |
| `split` | 3.1+ | Uses split window (always works) |

If you request `overlay` or `popup` on tmux 3.1, the plugin will automatically fall back to `split` mode with an informative message.

## Usage

1. Press `prefix + N` (or your configured key) to enter line numbers view
2. Navigate using vim-style motion keys
3. Press `L` to cycle between display modes (absolute -> relative -> hybrid)
4. Select and yank text using visual mode
5. Press `q` or `Escape` to exit and return to your shell

### Vim-Style Keybindings

#### Motion Keys

| Key | Motion | Description |
|-----|--------|-------------|
| `j` | Down | Move cursor down one line |
| `k` | Up | Move cursor up one line |
| `h` | Left | Move cursor left one character |
| `l` | Right | Move cursor right one character |
| `w` | Word forward | Jump to start of next word |
| `b` | Word backward | Jump to start of previous word |
| `e` | Word end | Jump to end of current/next word |
| `E` | WORD end | Jump to end of whitespace-separated WORD |
| `B` | WORD backward | Jump to start of previous whitespace-separated WORD |
| `0` | Line start | Jump to beginning of line |
| `^` | First non-blank | Jump to first non-whitespace character |
| `$` | Line end | Jump to end of line |
| `gg` | First line | Jump to first line of document |
| `G` | Last line | Jump to last line of document |
| `Ctrl-u` | Half page up | Scroll up half a page |
| `Ctrl-d` | Half page down | Scroll down half a page |
| `zt` | Viewport top | Position current line at top of viewport |
| `zz` | Viewport center | Position current line at center of viewport |
| `zb` | Viewport bottom | Position current line at bottom of viewport |

#### Visual Mode & Yanking

| Key | Action | Description |
|-----|--------|-------------|
| `v` | Visual char | Enter character-wise visual mode |
| `V` | Visual line | Enter line-wise visual mode |
| `y` | Yank | Yank selected text and exit |
| `Enter` | Yank | Yank selected text and exit (same as `y`) |
| `Escape` | Exit visual | Return to normal mode |

#### Other Keys

| Key | Action | Description |
|-----|--------|-------------|
| `L` | Toggle mode | Cycle through line number modes |
| `q` | Quit | Exit line numbers view |

#### Count Prefixes

All motion keys support count prefixes (like vim):
- `5j` - Move down 5 lines
- `3w` - Jump forward 3 words
- `10G` - Jump to line 10
- `2$` - Jump to end of next line

## How It Works

The plugin uses a **Go TUI** that renders line numbers and handles vim-style navigation:

1. Launcher script (`scripts/launch_yankee.sh`) gathers pane context
2. Depending on `@yankee_display_mode`:
   - **overlay**: Creates helper window with TUI, uses `swap-pane` to place it in active pane position. Swaps back on exit to preserve shell state (like tmux-fingers/tmux-thumbs)
   - **popup**: Launches centered popup (90% width/height)
   - **split**: Creates horizontal split window
3. Go binary (`bin/tmux-yankee`) captures pane content and renders realtime TUI
4. User navigates with vim motions, selects text with visual mode
5. On yank, text is copied to clipboard and tmux buffer (line numbers stripped)
6. TUI exits, original pane restored with shell history and contents intact (overlay mode)

The TUI shows a **snapshot** of pane content at launch time. Colors and formatting are preserved via ANSI code parsing.

## Architecture

```
yank.tmux                   TPM entry point, keybinding setup
scripts/
  launch_yankee.sh         Display mode dispatcher and launcher
  helpers.sh                Vendored tmux-yank clipboard helpers
  copy_stdin.sh             Clipboard copy wrapper
  copy_line.sh              Vendored tmux-yank line copy
  copy_pane_pwd.sh          Vendored tmux-yank pwd copy
cmd/tmux-yankee/
  main.go                   Go binary entry point
internal/
  ui/                       TUI rendering and event loop
  input/                    Vim-style input parser
  motion/                   Vim motion handlers
  selection/                Visual mode selection logic
  linenums/                 Line number formatting (absolute/relative/hybrid)
  tmux/                     Tmux client wrapper
```

## Known Limitations

- **Snapshot view:** Content is captured at launch time; live scrolling is not supported
- **Overlay mode:** Uses swap-pane to cover active pane; shell process and history are preserved
- **Popup mode:** Requires tmux 3.2+ (auto-falls back to split on tmux 3.1)

## License

MIT
