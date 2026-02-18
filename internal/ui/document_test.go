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
