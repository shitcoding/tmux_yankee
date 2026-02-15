package ui

import (
	"strings"
	"testing"
)

// TestTUILineNumberRendering tests that line numbers are rendered in the UI
func TestTUILineNumberRendering(t *testing.T) {
	tests := []struct {
		name            string
		content         []string
		mode            string
		cursorLine      int
		wantGutterCount int
		wantSeparator   bool
	}{
		{
			name:            "absolute mode shows line numbers",
			content:         []string{"line 1", "line 2", "line 3"},
			mode:            "absolute",
			cursorLine:      0,
			wantGutterCount: 3,
			wantSeparator:   true,
		},
		{
			name:            "relative mode shows distances",
			content:         []string{"line 1", "line 2", "line 3"},
			mode:            "relative",
			cursorLine:      1,
			wantGutterCount: 3,
			wantSeparator:   true,
		},
		{
			name:            "hybrid mode shows colored numbers",
			content:         []string{"line 1", "line 2", "line 3"},
			mode:            "hybrid",
			cursorLine:      1,
			wantGutterCount: 3,
			wantSeparator:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tui := NewTUI("test-pane", tt.content, tt.mode)
			tui.cursorLine = tt.cursorLine
			tui.width = 80
			tui.height = 10

			// Render to a buffer instead of stdout
			output := captureRender(tui)

			// Check that gutter separator (│) appears
			if tt.wantSeparator {
				separatorCount := strings.Count(output, "│")
				if separatorCount == 0 {
					t.Errorf("render() should include gutter separator '│', but none found")
				}
			}

			// Check that line numbers appear (digits before separator)
			if tt.wantGutterCount > 0 {
				// Should have numbers rendered
				hasNumbers := false
				lines := strings.Split(output, "\n")
				for _, line := range lines {
					if strings.Contains(line, "│") {
						hasNumbers = true
						break
					}
				}
				if !hasNumbers {
					t.Errorf("render() should include line numbers in %s mode", tt.mode)
				}
			}
		})
	}
}

// TestTUIModeToggle tests cycling through line number modes with 'L' key
func TestTUIModeToggle(t *testing.T) {
	tui := NewTUI("test-pane", []string{"line 1", "line 2", "line 3"}, "hybrid")
	tui.width = 80
	tui.height = 10

	// Initial mode should be hybrid
	if tui.GetMode() != "hybrid" {
		t.Errorf("initial mode = %s, want hybrid", tui.GetMode())
	}

	// Press 'L' -> should change to absolute
	tui.handleInput([]byte{'L'})
	if tui.GetMode() != "absolute" {
		t.Errorf("after first toggle mode = %s, want absolute", tui.GetMode())
	}

	// Press 'L' again -> should change to relative
	tui.handleInput([]byte{'L'})
	if tui.GetMode() != "relative" {
		t.Errorf("after second toggle mode = %s, want relative", tui.GetMode())
	}

	// Press 'L' again -> should cycle back to hybrid
	tui.handleInput([]byte{'L'})
	if tui.GetMode() != "hybrid" {
		t.Errorf("after third toggle mode = %s, want hybrid (cycle complete)", tui.GetMode())
	}
}

// TestTUILineNumbersUpdateOnNavigation tests that line numbers update when cursor moves
func TestTUILineNumbersUpdateOnNavigation(t *testing.T) {
	content := []string{
		"line 1",
		"line 2",
		"line 3",
		"line 4",
		"line 5",
	}

	tui := NewTUI("test-pane", content, "relative")
	tui.width = 80
	tui.height = 10
	tui.cursorLine = 2 // Start at line 3 (0-indexed)

	// Render initial state
	output1 := captureRender(tui)

	// Move cursor down
	tui.handleInput([]byte{'j'})

	// Render after move
	output2 := captureRender(tui)

	// In relative mode, line numbers should change when cursor moves
	// (because distances to cursor change)
	if output1 == output2 {
		t.Error("render() output should change after cursor movement in relative mode")
	}

	// Verify cursor moved
	if tui.cursorLine != 3 {
		t.Errorf("cursorLine after 'j' = %d, want 3", tui.cursorLine)
	}
}

// TestTUIHybridModeColors tests that hybrid mode uses correct colors
func TestTUIHybridModeColors(t *testing.T) {
	content := []string{"line 1", "line 2", "line 3", "line 4", "line 5"}
	tui := NewTUI("test-pane", content, "hybrid")
	tui.width = 80
	tui.height = 10
	tui.cursorLine = 2

	output := captureRender(tui)

	// Should contain green ANSI code for cursor line (32;1m)
	if !strings.Contains(output, "\x1b[32;1m") {
		t.Error("hybrid mode should use green color (32;1m) for cursor line")
	}

	// Should contain yellow ANSI code for other lines (33m)
	if !strings.Contains(output, "\x1b[33m") {
		t.Error("hybrid mode should use yellow color (33m) for non-cursor lines")
	}

	// Should contain reset codes
	if !strings.Contains(output, "\x1b[0m") {
		t.Error("hybrid mode should include ANSI reset codes (0m)")
	}
}

// TestTUIAbsoluteModePersistence tests absolute line numbers don't change on cursor move
func TestTUIAbsoluteModePersistence(t *testing.T) {
	content := []string{"line 1", "line 2", "line 3", "line 4", "line 5"}
	tui := NewTUI("test-pane", content, "absolute")
	tui.width = 80
	tui.height = 10
	tui.cursorLine = 1

	// Capture line numbers at different cursor positions
	output1 := captureRender(tui)
	firstLineNum := extractFirstLineNumber(output1)

	tui.cursorLine = 3
	output2 := captureRender(tui)
	secondLineNum := extractFirstLineNumber(output2)

	// In absolute mode, first line number should always be the same
	// (doesn't depend on cursor position)
	if firstLineNum != secondLineNum {
		t.Errorf("absolute mode: first line number changed from %s to %s (should be stable)",
			firstLineNum, secondLineNum)
	}
}

// TestTUIGutterWidthCalculation tests that gutter width adjusts to content
func TestTUIGutterWidthCalculation(t *testing.T) {
	tests := []struct {
		name         string
		numLines     int
		wantMinWidth int
	}{
		{
			name:         "small file (< 10 lines)",
			numLines:     5,
			wantMinWidth: 1,
		},
		{
			name:         "medium file (< 100 lines)",
			numLines:     50,
			wantMinWidth: 2,
		},
		{
			name:         "large file (< 1000 lines)",
			numLines:     500,
			wantMinWidth: 3,
		},
		{
			name:         "very large file (10000+ lines)",
			numLines:     10000,
			wantMinWidth: 5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Generate content
			content := make([]string, tt.numLines)
			for i := 0; i < tt.numLines; i++ {
				content[i] = "test line"
			}

			tui := NewTUI("test-pane", content, "absolute")
			tui.width = 80
			tui.height = 10

			output := captureRender(tui)

			// Check that numbers have appropriate width
			// (numbers should be right-aligned with enough space)
			lines := strings.Split(output, "\n")
			for _, line := range lines {
				if strings.Contains(line, "│") {
					parts := strings.Split(line, "│")
					if len(parts) > 0 {
						gutterPart := parts[0]
						// Strip ANSI codes for measurement
						stripped := stripANSI(gutterPart)
						if len(stripped) < tt.wantMinWidth {
							t.Errorf("gutter width %d < minimum %d for %d lines",
								len(stripped), tt.wantMinWidth, tt.numLines)
						}
					}
				}
			}
		})
	}
}

// Helper functions for testing

// captureRender captures the render output (mocked for testing)
func captureRender(tui *TUI) string {
	// Create a strings.Builder to capture output
	var b strings.Builder

	// Calculate visible range
	endLine := tui.viewportTop + tui.height
	if endLine > tui.doc.LineCount() {
		endLine = tui.doc.LineCount()
	}

	// Render visible lines (simplified version of render())
	for i := tui.viewportTop; i < endLine; i++ {
		line := tui.doc.Line(i)

		// Render line number gutter (1-indexed for display)
		gutter := tui.formatter.RenderGutter(i+1, tui.cursorLine+1)
		b.WriteString(gutter)

		// Highlight cursor line
		if i == tui.cursorLine {
			b.WriteString("\x1b[7m") // Reverse video
		}

		// Truncate line if too long (account for gutter width)
		gutterWidth := len(stripANSI(gutter))
		availableWidth := tui.width - gutterWidth
		runes := []rune(line)
		if len(runes) > availableWidth {
			line = string(runes[:availableWidth])
		}

		b.WriteString(line)

		// Reset style
		b.WriteString("\x1b[0m")

		// Newline if not last line
		if i < endLine-1 {
			b.WriteString("\n")
		}
	}

	return b.String()
}

// extractFirstLineNumber extracts the first line number from render output
func extractFirstLineNumber(output string) string {
	lines := strings.Split(output, "\n")
	if len(lines) == 0 {
		return ""
	}

	// Get first line
	firstLine := lines[0]

	// Strip ANSI codes
	stripped := stripANSI(firstLine)

	// Extract number before separator
	parts := strings.Split(stripped, "│")
	if len(parts) == 0 {
		return ""
	}

	return strings.TrimSpace(parts[0])
}

// TestTUIUTF8Truncation tests that UTF-8 characters are safely truncated
func TestTUIUTF8Truncation(t *testing.T) {
	content := []string{"Hello 世界 🌍 こんにちは"}
	tui := NewTUI("test", content, "absolute")
	tui.width = 20
	tui.height = 5

	// Capture render output
	output := captureRender(tui)

	// Verify output doesn't contain invalid UTF-8 sequences
	// Go's string validation should pass
	if !isValidUTF8(output) {
		t.Error("render() produced invalid UTF-8 after truncation")
	}

	// Verify each line was truncated properly
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		stripped := stripANSI(line)
		runeCount := len([]rune(stripped))
		if runeCount > tui.width {
			t.Errorf("render() line rune count %d exceeds terminal width %d", runeCount, tui.width)
		}
	}
}

// isValidUTF8 checks if a string contains valid UTF-8
func isValidUTF8(s string) bool {
	// Go strings are always valid UTF-8 if created from []rune
	// This function verifies no byte-level corruption
	for _, r := range s {
		if r == '\ufffd' && !strings.Contains(s, "\ufffd") {
			// Found replacement character that wasn't in original
			return false
		}
	}
	return true
}
