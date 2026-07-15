# Changelog

All notable changes to tmux-yankee are documented here. The format follows
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and the project
loosely follows [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.0.2] — 2026-07-14

Maintenance release: the plugin now upgrades like a regular tmux plugin.

### Added

- `tmux-yankee -version` prints the installed version.

### Fixed

- The bundled binary now upgrades automatically when the plugin is updated
  (`prefix + U` with TPM, or `git pull` for a manual install). Previously the
  installer skipped the download whenever *any* binary already existed, so an
  update pulled new source but kept the old binary — you stayed on the old
  version indefinitely. A `VERSION` file is now the source of truth: the binary
  is stamped with it, and `scripts/install.sh` re-downloads the matching
  release (`releases/tags/v<VERSION>`) whenever the installed binary's version
  differs. Local `go build` dev binaries (version `dev`) are left untouched, and
  a failed upgrade keeps the existing binary working instead of erroring.

## [1.0.1] — 2026-07-14

Maintenance release: bug fixes and internal cleanup. No new features and no
configuration changes — upgrading is a drop-in replacement.

### Fixed

- Demo mode (`--demo`): yanking no longer crashes. Demo mode has no tmux
  backend, and the default copy target dereferenced the nil tmux client.
- Input: an escape sequence fragmented across reads (e.g. an SGR mouse
  sequence split by a TCP-segment boundary over SSH) is no longer misparsed
  as a lone `Esc` followed by literal keys. A short (25 ms) escape-flush
  debounce waits for the rest of the sequence before treating `Esc` as
  standalone — standalone `Esc` stays responsive.
- Resize: the SIGWINCH debounce no longer races a superseded timer, so a
  stray resize can no longer render the pane at an intermediate (mid
  zoom-swap) size. Both debounces now use race-free timers.

### Changed

- Internal: removed dead code across the Go packages, consolidated duplicated
  helpers (repo-wide `min`/`max` builtins, a single hex-color parser and
  escape-flush path), and split the monolithic `internal/ui/tui.go` into
  focused files (wrap-mode viewport math, mouse handling, yank). No
  user-visible behavior change.

## [1.0.0] — 2026-06-12

First public release. Replaces tmux's native copy-mode with a Go TUI overlay
featuring vim motions, visual selection (char/line/block), incremental search,
flash-style label jumps, text objects, multiple themes, and clipboard
integration.

### Added

#### Navigation & motions

- Vim motions: `h/j/k/l`, `w/b/e/ge/gE`, `0/^/$/g_`, `gg/G`, `H/M/L`, `f/F/t/T/;/,`, `%`
- Page motions: `Ctrl-D/U/B/F`, line scrolls `Ctrl-E/Y`
- Viewport repositioning: `zt/zz/zb`
- Marks: `m{a-z}`, jumps via `` `{a-z} `` and `'{a-z}`
- Jump list: `Ctrl-O` / `Ctrl-I`
- Paragraph motions: `{` / `}`
- Wrap-mode display-line navigation: `gj`/`gk`, `gw` to toggle wrap
- Numeric count prefixes on most motions (`5j`, `42gg`, `H 3`, `L 2`, …)
- Horizontal scroll for long unwrapped lines with `<`/`>` indicators

#### Selection & yank

- Visual modes: char (`v`), line (`V`), block (`Ctrl-V`)
- Visual-mode `o`/`O` swap cursor end (block: column-only)
- Yank with `y` (selection) and `yy` (line)
- Block selection extends past EOL on shorter lines (vim-faithful)
- Mouse drag-select with live highlighting and selection extension
- Clipboard via cross-platform `copy_stdin.sh` (`pbcopy` → `wl-copy` → `xsel` → `xclip` → `clip.exe` → `putclip`)
- Tmux paste buffer integration via `load-buffer -` (stdin, not argv)
- Line-number gutter is stripped from yanked text
- `@yankee_copy_target`: `both` (default), `tmux`, or `clipboard`
- `@yankee_exit_on_yank`: stay in TUI after yank when `off`

#### Line numbers

- Three modes: `absolute`, `relative`, `hybrid` (cursor absolute, others relative)
- Configurable per-element styling (FG, BG, bold, dim, italic)
- Cycle line-number modes with `Alt+Shift+L` (rebindable via `@yankee_bind_<key>` / `@yankee_nbind_<key>` overrides)
- Configurable separator character and color
- Demo mode (`--demo`) cycles four content fixtures

#### Search

- `/pattern` (forward) and `?pattern` (backward) incremental search with live highlighting
- `n`/`N` next/prev match (from cursor position)
- `*` / `#` word-under-cursor search
- `gn` / `gN` select next/prev match
- Esc cancel restores the pre-search pattern, matches, direction, viewport, and cursor

#### Flash navigation (port of flash.nvim)

- `s` enters flash search mode — type any character to label all visible matches; press a label letter to jump
- `f` / `t` / `F` / `T` optionally augmented with flash labels for multi-match disambiguation
- Configurable jump positions (`match_start`, `match_end`, `word_start`, `word_end`)
- Smartcase, label-vs-pattern disambiguation, wrap-aware visible matcher

#### Text objects

- `iw`/`aw` word, `iW`/`aW` WORD
- `ip`/`ap` paragraph
- Quote objects: `i"`, `a"`, `i'`, `a'`, `` i` ``, `` a` ``
- Bracket objects: `i(`, `a(`, `i[`, `a[`, `i{`, `a{`, `i<`, `a<`
- Multi-strategy bracket search with backward fallback (Neovim-faithful)

#### Configurable keymap

- `@yankee_bind_<key>` / `@yankee_unbind_<key>`: shared overrides
- `@yankee_nbind_<key>` / `@yankee_nunbind_<key>`: normal-mode-only overrides
- `@yankee_vbind_<key>` / `@yankee_vunbind_<key>`: visual-mode-only overrides
- Trie-based prefix matching, Alt+key safe (no leakage to tmux)
- ~80 default actions exposed by name

#### Themes & styling

- Five preset themes: `default`, `dracula`, `gruvbox`, `nord`, `solarized`
- Cycle themes interactively with `Alt+t`
- Per-element foreground / background / style overrides for cursor, selection, gutter, line numbers, status bar, search match/current, flash label/match/backdrop
- Powerline status bar with vim-airline-inspired palettes

#### Mouse

- `@yankee_mouse on` (opt-in): mouse wheel launches yankee in regular panes; wheel inside yankee scrolls the viewport
- SGR mouse parsing for click, drag, release, and wheel events

#### Distribution

- TPM install: `set -g @plugin 'shitcoding/tmux-yankee'`; binary auto-downloads from the latest GitHub Release on first plugin load
- `scripts/install.sh` detects platform/architecture (darwin/linux × amd64/arm64) and downloads with atomic `mktemp` + `mv`
- Fallback: `make build` from source
- GitHub Actions release workflow cross-compiles all four binaries on a `v*` tag push

#### Robustness & safety

- Per-pane locks under `$TMPDIR/tmux-yankee/<server-key>/pane-<id>.lock` for concurrent yankee instances
- Atomic busy gate at the tmux binding level (`set-option -opq @yankee_busy 1`) serializes rapid scroll-launches
- Per-pane deterministic helper-session names; crash-recoverable
- 50 ms SIGWINCH debounce mitigates the zoomed-pane swap-pane glitch
- 10 s timeout on every tmux subprocess call (`exec.CommandContext`)
- Bash strict mode (`set -euo pipefail`) and shellcheck-clean across all shell scripts
- macOS bash 3.2 compatibility verified
- Strict config validation: hex colors, on/off enums, integer ranges, key codes
- ECMA-48 escape-sequence scanner shared by `stripANSI` (yank/search path) and `ParseANSILine` (render path): closes terminal-escape-injection paths via OSC/DCS/APC/PM/SOS/SS2/SS3/charset designation in captured content
- Wrap-mode viewport math overhauled: every scroll/motion path (mouse scroll, `Ctrl-E/Y`, `Ctrl-D/U/F/B`, `zt/zb`, `H/M/L`) compares display rows, not source lines

#### Tooling

- Docker-based Linux integration test harness (`Dockerfile.test-linux`, `tests/run_linux_tests.sh`)
- Bash unit + integration test runners
- Visual asset pipeline via `vhs`, `freeze`, `asciinema`

### Known limitations

- Single source lines that wrap past the visible viewport cannot be scrolled within (no intra-line display-row offset)
- `display-popup -K` flag does not exist in any released tmux; yankee uses the overlay (swap-pane) display mode exclusively
- CJK rune-width handling in horizontal scroll is out of scope (use wrap mode for full-width content)

## [0.1.0] — 2026-02-14

Initial shell-only prototype (superseded the same day by the Go TUI rewrite
under ADR-002). Kept here only for historical reference; no published artifact.

### Added

- Line-number rendering with absolute / relative / hybrid modes
- Mode cycling via toggle key
- Customizable per-element styling
- Copy filtering that strips the line-number gutter when yanking
- Opt-in `prefix + N` binding
- Respawn-pane lifecycle with trap-based cleanup
- 34 tests (23 unit + 11 integration), shellcheck-clean
