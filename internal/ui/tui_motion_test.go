package ui

import (
	"testing"
)

// TestMotion_CountedDown tests counted down motion (5j)
func TestMotion_CountedDown(t *testing.T) {
	// Setup: TUI with 10 lines
	content := make([]string, 10)
	for i := range content {
		content[i] = "line content"
	}

	tui := NewTUI("test-pane", content, "absolute")
	tui.height = 5
	tui.cursorLine = 0 // explicitly start at top for navigation test

	// Execute "5j" (down 5 lines)
	tui.handleInput([]byte{'5'})
	tui.handleInput([]byte{'j'})

	// Assert: cursor should be at line 5
	if tui.cursorLine != 5 {
		t.Errorf("After 5j, expected cursor at line 5, got %d", tui.cursorLine)
	}
}

// TestMotion_CountedUp tests counted up motion (3k)
func TestMotion_CountedUp(t *testing.T) {
	// Setup: TUI with 10 lines
	content := make([]string, 10)
	for i := range content {
		content[i] = "line content"
	}

	tui := NewTUI("test-pane", content, "absolute")
	tui.height = 5
	tui.cursorLine = 7 // Start at line 7

	// Execute "3k" (up 3 lines)
	tui.handleInput([]byte{'3'})
	tui.handleInput([]byte{'k'})

	// Assert: cursor should be at line 4
	if tui.cursorLine != 4 {
		t.Errorf("After 3k, expected cursor at line 4, got %d", tui.cursorLine)
	}
}

// TestMotion_gg tests gg (go to first line)
func TestMotion_gg(t *testing.T) {
	// Setup: TUI with 10 lines
	content := make([]string, 10)
	for i := range content {
		content[i] = "line content"
	}

	tui := NewTUI("test-pane", content, "absolute")
	tui.height = 5
	tui.cursorLine = 5 // Start at line 5

	// Execute "gg"
	tui.handleInput([]byte{'g'})
	tui.handleInput([]byte{'g'})

	// Assert: cursor should be at line 0
	if tui.cursorLine != 0 {
		t.Errorf("After gg, expected cursor at line 0, got %d", tui.cursorLine)
	}
}

// TestMotion_CountedGG tests 5gg (go to line 5)
func TestMotion_CountedGG(t *testing.T) {
	// Setup: TUI with 10 lines
	content := make([]string, 10)
	for i := range content {
		content[i] = "line content"
	}

	tui := NewTUI("test-pane", content, "absolute")
	tui.height = 5

	// Execute "5gg" (go to line 5, 1-indexed)
	tui.handleInput([]byte{'5'})
	tui.handleInput([]byte{'g'})
	tui.handleInput([]byte{'g'})

	// Assert: cursor should be at line 4 (0-indexed)
	if tui.cursorLine != 4 {
		t.Errorf("After 5gg, expected cursor at line 4, got %d", tui.cursorLine)
	}
}

// TestMotion_G tests G (go to last line)
func TestMotion_G(t *testing.T) {
	// Setup: TUI with 10 lines
	content := make([]string, 10)
	for i := range content {
		content[i] = "line content"
	}

	tui := NewTUI("test-pane", content, "absolute")
	tui.height = 5

	// Execute "G"
	tui.handleInput([]byte{'G'})

	// Assert: cursor should be at line 9 (last line, 0-indexed)
	if tui.cursorLine != 9 {
		t.Errorf("After G, expected cursor at line 9, got %d", tui.cursorLine)
	}
}

// TestMotion_CountedG tests 3G (go to line 3)
func TestMotion_CountedG(t *testing.T) {
	// Setup: TUI with 10 lines
	content := make([]string, 10)
	for i := range content {
		content[i] = "line content"
	}

	tui := NewTUI("test-pane", content, "absolute")
	tui.height = 5

	// Execute "3G" (go to line 3, 1-indexed)
	tui.handleInput([]byte{'3'})
	tui.handleInput([]byte{'G'})

	// Assert: cursor should be at line 2 (0-indexed)
	if tui.cursorLine != 2 {
		t.Errorf("After 3G, expected cursor at line 2, got %d", tui.cursorLine)
	}
}

// TestMotion_ZeroLineStart tests 0 (go to line start)
func TestMotion_ZeroLineStart(t *testing.T) {
	// Setup: TUI with wide content
	content := []string{"this is a long line of text"}

	tui := NewTUI("test-pane", content, "absolute")
	tui.height = 5
	tui.cursorCol = 10 // Start at column 10

	// Execute "0"
	tui.handleInput([]byte{'0'})

	// Assert: cursor column should be 0
	if tui.cursorCol != 0 {
		t.Errorf("After 0, expected cursor col at 0, got %d", tui.cursorCol)
	}
	if tui.cursorLine != 0 {
		t.Errorf("After 0, cursor line should remain 0, got %d", tui.cursorLine)
	}
}

// TestMotion_CountWith10 tests "10j" (count starting with 1 then 0)
func TestMotion_CountWith10(t *testing.T) {
	// Setup: TUI with 20 lines
	content := make([]string, 20)
	for i := range content {
		content[i] = "line content"
	}

	tui := NewTUI("test-pane", content, "absolute")
	tui.height = 5
	tui.cursorLine = 0 // explicitly start at top for navigation test

	// Execute "10j" (down 10 lines)
	tui.handleInput([]byte{'1'})
	tui.handleInput([]byte{'0'})
	tui.handleInput([]byte{'j'})

	// Assert: cursor should be at line 10
	if tui.cursorLine != 10 {
		t.Errorf("After 10j, expected cursor at line 10, got %d", tui.cursorLine)
	}
}

// TestMotion_ViewportScroll tests that viewport scrolls with cursor
func TestMotion_ViewportScroll(t *testing.T) {
	// Setup: TUI with 20 lines, height=5
	content := make([]string, 20)
	for i := range content {
		content[i] = "line content"
	}

	tui := NewTUI("test-pane", content, "absolute")
	tui.height = 5
	tui.cursorLine = 0 // explicitly start at top for navigation test
	tui.viewportTop = 0

	// Execute "10j" to move cursor past viewport
	tui.handleInput([]byte{'1'})
	tui.handleInput([]byte{'0'})
	tui.handleInput([]byte{'j'})

	// Assert: cursor should be at line 10
	if tui.cursorLine != 10 {
		t.Errorf("Expected cursor at line 10, got %d", tui.cursorLine)
	}

	// Assert: viewport should have scrolled to keep cursor visible
	// Viewport should start at line 6 (so cursor at line 10 is at bottom)
	if tui.viewportTop > tui.cursorLine {
		t.Errorf("Viewport top (%d) should not be below cursor (%d)", tui.viewportTop, tui.cursorLine)
	}
	if tui.viewportTop+tui.height-1 < tui.cursorLine {
		t.Errorf("Cursor (%d) should be visible in viewport [%d, %d]",
			tui.cursorLine, tui.viewportTop, tui.viewportTop+tui.height-1)
	}
}

// TestMotion_DollarLineEnd tests $ (go to line end)
func TestMotion_DollarLineEnd(t *testing.T) {
	// Setup: TUI with content
	content := []string{"hello world"}

	tui := NewTUI("test-pane", content, "absolute")
	tui.height = 5
	tui.cursorCol = 0 // Start at beginning

	// Execute "$"
	tui.handleInput([]byte{'$'})

	// Assert: cursor should be at end of line (11 runes)
	expectedCol := len([]rune("hello world"))
	if tui.cursorCol != expectedCol {
		t.Errorf("After $, expected cursor col at %d, got %d", expectedCol, tui.cursorCol)
	}
}

// TestMotion_HorizontalMovement tests h and l
func TestMotion_HorizontalMovement(t *testing.T) {
	// Setup: TUI with content
	content := []string{"hello world"}

	tui := NewTUI("test-pane", content, "absolute")
	tui.height = 5
	tui.cursorCol = 5 // Start at middle

	// Execute "3h" (left 3)
	tui.handleInput([]byte{'3'})
	tui.handleInput([]byte{'h'})

	if tui.cursorCol != 2 {
		t.Errorf("After 3h from col 5, expected col 2, got %d", tui.cursorCol)
	}

	// Execute "5l" (right 5)
	tui.handleInput([]byte{'5'})
	tui.handleInput([]byte{'l'})

	if tui.cursorCol != 7 {
		t.Errorf("After 5l from col 2, expected col 7, got %d", tui.cursorCol)
	}
}

// TestMotion_CountedMotionWithVisualMode tests that motions extend selection
func TestMotion_CountedMotionWithVisualMode(t *testing.T) {
	// Setup: TUI with 10 lines
	content := make([]string, 10)
	for i := range content {
		content[i] = "line content"
	}

	tui := NewTUI("test-pane", content, "absolute")
	tui.height = 5
	tui.cursorLine = 0 // explicitly start at top for navigation test

	// Activate visual mode at line 0
	tui.handleInput([]byte{'v'})

	// Execute "3j" (down 3 lines)
	tui.handleInput([]byte{'3'})
	tui.handleInput([]byte{'j'})

	// Assert: cursor should be at line 3
	if tui.cursorLine != 3 {
		t.Errorf("After 3j, expected cursor at line 3, got %d", tui.cursorLine)
	}

	// Assert: selection should span lines 0-3
	region := tui.modeMachine.Region()
	if region.Start.Line != 0 || region.End.Line != 3 {
		t.Errorf("Expected selection from line 0-3, got %d-%d",
			region.Start.Line, region.End.Line)
	}
}
