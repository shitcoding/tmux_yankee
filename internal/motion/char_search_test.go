package motion

import "testing"

type charSearchDoc struct {
	lines []string
}

func (d *charSearchDoc) LineCount() int { return len(d.lines) }
func (d *charSearchDoc) Line(i int) string {
	if i >= 0 && i < len(d.lines) {
		return d.lines[i]
	}
	return ""
}
func (d *charSearchDoc) LineRuneCount(i int) int { return len([]rune(d.Line(i))) }

func TestApplyCharSearch_FindForward(t *testing.T) {
	doc := &charSearchDoc{lines: []string{"hello world"}}
	h := NewVimHandler()

	tests := []struct {
		name    string
		cursor  Cursor
		char    byte
		count   int
		wantCol int
	}{
		{"f finds first match", Cursor{0, 0}, 'o', 1, 4},
		{"f finds second match with count", Cursor{0, 0}, 'o', 2, 7},
		{"f no match stays put", Cursor{0, 0}, 'z', 1, 0},
		{"f from middle of line", Cursor{0, 5}, 'o', 1, 7},
		{"f count exceeds matches stays put", Cursor{0, 0}, 'o', 5, 0},
		{"f on current char skips it", Cursor{0, 4}, 'o', 1, 7},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := h.ApplyCharSearch(doc, tt.cursor, CharSearchFindForward, tt.char, tt.count)
			if got.Col != tt.wantCol {
				t.Errorf("col = %d, want %d", got.Col, tt.wantCol)
			}
			if got.Line != tt.cursor.Line {
				t.Errorf("line = %d, want %d", got.Line, tt.cursor.Line)
			}
		})
	}
}

func TestApplyCharSearch_TillForward(t *testing.T) {
	doc := &charSearchDoc{lines: []string{"hello world"}}
	h := NewVimHandler()

	tests := []struct {
		name    string
		cursor  Cursor
		char    byte
		count   int
		wantCol int
	}{
		{"t stops before match", Cursor{0, 0}, 'o', 1, 3},
		{"t with count=2 stops before 2nd", Cursor{0, 0}, 'o', 2, 6},
		{"t no match stays put", Cursor{0, 0}, 'z', 1, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := h.ApplyCharSearch(doc, tt.cursor, CharSearchTillForward, tt.char, tt.count)
			if got.Col != tt.wantCol {
				t.Errorf("col = %d, want %d", got.Col, tt.wantCol)
			}
		})
	}
}

func TestApplyCharSearch_FindBackward(t *testing.T) {
	doc := &charSearchDoc{lines: []string{"hello world"}}
	h := NewVimHandler()

	tests := []struct {
		name    string
		cursor  Cursor
		char    byte
		count   int
		wantCol int
	}{
		{"F finds match backward", Cursor{0, 10}, 'o', 1, 7},
		{"F finds 2nd match backward", Cursor{0, 10}, 'o', 2, 4},
		{"F no match stays put", Cursor{0, 10}, 'z', 1, 10},
		{"F from middle", Cursor{0, 6}, 'l', 1, 3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := h.ApplyCharSearch(doc, tt.cursor, CharSearchFindBackward, tt.char, tt.count)
			if got.Col != tt.wantCol {
				t.Errorf("col = %d, want %d", got.Col, tt.wantCol)
			}
		})
	}
}

func TestApplyCharSearch_TillBackward(t *testing.T) {
	doc := &charSearchDoc{lines: []string{"hello world"}}
	h := NewVimHandler()

	tests := []struct {
		name    string
		cursor  Cursor
		char    byte
		count   int
		wantCol int
	}{
		{"T stops after match backward", Cursor{0, 10}, 'o', 1, 8},
		{"T with count=2", Cursor{0, 10}, 'o', 2, 5},
		{"T no match stays put", Cursor{0, 10}, 'z', 1, 10},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := h.ApplyCharSearch(doc, tt.cursor, CharSearchTillBackward, tt.char, tt.count)
			if got.Col != tt.wantCol {
				t.Errorf("col = %d, want %d", got.Col, tt.wantCol)
			}
		})
	}
}

func TestRepeatCharSearch(t *testing.T) {
	doc := &charSearchDoc{lines: []string{"abracadabra"}}
	h := NewVimHandler()

	// No prior search — no-op
	got := h.RepeatCharSearch(doc, Cursor{0, 0}, 1)
	if got.Col != 0 {
		t.Errorf("no prior search: col = %d, want 0", got.Col)
	}

	// fa from col 0 → 'a' at col 3 (skips col 0 since search starts at col+1)
	h.ApplyCharSearch(doc, Cursor{0, 0}, CharSearchFindForward, 'a', 1)

	// Repeat from col 3 → next 'a' at col 5
	got = h.RepeatCharSearch(doc, Cursor{0, 3}, 1)
	if got.Col != 5 {
		t.Errorf("repeat: col = %d, want 5", got.Col)
	}
}

func TestRepeatCharSearchReverse(t *testing.T) {
	doc := &charSearchDoc{lines: []string{"abracadabra"}}
	h := NewVimHandler()

	// fa from col 0 → col 3
	h.ApplyCharSearch(doc, Cursor{0, 0}, CharSearchFindForward, 'a', 1)

	// Reverse from col 5 → backward find 'a' → col 3
	got := h.RepeatCharSearchReverse(doc, Cursor{0, 5}, 1)
	if got.Col != 3 {
		t.Errorf("reverse: col = %d, want 3", got.Col)
	}
}

func TestRepeatCharSearch_AfterTill(t *testing.T) {
	// "abracadabra" = a(0)b(1)r(2)a(3)c(4)a(5)d(6)a(7)b(8)r(9)a(10)
	// ta from col 0: finds 'a' at col 3, till → col 2
	// ; repeat from col 2: with repeat skip, start = 2+1+1=4, finds 'a' at col 5, till → col 4
	doc := &charSearchDoc{lines: []string{"abracadabra"}}
	h := NewVimHandler()

	got := h.ApplyCharSearch(doc, Cursor{0, 0}, CharSearchTillForward, 'a', 1)
	if got.Col != 2 {
		t.Fatalf("ta from 0: col = %d, want 2", got.Col)
	}

	got = h.RepeatCharSearch(doc, Cursor{0, 2}, 1)
	if got.Col != 4 {
		t.Errorf("; after ta: col = %d, want 4", got.Col)
	}
}

func TestRepeatCharSearchReverse_AfterFindBackward(t *testing.T) {
	// "abracadabra" — Fa from col 10 should land at col 7
	// , (reverse) from col 7 should search forward for 'a' → col 10
	// "abracadabra" = a(0)b(1)r(2)a(3)c(4)a(5)d(6)a(7)b(8)r(9)a(10)
	// Fa reverses to find-forward; from col 7, search starts at col 8 → a at col 10
	doc := &charSearchDoc{lines: []string{"abracadabra"}}
	h := NewVimHandler()

	got := h.ApplyCharSearch(doc, Cursor{0, 10}, CharSearchFindBackward, 'a', 1)
	if got.Col != 7 {
		t.Fatalf("Fa from 10: col = %d, want 7", got.Col)
	}

	got = h.RepeatCharSearchReverse(doc, Cursor{0, 7}, 1)
	if got.Col != 10 {
		t.Errorf(", after Fa: col = %d, want 10", got.Col)
	}
}

func TestApplyCharSearch_NonASCIINoFalseMatch(t *testing.T) {
	// U+0161 (š) has byte value 0x61 which is 'a' — must NOT match 'a'
	doc := &charSearchDoc{lines: []string{"xšy"}}
	h := NewVimHandler()
	got := h.ApplyCharSearch(doc, Cursor{0, 0}, CharSearchFindForward, 'a', 1)
	if got.Col != 0 {
		t.Errorf("non-ASCII false match: col = %d, want 0 (no match)", got.Col)
	}
}

func TestApplyCharSearch_Latin1Character(t *testing.T) {
	// 'é' (U+00E9, byte 0xE9) should be findable via char search
	doc := &charSearchDoc{lines: []string{"café"}}
	h := NewVimHandler()
	got := h.ApplyCharSearch(doc, Cursor{0, 0}, CharSearchFindForward, 0xE9, 1)
	if got.Col != 3 {
		t.Errorf("Latin-1 char search: col = %d, want 3", got.Col)
	}
}

func TestApplyCharSearch_Latin1Backward(t *testing.T) {
	// Backward search for 'é' from end
	doc := &charSearchDoc{lines: []string{"café latte"}}
	h := NewVimHandler()
	got := h.ApplyCharSearch(doc, Cursor{0, 9}, CharSearchFindBackward, 0xE9, 1)
	if got.Col != 3 {
		t.Errorf("Latin-1 backward: col = %d, want 3", got.Col)
	}
}

func TestApplyCharSearch_EmptyLine(t *testing.T) {
	doc := &charSearchDoc{lines: []string{""}}
	h := NewVimHandler()
	got := h.ApplyCharSearch(doc, Cursor{0, 0}, CharSearchFindForward, 'a', 1)
	if got.Col != 0 {
		t.Errorf("empty: col = %d, want 0", got.Col)
	}
}

func TestApplyCharSearch_CountZeroDefaultsToOne(t *testing.T) {
	doc := &charSearchDoc{lines: []string{"hello"}}
	h := NewVimHandler()
	got := h.ApplyCharSearch(doc, Cursor{0, 0}, CharSearchFindForward, 'l', 0)
	if got.Col != 2 {
		t.Errorf("count=0: col = %d, want 2", got.Col)
	}
}
