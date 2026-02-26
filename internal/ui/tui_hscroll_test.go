package ui

import (
	"strings"
	"testing"

	"github.com/shitcoding/tmux_yankee/internal/theme"
)

// --- RenderCellsWithPalette horizontal offset tests ---

func makeCells(s string) []Cell {
	var cells []Cell
	for _, r := range s {
		cells = append(cells, Cell{Rune: r, Style: DefaultStyle()})
	}
	return cells
}

func renderedText(cells []Cell, cursorCol, selStart, selEnd, startCol, maxWidth int) string {
	raw := RenderCellsWithPalette(cells, cursorCol, selStart, selEnd, nil, [2]int{-1, -1}, startCol, maxWidth, theme.Palette{})
	return stripANSI(raw)
}

func TestRenderCells_StartColZero(t *testing.T) {
	cells := makeCells("abcdefghij") // 10 chars
	got := renderedText(cells, -1, -1, -1, 0, 10)
	if got != "abcdefghij" {
		t.Errorf("startCol=0, maxWidth=10: got %q, want %q", got, "abcdefghij")
	}
}

func TestRenderCells_StartColTruncates(t *testing.T) {
	cells := makeCells("abcdefghij") // 10 chars
	// maxWidth=5, startCol=0 -> "abcde"
	got := renderedText(cells, -1, -1, -1, 0, 5)
	if got != "abcde" {
		t.Errorf("startCol=0, maxWidth=5: got %q, want %q", got, "abcde")
	}
}

func TestRenderCells_StartColOffset(t *testing.T) {
	cells := makeCells("abcdefghij") // 10 chars
	// startCol=3, maxWidth=5 -> "defgh"
	got := renderedText(cells, -1, -1, -1, 3, 5)
	if got != "defgh" {
		t.Errorf("startCol=3, maxWidth=5: got %q, want %q", got, "defgh")
	}
}

func TestRenderCells_StartColAtEnd(t *testing.T) {
	cells := makeCells("abcdefghij") // 10 chars
	// startCol=8, maxWidth=5 -> only "ij" remaining
	got := renderedText(cells, -1, -1, -1, 8, 5)
	if got != "ij" {
		t.Errorf("startCol=8, maxWidth=5: got %q, want %q", got, "ij")
	}
}

func TestRenderCells_StartColBeyondEnd(t *testing.T) {
	cells := makeCells("abc")
	got := renderedText(cells, -1, -1, -1, 10, 5)
	if got != "" {
		t.Errorf("startCol=10 (beyond 3 cells): got %q, want %q", got, "")
	}
}

func TestRenderCells_NegativeStartCol(t *testing.T) {
	cells := makeCells("abcde")
	got := renderedText(cells, -1, -1, -1, -5, 5)
	if got != "abcde" {
		t.Errorf("negative startCol should clamp to 0: got %q, want %q", got, "abcde")
	}
}

func testPalette() theme.Palette {
	return theme.Palette{
		Cursor:    theme.CellPalette{FG: "#ff0000", BG: "#0000ff"},
		Selection: theme.CellPalette{FG: "#00ff00", BG: "#ff00ff"},
	}
}

func TestRenderCells_CursorWithOffset(t *testing.T) {
	cells := makeCells("abcdefghij")
	pal := testPalette()
	// startCol=3, maxWidth=5, cursor at col 5 (visible as position 2 in viewport)
	raw := RenderCellsWithPalette(cells, 5, -1, -1, nil, [2]int{-1, -1}, 3, 5, pal)
	// The cursor cell ('f') should have cursor BG color (#0000ff = 0,0,255)
	if !strings.Contains(raw, "48;2;0;0;255") {
		t.Errorf("cursor at col 5 (visible with startCol=3) should have cursor BG, got: %q", raw)
	}
}

func TestRenderCells_CursorOutOfViewport(t *testing.T) {
	cells := makeCells("abcdefghij")
	pal := testPalette()
	// startCol=5, maxWidth=5, cursor at col 2 (before viewport)
	raw := RenderCellsWithPalette(cells, 2, -1, -1, nil, [2]int{-1, -1}, 5, 5, pal)
	// Cursor is before viewport, should NOT have cursor BG
	if strings.Contains(raw, "48;2;0;0;255") {
		t.Error("cursor at col 2 should not be visible with startCol=5")
	}
}

func TestRenderCells_SelectionWithOffset(t *testing.T) {
	cells := makeCells("abcdefghij")
	pal := testPalette()
	// startCol=2, maxWidth=6, selection from col 3 to col 6 (absolute)
	raw := RenderCellsWithPalette(cells, -1, 3, 6, nil, [2]int{-1, -1}, 2, 6, pal)
	stripped := stripANSI(raw)
	// Visible content: "cdefgh" (cols 2-7)
	// Selection covers cols 3-6 within that range
	if stripped != "cdefgh" {
		t.Errorf("visible content should be 'cdefgh', got %q", stripped)
	}
	// Selection styling should be present (selection BG #ff00ff = 255,0,255)
	if !strings.Contains(raw, "48;2;255;0;255") {
		t.Errorf("selection cols 3-6 should have selection BG, got: %q", raw)
	}
}

// --- ensureCursorVisibleH tests ---

func TestEnsureCursorVisibleH_CursorInView(t *testing.T) {
	tui := newTestTUI("test", []string{"abcdefghijklmnop"}, "absolute")
	tui.width = 80
	tui.height = 10
	tui.cursorCol = 5
	tui.hOffset = 0
	tui.ensureCursorVisibleH(20)
	if tui.hOffset != 0 {
		t.Errorf("cursor in view: hOffset should stay 0, got %d", tui.hOffset)
	}
}

func TestEnsureCursorVisibleH_CursorRight(t *testing.T) {
	tui := newTestTUI("test", []string{"abcdefghijklmnop"}, "absolute")
	tui.width = 80
	tui.height = 10
	tui.cursorCol = 25
	tui.hOffset = 0
	tui.ensureCursorVisibleH(10)
	// cursor at 25, contentWidth 10 -> hOffset = 25 - 10 + 1 = 16
	if tui.hOffset != 16 {
		t.Errorf("cursor right of viewport: hOffset should be 16, got %d", tui.hOffset)
	}
}

func TestEnsureCursorVisibleH_CursorLeft(t *testing.T) {
	tui := newTestTUI("test", []string{"abcdefghijklmnop"}, "absolute")
	tui.width = 80
	tui.height = 10
	tui.cursorCol = 3
	tui.hOffset = 10
	tui.ensureCursorVisibleH(20)
	// cursor at 3, hOffset at 10 -> should snap to 3
	if tui.hOffset != 3 {
		t.Errorf("cursor left of viewport: hOffset should be 3, got %d", tui.hOffset)
	}
}

func TestEnsureCursorVisibleH_ZeroWidth(t *testing.T) {
	tui := newTestTUI("test", []string{"abc"}, "absolute")
	tui.width = 80
	tui.height = 10
	tui.hOffset = 5
	tui.ensureCursorVisibleH(0)
	if tui.hOffset != 0 {
		t.Errorf("zero contentWidth: hOffset should reset to 0, got %d", tui.hOffset)
	}
}

func TestEnsureCursorVisibleH_PersistsAcrossLines(t *testing.T) {
	tui := newTestTUI("test", []string{
		"short",
		"this is a much longer line that extends beyond the viewport",
	}, "absolute")
	tui.width = 20
	tui.height = 10
	// Simulate being scrolled right on long line
	tui.cursorLine = 1
	tui.cursorCol = 30
	tui.hOffset = 20
	tui.ensureCursorVisibleH(15)
	// cursor at 30, hOffset at 20, contentWidth=15 -> visible range [20,35), cursor 30 is in range
	if tui.hOffset != 20 {
		t.Errorf("long line: hOffset should stay 20 (cursor in range), got %d", tui.hOffset)
	}

	// Now move to short line (cursor gets clamped to col 0 by clampViewportAndCursor)
	tui.cursorLine = 0
	tui.cursorCol = 0
	tui.ensureCursorVisibleH(15)
	// cursor at 0, hOffset at 16 -> snaps left to 0
	if tui.hOffset != 0 {
		t.Errorf("short line after long: hOffset should snap to 0, got %d", tui.hOffset)
	}
}
