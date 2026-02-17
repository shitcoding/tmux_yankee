package theme

// Presets maps theme names to their full palette definitions.
var Presets = map[ThemeName]Palette{
	ThemeDefault: {
		Cursor:    CellPalette{FG: "#1d2021", BG: "#fe8018"},
		Selection: CellPalette{FG: "#fbf1c7", BG: "#458588"},
		Gutter:    GutterPalette{FG: "#a89984", BG: "", Separator: "#665c54"},
		LineNum:   LineNumPalette{AbsoluteFG: "#d5c4a1", RelativeFG: "#fabd2f", CursorFG: "#b8bb26", CursorBold: true},
		Status:    StatusPalette{FG: "#ebdbb2", BG: "#3c3836"},
	},
	ThemeDracula: {
		Cursor:    CellPalette{FG: "#282a36", BG: "#ffb86c"},
		Selection: CellPalette{FG: "#f8f8f2", BG: "#44475a"},
		Gutter:    GutterPalette{FG: "#6272a4", BG: "", Separator: "#44475a"},
		LineNum:   LineNumPalette{AbsoluteFG: "#bd93f9", RelativeFG: "#6272a4", CursorFG: "#50fa7b", CursorBold: true},
		Status:    StatusPalette{FG: "#f8f8f2", BG: "#44475a"},
	},
	ThemeGruvbox: {
		Cursor:    CellPalette{FG: "#282828", BG: "#fe8019"},
		Selection: CellPalette{FG: "#fbf1c7", BG: "#458588"},
		Gutter:    GutterPalette{FG: "#928374", BG: "", Separator: "#665c54"},
		LineNum:   LineNumPalette{AbsoluteFG: "#d5c4a1", RelativeFG: "#d79921", CursorFG: "#b8bb26", CursorBold: true},
		Status:    StatusPalette{FG: "#ebdbb2", BG: "#3c3836"},
	},
	ThemeNord: {
		Cursor:    CellPalette{FG: "#2e3440", BG: "#88c0d0"},
		Selection: CellPalette{FG: "#eceff4", BG: "#5e81ac"},
		Gutter:    GutterPalette{FG: "#4c566a", BG: "", Separator: "#434c5e"},
		LineNum:   LineNumPalette{AbsoluteFG: "#d8dee9", RelativeFG: "#81a1c1", CursorFG: "#a3be8c", CursorBold: true},
		Status:    StatusPalette{FG: "#eceff4", BG: "#3b4252"},
	},
	ThemeSolarized: {
		Cursor:    CellPalette{FG: "#002b36", BG: "#cb4b16"},
		Selection: CellPalette{FG: "#eee8d5", BG: "#073642"},
		Gutter:    GutterPalette{FG: "#586e75", BG: "", Separator: "#657b83"},
		LineNum:   LineNumPalette{AbsoluteFG: "#93a1a1", RelativeFG: "#b58900", CursorFG: "#2aa198", CursorBold: true},
		Status:    StatusPalette{FG: "#eee8d5", BG: "#073642"},
	},
}
