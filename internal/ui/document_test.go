package ui

import "testing"

func TestDocument_CellsCached(t *testing.T) {
	raw := "\x1b[1;31mA\x1b[0mB"
	doc := NewDocument([]string{raw})

	cells := doc.Cells(0)
	if len(cells) != 2 {
		t.Fatalf("Cells(0): got %d cells, want 2", len(cells))
	}
	if cells[0].Rune != 'A' {
		t.Errorf("cells[0].Rune: got %q, want 'A'", cells[0].Rune)
	}
	if !cells[0].Style.Bold {
		t.Errorf("cells[0].Style.Bold: got false, want true")
	}
	if cells[1].Rune != 'B' {
		t.Errorf("cells[1].Rune: got %q, want 'B'", cells[1].Rune)
	}
	if cells[1].Style.Bold {
		t.Errorf("cells[1].Style.Bold: got true, want false (reset)")
	}

	// Second call must return the same slice (identity check)
	cells2 := doc.Cells(0)
	if &cells[0] != &cells2[0] {
		t.Errorf("Cells(0) returned different slices — caching not working")
	}
}

func TestDocument_CellsOutOfBounds(t *testing.T) {
	doc := NewDocument([]string{"hello"})
	if cells := doc.Cells(-1); cells != nil {
		t.Errorf("Cells(-1): got %v, want nil", cells)
	}
	if cells := doc.Cells(1); cells != nil {
		t.Errorf("Cells(1): got %v, want nil", cells)
	}
}

func TestDocument_CellsEmptyLine(t *testing.T) {
	doc := NewDocument([]string{""})
	cells := doc.Cells(0)
	if len(cells) != 0 {
		t.Errorf("Cells(0) on empty line: got %d cells, want 0", len(cells))
	}
}

func TestDocument_CellsMultipleLines(t *testing.T) {
	doc := NewDocument([]string{"AB", "\x1b[32mCD\x1b[0m", "EF"})

	if n := len(doc.Cells(0)); n != 2 {
		t.Errorf("line 0: got %d cells, want 2", n)
	}
	if n := len(doc.Cells(1)); n != 2 {
		t.Errorf("line 1: got %d cells, want 2", n)
	}
	if n := len(doc.Cells(2)); n != 2 {
		t.Errorf("line 2: got %d cells, want 2", n)
	}

	// Verify line 1 has green foreground
	cells := doc.Cells(1)
	if cells[0].Rune != 'C' {
		t.Errorf("line 1 cells[0].Rune: got %q, want 'C'", cells[0].Rune)
	}
	if cells[0].Style.FgColor != 32 {
		t.Errorf("line 1 cells[0].Style.FgColor: got %d, want 32", cells[0].Style.FgColor)
	}
}

func TestParseANSILine_Truecolor24Bit(t *testing.T) {
	// 24-bit foreground: ESC[38;2;255;128;0m  (orange foreground)
	// 24-bit background: ESC[48;2;0;0;128m    (dark blue background)
	raw := "\x1b[38;2;255;128;0mA\x1b[48;2;0;0;128mB\x1b[0mC"
	cells := ParseANSILine(raw)

	if len(cells) != 3 {
		t.Fatalf("got %d cells, want 3", len(cells))
	}

	// Cell A: 24-bit foreground, no background
	if cells[0].Rune != 'A' {
		t.Errorf("cells[0].Rune: got %q, want 'A'", cells[0].Rune)
	}
	if cells[0].Style.FgColor != -1 {
		t.Errorf("cells[0].FgColor: got %d, want -1 (truecolor sentinel)", cells[0].Style.FgColor)
	}
	if cells[0].Style.FgR != 255 || cells[0].Style.FgG != 128 || cells[0].Style.FgB != 0 {
		t.Errorf("cells[0].Fg RGB: got (%d,%d,%d), want (255,128,0)",
			cells[0].Style.FgR, cells[0].Style.FgG, cells[0].Style.FgB)
	}

	// Cell B: 24-bit foreground (inherited) + 24-bit background
	if cells[1].Rune != 'B' {
		t.Errorf("cells[1].Rune: got %q, want 'B'", cells[1].Rune)
	}
	if cells[1].Style.BgColor != -1 {
		t.Errorf("cells[1].BgColor: got %d, want -1 (truecolor sentinel)", cells[1].Style.BgColor)
	}
	if cells[1].Style.BgR != 0 || cells[1].Style.BgG != 0 || cells[1].Style.BgB != 128 {
		t.Errorf("cells[1].Bg RGB: got (%d,%d,%d), want (0,0,128)",
			cells[1].Style.BgR, cells[1].Style.BgG, cells[1].Style.BgB)
	}

	// Cell C: reset — back to default
	if cells[2].Rune != 'C' {
		t.Errorf("cells[2].Rune: got %q, want 'C'", cells[2].Rune)
	}
	if cells[2].Style.FgColor != 0 {
		t.Errorf("cells[2].FgColor: got %d, want 0 (default)", cells[2].Style.FgColor)
	}
	if cells[2].Style.BgColor != 0 {
		t.Errorf("cells[2].BgColor: got %d, want 0 (default)", cells[2].Style.BgColor)
	}
}
