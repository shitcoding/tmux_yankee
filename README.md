# tmux-yankee

Vim inside tmux. Kind of.

It started as "I just want line numbers in tmux yank mode" and spiraled into rebuilding half of Vim/Neovim and flash.nvim as a tmux plugin. No regrets. YOLO!

![tmux-yankee workflow](assets/hero-workflow.gif)

tmux-yankee captures your pane content into a Go TUI with line numbers, vim motions, visual selection, incremental search, flash navigation, text objects, and block select. You navigate with the same muscle memory as Vim, yank what you need, and it goes straight to your clipboard. The line number gutter is automatically stripped from yanked text.

## How It Actually Works

tmux doesn't let you draw arbitrary UI on top of a running pane. So yankee uses a trick borrowed from [tmux-fingers](https://github.com/Morantron/tmux-fingers): it creates a **temporary helper session**, runs the TUI there, and `swap-pane` puts it where your original pane was. When you quit, it swaps back. Your shell process, history, environment variables, working directory -- everything is exactly where you left it. 

Three display modes are available:
- **Overlay** (default) -- the swap-pane trick described above. Works on tmux 3.1+.
- **Popup** -- centered floating window. Requires tmux 3.2+.
- **Split** -- boring horizontal split. Always works.

## Installation

### With [TPM](https://github.com/tmux-plugins/tpm)

Add to `~/.tmux.conf`:

```tmux
set -g @plugin 'shitcoding/tmux-yankee'
```

Press `prefix + I` to install. The Go binary is downloaded automatically from the latest GitHub release -- no build tools needed.

### Manual

```bash
git clone https://github.com/shitcoding/tmux-yankee ~/.tmux/plugins/tmux-yankee
```

Add to `~/.tmux.conf`:

```tmux
run-shell ~/.tmux/plugins/tmux-yankee/yank.tmux
```

The binary will be downloaded on first run. To build from source instead:

```bash
cd ~/.tmux/plugins/tmux-yankee && make build
```

## Requirements

- tmux 3.1+ (3.2+ recommended for popup mode)
- Bash 4+
- `curl` (for automatic binary download)
- Go 1.19+ (only if building from source)

## Quick Start

1. Press `prefix + N` to launch yankee
2. Navigate with vim motions (`j`/`k`, `w`/`b`, `gg`/`G`, `/pattern`)
3. Press `v` for visual select, `V` for line select, `Ctrl-v` for block select
4. Press `y` to yank and exit
5. Press `q` to quit without yanking

## Features

### Search and Navigation

Incremental regex search with match highlighting. `n`/`N` to cycle matches, `*`/`#` to search the word under cursor. All the vim motion keys you'd expect: `w`/`b`/`e`, `gg`/`G`, `{`/`}`, `H`/`M`/`L`, `Ctrl-u`/`Ctrl-d`, and count prefixes (`5j`, `10G`, `3w`).

![Search and navigation](assets/search-navigation.gif)

### Flash Jump

Inspired by [flash.nvim](https://github.com/folke/flash.nvim). Press `s`, type a pattern, and labeled jump targets appear on every match. Press the label key to teleport there instantly. Works in normal mode (jump) and visual mode (extend selection to target).

![Flash jump](assets/flash-jump.gif)

### Flash + Visual Select

Enter visual mode with `v`, then use flash (`s`) to extend your selection to a distant target. Chain multiple flash jumps to select exactly the range you need without scrolling.

![Flash visual selection](assets/flash-visual.gif)

### Text Objects

Vim text objects for selecting inside/around quotes, brackets, braces, parentheses, words, and paragraphs. `vi"` to select inside double quotes, `va{` to select around curly braces, `iw` for inner word, etc.

![Text objects](assets/text-objects.gif)

### Block Select

Visual block mode (`Ctrl-v`) for column selection. Select rectangular regions across multiple lines -- useful for grabbing a specific column from tabular output like `ps aux`.

![Block selection](assets/block-select.gif)

### Line Number Modes

Three modes just like Vim: absolute (`set number`), relative (`set relativenumber`), and hybrid (`set number relativenumber`). Toggle with `Alt+Shift+L`.

![Line number modes](assets/line-number-modes.gif)

### Themes

Five built-in themes. Cycle through them at runtime with `Alt+t`.

![Theme cycling](assets/theme-cycling.gif)

![All themes](assets/themes-composite.png)

### Mouse Support

When `set -g mouse on` is set in tmux:

| Action | What happens |
|--------|-------------|
| Scroll up in shell | Launches yankee (replaces tmux copy-mode) |
| Left-click in yankee | Set cursor position |
| Left-click drag | Character-wise selection |
| Scroll in yankee | Navigate up/down |

Mouse-aware apps (vim, less, etc.) and alternate-screen programs are detected and left alone.

## Keybindings

### Motions

All motions support count prefixes (`5j`, `3w`, `10G`).

| Key | Action |
|-----|--------|
| `h`/`j`/`k`/`l` | Left / Down / Up / Right |
| `w`/`b`/`e` | Word forward / backward / end |
| `W`/`B`/`E` | WORD forward / backward / end |
| `0` / `$` | Line start / end |
| `^` / `g_` | First / last non-blank character |
| `gg` / `G` | First / last line (or `{count}G` for goto line) |
| `{` / `}` | Paragraph backward / forward |
| `H` / `M` / `L` | Screen top / middle / bottom |
| `%` | Matching bracket (or `{count}%` for percentage) |
| `Ctrl-u` / `Ctrl-d` | Half page up / down |
| `Ctrl-f` / `Ctrl-b` | Full page up / down |
| `Ctrl-y` / `Ctrl-e` | Scroll viewport one line up / down |
| `zt` / `zz` / `zb` | Position cursor line at top / center / bottom |
| `gj` / `gk` | Display line down / up (when wrap mode is on) |

### Search

| Key | Action |
|-----|--------|
| `/` | Forward search |
| `?` | Backward search |
| `n` / `N` | Next / previous match |
| `*` / `#` | Search word under cursor forward / backward |
| `gn` / `gN` | Select next / previous match |
| `\` | Clear search highlights |

### Character Search

| Key | Action |
|-----|--------|
| `f{char}` / `F{char}` | Find char forward / backward |
| `t{char}` / `T{char}` | Till char forward / backward |
| `;` / `,` | Repeat / reverse last char search |

### Flash Navigation

| Key | Action |
|-----|--------|
| `s` | Enter flash mode -- type a pattern, press label to jump |

Uppercase labels offer an alternative jump position (configurable). Works in both normal and visual mode. In visual mode, flash extends the selection to the target.

### Visual Mode and Yanking

| Key | Action |
|-----|--------|
| `v` | Character-wise visual mode |
| `V` | Line-wise visual mode |
| `Ctrl-v` | Block-wise (column) visual mode |
| `o` / `O` | Swap cursor to other end / corner |
| `y` / `Enter` | Yank selection |
| `yy` | Yank current line (normal mode) |

### Text Objects (in visual mode)

| Key | Selects |
|-----|---------|
| `iw` / `aw` | Inner / around word |
| `iW` / `aW` | Inner / around WORD |
| `ip` / `ap` | Inner / around paragraph |
| `i"` / `a"` | Inside / around double quotes |
| `i'` / `a'` | Inside / around single quotes |
| `` i` `` / `` a` `` | Inside / around backticks |
| `ib` / `ab` or `i(` / `a(` | Inside / around parentheses |
| `iB` / `aB` or `i{` / `a{` | Inside / around braces |
| `i[` / `a[` | Inside / around square brackets |
| `i<` / `a<` | Inside / around angle brackets |

### Marks

| Key | Action |
|-----|--------|
| `m{a-z}` | Set mark |
| `` `{a-z} `` | Jump to mark (exact position) |
| `'{a-z}` | Jump to mark line |

### Other

| Key | Action |
|-----|--------|
| `Alt+Shift+L` | Cycle line number modes |
| `Alt+t` | Cycle themes |
| `gw` | Toggle word wrap |
| `Ctrl-o` / `Ctrl-i` | Jump list backward / forward |
| `q` / `Ctrl-c` | Quit |

## Configuration

All options go in `~/.tmux.conf` before the plugin is loaded. They use the `@yankee_` prefix.

### Display

| Option | Default | Values | Description |
|--------|---------|--------|-------------|
| `@yankee_display_mode` | `overlay` | `overlay`, `popup`, `split` | How the TUI appears |
| `@yankee_key` | `N` | single key | Key to trigger yankee (with prefix) |
| `@yankee_key_table` | `prefix` | `prefix`, `root` | Key table for the trigger key |
| `@yankee_start_position` | `bottom` | `top`, `middle`, `bottom` | Initial cursor position |

### Line Numbers

| Option | Default | Values | Description |
|--------|---------|--------|-------------|
| `@yankee_mode` | `hybrid` | `absolute`, `relative`, `hybrid` | Line number display mode |
| `@yankee_scrollback_lines` | `2000` | `100`..`200000` | Lines of scrollback to capture |

### Behavior

| Option | Default | Values | Description |
|--------|---------|--------|-------------|
| `@yankee_copy_target` | `both` | `both`, `tmux`, `clipboard` | Where yanked text goes |
| `@yankee_exit_on_yank` | `on` | `on`, `off` | Close after yanking |
| `@yankee_wrap_mode` | `off` | `on`, `off` | Word wrap for long lines |
| `@yankee_mouse` | `off` | `on`, `off` | Mouse support (click, drag, scroll) |
| `@yankee_status_bar` | `on` | `on`, `off` | Show the status bar |

### Flash

| Option | Default | Values | Description |
|--------|---------|--------|-------------|
| `@yankee_flash` | `on` | `on`, `off` | Enable flash navigation |
| `@yankee_flash_min_chars` | `1` | number | Min pattern length before labels appear |
| `@yankee_flash_ft` | `off` | `on`, `off` | Use flash labels for f/t motions |
| `@yankee_flash_jump_pos` | `match_end` | `match_start`, `match_end`, `word_start`, `word_end` | Where label jump lands |
| `@yankee_flash_alt_jump_pos` | `match_start` | same as above | Where uppercase label lands |

### Theme

| Option | Default | Values | Description |
|--------|---------|--------|-------------|
| `@yankee_theme` | `default` | `default`, `dracula`, `gruvbox`, `nord`, `solarized` | Color theme |

Individual color overrides (`#RRGGBB` format) can be layered on top of any theme:

| Option | Element |
|--------|---------|
| `@yankee_cursor_fg` / `_bg` | Cursor line |
| `@yankee_selection_fg` / `_bg` | Visual selection |
| `@yankee_gutter_fg` / `_bg` | Gutter background |
| `@yankee_gutter_separator_fg` | Separator between gutter and content |
| `@yankee_linenum_absolute_fg` | Absolute line numbers |
| `@yankee_linenum_relative_fg` | Relative line numbers |
| `@yankee_linenum_cursor_fg` | Cursor line number |
| `@yankee_status_fg` / `_bg` | Status bar |
| `@yankee_flash_label_fg` / `_bg` | Flash labels |
| `@yankee_flash_match_fg` / `_bg` | Flash matches |
| `@yankee_flash_backdrop` | Flash dimmed background |

### Custom Keybindings

Rebind, unbind, or add mode-specific bindings:

```tmux
# Remap Ctrl-d to a different action
set -g @yankee_bind_C-d half_page_down

# Remove a default binding
set -g @yankee_unbind_H ""

# Normal-mode only binding
set -g @yankee_nbind_x some_action

# Visual-mode only binding
set -g @yankee_vbind_x some_action
```

### Example Config

```tmux
# Dracula theme, popup mode, generous scrollback
set -g @yankee_theme "dracula"
set -g @yankee_display_mode "popup"
set -g @yankee_scrollback_lines 5000

# Don't close after yank -- browse and yank multiple times
set -g @yankee_exit_on_yank "off"

# Start at top of content
set -g @yankee_start_position "top"

# Flash with 2-char minimum before labels
set -g @yankee_flash_min_chars 2

# Copy to clipboard only (skip tmux buffer)
set -g @yankee_copy_target "clipboard"
```

## Clipboard Support

Yankee detects your system clipboard automatically:

| Platform | Backend |
|----------|---------|
| macOS | `pbcopy` |
| Linux (X11) | `xclip` or `xsel` |
| Linux (Wayland) | `wl-copy` |
| WSL | `clip.exe` |

By default, yanked text goes to both the system clipboard and the tmux paste buffer. Change with `@yankee_copy_target`.

## Known Limitations

- **Snapshot view** -- content is captured at launch time. If your pane keeps producing output, you won't see it until you relaunch.
- **Overlay mode on tmux 3.1** -- works, but tmux 3.2+ is smoother.
- **No true vim registers** -- there's one yank destination (clipboard + tmux buffer), not 26 named registers.

## License

MIT
