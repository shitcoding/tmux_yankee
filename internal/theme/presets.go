package theme

// Presets maps theme names to their full palette definitions.
var Presets = map[ThemeName]Palette{
	ThemeDefault: {
		Cursor:    CellPalette{FG: "#ebdbb2", BG: "#FF8700"},
		Selection: CellPalette{FG: "", BG: "#FF8700"},
		Gutter:    GutterPalette{FG: "#665c54", BG: "", SeparatorChar: "│"},
		LineNum:   LineNumPalette{AbsoluteFG: "#7c6f64", RelativeFG: "#7c6f64", CursorFG: "#FF8700", CursorStyle: TextStyle{Bold: true}},
		Status:    StatusPalette{FG: "#a89984", BG: "#3c3836"},
	},
	ThemeDracula: {
		Cursor:    CellPalette{FG: "#282a36", BG: "#ffb86c"},
		Selection: CellPalette{FG: "#f8f8f2", BG: "#44475a"},
		Gutter:    GutterPalette{FG: "#6272a4", BG: "", SeparatorChar: "│"},
		LineNum:   LineNumPalette{AbsoluteFG: "#bd93f9", RelativeFG: "#6272a4", CursorFG: "#50fa7b", CursorStyle: TextStyle{Bold: true}},
		Status:    StatusPalette{FG: "#f8f8f2", BG: "#44475a"},
	},
	ThemeGruvbox: {
		Cursor:    CellPalette{FG: "#ebdbb2", BG: "#3c3836"},                         // CursorLine: fg1 on bg1
		Selection: CellPalette{FG: "", BG: "#665c54"},                                 // Visual: bg3
		Gutter:    GutterPalette{FG: "#665c54", BG: "", SeparatorChar: "│"},            // VertSplit: bg3
		LineNum:   LineNumPalette{AbsoluteFG: "#7c6f64", RelativeFG: "#7c6f64", CursorFG: "#fabd2f", CursorStyle: TextStyle{Bold: true}}, // LineNr: bg4, CursorLineNr: bright_yellow
		Status:    StatusPalette{FG: "#a89984", BG: "#3c3836"},                        // StatusLineNC: fg4 on bg1
	},
	ThemeNord: {
		Cursor:    CellPalette{FG: "#2e3440", BG: "#88c0d0"},
		Selection: CellPalette{FG: "#eceff4", BG: "#5e81ac"},
		Gutter:    GutterPalette{FG: "#4c566a", BG: "", SeparatorChar: "│"},
		LineNum:   LineNumPalette{AbsoluteFG: "#d8dee9", RelativeFG: "#81a1c1", CursorFG: "#a3be8c", CursorStyle: TextStyle{Bold: true}},
		Status:    StatusPalette{FG: "#eceff4", BG: "#3b4252"},
	},
	ThemeSolarized: {
		Cursor:    CellPalette{FG: "#002b36", BG: "#cb4b16"},
		Selection: CellPalette{FG: "#eee8d5", BG: "#073642"},
		Gutter:    GutterPalette{FG: "#586e75", BG: "", SeparatorChar: "│"},
		LineNum:   LineNumPalette{AbsoluteFG: "#93a1a1", RelativeFG: "#b58900", CursorFG: "#2aa198", CursorStyle: TextStyle{Bold: true}},
		Status:    StatusPalette{FG: "#eee8d5", BG: "#073642"},
	},
}
