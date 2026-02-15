package linenums

import (
	"fmt"
	"strings"
)

// Mode represents the line number display mode
type Mode int

const (
	// ModeAbsolute displays actual line numbers
	ModeAbsolute Mode = iota
	// ModeRelative displays distance from cursor
	ModeRelative
	// ModeHybrid displays absolute at cursor (green), relative elsewhere (yellow)
	ModeHybrid
)

// Formatter handles line number formatting for the gutter
type Formatter struct {
	mode        Mode
	gutterWidth int
}

// NewFormatter creates a new Formatter with the given mode and max line number
func NewFormatter(mode Mode, maxLine int) *Formatter {
	f := &Formatter{
		mode: mode,
	}
	f.gutterWidth = f.CalculateGutterWidth(maxLine)
	return f
}

// RenderGutter renders the line number gutter for a given line
// Returns a formatted string with line number and separator
func (f *Formatter) RenderGutter(lineNum, cursorLine int) string {
	switch f.mode {
	case ModeAbsolute:
		return fmt.Sprintf("%*d │ ", f.gutterWidth, lineNum)
	case ModeRelative:
		dist := abs(lineNum - cursorLine)
		return fmt.Sprintf("%*d │ ", f.gutterWidth, dist)
	case ModeHybrid:
		if lineNum == cursorLine {
			return fmt.Sprintf("\x1b[32;1m%*d\x1b[0m │ ", f.gutterWidth, lineNum)
		}
		dist := abs(lineNum - cursorLine)
		return fmt.Sprintf("\x1b[33m%*d\x1b[0m │ ", f.gutterWidth, dist)
	}
	return ""
}

// CalculateGutterWidth calculates the width needed for the gutter
// based on the maximum line number
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
