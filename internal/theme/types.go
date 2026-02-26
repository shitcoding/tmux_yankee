package theme

// ThemeName is the name of a built-in color theme.
type ThemeName string

const (
	ThemeDefault   ThemeName = "default"
	ThemeDracula   ThemeName = "dracula"
	ThemeGruvbox   ThemeName = "gruvbox"
	ThemeNord      ThemeName = "nord"
	ThemeSolarized ThemeName = "solarized"
)

// ThemeOrder lists all built-in themes in a stable cycle order.
var ThemeOrder = []ThemeName{ThemeDefault, ThemeDracula, ThemeGruvbox, ThemeNord, ThemeSolarized}

// HexColor is a normalized "#rrggbb" hex color string, or empty string for transparent/default.
type HexColor string

// TextStyle holds per-element text decoration flags.
type TextStyle struct {
	Bold      bool
	Dim       bool
	Italic    bool
	Underline bool
}

// CellPalette holds foreground and background colors for a UI element.
type CellPalette struct {
	FG    HexColor
	BG    HexColor
	Style TextStyle
}

// GutterPalette holds colors for the line-number gutter.
type GutterPalette struct {
	FG             HexColor
	BG             HexColor
	SeparatorFG    HexColor
	SeparatorBG    HexColor
	SeparatorChar  string
	SeparatorStyle TextStyle
}

// LineNumPalette holds colors for line number text within the gutter.
type LineNumPalette struct {
	AbsoluteFG    HexColor
	RelativeFG    HexColor
	CursorFG      HexColor
	CursorStyle   TextStyle
	AbsoluteStyle TextStyle
	RelativeStyle TextStyle
}

// StatusPalette holds colors for the legacy status bar (demo mode fallback).
type StatusPalette struct {
	FG    HexColor
	BG    HexColor
	Style TextStyle
}

// StatusBarPalette holds per-mode colors for the powerline status bar.
type StatusBarPalette struct {
	ModeNormal     CellPalette // NORMAL mode segment
	ModeVisualChar CellPalette // VISUAL mode segment
	ModeVisualLine  CellPalette // V-LINE mode segment
	ModeVisualBlock CellPalette // V-BLOCK mode segment
	InfoPrimary     CellPalette // position/percentage segments
	InfoSecondary  CellPalette // secondary info (wrap, line mode)
	Fill           CellPalette // middle fill area
}

// Palette is the full set of colors used by the TUI.
type Palette struct {
	Cursor        CellPalette
	Selection     CellPalette
	SearchMatch   CellPalette // all search matches (yellow bg)
	SearchCurrent CellPalette // current/active search match (orange/pink bg)
	Gutter        GutterPalette
	LineNum       LineNumPalette
	Status        StatusPalette
	StatusBar     StatusBarPalette
}

// ThemeOverrides holds per-field color overrides supplied via CLI flags.
// An empty string means "use the preset value".
type ThemeOverrides struct {
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
	LineNumCursorBold string // "on"/"off"/"" — backward compat, maps to CursorStyle.Bold
	StatusFG          string
	StatusBG          string

	// TextStyle overrides ("on"/"off"/"")
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
}
