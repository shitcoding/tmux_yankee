package config

import (
	"fmt"
	"regexp"
	"strconv"
	"unicode"
	"unicode/utf8"
)

var hexColorRe = regexp.MustCompile(`^#[0-9a-fA-F]{6}$`)

// Validate checks all CLIOptions fields for correctness.
// ScrollbackLines is clamped (not an error) — the caller handles clamping.
func Validate(opts CLIOptions) error {
	// PaneID: required unless --demo
	if opts.PaneID == "" && !opts.Demo {
		return fmt.Errorf("--pane is required (unless --demo is set)")
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
		{"gutter-separator-bg", opts.GutterSeparatorBG},
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

	// GutterSeparatorChar: must be exactly 1 printable rune when non-empty
	if opts.GutterSeparatorChar != "" {
		if utf8.RuneCountInString(opts.GutterSeparatorChar) != 1 {
			return fmt.Errorf("invalid gutter-separator-char %q: must be exactly one printable character", opts.GutterSeparatorChar)
		}
		r, _ := utf8.DecodeRuneInString(opts.GutterSeparatorChar)
		if !unicode.IsPrint(r) {
			return fmt.Errorf("invalid gutter-separator-char %q: must be a printable character", opts.GutterSeparatorChar)
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

	// WrapKey: exactly 1 printable ASCII character
	if len(opts.WrapKey) != 1 {
		return fmt.Errorf("invalid wrap-key %q: must be exactly one ASCII character", opts.WrapKey)
	}
	wk := opts.WrapKey[0]
	if wk < 0x20 || wk > 0x7e {
		return fmt.Errorf("invalid wrap-key %q: must be a printable ASCII character", opts.WrapKey)
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

	// StatusBar
	switch opts.StatusBar {
	case "on", "off":
		// valid
	default:
		return fmt.Errorf("invalid status-bar %q: must be on or off", opts.StatusBar)
	}

	// WrapMode
	switch opts.WrapMode {
	case "on", "off":
		// valid
	default:
		return fmt.Errorf("invalid wrap-mode %q: must be on or off", opts.WrapMode)
	}

	// FlashMinChars: must be a positive integer when non-empty
	if opts.FlashMinChars != "" {
		n, err := strconv.Atoi(opts.FlashMinChars)
		if err != nil || n < 1 {
			return fmt.Errorf("invalid flash-min-chars %q: must be a positive integer", opts.FlashMinChars)
		}
	}

	// Flash jump positions
	validJumpPos := map[string]bool{
		"match_end": true, "match_start": true,
		"word_start": true, "word_end": true,
		"off": true, "": true,
	}
	if !validJumpPos[opts.FlashJumpPos] {
		return fmt.Errorf("invalid flash-jump-pos %q: must be one of match_end, match_start, word_start, word_end, off", opts.FlashJumpPos)
	}
	if !validJumpPos[opts.FlashAltJumpPos] {
		return fmt.Errorf("invalid flash-alt-jump-pos %q: must be one of match_end, match_start, word_start, word_end, off", opts.FlashAltJumpPos)
	}

	// On/off boolean style fields
	boolFields := []struct {
		name string
		val  string
	}{
		{"linenum-cursor-bold", opts.LineNumCursorBold},
		{"linenum-absolute-bold", opts.LineNumAbsoluteBold},
		{"linenum-absolute-dim", opts.LineNumAbsoluteDim},
		{"linenum-absolute-italic", opts.LineNumAbsoluteItalic},
		{"linenum-relative-bold", opts.LineNumRelativeBold},
		{"linenum-relative-dim", opts.LineNumRelativeDim},
		{"linenum-relative-italic", opts.LineNumRelativeItalic},
		{"linenum-cursor-dim", opts.LineNumCursorDim},
		{"linenum-cursor-italic", opts.LineNumCursorItalic},
		{"status-bold", opts.StatusBold},
		{"status-dim", opts.StatusDim},
		{"cursor-dim", opts.CursorDim},
		{"cursor-italic", opts.CursorItalic},
		{"selection-dim", opts.SelectionDim},
		{"selection-italic", opts.SelectionItalic},
	}
	for _, f := range boolFields {
		switch f.val {
		case "on", "off", "":
			// valid
		default:
			return fmt.Errorf("invalid %s %q: must be on, off, or empty", f.name, f.val)
		}
	}

	return nil
}
