package config

import (
	"github.com/shitcoding/tmux_yankee/internal/keymap"
	"github.com/shitcoding/tmux_yankee/internal/theme"
)

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
	WrapModeOff WrapMode = "off"
	WrapModeOn  WrapMode = "on"
)

type StatusBarMode string

const (
	StatusBarOn  StatusBarMode = "on"
	StatusBarOff StatusBarMode = "off"
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
	WrapKey       string
	CopyTarget    string
	ExitOnYank    string
	StartPosition string
	WrapMode      string
	StatusBar      string
	Bindings       string
	NormalBindings string
	VisualBindings string

	// Flash navigation
	Flash         string // "on" or "off"
	FlashMinChars string // digits: minimum chars before labels
	FlashFT       string // "on" or "off"
	FlashLabelFG  string
	FlashLabelBG  string
	FlashMatchFG  string
	FlashMatchBG  string
	FlashBackdrop   string
	FlashJumpPos    string // "match_start", "match_end", "word_start", "word_end"
	FlashAltJumpPos string // same values + "off"
}

// Settings is the validated, typed settings passed into the TUI.
type Settings struct {
	PaneID          string
	Mode            LineNumberMode
	ScrollbackLines int
	Demo            bool
	ThemeName       string

	Palette theme.Palette

	ToggleModeKey byte
	WrapKey       byte
	CopyTarget    CopyTarget
	ExitOnYank    bool
	StartPosition StartPosition
	WrapMode      WrapMode
	StatusBar     StatusBarMode
	ModeKeymap    keymap.ModeKeymap

	// Flash navigation
	FlashEnabled   bool
	FlashMinChars  int
	FlashFTEnabled  bool
	FlashJumpPos    int // matches flash.JumpPos values
	FlashAltJumpPos int // matches flash.JumpPos values
}
