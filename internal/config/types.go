package config

type LineNumberMode string

const (
	LineNumberModeAbsolute LineNumberMode = "absolute"
	LineNumberModeRelative LineNumberMode = "relative"
	LineNumberModeHybrid   LineNumberMode = "hybrid"
)

type ThemeName string

const (
	ThemeDefault   ThemeName = "default"
	ThemeDracula   ThemeName = "dracula"
	ThemeGruvbox   ThemeName = "gruvbox"
	ThemeNord      ThemeName = "nord"
	ThemeSolarized ThemeName = "solarized"
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

type HexColor string // normalized "#rrggbb" or ""

type CellPalette struct {
	FG   HexColor
	BG   HexColor
	Bold bool
}

type GutterPalette struct {
	FG        HexColor
	BG        HexColor
	Separator HexColor
}

type LineNumPalette struct {
	AbsoluteFG HexColor
	RelativeFG HexColor
	CursorFG   HexColor
	CursorBold bool
}

type StatusPalette struct {
	FG HexColor
	BG HexColor
}

type Palette struct {
	Cursor    CellPalette
	Selection CellPalette
	Gutter    GutterPalette
	LineNum   LineNumPalette
	Status    StatusPalette
}

type ThemeOverrides struct {
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
	LineNumCursorBold string // "on"/"off"/""
	StatusFG          string
	StatusBG          string
}

// CLIOptions holds raw string values from CLI flags before validation.
type CLIOptions struct {
	PaneID          string
	Mode            string
	ScrollbackLines int
	Theme           string
	StatusIndicator string

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

	Palette         Palette
	StatusIndicator bool

	ToggleModeKey byte
	CopyTarget    CopyTarget
	ExitOnYank    bool
	StartPosition StartPosition
}
