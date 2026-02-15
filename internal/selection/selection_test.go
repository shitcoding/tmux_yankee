package selection

import (
	"strconv"
	"strings"
	"testing"
)

// TestSelectionToggle tests toggling selection mode on/off
func TestSelectionToggle(t *testing.T) {
	tests := []struct {
		name          string
		initialActive bool
		wantActive    bool
		toggleCount   int
	}{
		{
			name:          "inactive to active",
			initialActive: false,
			wantActive:    true,
			toggleCount:   1,
		},
		{
			name:          "active to inactive",
			initialActive: true,
			wantActive:    false,
			toggleCount:   1,
		},
		{
			name:          "double toggle returns to initial",
			initialActive: false,
			wantActive:    false,
			toggleCount:   2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Selection{active: tt.initialActive}

			for i := 0; i < tt.toggleCount; i++ {
				s.Toggle()
			}

			if s.active != tt.wantActive {
				t.Errorf("Toggle() active = %v, want %v", s.active, tt.wantActive)
			}
		})
	}
}

// TestSelectionUpdateEnd tests updating the selection end as cursor moves
func TestSelectionUpdateEnd(t *testing.T) {
	tests := []struct {
		name      string
		startLine int
		startCol  int
		endLine   int
		endCol    int
	}{
		{
			name:      "update to same line different column",
			startLine: 10,
			startCol:  5,
			endLine:   10,
			endCol:    20,
		},
		{
			name:      "update to line below",
			startLine: 10,
			startCol:  5,
			endLine:   15,
			endCol:    0,
		},
		{
			name:      "update to line above",
			startLine: 10,
			startCol:  5,
			endLine:   5,
			endCol:    10,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Selection{
				active:    true,
				startLine: tt.startLine,
				startCol:  tt.startCol,
			}

			s.UpdateEnd(tt.endLine, tt.endCol)

			if s.endLine != tt.endLine {
				t.Errorf("UpdateEnd() endLine = %d, want %d", s.endLine, tt.endLine)
			}
			if s.endCol != tt.endCol {
				t.Errorf("UpdateEnd() endCol = %d, want %d", s.endCol, tt.endCol)
			}
		})
	}
}

// TestSelectionExtractSingleLine tests extracting text from a single line
func TestSelectionExtractSingleLine(t *testing.T) {
	tests := []struct {
		name      string
		content   []string
		startLine int
		endLine   int
		want      string
		wantErr   bool
	}{
		{
			name: "single line without gutter",
			content: []string{
				"  1 │ Hello world",
				"  2 │ Second line",
				"  3 │ Third line",
			},
			startLine: 0,
			endLine:   0,
			want:      "Hello world",
			wantErr:   false,
		},
		{
			name: "single line with hybrid mode gutter (green)",
			content: []string{
				"\x1b[32;1m  1\x1b[0m │ First line",
				"\x1b[33m  1\x1b[0m │ Second line",
				"\x1b[33m  2\x1b[0m │ Third line",
			},
			startLine: 0,
			endLine:   0,
			want:      "First line",
			wantErr:   false,
		},
		{
			name: "single line middle of content",
			content: []string{
				"  1 │ Line one",
				"  2 │ Line two",
				"  3 │ Line three",
				"  4 │ Line four",
			},
			startLine: 2,
			endLine:   2,
			want:      "Line three",
			wantErr:   false,
		},
		{
			name: "single line with wide gutter (4 digits)",
			content: []string{
				" 999 │ Almost there",
				"1000 │ Thousand",
				"1001 │ Beyond",
			},
			startLine: 1,
			endLine:   1,
			want:      "Thousand",
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Selection{
				active:    true,
				startLine: tt.startLine,
				endLine:   tt.endLine,
			}

			got, err := s.Extract(tt.content)
			if (err != nil) != tt.wantErr {
				t.Errorf("Extract() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("Extract() = %q, want %q", got, tt.want)
			}
		})
	}
}

// TestSelectionExtractMultiLine tests extracting text from multiple lines
func TestSelectionExtractMultiLine(t *testing.T) {
	tests := []struct {
		name      string
		content   []string
		startLine int
		endLine   int
		want      string
		wantErr   bool
	}{
		{
			name: "two adjacent lines",
			content: []string{
				"  1 │ First line",
				"  2 │ Second line",
				"  3 │ Third line",
			},
			startLine: 0,
			endLine:   1,
			want:      "First line\nSecond line",
			wantErr:   false,
		},
		{
			name: "multiple lines with hybrid gutter",
			content: []string{
				"\x1b[33m  2\x1b[0m │ Line A",
				"\x1b[33m  1\x1b[0m │ Line B",
				"\x1b[32;1m  5\x1b[0m │ Line C",
				"\x1b[33m  1\x1b[0m │ Line D",
			},
			startLine: 1,
			endLine:   3,
			want:      "Line B\nLine C\nLine D",
			wantErr:   false,
		},
		{
			name: "select all content",
			content: []string{
				"  1 │ Alpha",
				"  2 │ Beta",
				"  3 │ Gamma",
			},
			startLine: 0,
			endLine:   2,
			want:      "Alpha\nBeta\nGamma",
			wantErr:   false,
		},
		{
			name: "reversed selection (end before start)",
			content: []string{
				"  1 │ First",
				"  2 │ Second",
				"  3 │ Third",
				"  4 │ Fourth",
			},
			startLine: 3,
			endLine:   1,
			want:      "Second\nThird\nFourth",
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Selection{
				active:    true,
				startLine: tt.startLine,
				endLine:   tt.endLine,
			}

			got, err := s.Extract(tt.content)
			if (err != nil) != tt.wantErr {
				t.Errorf("Extract() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("Extract() = %q, want %q", got, tt.want)
			}
		})
	}
}

// TestSelectionExtractGutterStripping tests that line number gutters are correctly stripped
func TestSelectionExtractGutterStripping(t *testing.T) {
	tests := []struct {
		name    string
		content []string
		want    string
	}{
		{
			name: "absolute mode gutter",
			content: []string{
				"  1 │ Hello",
				" 42 │ World",
				"123 │ Test",
			},
			want: "Hello\nWorld\nTest",
		},
		{
			name: "relative mode gutter",
			content: []string{
				"  5 │ Above",
				"  0 │ Cursor line",
				"  3 │ Below",
			},
			want: "Above\nCursor line\nBelow",
		},
		{
			name: "hybrid mode with ANSI colors",
			content: []string{
				"\x1b[33m  5\x1b[0m │ Far above",
				"\x1b[33m  1\x1b[0m │ Close above",
				"\x1b[32;1m 42\x1b[0m │ Current",
				"\x1b[33m  1\x1b[0m │ Close below",
			},
			want: "Far above\nClose above\nCurrent\nClose below",
		},
		{
			name: "wide gutter (5 digits)",
			content: []string{
				" 9998 │ Almost",
				" 9999 │ There",
				"10000 │ Big file",
			},
			want: "Almost\nThere\nBig file",
		},
		{
			name: "mixed whitespace after separator",
			content: []string{
				"  1 │ NoSpace",
				"  2 │  TwoSpaces",
				"  3 │ 	Tab",
			},
			want: "NoSpace\n TwoSpaces\n\tTab",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Selection{
				active:    true,
				startLine: 0,
				endLine:   len(tt.content) - 1,
			}

			got, err := s.Extract(tt.content)
			if err != nil {
				t.Fatalf("Extract() error = %v", err)
			}

			if got != tt.want {
				t.Errorf("Extract() = %q, want %q", got, tt.want)
			}

			// Ensure gutter components are NOT present
			for _, line := range strings.Split(got, "\n") {
				if strings.Contains(line, "│") {
					t.Errorf("Extract() result contains gutter separator: %q", line)
				}
				// Check for numeric prefix followed by space
				fields := strings.Fields(line)
				if len(fields) > 0 {
					first := fields[0]
					// First field should not be a pure number (would indicate gutter leak)
					if _, err := strconv.Atoi(first); err == nil && len(first) <= 5 {
						// Could be a legitimate number in content, so check context
						// If it's followed by │, it's definitely a gutter leak
						continue
					}
				}
			}
		})
	}
}

// TestSelectionExtractUTF8 tests that UTF-8 characters are preserved
func TestSelectionExtractUTF8(t *testing.T) {
	tests := []struct {
		name    string
		content []string
		want    string
	}{
		{
			name: "emoji preservation",
			content: []string{
				"  1 │ Hello 👋 world",
				"  2 │ Test 🚀 rockets",
			},
			want: "Hello 👋 world\nTest 🚀 rockets",
		},
		{
			name: "CJK characters",
			content: []string{
				"  1 │ 你好世界",
				"  2 │ こんにちは",
				"  3 │ 안녕하세요",
			},
			want: "你好世界\nこんにちは\n안녕하세요",
		},
		{
			name: "mixed scripts",
			content: []string{
				"  1 │ Hello Мир",
				"  2 │ Café ☕",
			},
			want: "Hello Мир\nCafé ☕",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Selection{
				active:    true,
				startLine: 0,
				endLine:   len(tt.content) - 1,
			}

			got, err := s.Extract(tt.content)
			if err != nil {
				t.Fatalf("Extract() error = %v", err)
			}
			if got != tt.want {
				t.Errorf("Extract() = %q, want %q", got, tt.want)
			}
		})
	}
}

// TestSelectionExtractEmpty tests edge cases with empty content
func TestSelectionExtractEmpty(t *testing.T) {
	tests := []struct {
		name    string
		content []string
		active  bool
		wantErr bool
	}{
		{
			name:    "empty content array",
			content: []string{},
			active:  true,
			wantErr: true,
		},
		{
			name: "selection not active",
			content: []string{
				"  1 │ Line one",
				"  2 │ Line two",
			},
			active:  false,
			wantErr: true,
		},
		{
			name: "empty lines in content",
			content: []string{
				"  1 │ ",
				"  2 │ ",
				"  3 │ ",
			},
			active:  true,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Selection{
				active:    tt.active,
				startLine: 0,
				endLine:   len(tt.content) - 1,
			}

			_, err := s.Extract(tt.content)
			if (err != nil) != tt.wantErr {
				t.Errorf("Extract() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestSelectionBoundaries tests selection boundary handling
func TestSelectionBoundaries(t *testing.T) {
	content := []string{
		"  1 │ Line 1",
		"  2 │ Line 2",
		"  3 │ Line 3",
		"  4 │ Line 4",
		"  5 │ Line 5",
	}

	tests := []struct {
		name      string
		startLine int
		endLine   int
		want      string
		wantErr   bool
	}{
		{
			name:      "first line only",
			startLine: 0,
			endLine:   0,
			want:      "Line 1",
			wantErr:   false,
		},
		{
			name:      "last line only",
			startLine: 4,
			endLine:   4,
			want:      "Line 5",
			wantErr:   false,
		},
		{
			name:      "start beyond content",
			startLine: 10,
			endLine:   10,
			want:      "",
			wantErr:   true,
		},
		{
			name:      "end beyond content",
			startLine: 0,
			endLine:   10,
			want:      "",
			wantErr:   true,
		},
		{
			name:      "negative start",
			startLine: -1,
			endLine:   2,
			want:      "",
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Selection{
				active:    true,
				startLine: tt.startLine,
				endLine:   tt.endLine,
			}

			got, err := s.Extract(content)
			if (err != nil) != tt.wantErr {
				t.Errorf("Extract() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("Extract() = %q, want %q", got, tt.want)
			}
		})
	}
}

// TestSelectionRange tests the Range() method if it exists
func TestSelectionRange(t *testing.T) {
	tests := []struct {
		name          string
		startLine     int
		endLine       int
		wantStartLine int
		wantEndLine   int
	}{
		{
			name:          "forward selection",
			startLine:     5,
			endLine:       10,
			wantStartLine: 5,
			wantEndLine:   10,
		},
		{
			name:          "backward selection (normalized)",
			startLine:     10,
			endLine:       5,
			wantStartLine: 5,
			wantEndLine:   10,
		},
		{
			name:          "single line",
			startLine:     7,
			endLine:       7,
			wantStartLine: 7,
			wantEndLine:   7,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Selection{
				active:    true,
				startLine: tt.startLine,
				endLine:   tt.endLine,
			}

			start, end := s.Range()
			if start != tt.wantStartLine {
				t.Errorf("Range() start = %d, want %d", start, tt.wantStartLine)
			}
			if end != tt.wantEndLine {
				t.Errorf("Range() end = %d, want %d", end, tt.wantEndLine)
			}
		})
	}
}

// TestSelectionIsActive tests checking if selection is active
func TestSelectionIsActive(t *testing.T) {
	tests := []struct {
		name   string
		active bool
		want   bool
	}{
		{
			name:   "active selection",
			active: true,
			want:   true,
		},
		{
			name:   "inactive selection",
			active: false,
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Selection{active: tt.active}
			if got := s.IsActive(); got != tt.want {
				t.Errorf("IsActive() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestSelectionClear tests clearing/resetting selection state
func TestSelectionClear(t *testing.T) {
	s := &Selection{
		active:    true,
		startLine: 10,
		startCol:  5,
		endLine:   20,
		endCol:    15,
	}

	s.Clear()

	if s.active {
		t.Error("Clear() should set active to false")
	}
	// Note: startLine/endLine/cols may or may not be reset to 0
	// depending on implementation choice
}
