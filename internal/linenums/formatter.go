package linenums

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/shitcoding/tmux_yankee/internal/theme"
)

// Mode represents the line number display mode
type Mode int

const (
	// ModeAbsolute displays actual line numbers
	ModeAbsolute Mode = iota
	// ModeRelative displays distance from cursor
	ModeRelative
	// ModeHybrid displays absolute at cursor, relative elsewhere, colored via palette
	ModeHybrid
)

// Formatter handles line number formatting for the gutter
type Formatter struct {
	mode        Mode
	gutterWidth int
	lineNumPal  theme.LineNumPalette
}

// NewFormatter creates a new Formatter with the given mode and max line number.
// Colors default to empty (terminal default — no color output).
func NewFormatter(mode Mode, maxLine int) *Formatter {
	return NewFormatterWithPalette(mode, maxLine, theme.LineNumPalette{})
}

// NewFormatterWithPalette creates a Formatter that uses the provided palette for
// line number coloring in hybrid (and optionally absolute/relative) modes.
func NewFormatterWithPalette(mode Mode, maxLine int, pal theme.LineNumPalette) *Formatter {
	f := &Formatter{
		mode:       mode,
		lineNumPal: pal,
	}
	f.gutterWidth = f.CalculateGutterWidth(maxLine)
	return f
}

// RenderGutter renders the line number gutter for a given line.
// Returns a formatted string with line number and separator.
func (f *Formatter) RenderGutter(lineNum, cursorLine int) string {
	switch f.mode {
	case ModeAbsolute:
		num := fmt.Sprintf("%*d", f.gutterWidth, lineNum)
		if f.lineNumPal.AbsoluteFG != "" {
			return hexFGWrap(string(f.lineNumPal.AbsoluteFG), num) + " │ "
		}
		return num + " │ "

	case ModeRelative:
		dist := abs(lineNum - cursorLine)
		num := fmt.Sprintf("%*d", f.gutterWidth, dist)
		if f.lineNumPal.RelativeFG != "" {
			return hexFGWrap(string(f.lineNumPal.RelativeFG), num) + " │ "
		}
		return num + " │ "

	case ModeHybrid:
		if lineNum == cursorLine {
			num := fmt.Sprintf("%*d", f.gutterWidth, lineNum)
			if f.lineNumPal.CursorFG != "" {
				fg := string(f.lineNumPal.CursorFG)
				styled := hexFGWrap(fg, num)
				if f.lineNumPal.CursorBold {
					styled = "\x1b[1m" + hexFGWrap(fg, num) + "\x1b[0m"
					// Re-wrap so reset appears only once at end
					styled = hexFGBoldWrap(fg, num)
				}
				return styled + " │ "
			}
			// No palette color: fall back to plain bold
			return fmt.Sprintf("\x1b[1m%*d\x1b[0m │ ", f.gutterWidth, lineNum)
		}
		dist := abs(lineNum - cursorLine)
		num := fmt.Sprintf("%*d", f.gutterWidth, dist)
		if f.lineNumPal.RelativeFG != "" {
			return hexFGWrap(string(f.lineNumPal.RelativeFG), num) + " │ "
		}
		return num + " │ "
	}
	return ""
}

// RenderBlankGutter returns a blank gutter of the same visual width as a
// normal line-number gutter. Used for wrap-continuation rows.
func (f *Formatter) RenderBlankGutter() string {
	return strings.Repeat(" ", f.gutterWidth) + " │ "
}

// hexFGWrap wraps text in a 24-bit foreground color escape + reset.
// hex must be a "#rrggbb" string.
func hexFGWrap(hex, text string) string {
	r, g, b, ok := parseHex(hex)
	if !ok {
		return text
	}
	return fmt.Sprintf("\x1b[38;2;%d;%d;%dm%s\x1b[0m", r, g, b, text)
}

// hexFGBoldWrap wraps text in bold + 24-bit foreground color + reset.
func hexFGBoldWrap(hex, text string) string {
	r, g, b, ok := parseHex(hex)
	if !ok {
		return "\x1b[1m" + text + "\x1b[0m"
	}
	return fmt.Sprintf("\x1b[1;38;2;%d;%d;%dm%s\x1b[0m", r, g, b, text)
}

// parseHex parses a "#rrggbb" string into r, g, b components.
func parseHex(hex string) (r, g, b int, ok bool) {
	hex = strings.TrimPrefix(hex, "#")
	if len(hex) != 6 {
		return 0, 0, 0, false
	}
	rv, err := strconv.ParseUint(hex[0:2], 16, 8)
	if err != nil {
		return 0, 0, 0, false
	}
	gv, err := strconv.ParseUint(hex[2:4], 16, 8)
	if err != nil {
		return 0, 0, 0, false
	}
	bv, err := strconv.ParseUint(hex[4:6], 16, 8)
	if err != nil {
		return 0, 0, 0, false
	}
	return int(rv), int(gv), int(bv), true
}

// CalculateGutterWidth calculates the width needed for the gutter
// based on the maximum line number.
func (f *Formatter) CalculateGutterWidth(maxLine int) int {
	return len(fmt.Sprintf("%d", maxLine))
}

// ToggleMode cycles through modes: hybrid → absolute → relative → hybrid
func (f *Formatter) ToggleMode() {
	switch f.mode {
	case ModeHybrid:
		f.mode = ModeAbsolute
	case ModeAbsolute:
		f.mode = ModeRelative
	case ModeRelative:
		f.mode = ModeHybrid
	}
}

// CurrentMode returns the current mode
func (f *Formatter) CurrentMode() Mode {
	return f.mode
}

// ModeFromString parses a mode string into a Mode type
// Returns ModeHybrid and error if input is invalid
func ModeFromString(s string) (Mode, error) {
	switch strings.ToLower(s) {
	case "absolute":
		return ModeAbsolute, nil
	case "relative":
		return ModeRelative, nil
	case "hybrid":
		return ModeHybrid, nil
	default:
		return ModeHybrid, fmt.Errorf("invalid mode: %s", s)
	}
}

// abs returns the absolute value of x
func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}
