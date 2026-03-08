package config

import (
	"fmt"
	"strconv"

	"github.com/shitcoding/tmux_yankee/internal/flash"
	"github.com/shitcoding/tmux_yankee/internal/keymap"
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

	// Parse ToggleModeKey and WrapKey to byte
	toggleKey := opts.ToggleModeKey[0]
	wrapKey := opts.WrapKey[0]

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
		FlashLabelFG:          opts.FlashLabelFG,
		FlashLabelBG:          opts.FlashLabelBG,
		FlashMatchFG:          opts.FlashMatchFG,
		FlashMatchBG:          opts.FlashMatchBG,
		FlashBackdrop:         opts.FlashBackdrop,
	}

	palette, err := theme.Resolve(theme.ThemeName(opts.Theme), overrides)
	if err != nil {
		return Settings{}, fmt.Errorf("theme: %w", err)
	}

	wrapMode := WrapMode(opts.WrapMode)
	if wrapMode != WrapModeOff && wrapMode != WrapModeOn {
		wrapMode = WrapModeOff
	}

	statusBar := StatusBarMode(opts.StatusBar)
	if statusBar != StatusBarOn && statusBar != StatusBarOff {
		statusBar = StatusBarOn
	}

	// Build keymap: defaults + user overrides from --bindings flag
	km := keymap.DefaultKeymap()
	if opts.Bindings != "" {
		overrides, err := keymap.ParseBindings(opts.Bindings)
		if err != nil {
			return Settings{}, fmt.Errorf("bindings: %w", err)
		}
		km = km.Merge(overrides)
	}

	// Flash settings
	flashEnabled := opts.Flash != "off"
	flashMinChars := 1
	if opts.FlashMinChars != "" {
		if n, err := strconv.Atoi(opts.FlashMinChars); err == nil && n > 0 {
			flashMinChars = n
		}
	}
	flashFT := opts.FlashFT == "on"
	flashJumpPos := int(flash.ParseJumpPos(opts.FlashJumpPos, flash.JumpPosMatchEnd))
	flashAltJumpPos := int(flash.ParseJumpPos(opts.FlashAltJumpPos, flash.JumpPosMatchStart))

	return Settings{
		PaneID:          opts.PaneID,
		Mode:            LineNumberMode(opts.Mode),
		ScrollbackLines: scrollback,
		Demo:            opts.Demo,
		ThemeName:       opts.Theme,
		Palette:         palette,
		ToggleModeKey:   toggleKey,
		WrapKey:         wrapKey,
		CopyTarget:      CopyTarget(opts.CopyTarget),
		ExitOnYank:      exitOnYank,
		StartPosition:   StartPosition(opts.StartPosition),
		WrapMode:        wrapMode,
		StatusBar:       statusBar,
		Keymap:          km,
		FlashEnabled:    flashEnabled,
		FlashMinChars:   flashMinChars,
		FlashFTEnabled:  flashFT,
		FlashJumpPos:    flashJumpPos,
		FlashAltJumpPos: flashAltJumpPos,
	}, nil
}
