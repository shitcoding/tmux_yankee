package config

import "github.com/shitcoding/tmux_yankee/internal/theme"

type LineNumberMode string

const (
	LineNumberModeAbsolute LineNumberMode = "absolute"
	LineNumberModeRelative LineNumberMode = "relative"
	LineNumberModeHybrid   LineNumberMode = "hybrid"
)

type CopyTarget string

const (
	CopyTargetBoth      CopyTarget = "both"
	CopyTargetTmux      CopyTarget = "tmux"
	CopyTargetClipboard CopyTarget = "clipboard"
)

type StartPosition string

const (
	StartPositionTop    StartPosition = "top"
	StartPositionMiddle StartPosition = "middle"
	StartPositionBottom StartPosition = "bottom"
)

// CLIOptions holds raw string values from CLI flags before validation.
type CLIOptions struct {
	PaneID          string
	Mode            string
	ScrollbackLines int
	Theme           string

	CursorFG          string
	CursorBG          string
	SelectionFG       string
	SelectionBG       string
	GutterFG          string
	GutterBG          string
	GutterSeparatorFG string
	LineNumAbsoluteFG string
	LineNumRelativeFG string
	LineNumCursorFG   string
	LineNumCursorBold string
	StatusFG          string
	StatusBG          string

	ToggleModeKey string
	CopyTarget    string
	ExitOnYank    string
	StartPosition string
}

// Settings is the validated, typed settings passed into the TUI.
type Settings struct {
	PaneID          string
	Mode            LineNumberMode
	ScrollbackLines int

	Palette theme.Palette

	ToggleModeKey byte
	CopyTarget    CopyTarget
	ExitOnYank    bool
	StartPosition StartPosition
}
