package config

import (
	"fmt"
	"regexp"
)

var hexColorRe = regexp.MustCompile(`^#[0-9a-fA-F]{6}$`)

// Validate checks all CLIOptions fields for correctness.
// ScrollbackLines is clamped (not an error) — the caller handles clamping.
func Validate(opts CLIOptions) error {
	// PaneID
	if opts.PaneID == "" {
		return fmt.Errorf("--pane is required")
	}

	// Mode
	switch opts.Mode {
	case "absolute", "relative", "hybrid":
		// valid
	default:
		return fmt.Errorf("invalid mode %q: must be one of absolute, relative, hybrid", opts.Mode)
	}

	// Theme
	switch opts.Theme {
	case "default", "dracula", "gruvbox", "nord", "solarized":
		// valid
	default:
		return fmt.Errorf("invalid theme %q: must be one of default, dracula, gruvbox, nord, solarized", opts.Theme)
	}

	// Color fields (ordered for deterministic error messages)
	colorFields := []struct {
		name string
		val  string
	}{
		{"cursor-fg", opts.CursorFG},
		{"cursor-bg", opts.CursorBG},
		{"selection-fg", opts.SelectionFG},
		{"selection-bg", opts.SelectionBG},
		{"gutter-fg", opts.GutterFG},
		{"gutter-bg", opts.GutterBG},
		{"gutter-separator-fg", opts.GutterSeparatorFG},
		{"linenum-absolute-fg", opts.LineNumAbsoluteFG},
		{"linenum-relative-fg", opts.LineNumRelativeFG},
		{"linenum-cursor-fg", opts.LineNumCursorFG},
		{"status-fg", opts.StatusFG},
		{"status-bg", opts.StatusBG},
	}
	for _, f := range colorFields {
		if f.val != "" && !hexColorRe.MatchString(f.val) {
			return fmt.Errorf("invalid color for --%s: %q (must be #RRGGBB)", f.name, f.val)
		}
	}

	// ToggleModeKey: exactly 1 printable ASCII character
	if len(opts.ToggleModeKey) != 1 {
		return fmt.Errorf("invalid toggle-mode-key %q: must be exactly one ASCII character", opts.ToggleModeKey)
	}
	b := opts.ToggleModeKey[0]
	if b < 0x20 || b > 0x7e {
		return fmt.Errorf("invalid toggle-mode-key %q: must be a printable ASCII character", opts.ToggleModeKey)
	}

	// CopyTarget
	switch opts.CopyTarget {
	case "both", "tmux", "clipboard":
		// valid
	default:
		return fmt.Errorf("invalid copy-target %q: must be one of both, tmux, clipboard", opts.CopyTarget)
	}

	// ExitOnYank
	switch opts.ExitOnYank {
	case "on", "off":
		// valid
	default:
		return fmt.Errorf("invalid exit-on-yank %q: must be on or off", opts.ExitOnYank)
	}

	// StartPosition
	switch opts.StartPosition {
	case "top", "middle", "bottom":
		// valid
	default:
		return fmt.Errorf("invalid start-position %q: must be one of top, middle, bottom", opts.StartPosition)
	}

	// LineNumCursorBold
	switch opts.LineNumCursorBold {
	case "on", "off", "":
		// valid
	default:
		return fmt.Errorf("invalid linenum-cursor-bold %q: must be on, off, or empty", opts.LineNumCursorBold)
	}

	return nil
}
