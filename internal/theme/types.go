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

// HexColor is a normalized "#rrggbb" hex color string, or empty string for transparent/default.
type HexColor string

// CellPalette holds foreground and background colors for a UI element.
type CellPalette struct {
	FG   HexColor
	BG   HexColor
	Bold bool
}

// GutterPalette holds colors for the line-number gutter.
type GutterPalette struct {
	FG        HexColor
	BG        HexColor
	Separator HexColor
}

// LineNumPalette holds colors for line number text within the gutter.
type LineNumPalette struct {
	AbsoluteFG HexColor
	RelativeFG HexColor
	CursorFG   HexColor
	CursorBold bool
}

// StatusPalette holds colors for the status bar.
type StatusPalette struct {
	FG HexColor
	BG HexColor
}

// Palette is the full set of colors used by the TUI.
type Palette struct {
	Cursor    CellPalette
	Selection CellPalette
	Gutter    GutterPalette
	LineNum   LineNumPalette
	Status    StatusPalette
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
	LineNumAbsoluteFG string
	LineNumRelativeFG string
	LineNumCursorFG   string
	LineNumCursorBold string // "on"/"off"/""
	StatusFG          string
	StatusBG          string
}
