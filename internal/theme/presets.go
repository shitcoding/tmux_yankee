package theme

// Presets maps theme names to their full palette definitions.
//
// StatusBar colors are sourced from official vim-airline theme definitions:
//   - Gruvbox: github.com/morhetz/gruvbox/autoload/airline/themes/gruvbox.vim
//   - Dracula: github.com/dracula/vim/autoload/airline/themes/dracula.vim
//   - Nord:    github.com/nordtheme/vim/autoload/airline/themes/nord.vim
//   - Solarized: github.com/vim-airline/vim-airline-themes/autoload/airline/themes/solarized.vim
var Presets = map[ThemeName]Palette{
	// Default theme uses Gruvbox airline colors.
	ThemeDefault: {
		Cursor:    CellPalette{FG: "#ebdbb2", BG: "#FF8700"},
		Selection: CellPalette{FG: "", BG: "#FF8700"},
		Gutter:    GutterPalette{FG: "#665c54", BG: "", SeparatorChar: "│"},
		LineNum:   LineNumPalette{AbsoluteFG: "#7c6f64", RelativeFG: "#7c6f64", CursorFG: "#FF8700", CursorStyle: TextStyle{Bold: true}},
		Status:    StatusPalette{FG: "#a89984", BG: "#3c3836"},
		StatusBar: StatusBarPalette{
			// Gruvbox airline: Normal=bright_green, Visual=bright_purple, Insert=bright_blue
			ModeNormal:     CellPalette{FG: "#3c3836", BG: "#b8bb26", Style: TextStyle{Bold: true}}, // dark1 on bright_green
			ModeVisualChar: CellPalette{FG: "#3c3836", BG: "#d3869b", Style: TextStyle{Bold: true}}, // dark1 on bright_purple
			ModeVisualLine: CellPalette{FG: "#3c3836", BG: "#83a598", Style: TextStyle{Bold: true}}, // dark1 on bright_blue
			InfoPrimary:    CellPalette{FG: "#ebdbb2", BG: "#504945"},                               // light1 on dark2
			InfoSecondary:  CellPalette{FG: "#fe8019", BG: "#3c3836"},                               // bright_orange on dark1
			Fill:           CellPalette{FG: "#fe8019", BG: "#3c3836"},                               // bright_orange on dark1
		},
	},
	ThemeDracula: {
		Cursor:    CellPalette{FG: "#282a36", BG: "#ffb86c"},
		Selection: CellPalette{FG: "#f8f8f2", BG: "#44475a"},
		Gutter:    GutterPalette{FG: "#6272a4", BG: "", SeparatorChar: "│"},
		LineNum:   LineNumPalette{AbsoluteFG: "#bd93f9", RelativeFG: "#6272a4", CursorFG: "#50fa7b", CursorStyle: TextStyle{Bold: true}},
		Status:    StatusPalette{FG: "#f8f8f2", BG: "#44475a"},
		StatusBar: StatusBarPalette{
			// Dracula airline: Normal=purple, Visual=yellow, Insert=green
			ModeNormal:     CellPalette{FG: "#282a36", BG: "#bd93f9", Style: TextStyle{Bold: true}}, // bg on purple
			ModeVisualChar: CellPalette{FG: "#282a36", BG: "#f1fa8c", Style: TextStyle{Bold: true}}, // bg on yellow
			ModeVisualLine: CellPalette{FG: "#282a36", BG: "#50fa7b", Style: TextStyle{Bold: true}}, // bg on green
			InfoPrimary:    CellPalette{FG: "#f8f8f2", BG: "#6272a4"},                               // fg on comment
			InfoSecondary:  CellPalette{FG: "#f8f8f2", BG: "#44475a"},                               // fg on selection
			Fill:           CellPalette{FG: "#f8f8f2", BG: "#44475a"},                               // fg on selection
		},
	},
	// Gruvbox airline: Normal=bright_green, Visual=bright_purple, Insert=bright_blue
	ThemeGruvbox: {
		Cursor:    CellPalette{FG: "#ebdbb2", BG: "#3c3836"},
		Selection: CellPalette{FG: "", BG: "#665c54"},
		Gutter:    GutterPalette{FG: "#665c54", BG: "", SeparatorChar: "│"},
		LineNum:   LineNumPalette{AbsoluteFG: "#7c6f64", RelativeFG: "#7c6f64", CursorFG: "#fabd2f", CursorStyle: TextStyle{Bold: true}},
		Status:    StatusPalette{FG: "#a89984", BG: "#3c3836"},
		StatusBar: StatusBarPalette{
			ModeNormal:     CellPalette{FG: "#3c3836", BG: "#b8bb26", Style: TextStyle{Bold: true}}, // dark1 on bright_green
			ModeVisualChar: CellPalette{FG: "#3c3836", BG: "#d3869b", Style: TextStyle{Bold: true}}, // dark1 on bright_purple
			ModeVisualLine: CellPalette{FG: "#3c3836", BG: "#83a598", Style: TextStyle{Bold: true}}, // dark1 on bright_blue
			InfoPrimary:    CellPalette{FG: "#ebdbb2", BG: "#504945"},                               // light1 on dark2
			InfoSecondary:  CellPalette{FG: "#fe8019", BG: "#3c3836"},                               // bright_orange on dark1
			Fill:           CellPalette{FG: "#fe8019", BG: "#3c3836"},                               // bright_orange on dark1
		},
	},
	// Nord airline: Normal=nord8(cyan), Visual=nord7(teal), Insert=nord14(green)
	ThemeNord: {
		Cursor:    CellPalette{FG: "#2e3440", BG: "#88c0d0"},
		Selection: CellPalette{FG: "#eceff4", BG: "#5e81ac"},
		Gutter:    GutterPalette{FG: "#4c566a", BG: "", SeparatorChar: "│"},
		LineNum:   LineNumPalette{AbsoluteFG: "#d8dee9", RelativeFG: "#81a1c1", CursorFG: "#a3be8c", CursorStyle: TextStyle{Bold: true}},
		Status:    StatusPalette{FG: "#eceff4", BG: "#3b4252"},
		StatusBar: StatusBarPalette{
			ModeNormal:     CellPalette{FG: "#3b4252", BG: "#88c0d0", Style: TextStyle{Bold: true}}, // nord1 on nord8 (cyan)
			ModeVisualChar: CellPalette{FG: "#3b4252", BG: "#8fbcbb", Style: TextStyle{Bold: true}}, // nord1 on nord7 (teal)
			ModeVisualLine: CellPalette{FG: "#3b4252", BG: "#a3be8c", Style: TextStyle{Bold: true}}, // nord1 on nord14 (green)
			InfoPrimary:    CellPalette{FG: "#e5e9f0", BG: "#81a1c1"},                               // nord5 on nord9 (blue)
			InfoSecondary:  CellPalette{FG: "#e5e9f0", BG: "#4c566a"},                               // nord5 on nord3
			Fill:           CellPalette{FG: "#e5e9f0", BG: "#4c566a"},                               // nord5 on nord3
		},
	},
	// Solarized airline: Normal=green, Visual=magenta, Insert=yellow
	ThemeSolarized: {
		Cursor:    CellPalette{FG: "#002b36", BG: "#cb4b16"},
		Selection: CellPalette{FG: "#eee8d5", BG: "#073642"},
		Gutter:    GutterPalette{FG: "#586e75", BG: "", SeparatorChar: "│"},
		LineNum:   LineNumPalette{AbsoluteFG: "#93a1a1", RelativeFG: "#b58900", CursorFG: "#2aa198", CursorStyle: TextStyle{Bold: true}},
		Status:    StatusPalette{FG: "#eee8d5", BG: "#073642"},
		StatusBar: StatusBarPalette{
			ModeNormal:     CellPalette{FG: "#fdf6e3", BG: "#859900", Style: TextStyle{Bold: true}}, // base3 on green
			ModeVisualChar: CellPalette{FG: "#fdf6e3", BG: "#d33682", Style: TextStyle{Bold: true}}, // base3 on magenta
			ModeVisualLine: CellPalette{FG: "#fdf6e3", BG: "#b58900", Style: TextStyle{Bold: true}}, // base3 on yellow
			InfoPrimary:    CellPalette{FG: "#eee8d5", BG: "#586e75"},                               // base2 on base01
			InfoSecondary:  CellPalette{FG: "#586e75", BG: "#073642"},                               // base01 on base02
			Fill:           CellPalette{FG: "#586e75", BG: "#073642"},                               // base01 on base02
		},
	},
}
