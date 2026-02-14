# Manual Verification Checklist - Phase 2

This checklist covers items that require visual inspection or subjective assessment
and cannot be reliably automated.

## Visual Appearance

- [ ] Line numbers are right-aligned in the gutter (no jagged edges)
- [ ] Separator "|" is consistently positioned across all lines
- [ ] Gutter width is uniform (same width for all lines in viewport)
- [ ] Colors render correctly on dark background terminal
- [ ] Colors render correctly on light background terminal
- [ ] Cursor line is visually distinct (green, bold) from other lines
- [ ] Absolute mode numbers are white (or configured color)
- [ ] Relative mode numbers are yellow (or configured color)
- [ ] Hybrid mode shows mixed colors correctly (cursor=green, others=yellow)
- [ ] ANSI reset works properly (no color bleeding into content area)
- [ ] Content text after the gutter is readable and not garbled
- [ ] Empty lines show numbers correctly (number + empty content)

## Mode Toggle (L key)

- [ ] Toggle from hybrid to absolute is visually smooth (no flicker)
- [ ] Toggle from absolute to relative is visually smooth
- [ ] Toggle from relative to hybrid is visually smooth
- [ ] Mode indicator message appears briefly in status line
- [ ] Numbers update correctly after each toggle

## Performance Feel

- [ ] Entry into line-numbered view feels instant (< 100ms subjective)
- [ ] Mode toggle feels instant (< 50ms subjective)
- [ ] Copy-and-paste completes without noticeable delay
- [ ] Exit (q/Escape) restores original pane instantly

## Large Content

- [ ] 50,000 line scrollback: entry time still feels instant
- [ ] 50,000 line scrollback: gutter accommodates 5-digit numbers
- [ ] 50,000 line scrollback: no visible performance degradation

## Pane Sizes

- [ ] Narrow pane (30 cols): content still readable after gutter
- [ ] Very wide pane (200 cols): no wrapping issues or alignment problems
- [ ] Standard pane (80 cols): looks good as the default case
- [ ] Short pane (10 rows): all rows have numbers
- [ ] Tall pane (50+ rows): all rows have numbers, performance OK

## Copy-Mode Navigation

- [ ] Arrow keys work for cursor movement in numbered view
- [ ] vi motions (h/j/k/l) work for movement
- [ ] Page up/down (Ctrl-b/Ctrl-f) work
- [ ] Search (/) works in numbered view
- [ ] Selection (v, V) works and highlights correctly
- [ ] Yank (y) copies text WITHOUT line numbers
- [ ] Enter copies text WITHOUT line numbers
- [ ] Copied text pastes correctly via prefix+]
- [ ] Copied text is available in system clipboard (if supported)

## Exit Behavior

- [ ] Press q: exits cleanly, original pane restored
- [ ] Press Escape: exits cleanly, original pane restored
- [ ] After exit: standard copy-mode (prefix+[) works normally
- [ ] After exit: vi motions in copy-mode work as expected
- [ ] After exit: q in standard copy-mode exits copy-mode (not plugin)
- [ ] After exit: no orphaned panes visible in `tmux list-panes`

## Edge Cases

- [ ] Zoomed pane: line numbers work, zoom state preserved after exit
- [ ] Split panes: only the target pane is affected, others unchanged
- [ ] Multiple windows: only the current window is affected
- [ ] Copy-mode already active: plugin captures correct scroll position
- [ ] No scrollback (fresh pane): line numbers start from 0, works correctly
- [ ] Mouse mode (set -g mouse on): mouse scrolling works in numbered view

## Plugin Loading

- [ ] TPM installation: `prefix + I` installs plugin correctly
- [ ] Plugin source: `tmux source-file ~/.tmux.conf` loads without errors
- [ ] Idempotent: sourcing plugin multiple times causes no issues
- [ ] No stdout output during loading (would break tmux)
- [ ] Plugin loads quickly (does not delay tmux startup)

## Compatibility

- [ ] Works with tmux-yank plugin installed simultaneously
- [ ] Works with tmux-fingers plugin installed simultaneously
- [ ] Works in tmux inside tmux (nested sessions)
- [ ] Works on macOS with pbcopy for clipboard
- [ ] Works on Linux/X11 with xclip for clipboard
- [ ] Standard prefix+[ copy-mode is completely unaffected

## Configuration

- [ ] Custom mode via `@linenumbers-mode "absolute"` works
- [ ] Custom styles via `@linenumbers-style-*` options work
- [ ] Custom toggle key via `@linenumbers-toggle-key` works
- [ ] Custom binding key via `@linenumbers-custom-key` works
- [ ] Disabling binding via `@linenumbers-enable-binding "off"` works
- [ ] All options have sensible defaults without explicit configuration

## Error Handling

- [ ] tmux version < 3.1: graceful error message, no crash
- [ ] Killing the script mid-execution: cleanup runs, no orphaned panes
- [ ] Closing terminal window during view: no persistent state leaks
- [ ] SIGTERM/SIGINT: cleanup trap fires correctly
