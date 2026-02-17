package ui

import (
	"strings"
	"testing"

	"github.com/shitcoding/tmux_yankee/internal/theme"
)

// TestRenderCell_CursorColorFromPalette verifies the cursor cell uses palette.Cursor.BG/FG
// rather than any hardcoded orange color.
func TestRenderCell_CursorColorFromPalette(t *testing.T) {
	pal := theme.Palette{
		Cursor: theme.CellPalette{FG: "#ff0000", BG: "#0000ff"},
	}

	cell := Cell{Rune: 'A', Style: DefaultStyle()}
	rendered := RenderCellWithPalette(cell, true, false, pal)

	// Must contain the blue background we set (#0000ff = rgb(0,0,255))
	if !strings.Contains(rendered, "48;2;0;0;255") {
		t.Errorf("cursor cell should use palette.Cursor.BG (#0000ff), got: %q", rendered)
	}

	// Must contain the red foreground we set (#ff0000 = rgb(255,0,0))
	if !strings.Contains(rendered, "38;2;255;0;0") {
		t.Errorf("cursor cell should use palette.Cursor.FG (#ff0000), got: %q", rendered)
	}

	// Must NOT contain the old hardcoded orange background (254,128,24)
	if strings.Contains(rendered, "48;2;254;128;24") {
		t.Errorf("cursor cell must not use hardcoded orange background, got: %q", rendered)
	}
}

// TestRenderCell_SelectionColorFromPalette verifies the selection cell uses palette.Selection.BG/FG
// rather than any hardcoded blue color.
func TestRenderCell_SelectionColorFromPalette(t *testing.T) {
	pal := theme.Palette{
		Selection: theme.CellPalette{FG: "#00ff00", BG: "#ff00ff"},
	}

	cell := Cell{Rune: 'B', Style: DefaultStyle()}
	rendered := RenderCellWithPalette(cell, false, true, pal)

	// Must contain the magenta background we set (#ff00ff = rgb(255,0,255))
	if !strings.Contains(rendered, "48;2;255;0;255") {
		t.Errorf("selection cell should use palette.Selection.BG (#ff00ff), got: %q", rendered)
	}

	// Must contain the green foreground we set (#00ff00 = rgb(0,255,0))
	if !strings.Contains(rendered, "38;2;0;255;0") {
		t.Errorf("selection cell should use palette.Selection.FG (#00ff00), got: %q", rendered)
	}

	// Must NOT contain any hardcoded blue-ish backgrounds
	if strings.Contains(rendered, "48;2;70;130;180") {
		t.Errorf("selection cell must not use hardcoded blue background, got: %q", rendered)
	}
}

// TestRenderCell_EmptyPaletteUsesTerminalDefault verifies that empty HexColor
// does not emit any background/foreground escape sequences.
func TestRenderCell_EmptyPaletteUsesTerminalDefault(t *testing.T) {
	pal := theme.Palette{
		Cursor: theme.CellPalette{FG: "", BG: ""},
	}

	cell := Cell{Rune: 'C', Style: DefaultStyle()}
	rendered := RenderCellWithPalette(cell, true, false, pal)

	// With empty colors, no 48;2 or 38;2 sequences should appear
	if strings.Contains(rendered, "48;2") {
		t.Errorf("empty BG should not emit background color sequence, got: %q", rendered)
	}
	if strings.Contains(rendered, "38;2") {
		t.Errorf("empty FG should not emit foreground color sequence, got: %q", rendered)
	}
}
