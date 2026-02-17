# Changelog

## [Unreleased]

### Added
- Full settings system via `@yankee_*` tmux options
- 5 built-in themes: `default`, `dracula`, `gruvbox`, `nord`, `solarized`
- Per-element color overrides (cursor, selection, gutter, line numbers)
- `@yankee_scrollback_lines`: configurable history capture (100-200000, default 2000)
- `@yankee_copy_target`: copy to `both` (default), `tmux`, or `clipboard` only
- `@yankee_exit_on_yank`: keep TUI open after yanking when set to `off`
- `@yankee_start_position`: start cursor at `top`, `middle`, or `bottom` (default)
- `@yankee_toggle_mode_key`: configurable line number mode toggle key (default `L`)
- Mouse scroll integration: scroll-up launches yankee, wheel scrolls content, overscroll-down exits (requires `set -g mouse on`)

## 0.1.0 - 2026-02-14

Initial release. Phase 2 implementation: capture-and-replace line numbers for tmux copy-mode.

### Added

- **Line number rendering** with three display modes: absolute, relative, hybrid
- **Mode cycling** via configurable toggle key (`L` by default) during line numbers view
- **Customizable styles** for absolute, relative, and cursor line numbers (`@linenumbers-style-*` options)
- **Copy filtering** that automatically strips the line number gutter when yanking text
- **Opt-in keybinding** (`prefix + N` by default) -- does not override native `[`
- **Respawn-pane architecture** for stable pane IDs, zoom safety, and simpler lifecycle
- **Trap-based cleanup** on all exit paths (q, Escape, SIGTERM, SIGINT)
- **Comprehensive test suite**: 23 unit tests + 11 integration tests (34 total), shellcheck clean

### Architecture

Uses the capture-and-replace pattern (same approach as tmux-fingers, tmux-thumbs):
- Captures viewport content and scroll state
- Renders numbered content to a temp file
- Replaces pane via `respawn-pane -k` (keeps pane ID stable)
- Enters copy-mode for navigation
- Restores original shell on exit

### Known Issues

- CWD not preserved across respawn-pane lifecycle (user lands in `$HOME` after exit)
- Shell in-memory history lost on respawn (inherent trade-off)
- Renderer calls `tmux_style_to_ansi` per line in subshell (performance optimization opportunity)
- State directory discovery scans `/tmp` glob (fragile with concurrent sessions)

### Files

Production (766 lines):
- `plugin.tmux` (25 lines)
- `scripts/utils.sh` (57 lines)
- `scripts/config.sh` (84 lines)
- `scripts/renderer.sh` (152 lines)
- `scripts/state_cleanup.sh` (101 lines)
- `scripts/copy_filter.sh` (58 lines)
- `scripts/line_numbers.sh` (194 lines)
- `scripts/toggle_and_rerender.sh` (75 lines)
- `scripts/init.sh` (20 lines)

Tests (2666 lines):
- `tests/run_all_tests.sh`
- `tests/test_helpers.sh`
- `tests/unit/test_renderer.sh`
- `tests/unit/test_config.sh`
- `tests/unit/test_copy_filter.sh`
- `tests/unit/test_math.sh`
- `tests/integration/test_basic_flow.sh`
- `tests/integration/test_cleanup.sh`
- `tests/integration/test_toggle.sh`
- `tests/integration/test_edge_cases.sh`
- `tests/manual_verification.md`
