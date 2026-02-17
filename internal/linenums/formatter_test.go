package linenums

import (
	"strings"
	"testing"

	"github.com/shitcoding/tmux_yankee/internal/theme"
)

// TestFormatterRenderGutterAbsoluteMode tests absolute line number rendering
func TestFormatterRenderGutterAbsoluteMode(t *testing.T) {
	tests := []struct {
		name         string
		lineNum      int
		cursorLine   int
		gutterWidth  int
		wantContains string
		wantPrefix   string
	}{
		{
			name:         "single digit line",
			lineNum:      1,
			cursorLine:   5,
			gutterWidth:  3,
			wantContains: "1",
			wantPrefix:   "  1 │ ",
		},
		{
			name:         "double digit line",
			lineNum:      42,
			cursorLine:   5,
			gutterWidth:  3,
			wantContains: "42",
			wantPrefix:   " 42 │ ",
		},
		{
			name:         "triple digit line",
			lineNum:      123,
			cursorLine:   50,
			gutterWidth:  4,
			wantContains: "123",
			wantPrefix:   " 123 │ ",
		},
		{
			name:         "line at cursor position",
			lineNum:      10,
			cursorLine:   10,
			gutterWidth:  3,
			wantContains: "10",
			wantPrefix:   " 10 │ ",
		},
		{
			name:         "wide gutter for large files",
			lineNum:      9999,
			cursorLine:   5000,
			gutterWidth:  5,
			wantContains: "9999",
			wantPrefix:   " 9999 │ ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := &Formatter{
				mode:        ModeAbsolute,
				gutterWidth: tt.gutterWidth,
			}

			got := f.RenderGutter(tt.lineNum, tt.cursorLine)

			if !strings.Contains(got, tt.wantContains) {
				t.Errorf("RenderGutter() = %q, want to contain %q", got, tt.wantContains)
			}

			if got != tt.wantPrefix {
				t.Errorf("RenderGutter() = %q, want %q", got, tt.wantPrefix)
			}

			// Should not contain ANSI color codes in absolute mode
			if strings.Contains(got, "\x1b[") {
				t.Errorf("RenderGutter() absolute mode should not contain ANSI codes, got %q", got)
			}
		})
	}
}

// TestFormatterRenderGutterRelativeMode tests relative line number rendering
func TestFormatterRenderGutterRelativeMode(t *testing.T) {
	tests := []struct {
		name         string
		lineNum      int
		cursorLine   int
		gutterWidth  int
		wantDistance int
		wantPrefix   string
	}{
		{
			name:         "cursor line shows 0",
			lineNum:      10,
			cursorLine:   10,
			gutterWidth:  3,
			wantDistance: 0,
			wantPrefix:   "  0 │ ",
		},
		{
			name:         "line above cursor",
			lineNum:      5,
			cursorLine:   10,
			gutterWidth:  3,
			wantDistance: 5,
			wantPrefix:   "  5 │ ",
		},
		{
			name:         "line below cursor",
			lineNum:      15,
			cursorLine:   10,
			gutterWidth:  3,
			wantDistance: 5,
			wantPrefix:   "  5 │ ",
		},
		{
			name:         "adjacent line above",
			lineNum:      9,
			cursorLine:   10,
			gutterWidth:  3,
			wantDistance: 1,
			wantPrefix:   "  1 │ ",
		},
		{
			name:         "adjacent line below",
			lineNum:      11,
			cursorLine:   10,
			gutterWidth:  3,
			wantDistance: 1,
			wantPrefix:   "  1 │ ",
		},
		{
			name:         "far from cursor",
			lineNum:      100,
			cursorLine:   10,
			gutterWidth:  3,
			wantDistance: 90,
			wantPrefix:   " 90 │ ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := &Formatter{
				mode:        ModeRelative,
				gutterWidth: tt.gutterWidth,
			}

			got := f.RenderGutter(tt.lineNum, tt.cursorLine)

			if got != tt.wantPrefix {
				t.Errorf("RenderGutter() = %q, want %q", got, tt.wantPrefix)
			}

			// Should not contain ANSI color codes in relative mode
			if strings.Contains(got, "\x1b[") {
				t.Errorf("RenderGutter() relative mode should not contain ANSI codes, got %q", got)
			}
		})
	}
}

// TestFormatterRenderGutterHybridMode tests hybrid line number rendering
func TestFormatterRenderGutterHybridMode(t *testing.T) {
	// Use default theme palette colors:
	//   CursorFG   = "#b8bb26" → rgb(184,187,38)  → 38;2;184;187;38
	//   RelativeFG = "#fabd2f" → rgb(250,189,47)  → 38;2;250;189;47
	defaultPal := theme.Presets[theme.ThemeDefault].LineNum

	tests := []struct {
		name              string
		lineNum           int
		cursorLine        int
		gutterWidth       int
		wantNumber        string
		wantColorContains string // expected ANSI fragment
		shouldContainANSI bool
	}{
		{
			name:              "cursor line - palette cursor color absolute",
			lineNum:           10,
			cursorLine:        10,
			gutterWidth:       3,
			wantNumber:        " 10 │ ",
			wantColorContains: "38;2;184;187;38",
			shouldContainANSI: true,
		},
		{
			name:              "line above - palette relative color",
			lineNum:           8,
			cursorLine:        10,
			gutterWidth:       3,
			wantNumber:        "  2 │ ", // distance is 2
			wantColorContains: "38;2;250;189;47",
			shouldContainANSI: true,
		},
		{
			name:              "line below - palette relative color",
			lineNum:           15,
			cursorLine:        10,
			gutterWidth:       3,
			wantNumber:        "  5 │ ", // distance is 5
			wantColorContains: "38;2;250;189;47",
			shouldContainANSI: true,
		},
		{
			name:              "triple digit cursor line",
			lineNum:           123,
			cursorLine:        123,
			gutterWidth:       4,
			wantNumber:        " 123 │ ",
			wantColorContains: "38;2;184;187;38",
			shouldContainANSI: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := NewFormatterWithPalette(ModeHybrid, 999, defaultPal)
			f.gutterWidth = tt.gutterWidth

			got := f.RenderGutter(tt.lineNum, tt.cursorLine)

			// Check for ANSI codes
			if tt.shouldContainANSI && !strings.Contains(got, "\x1b[") {
				t.Errorf("RenderGutter() hybrid mode should contain ANSI codes, got %q", got)
			}

			// Check for correct color fragment
			if tt.wantColorContains != "" && !strings.Contains(got, tt.wantColorContains) {
				t.Errorf("RenderGutter() should contain color %q, got %q", tt.wantColorContains, got)
			}

			// Must NOT contain old hardcoded green/yellow
			if strings.Contains(got, "\x1b[32;1m") {
				t.Errorf("RenderGutter() must not use hardcoded green (32;1m), got %q", got)
			}
			if strings.Contains(got, "\x1b[33m") {
				t.Errorf("RenderGutter() must not use hardcoded yellow (33m), got %q", got)
			}

			// Check that reset code is present
			if tt.shouldContainANSI && !strings.Contains(got, "\x1b[0m") {
				t.Errorf("RenderGutter() should include reset code (0m), got %q", got)
			}

			// Strip ANSI codes and verify number format
			stripped := stripANSI(got)
			if stripped != tt.wantNumber {
				t.Errorf("RenderGutter() after stripping ANSI = %q, want %q", stripped, tt.wantNumber)
			}
		})
	}
}

// TestFormatterCalculateGutterWidth tests gutter width calculation
func TestFormatterCalculateGutterWidth(t *testing.T) {
	tests := []struct {
		name      string
		maxLine   int
		wantWidth int
	}{
		{
			name:      "single digit max",
			maxLine:   9,
			wantWidth: 1,
		},
		{
			name:      "double digit max",
			maxLine:   99,
			wantWidth: 2,
		},
		{
			name:      "triple digit max",
			maxLine:   999,
			wantWidth: 3,
		},
		{
			name:      "four digit max",
			maxLine:   9999,
			wantWidth: 4,
		},
		{
			name:      "five digit max",
			maxLine:   50000,
			wantWidth: 5,
		},
		{
			name:      "boundary case 10",
			maxLine:   10,
			wantWidth: 2,
		},
		{
			name:      "boundary case 100",
			maxLine:   100,
			wantWidth: 3,
		},
		{
			name:      "boundary case 1000",
			maxLine:   1000,
			wantWidth: 4,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := &Formatter{}
			got := f.CalculateGutterWidth(tt.maxLine)
			if got != tt.wantWidth {
				t.Errorf("CalculateGutterWidth(%d) = %d, want %d", tt.maxLine, got, tt.wantWidth)
			}
		})
	}
}

// TestFormatterModeToggle tests cycling through modes
func TestFormatterModeToggle(t *testing.T) {
	tests := []struct {
		name         string
		currentMode  Mode
		wantNextMode Mode
	}{
		{
			name:         "hybrid to absolute",
			currentMode:  ModeHybrid,
			wantNextMode: ModeAbsolute,
		},
		{
			name:         "absolute to relative",
			currentMode:  ModeAbsolute,
			wantNextMode: ModeRelative,
		},
		{
			name:         "relative to hybrid",
			currentMode:  ModeRelative,
			wantNextMode: ModeHybrid,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := &Formatter{mode: tt.currentMode}
			f.ToggleMode()
			if f.mode != tt.wantNextMode {
				t.Errorf("ToggleMode() from %v = %v, want %v", tt.currentMode, f.mode, tt.wantNextMode)
			}
		})
	}
}

// TestFormatterNewFormatter tests formatter initialization
func TestFormatterNewFormatter(t *testing.T) {
	tests := []struct {
		name         string
		mode         Mode
		maxLine      int
		wantMode     Mode
		wantGutWidth int
	}{
		{
			name:         "absolute mode small file",
			mode:         ModeAbsolute,
			maxLine:      50,
			wantMode:     ModeAbsolute,
			wantGutWidth: 2,
		},
		{
			name:         "relative mode large file",
			mode:         ModeRelative,
			maxLine:      5000,
			wantMode:     ModeRelative,
			wantGutWidth: 4,
		},
		{
			name:         "hybrid mode default",
			mode:         ModeHybrid,
			maxLine:      999,
			wantMode:     ModeHybrid,
			wantGutWidth: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := NewFormatter(tt.mode, tt.maxLine)
			if f.mode != tt.wantMode {
				t.Errorf("NewFormatter() mode = %v, want %v", f.mode, tt.wantMode)
			}
			if f.gutterWidth != tt.wantGutWidth {
				t.Errorf("NewFormatter() gutterWidth = %d, want %d", f.gutterWidth, tt.wantGutWidth)
			}
		})
	}
}

// TestFormatterRenderGutterAlignment tests right-alignment of numbers
func TestFormatterRenderGutterAlignment(t *testing.T) {
	f := &Formatter{
		mode:        ModeAbsolute,
		gutterWidth: 4,
	}

	tests := []struct {
		lineNum int
		want    string
	}{
		{1, "   1 │ "},
		{10, "  10 │ "},
		{100, " 100 │ "},
		{1000, "1000 │ "},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := f.RenderGutter(tt.lineNum, 0)
			if got != tt.want {
				t.Errorf("RenderGutter(%d) = %q, want %q (check alignment)", tt.lineNum, got, tt.want)
			}
		})
	}
}

// TestFormatterModeFromString tests parsing mode from string
func TestFormatterModeFromString(t *testing.T) {
	tests := []struct {
		input   string
		want    Mode
		wantErr bool
	}{
		{"absolute", ModeAbsolute, false},
		{"relative", ModeRelative, false},
		{"hybrid", ModeHybrid, false},
		{"ABSOLUTE", ModeAbsolute, false}, // case insensitive
		{"Hybrid", ModeHybrid, false},
		{"invalid", ModeHybrid, true}, // default to hybrid on error
		{"", ModeHybrid, true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := ModeFromString(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ModeFromString(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("ModeFromString(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

// stripANSI removes ANSI escape codes from a string
func stripANSI(s string) string {
	// Simple ANSI stripper for testing
	result := ""
	inEscape := false
	for _, r := range s {
		if r == '\x1b' {
			inEscape = true
			continue
		}
		if inEscape {
			if r == 'm' {
				inEscape = false
			}
			continue
		}
		result += string(r)
	}
	return result
}

// Benchmark tests for performance validation
func BenchmarkFormatterRenderGutterAbsolute(b *testing.B) {
	f := &Formatter{
		mode:        ModeAbsolute,
		gutterWidth: 4,
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		f.RenderGutter(i%10000, 5000)
	}
}

func BenchmarkFormatterRenderGutterRelative(b *testing.B) {
	f := &Formatter{
		mode:        ModeRelative,
		gutterWidth: 4,
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		f.RenderGutter(i%10000, 5000)
	}
}

func BenchmarkFormatterRenderGutterHybrid(b *testing.B) {
	f := &Formatter{
		mode:        ModeHybrid,
		gutterWidth: 4,
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		f.RenderGutter(i%10000, 5000)
	}
}
