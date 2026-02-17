package config

import "flag"

// RegisterFlags binds all CLI flags to opts fields.
func RegisterFlags(fs *flag.FlagSet, opts *CLIOptions) {
	fs.StringVar(&opts.PaneID, "pane", "", "Target tmux pane ID (required)")
	fs.StringVar(&opts.Mode, "mode", DefaultMode, "Line number mode (absolute, relative, hybrid)")
	fs.IntVar(&opts.ScrollbackLines, "scrollback-lines", DefaultScrollbackLines, "Lines of scrollback to capture (100-200000)")
	fs.StringVar(&opts.Theme, "theme", DefaultTheme, "Theme preset (default, dracula, gruvbox, nord, solarized)")
	fs.StringVar(&opts.StatusIndicator, "status-indicator", DefaultStatusIndicator, "Show top-right status indicator (on, off)")
	fs.StringVar(&opts.CursorFG, "cursor-fg", "", "Cursor foreground color (#RRGGBB)")
	fs.StringVar(&opts.CursorBG, "cursor-bg", "", "Cursor background color (#RRGGBB)")
	fs.StringVar(&opts.SelectionFG, "selection-fg", "", "Selection foreground color (#RRGGBB)")
	fs.StringVar(&opts.SelectionBG, "selection-bg", "", "Selection background color (#RRGGBB)")
	fs.StringVar(&opts.GutterFG, "gutter-fg", "", "Gutter foreground color (#RRGGBB)")
	fs.StringVar(&opts.GutterBG, "gutter-bg", "", "Gutter background color (#RRGGBB)")
	fs.StringVar(&opts.GutterSeparatorFG, "gutter-separator-fg", "", "Gutter separator foreground color (#RRGGBB)")
	fs.StringVar(&opts.LineNumAbsoluteFG, "linenum-absolute-fg", "", "Absolute line number foreground color (#RRGGBB)")
	fs.StringVar(&opts.LineNumRelativeFG, "linenum-relative-fg", "", "Relative line number foreground color (#RRGGBB)")
	fs.StringVar(&opts.LineNumCursorFG, "linenum-cursor-fg", "", "Cursor line number foreground color (#RRGGBB)")
	fs.StringVar(&opts.LineNumCursorBold, "linenum-cursor-bold", "", "Cursor line bold (on, off)")
	fs.StringVar(&opts.StatusFG, "status-fg", "", "Status indicator foreground color (#RRGGBB)")
	fs.StringVar(&opts.StatusBG, "status-bg", "", "Status indicator background color (#RRGGBB)")
	fs.StringVar(&opts.ToggleModeKey, "toggle-mode-key", DefaultToggleModeKey, "Key to toggle line number mode (single ASCII char)")
	fs.StringVar(&opts.CopyTarget, "copy-target", DefaultCopyTarget, "Copy destination (both, tmux, clipboard)")
	fs.StringVar(&opts.ExitOnYank, "exit-on-yank", DefaultExitOnYank, "Exit TUI after yanking (on, off)")
	fs.StringVar(&opts.StartPosition, "start-position", DefaultStartPosition, "Initial cursor position (top, middle, bottom)")
}
