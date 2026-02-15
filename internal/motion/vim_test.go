package motion

import "testing"

// mockDocument is a simple in-memory document for testing.
type mockDocument struct {
	lines []string
}

func newMockDocument(lines []string) *mockDocument {
	return &mockDocument{lines: lines}
}

func (m *mockDocument) LineCount() int {
	return len(m.lines)
}

func (m *mockDocument) Line(index int) string {
	if index < 0 || index >= len(m.lines) {
		return ""
	}
	return m.lines[index]
}

func (m *mockDocument) LineRuneCount(index int) int {
	if index < 0 || index >= len(m.lines) {
		return 0
	}
	return len([]rune(m.lines[index]))
}

func TestVimHandler_MotionUp(t *testing.T) {
	doc := newMockDocument([]string{
		"line 0",
		"line 1",
		"line 2",
		"line 3",
	})

	tests := []struct {
		name     string
		cursor   Cursor
		count    int
		expected Cursor
	}{
		{
			name:     "single up from line 2",
			cursor:   Cursor{Line: 2, Col: 3},
			count:    1,
			expected: Cursor{Line: 1, Col: 3},
		},
		{
			name:     "multi up (3k from line 3)",
			cursor:   Cursor{Line: 3, Col: 2},
			count:    3,
			expected: Cursor{Line: 0, Col: 2},
		},
		{
			name:     "up beyond bounds (clamps to 0)",
			cursor:   Cursor{Line: 1, Col: 0},
			count:    5,
			expected: Cursor{Line: 0, Col: 0},
		},
		{
			name:     "up from line 0 (no-op)",
			cursor:   Cursor{Line: 0, Col: 4},
			count:    1,
			expected: Cursor{Line: 0, Col: 4},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := NewVimHandler()
			viewport := Viewport{Top: 0, Height: 10}
			result := handler.Apply(doc, tt.cursor, viewport, MotionUp, tt.count)

			if result.Cursor != tt.expected {
				t.Errorf("expected cursor %+v, got %+v", tt.expected, result.Cursor)
			}
		})
	}
}

func TestVimHandler_MotionDown(t *testing.T) {
	doc := newMockDocument([]string{
		"line 0",
		"line 1",
		"line 2",
		"line 3",
	})

	tests := []struct {
		name     string
		cursor   Cursor
		count    int
		expected Cursor
	}{
		{
			name:     "single down from line 0",
			cursor:   Cursor{Line: 0, Col: 2},
			count:    1,
			expected: Cursor{Line: 1, Col: 2},
		},
		{
			name:     "multi down (2j from line 0)",
			cursor:   Cursor{Line: 0, Col: 3},
			count:    2,
			expected: Cursor{Line: 2, Col: 3},
		},
		{
			name:     "down beyond bounds (clamps to last line)",
			cursor:   Cursor{Line: 2, Col: 1},
			count:    10,
			expected: Cursor{Line: 3, Col: 1},
		},
		{
			name:     "down from last line (no-op)",
			cursor:   Cursor{Line: 3, Col: 4},
			count:    1,
			expected: Cursor{Line: 3, Col: 4},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := NewVimHandler()
			viewport := Viewport{Top: 0, Height: 10}
			result := handler.Apply(doc, tt.cursor, viewport, MotionDown, tt.count)

			if result.Cursor != tt.expected {
				t.Errorf("expected cursor %+v, got %+v", tt.expected, result.Cursor)
			}
		})
	}
}

func TestVimHandler_GoalColumn(t *testing.T) {
	doc := newMockDocument([]string{
		"short",               // 5 chars
		"very long line here", // 20 chars
		"mid",                 // 3 chars
		"another long line",   // 18 chars
	})

	handler := NewVimHandler()
	viewport := Viewport{Top: 0, Height: 10}

	// Start at column 10 on long line
	cursor := Cursor{Line: 1, Col: 10}

	// Move down to short line (col should clamp to 3)
	result := handler.Apply(doc, cursor, viewport, MotionDown, 1)
	if result.Cursor.Line != 2 || result.Cursor.Col != 3 {
		t.Errorf("expected cursor at line 2 col 3, got %+v", result.Cursor)
	}

	// Move down to another long line (col should restore to 10)
	result = handler.Apply(doc, result.Cursor, viewport, MotionDown, 1)
	if result.Cursor.Line != 3 || result.Cursor.Col != 10 {
		t.Errorf("expected cursor at line 3 col 10 (goal column), got %+v", result.Cursor)
	}
}

func TestVimHandler_GoalColumnResetOnHorizontal(t *testing.T) {
	doc := newMockDocument([]string{
		"first line here",
		"second line",
		"third line here",
	})

	handler := NewVimHandler()
	viewport := Viewport{Top: 0, Height: 10}

	// Start at col 10
	cursor := Cursor{Line: 0, Col: 10}

	// Move down (goal col = 10)
	result := handler.Apply(doc, cursor, viewport, MotionDown, 1)

	// Move left (should set new goal col = 9)
	result = handler.Apply(doc, result.Cursor, viewport, MotionLeft, 1)
	if result.Cursor.Col != 9 {
		t.Errorf("expected col 9 after left, got %d", result.Cursor.Col)
	}

	// Move down again (should use new goal col = 9, not 10)
	result = handler.Apply(doc, result.Cursor, viewport, MotionDown, 1)
	if result.Cursor.Col != 9 {
		t.Errorf("expected col 9 (new goal column), got %d", result.Cursor.Col)
	}
}

func TestVimHandler_MotionLeft(t *testing.T) {
	doc := newMockDocument([]string{"hello world"})

	tests := []struct {
		name     string
		cursor   Cursor
		count    int
		expected Cursor
	}{
		{
			name:     "single left",
			cursor:   Cursor{Line: 0, Col: 5},
			count:    1,
			expected: Cursor{Line: 0, Col: 4},
		},
		{
			name:     "multi left (3h)",
			cursor:   Cursor{Line: 0, Col: 7},
			count:    3,
			expected: Cursor{Line: 0, Col: 4},
		},
		{
			name:     "left beyond bounds (clamps to 0)",
			cursor:   Cursor{Line: 0, Col: 2},
			count:    10,
			expected: Cursor{Line: 0, Col: 0},
		},
		{
			name:     "left from col 0 (no-op)",
			cursor:   Cursor{Line: 0, Col: 0},
			count:    1,
			expected: Cursor{Line: 0, Col: 0},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := NewVimHandler()
			viewport := Viewport{Top: 0, Height: 10}
			result := handler.Apply(doc, tt.cursor, viewport, MotionLeft, tt.count)

			if result.Cursor != tt.expected {
				t.Errorf("expected cursor %+v, got %+v", tt.expected, result.Cursor)
			}
		})
	}
}

func TestVimHandler_MotionRight(t *testing.T) {
	doc := newMockDocument([]string{"hello"}) // 5 chars

	tests := []struct {
		name     string
		cursor   Cursor
		count    int
		expected Cursor
	}{
		{
			name:     "single right",
			cursor:   Cursor{Line: 0, Col: 2},
			count:    1,
			expected: Cursor{Line: 0, Col: 3},
		},
		{
			name:     "multi right (2l)",
			cursor:   Cursor{Line: 0, Col: 1},
			count:    2,
			expected: Cursor{Line: 0, Col: 3},
		},
		{
			name:     "right beyond bounds (clamps to line length)",
			cursor:   Cursor{Line: 0, Col: 3},
			count:    10,
			expected: Cursor{Line: 0, Col: 5},
		},
		{
			name:     "right from end (no-op)",
			cursor:   Cursor{Line: 0, Col: 5},
			count:    1,
			expected: Cursor{Line: 0, Col: 5},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := NewVimHandler()
			viewport := Viewport{Top: 0, Height: 10}
			result := handler.Apply(doc, tt.cursor, viewport, MotionRight, tt.count)

			if result.Cursor != tt.expected {
				t.Errorf("expected cursor %+v, got %+v", tt.expected, result.Cursor)
			}
		})
	}
}

func TestVimHandler_MotionLineStart(t *testing.T) {
	doc := newMockDocument([]string{"hello world"})
	handler := NewVimHandler()
	viewport := Viewport{Top: 0, Height: 10}

	cursor := Cursor{Line: 0, Col: 7}
	result := handler.Apply(doc, cursor, viewport, MotionLineStart, 1)

	expected := Cursor{Line: 0, Col: 0}
	if result.Cursor != expected {
		t.Errorf("expected cursor %+v, got %+v", expected, result.Cursor)
	}
}

func TestVimHandler_MotionLineEnd(t *testing.T) {
	doc := newMockDocument([]string{"hello"}) // 5 chars
	handler := NewVimHandler()
	viewport := Viewport{Top: 0, Height: 10}

	cursor := Cursor{Line: 0, Col: 2}
	result := handler.Apply(doc, cursor, viewport, MotionLineEnd, 1)

	expected := Cursor{Line: 0, Col: 5}
	if result.Cursor != expected {
		t.Errorf("expected cursor %+v, got %+v", expected, result.Cursor)
	}
}

func TestVimHandler_MotionLineEndEmptyLine(t *testing.T) {
	doc := newMockDocument([]string{""}) // empty line
	handler := NewVimHandler()
	viewport := Viewport{Top: 0, Height: 10}

	cursor := Cursor{Line: 0, Col: 0}
	result := handler.Apply(doc, cursor, viewport, MotionLineEnd, 1)

	expected := Cursor{Line: 0, Col: 0}
	if result.Cursor != expected {
		t.Errorf("expected cursor %+v, got %+v", expected, result.Cursor)
	}
}

func TestVimHandler_MotionFirstLine(t *testing.T) {
	doc := newMockDocument([]string{
		"line 0",
		"line 1",
		"line 2",
		"line 3",
		"line 4",
	})

	tests := []struct {
		name     string
		cursor   Cursor
		count    int
		expected Cursor
	}{
		{
			name:     "gg (no count) from middle",
			cursor:   Cursor{Line: 3, Col: 2},
			count:    0,
			expected: Cursor{Line: 0, Col: 2},
		},
		{
			name:     "1gg (go to line 1 = index 0)",
			cursor:   Cursor{Line: 3, Col: 2},
			count:    1,
			expected: Cursor{Line: 0, Col: 2},
		},
		{
			name:     "5gg (go to line 5 = index 4)",
			cursor:   Cursor{Line: 0, Col: 3},
			count:    5,
			expected: Cursor{Line: 4, Col: 3},
		},
		{
			name:     "3gg (go to line 3 = index 2)",
			cursor:   Cursor{Line: 4, Col: 1},
			count:    3,
			expected: Cursor{Line: 2, Col: 1},
		},
		{
			name:     "100gg (beyond bounds, clamps to last line)",
			cursor:   Cursor{Line: 2, Col: 0},
			count:    100,
			expected: Cursor{Line: 4, Col: 0},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := NewVimHandler()
			viewport := Viewport{Top: 0, Height: 10}
			result := handler.Apply(doc, tt.cursor, viewport, MotionFirstLine, tt.count)

			if result.Cursor != tt.expected {
				t.Errorf("expected cursor %+v, got %+v", tt.expected, result.Cursor)
			}
		})
	}
}

func TestVimHandler_MotionLastLine(t *testing.T) {
	doc := newMockDocument([]string{
		"line 0",
		"line 1",
		"line 2",
		"line 3",
		"line 4",
	})

	tests := []struct {
		name     string
		cursor   Cursor
		count    int
		expected Cursor
	}{
		{
			name:     "G (no count) from middle",
			cursor:   Cursor{Line: 1, Col: 3},
			count:    0,
			expected: Cursor{Line: 4, Col: 3},
		},
		{
			name:     "1G (go to line 1 = index 0)",
			cursor:   Cursor{Line: 3, Col: 4},
			count:    1,
			expected: Cursor{Line: 0, Col: 4},
		},
		{
			name:     "3G (go to line 3 = index 2)",
			cursor:   Cursor{Line: 4, Col: 2},
			count:    3,
			expected: Cursor{Line: 2, Col: 2},
		},
		{
			name:     "5G (go to line 5 = index 4)",
			cursor:   Cursor{Line: 1, Col: 1},
			count:    5,
			expected: Cursor{Line: 4, Col: 1},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := NewVimHandler()
			viewport := Viewport{Top: 0, Height: 10}
			result := handler.Apply(doc, tt.cursor, viewport, MotionLastLine, tt.count)

			if result.Cursor != tt.expected {
				t.Errorf("expected cursor %+v, got %+v", tt.expected, result.Cursor)
			}
		})
	}
}

func TestVimHandler_MotionHalfPageUp(t *testing.T) {
	doc := newMockDocument([]string{
		"line 0", "line 1", "line 2", "line 3", "line 4",
		"line 5", "line 6", "line 7", "line 8", "line 9",
		"line 10", "line 11", "line 12", "line 13", "line 14",
	})

	tests := []struct {
		name             string
		cursor           Cursor
		viewport         Viewport
		expectedCursor   Cursor
		expectedViewport Viewport
	}{
		{
			name:             "Ctrl-U from middle",
			cursor:           Cursor{Line: 10, Col: 2},
			viewport:         Viewport{Top: 5, Height: 10},
			expectedCursor:   Cursor{Line: 5, Col: 2},
			expectedViewport: Viewport{Top: 0, Height: 10},
		},
		{
			name:             "Ctrl-U near top (clamps to 0)",
			cursor:           Cursor{Line: 3, Col: 1},
			viewport:         Viewport{Top: 2, Height: 10},
			expectedCursor:   Cursor{Line: 0, Col: 1},
			expectedViewport: Viewport{Top: 0, Height: 10},
		},
		{
			name:             "Ctrl-U from top (no-op)",
			cursor:           Cursor{Line: 0, Col: 0},
			viewport:         Viewport{Top: 0, Height: 10},
			expectedCursor:   Cursor{Line: 0, Col: 0},
			expectedViewport: Viewport{Top: 0, Height: 10},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := NewVimHandler()
			result := handler.Apply(doc, tt.cursor, tt.viewport, MotionHalfPageUp, 1)

			if result.Cursor != tt.expectedCursor {
				t.Errorf("expected cursor %+v, got %+v", tt.expectedCursor, result.Cursor)
			}
			if result.Viewport != tt.expectedViewport {
				t.Errorf("expected viewport %+v, got %+v", tt.expectedViewport, result.Viewport)
			}
		})
	}
}

func TestVimHandler_MotionHalfPageDown(t *testing.T) {
	doc := newMockDocument([]string{
		"line 0", "line 1", "line 2", "line 3", "line 4",
		"line 5", "line 6", "line 7", "line 8", "line 9",
		"line 10", "line 11", "line 12", "line 13", "line 14",
	})

	tests := []struct {
		name             string
		cursor           Cursor
		viewport         Viewport
		expectedCursor   Cursor
		expectedViewport Viewport
	}{
		{
			name:             "Ctrl-D from top",
			cursor:           Cursor{Line: 2, Col: 1},
			viewport:         Viewport{Top: 0, Height: 10},
			expectedCursor:   Cursor{Line: 7, Col: 1},
			expectedViewport: Viewport{Top: 5, Height: 10},
		},
		{
			name:             "Ctrl-D near bottom (clamps)",
			cursor:           Cursor{Line: 12, Col: 3},
			viewport:         Viewport{Top: 5, Height: 10},
			expectedCursor:   Cursor{Line: 14, Col: 3},
			expectedViewport: Viewport{Top: 5, Height: 10}, // Can't scroll past end
		},
		{
			name:             "Ctrl-D from last line (no-op)",
			cursor:           Cursor{Line: 14, Col: 2},
			viewport:         Viewport{Top: 5, Height: 10},
			expectedCursor:   Cursor{Line: 14, Col: 2},
			expectedViewport: Viewport{Top: 5, Height: 10},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := NewVimHandler()
			result := handler.Apply(doc, tt.cursor, tt.viewport, MotionHalfPageDown, 1)

			if result.Cursor != tt.expectedCursor {
				t.Errorf("expected cursor %+v, got %+v", tt.expectedCursor, result.Cursor)
			}
			if result.Viewport != tt.expectedViewport {
				t.Errorf("expected viewport %+v, got %+v", tt.expectedViewport, result.Viewport)
			}
		})
	}
}

func TestVimHandler_ViewportAdjustment(t *testing.T) {
	doc := newMockDocument([]string{
		"line 0", "line 1", "line 2", "line 3", "line 4",
		"line 5", "line 6", "line 7", "line 8", "line 9",
	})

	tests := []struct {
		name             string
		cursor           Cursor
		motion           Motion
		count            int
		viewport         Viewport
		expectedViewport Viewport
	}{
		{
			name:             "scroll down when cursor moves below viewport",
			cursor:           Cursor{Line: 4, Col: 0},
			motion:           MotionDown,
			count:            3,                           // Move to line 7
			viewport:         Viewport{Top: 0, Height: 5}, // Shows lines 0-4
			expectedViewport: Viewport{Top: 3, Height: 5}, // Scroll to show line 7
		},
		{
			name:             "scroll up when cursor moves above viewport",
			cursor:           Cursor{Line: 5, Col: 0},
			motion:           MotionUp,
			count:            3,                           // Move to line 2
			viewport:         Viewport{Top: 5, Height: 5}, // Shows lines 5-9
			expectedViewport: Viewport{Top: 2, Height: 5}, // Scroll to show line 2
		},
		{
			name:             "no scroll when cursor stays in viewport",
			cursor:           Cursor{Line: 3, Col: 0},
			motion:           MotionDown,
			count:            1, // Move to line 4
			viewport:         Viewport{Top: 0, Height: 10},
			expectedViewport: Viewport{Top: 0, Height: 10}, // No change
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := NewVimHandler()
			result := handler.Apply(doc, tt.cursor, tt.viewport, tt.motion, tt.count)

			if result.Viewport != tt.expectedViewport {
				t.Errorf("expected viewport %+v, got %+v", tt.expectedViewport, result.Viewport)
			}
		})
	}
}

func TestVimHandler_EmptyDocument(t *testing.T) {
	doc := newMockDocument([]string{""})
	handler := NewVimHandler()
	viewport := Viewport{Top: 0, Height: 10}
	cursor := Cursor{Line: 0, Col: 0}

	// All motions should handle empty document gracefully
	motions := []Motion{
		MotionUp, MotionDown, MotionLeft, MotionRight,
		MotionLineStart, MotionLineEnd,
		MotionFirstLine, MotionLastLine,
		MotionHalfPageUp, MotionHalfPageDown,
	}

	for _, motion := range motions {
		result := handler.Apply(doc, cursor, viewport, motion, 1)
		if result.Cursor.Line != 0 || result.Cursor.Col != 0 {
			t.Errorf("motion %v on empty doc should stay at 0,0, got %+v", motion, result.Cursor)
		}
	}
}

func TestVimHandler_SingleLineDocument(t *testing.T) {
	doc := newMockDocument([]string{"single line"})
	handler := NewVimHandler()
	viewport := Viewport{Top: 0, Height: 10}
	cursor := Cursor{Line: 0, Col: 5}

	// Vertical motions should stay on line 0
	result := handler.Apply(doc, cursor, viewport, MotionDown, 5)
	if result.Cursor.Line != 0 {
		t.Errorf("MotionDown on single line should stay on line 0, got %d", result.Cursor.Line)
	}

	result = handler.Apply(doc, cursor, viewport, MotionUp, 5)
	if result.Cursor.Line != 0 {
		t.Errorf("MotionUp on single line should stay on line 0, got %d", result.Cursor.Line)
	}

	result = handler.Apply(doc, cursor, viewport, MotionFirstLine, 1)
	if result.Cursor.Line != 0 {
		t.Errorf("MotionFirstLine on single line should stay on line 0, got %d", result.Cursor.Line)
	}

	result = handler.Apply(doc, cursor, viewport, MotionLastLine, 1)
	if result.Cursor.Line != 0 {
		t.Errorf("MotionLastLine on single line should stay on line 0, got %d", result.Cursor.Line)
	}
}

func TestVimHandler_UnicodeSupport(t *testing.T) {
	doc := newMockDocument([]string{
		"Hello 世界", // "Hello " = 6 chars, "世界" = 2 chars = 8 total runes
	})
	handler := NewVimHandler()
	viewport := Viewport{Top: 0, Height: 10}

	// Move to end of line (should count runes, not bytes)
	cursor := Cursor{Line: 0, Col: 0}
	result := handler.Apply(doc, cursor, viewport, MotionLineEnd, 1)

	expected := Cursor{Line: 0, Col: 8} // 8 runes total
	if result.Cursor != expected {
		t.Errorf("expected cursor at col 8 (8 runes), got %+v", result.Cursor)
	}

	// Move right from position 6 should go to 7 (first rune of 世)
	cursor = Cursor{Line: 0, Col: 6}
	result = handler.Apply(doc, cursor, viewport, MotionRight, 1)
	expected = Cursor{Line: 0, Col: 7}
	if result.Cursor != expected {
		t.Errorf("expected cursor at col 7, got %+v", result.Cursor)
	}
}

func TestVimHandler_ViewportLargerThanDocument(t *testing.T) {
	doc := newMockDocument([]string{
		"line 0",
		"line 1",
		"line 2",
	})
	handler := NewVimHandler()
	viewport := Viewport{Top: 0, Height: 20} // Viewport larger than doc
	cursor := Cursor{Line: 1, Col: 0}

	// Move down
	result := handler.Apply(doc, cursor, viewport, MotionDown, 1)
	if result.Viewport.Top != 0 {
		t.Errorf("viewport should stay at top 0 when larger than doc, got %d", result.Viewport.Top)
	}

	// gg should work
	result = handler.Apply(doc, cursor, viewport, MotionFirstLine, 1)
	if result.Cursor.Line != 0 {
		t.Errorf("gg should move to line 0, got %d", result.Cursor.Line)
	}
}

func TestVimHandler_CountZero(t *testing.T) {
	doc := newMockDocument([]string{"line 0", "line 1", "line 2", "line 3", "line 4"})
	viewport := Viewport{Top: 0, Height: 10}

	// For regular motions (j/k/h/l), count 0 should be treated as count 1
	handler1 := NewVimHandler()
	cursor := Cursor{Line: 1, Col: 2}
	result := handler1.Apply(doc, cursor, viewport, MotionDown, 0)
	expected := Cursor{Line: 2, Col: 2}
	if result.Cursor != expected {
		t.Errorf("count 0 with MotionDown should behave as count 1, expected %+v, got %+v", expected, result.Cursor)
	}

	// For gg/G, count 0 means "no count" (special behavior)
	// Use fresh handlers to avoid goal column interference
	handler2 := NewVimHandler()
	cursor = Cursor{Line: 2, Col: 1}
	result = handler2.Apply(doc, cursor, viewport, MotionFirstLine, 0)
	expected = Cursor{Line: 0, Col: 1}
	if result.Cursor != expected {
		t.Errorf("count 0 with MotionFirstLine (gg) should go to first line, expected %+v, got %+v", expected, result.Cursor)
	}

	handler3 := NewVimHandler()
	cursor = Cursor{Line: 1, Col: 3}
	result = handler3.Apply(doc, cursor, viewport, MotionLastLine, 0)
	expected = Cursor{Line: 4, Col: 3}
	if result.Cursor != expected {
		t.Errorf("count 0 with MotionLastLine (G) should go to last line, expected %+v, got %+v", expected, result.Cursor)
	}
}

func TestVimHandler_MotionWordForward(t *testing.T) {
	doc := newMockDocument([]string{
		"hello world test",
		"foo-bar_baz",
		"   leading spaces",
		"",
		"last line",
	})

	tests := []struct {
		name     string
		cursor   Cursor
		count    int
		expected Cursor
	}{
		{
			name:     "w from word start",
			cursor:   Cursor{Line: 0, Col: 0},
			count:    1,
			expected: Cursor{Line: 0, Col: 6}, // "hello" -> "world"
		},
		{
			name:     "w from word middle",
			cursor:   Cursor{Line: 0, Col: 2},
			count:    1,
			expected: Cursor{Line: 0, Col: 6}, // "hello" -> "world"
		},
		{
			name:     "3w multiple words",
			cursor:   Cursor{Line: 0, Col: 0},
			count:    3,
			expected: Cursor{Line: 1, Col: 0}, // "hello" -> "world" -> "test" -> "foo"
		},
		{
			name:     "w across punctuation",
			cursor:   Cursor{Line: 1, Col: 0},
			count:    1,
			expected: Cursor{Line: 1, Col: 3}, // "foo" -> "-"
		},
		{
			name:     "w over punctuation to word",
			cursor:   Cursor{Line: 1, Col: 3},
			count:    1,
			expected: Cursor{Line: 1, Col: 4}, // "-" -> "bar"
		},
		{
			name:     "w from middle of word to next line",
			cursor:   Cursor{Line: 0, Col: 12},
			count:    1,
			expected: Cursor{Line: 1, Col: 0}, // "test" -> "foo"
		},
		{
			name:     "w from end of line to next line",
			cursor:   Cursor{Line: 0, Col: 16},
			count:    1,
			expected: Cursor{Line: 1, Col: 0}, // end of line 0 -> "foo"
		},
		{
			name:     "w from whitespace to next word",
			cursor:   Cursor{Line: 2, Col: 10},
			count:    1,
			expected: Cursor{Line: 2, Col: 11}, // whitespace -> "spaces"
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := NewVimHandler()
			viewport := Viewport{Top: 0, Height: 10}
			result := handler.Apply(doc, tt.cursor, viewport, MotionWordForward, tt.count)

			if result.Cursor != tt.expected {
				t.Errorf("expected cursor %+v, got %+v", tt.expected, result.Cursor)
			}
		})
	}
}

func TestVimHandler_MotionWordBackward(t *testing.T) {
	doc := newMockDocument([]string{
		"hello world test",
		"foo-bar_baz",
		"   leading spaces",
		"",
		"last line",
	})

	tests := []struct {
		name     string
		cursor   Cursor
		count    int
		expected Cursor
	}{
		{
			name:     "b from word end",
			cursor:   Cursor{Line: 0, Col: 10},
			count:    1,
			expected: Cursor{Line: 0, Col: 6}, // "world" middle -> "world" start
		},
		{
			name:     "b from word start",
			cursor:   Cursor{Line: 0, Col: 6},
			count:    1,
			expected: Cursor{Line: 0, Col: 0}, // "world" start -> "hello" start
		},
		{
			name:     "3b multiple words back",
			cursor:   Cursor{Line: 1, Col: 7},
			count:    3,
			expected: Cursor{Line: 1, Col: 0}, // "_" -> "bar" -> "-" -> "foo"
		},
		{
			name:     "b across punctuation",
			cursor:   Cursor{Line: 1, Col: 4},
			count:    1,
			expected: Cursor{Line: 1, Col: 3}, // "bar" -> "-"
		},
		{
			name:     "b from line start to previous line",
			cursor:   Cursor{Line: 1, Col: 0},
			count:    1,
			expected: Cursor{Line: 0, Col: 12}, // "foo" start -> "test" start
		},
		{
			name:     "b from word start to previous line",
			cursor:   Cursor{Line: 2, Col: 3},
			count:    1,
			expected: Cursor{Line: 1, Col: 4}, // "leading" start -> "bar" start
		},
		{
			name:     "b from first position (no-op)",
			cursor:   Cursor{Line: 0, Col: 0},
			count:    1,
			expected: Cursor{Line: 0, Col: 0},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := NewVimHandler()
			viewport := Viewport{Top: 0, Height: 10}
			result := handler.Apply(doc, tt.cursor, viewport, MotionWordBackward, tt.count)

			if result.Cursor != tt.expected {
				t.Errorf("expected cursor %+v, got %+v", tt.expected, result.Cursor)
			}
		})
	}
}

func TestVimHandler_MotionWordEnd(t *testing.T) {
	doc := newMockDocument([]string{
		"hello world test",
		"foo-bar_baz",
		"   leading spaces",
		"",
		"last line",
	})

	tests := []struct {
		name     string
		cursor   Cursor
		count    int
		expected Cursor
	}{
		{
			name:     "e from word start",
			cursor:   Cursor{Line: 0, Col: 0},
			count:    1,
			expected: Cursor{Line: 0, Col: 4}, // "hello" start -> "hello" end
		},
		{
			name:     "e from word middle",
			cursor:   Cursor{Line: 0, Col: 2},
			count:    1,
			expected: Cursor{Line: 0, Col: 4}, // "hello" middle -> "hello" end
		},
		{
			name:     "e from word end",
			cursor:   Cursor{Line: 0, Col: 4},
			count:    1,
			expected: Cursor{Line: 0, Col: 10}, // "hello" end -> "world" end
		},
		{
			name:     "3e multiple word ends",
			cursor:   Cursor{Line: 0, Col: 0},
			count:    3,
			expected: Cursor{Line: 0, Col: 15}, // "hello" end -> "world" end -> "test" end
		},
		{
			name:     "e across punctuation",
			cursor:   Cursor{Line: 1, Col: 0},
			count:    1,
			expected: Cursor{Line: 1, Col: 2}, // "foo" end
		},
		{
			name:     "e to punctuation",
			cursor:   Cursor{Line: 1, Col: 2},
			count:    1,
			expected: Cursor{Line: 1, Col: 3}, // "-" end
		},
		{
			name:     "e skip whitespace to next word",
			cursor:   Cursor{Line: 0, Col: 15},
			count:    1,
			expected: Cursor{Line: 1, Col: 2}, // "test" end -> "foo" end
		},
		{
			name:     "e skip empty line",
			cursor:   Cursor{Line: 2, Col: 17},
			count:    1,
			expected: Cursor{Line: 4, Col: 3}, // "spaces" end -> "last" end
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := NewVimHandler()
			viewport := Viewport{Top: 0, Height: 10}
			result := handler.Apply(doc, tt.cursor, viewport, MotionWordEnd, tt.count)

			if result.Cursor != tt.expected {
				t.Errorf("expected cursor %+v, got %+v", tt.expected, result.Cursor)
			}
		})
	}
}
