package config

import "flag"

// RegisterFlags binds all CLI flags to opts fields.
func RegisterFlags(fs *flag.FlagSet, opts *CLIOptions) {
	fs.StringVar(&opts.PaneID, "pane", "", "Target tmux pane ID (required unless --demo)")
	fs.StringVar(&opts.Mode, "mode", DefaultMode, "Line number mode (absolute, relative, hybrid)")
	fs.IntVar(&opts.ScrollbackLines, "scrollback-lines", DefaultScrollbackLines, "Lines of scrollback to capture (100-200000)")
	fs.StringVar(&opts.Theme, "theme", DefaultTheme, "Theme preset (default, dracula, gruvbox, nord, solarized)")
	fs.BoolVar(&opts.Demo, "demo", false, "Run in demo mode (no tmux pane required)")
	fs.StringVar(&opts.CursorFG, "cursor-fg", "", "Cursor foreground color (#RRGGBB)")
	fs.StringVar(&opts.CursorBG, "cursor-bg", "", "Cursor background color (#RRGGBB)")
	fs.StringVar(&opts.SelectionFG, "selection-fg", "", "Selection foreground color (#RRGGBB)")
	fs.StringVar(&opts.SelectionBG, "selection-bg", "", "Selection background color (#RRGGBB)")
	fs.StringVar(&opts.GutterFG, "gutter-fg", "", "Gutter foreground color (#RRGGBB)")
	fs.StringVar(&opts.GutterBG, "gutter-bg", "", "Gutter background color (#RRGGBB)")
	fs.StringVar(&opts.GutterSeparatorFG, "gutter-separator-fg", "", "Gutter separator foreground color (#RRGGBB)")
	fs.StringVar(&opts.GutterSeparatorBG, "gutter-separator-bg", "", "Gutter separator background color (#RRGGBB)")
	fs.StringVar(&opts.GutterSeparatorChar, "gutter-separator-char", "", "Gutter separator character (single printable rune)")
	fs.StringVar(&opts.LineNumAbsoluteFG, "linenum-absolute-fg", "", "Absolute line number foreground color (#RRGGBB)")
	fs.StringVar(&opts.LineNumRelativeFG, "linenum-relative-fg", "", "Relative line number foreground color (#RRGGBB)")
	fs.StringVar(&opts.LineNumCursorFG, "linenum-cursor-fg", "", "Cursor line number foreground color (#RRGGBB)")
	fs.StringVar(&opts.LineNumCursorBold, "linenum-cursor-bold", "", "Cursor line bold (on, off)")
	fs.StringVar(&opts.StatusFG, "status-fg", "", "Status indicator foreground color (#RRGGBB)")
	fs.StringVar(&opts.StatusBG, "status-bg", "", "Status indicator background color (#RRGGBB)")

	// TextStyle override flags
	fs.StringVar(&opts.LineNumAbsoluteBold, "linenum-absolute-bold", "", "Absolute line number bold (on, off)")
	fs.StringVar(&opts.LineNumAbsoluteDim, "linenum-absolute-dim", "", "Absolute line number dim (on, off)")
	fs.StringVar(&opts.LineNumAbsoluteItalic, "linenum-absolute-italic", "", "Absolute line number italic (on, off)")
	fs.StringVar(&opts.LineNumRelativeBold, "linenum-relative-bold", "", "Relative line number bold (on, off)")
	fs.StringVar(&opts.LineNumRelativeDim, "linenum-relative-dim", "", "Relative line number dim (on, off)")
	fs.StringVar(&opts.LineNumRelativeItalic, "linenum-relative-italic", "", "Relative line number italic (on, off)")
	fs.StringVar(&opts.LineNumCursorDim, "linenum-cursor-dim", "", "Cursor line number dim (on, off)")
	fs.StringVar(&opts.LineNumCursorItalic, "linenum-cursor-italic", "", "Cursor line number italic (on, off)")
	fs.StringVar(&opts.StatusBold, "status-bold", "", "Status bar bold (on, off)")
	fs.StringVar(&opts.StatusDim, "status-dim", "", "Status bar dim (on, off)")
	fs.StringVar(&opts.CursorDim, "cursor-dim", "", "Cursor dim (on, off)")
	fs.StringVar(&opts.CursorItalic, "cursor-italic", "", "Cursor italic (on, off)")
	fs.StringVar(&opts.SelectionDim, "selection-dim", "", "Selection dim (on, off)")
	fs.StringVar(&opts.SelectionItalic, "selection-italic", "", "Selection italic (on, off)")

	fs.StringVar(&opts.ToggleModeKey, "toggle-mode-key", DefaultToggleModeKey, "Key to toggle line number mode (single ASCII char)")
	fs.StringVar(&opts.CopyTarget, "copy-target", DefaultCopyTarget, "Copy destination (both, tmux, clipboard)")
	fs.StringVar(&opts.ExitOnYank, "exit-on-yank", DefaultExitOnYank, "Exit TUI after yanking (on, off)")
	fs.StringVar(&opts.StartPosition, "start-position", DefaultStartPosition, "Initial cursor position (top, middle, bottom)")
	fs.StringVar(&opts.WrapMode, "wrap-mode", DefaultWrapMode, "Long line handling (scroll, wrap)")
}
