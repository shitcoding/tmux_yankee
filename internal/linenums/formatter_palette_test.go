package linenums

import (
	"strings"
	"testing"

	"github.com/shitcoding/tmux_yankee/internal/theme"
)

// TestFormatter_GutterColorFromPalette verifies that the formatter uses palette colors
// for hybrid mode rendering rather than hardcoded ANSI color codes.
func TestFormatter_GutterColorFromPalette(t *testing.T) {
	pal := theme.LineNumPalette{
		CursorFG:    "#00ffff", // cyan for cursor line
		RelativeFG:  "#ff00ff", // magenta for relative lines
		CursorStyle: theme.TextStyle{Bold: true},
	}

	f := NewFormatterWithPalette(ModeHybrid, 10, pal)

	// Test cursor line: should use cyan (#00ffff = rgb(0,255,255))
	cursorGutter := f.RenderGutter(5, 5) // lineNum == cursorLine
	if !strings.Contains(cursorGutter, "38;2;0;255;255") {
		t.Errorf("cursor line should use palette.CursorFG (#00ffff), got: %q", cursorGutter)
	}

	// Must NOT contain old hardcoded green (32;1m)
	if strings.Contains(cursorGutter, "\x1b[32;1m") {
		t.Errorf("cursor line must not use hardcoded green (32;1m), got: %q", cursorGutter)
	}

	// Test non-cursor line: should use magenta (#ff00ff = rgb(255,0,255))
	relGutter := f.RenderGutter(8, 5) // lineNum != cursorLine
	if !strings.Contains(relGutter, "38;2;255;0;255") {
		t.Errorf("relative line should use palette.RelativeFG (#ff00ff), got: %q", relGutter)
	}

	// Must NOT contain old hardcoded yellow (33m)
	if strings.Contains(relGutter, "\x1b[33m") {
		t.Errorf("relative line must not use hardcoded yellow (33m), got: %q", relGutter)
	}
}

// TestFormatter_GutterEmptyColorNoEscape verifies that empty palette colors
// don't emit any color escape sequences in absolute/relative modes.
func TestFormatter_GutterEmptyColorNoEscape(t *testing.T) {
	pal := theme.LineNumPalette{
		AbsoluteFG: "",
		RelativeFG: "",
		CursorFG:   "",
	}

	fAbs := NewFormatterWithPalette(ModeAbsolute, 10, pal)
	got := fAbs.RenderGutter(3, 5)
	if strings.Contains(got, "\x1b[") {
		t.Errorf("absolute mode with empty palette should not emit ANSI codes, got: %q", got)
	}

	fRel := NewFormatterWithPalette(ModeRelative, 10, pal)
	got = fRel.RenderGutter(3, 5)
	if strings.Contains(got, "\x1b[") {
		t.Errorf("relative mode with empty palette should not emit ANSI codes, got: %q", got)
	}
}
