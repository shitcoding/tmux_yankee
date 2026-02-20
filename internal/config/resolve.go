package config

import (
	"fmt"

	"github.com/shitcoding/tmux_yankee/internal/theme"
)

// Resolve validates opts and returns a fully populated Settings, or an error.
func Resolve(opts CLIOptions) (Settings, error) {
	if err := Validate(opts); err != nil {
		return Settings{}, err
	}

	// Clamp ScrollbackLines to [MinScrollbackLines, MaxScrollbackLines]
	scrollback := opts.ScrollbackLines
	if scrollback < MinScrollbackLines {
		scrollback = MinScrollbackLines
	}
	if scrollback > MaxScrollbackLines {
		scrollback = MaxScrollbackLines
	}

	// Parse bool strings
	exitOnYank := opts.ExitOnYank == "on"

	// Parse ToggleModeKey to byte
	toggleKey := opts.ToggleModeKey[0]

	// Build ThemeOverrides from CLI options
	overrides := theme.ThemeOverrides{
		CursorFG:              opts.CursorFG,
		CursorBG:              opts.CursorBG,
		SelectionFG:           opts.SelectionFG,
		SelectionBG:           opts.SelectionBG,
		GutterFG:              opts.GutterFG,
		GutterBG:              opts.GutterBG,
		GutterSeparatorFG:     opts.GutterSeparatorFG,
		GutterSeparatorBG:     opts.GutterSeparatorBG,
		GutterSeparatorChar:   opts.GutterSeparatorChar,
		LineNumAbsoluteFG:     opts.LineNumAbsoluteFG,
		LineNumRelativeFG:     opts.LineNumRelativeFG,
		LineNumCursorFG:       opts.LineNumCursorFG,
		LineNumCursorBold:     opts.LineNumCursorBold,
		StatusFG:              opts.StatusFG,
		StatusBG:              opts.StatusBG,
		LineNumAbsoluteBold:   opts.LineNumAbsoluteBold,
		LineNumAbsoluteDim:    opts.LineNumAbsoluteDim,
		LineNumAbsoluteItalic: opts.LineNumAbsoluteItalic,
		LineNumRelativeBold:   opts.LineNumRelativeBold,
		LineNumRelativeDim:    opts.LineNumRelativeDim,
		LineNumRelativeItalic: opts.LineNumRelativeItalic,
		LineNumCursorDim:      opts.LineNumCursorDim,
		LineNumCursorItalic:   opts.LineNumCursorItalic,
		StatusBold:            opts.StatusBold,
		StatusDim:             opts.StatusDim,
		CursorDim:             opts.CursorDim,
		CursorItalic:          opts.CursorItalic,
		SelectionDim:          opts.SelectionDim,
		SelectionItalic:       opts.SelectionItalic,
	}

	palette, err := theme.Resolve(theme.ThemeName(opts.Theme), overrides)
	if err != nil {
		return Settings{}, fmt.Errorf("theme: %w", err)
	}

	wrapMode := WrapMode(opts.WrapMode)
	if wrapMode != WrapModeScroll && wrapMode != WrapModeWrap {
		wrapMode = WrapModeScroll
	}

	return Settings{
		PaneID:          opts.PaneID,
		Mode:            LineNumberMode(opts.Mode),
		ScrollbackLines: scrollback,
		Demo:            opts.Demo,
		Palette:         palette,
		ToggleModeKey:   toggleKey,
		CopyTarget:      CopyTarget(opts.CopyTarget),
		ExitOnYank:      exitOnYank,
		StartPosition:   StartPosition(opts.StartPosition),
		WrapMode:        wrapMode,
	}, nil
}
