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

type WrapMode string

const (
	WrapModeScroll WrapMode = "scroll"
	WrapModeWrap   WrapMode = "wrap"
)

// CLIOptions holds raw string values from CLI flags before validation.
type CLIOptions struct {
	PaneID          string
	Mode            string
	ScrollbackLines int
	Theme           string
	Demo            bool

	CursorFG          string
	CursorBG          string
	SelectionFG       string
	SelectionBG       string
	GutterFG          string
	GutterBG          string
	GutterSeparatorFG string
	GutterSeparatorBG string
	GutterSeparatorChar string
	LineNumAbsoluteFG string
	LineNumRelativeFG string
	LineNumCursorFG   string
	LineNumCursorBold string
	StatusFG          string
	StatusBG          string

	// TextStyle override flags ("on"/"off"/"")
	LineNumAbsoluteBold   string
	LineNumAbsoluteDim    string
	LineNumAbsoluteItalic string
	LineNumRelativeBold   string
	LineNumRelativeDim    string
	LineNumRelativeItalic string
	LineNumCursorDim      string
	LineNumCursorItalic   string
	StatusBold            string
	StatusDim             string
	CursorDim             string
	CursorItalic          string
	SelectionDim          string
	SelectionItalic       string

	ToggleModeKey string
	CopyTarget    string
	ExitOnYank    string
	StartPosition string
	WrapMode      string
}

// Settings is the validated, typed settings passed into the TUI.
type Settings struct {
	PaneID          string
	Mode            LineNumberMode
	ScrollbackLines int
	Demo            bool

	Palette theme.Palette

	ToggleModeKey byte
	CopyTarget    CopyTarget
	ExitOnYank    bool
	StartPosition StartPosition
	WrapMode      WrapMode
}
