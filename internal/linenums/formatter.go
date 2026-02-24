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
	gutterPal   theme.GutterPalette
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
	return NewFormatterWithFullPalette(mode, maxLine, theme.GutterPalette{SeparatorChar: "│"}, pal)
}

// NewFormatterWithFullPalette creates a Formatter with full gutter and line-number palettes.
func NewFormatterWithFullPalette(mode Mode, maxLine int, gutterPal theme.GutterPalette, lineNumPal theme.LineNumPalette) *Formatter {
	if gutterPal.SeparatorChar == "" {
		gutterPal.SeparatorChar = "│"
	}
	f := &Formatter{
		mode:       mode,
		gutterPal:  gutterPal,
		lineNumPal: lineNumPal,
	}
	f.gutterWidth = f.CalculateGutterWidth(maxLine)
	return f
}

// formatNum formats an integer right-aligned in the gutter width, without fmt.Sprintf.
func (f *Formatter) formatNum(n int) string {
	// strconv.AppendInt into a stack-friendly buffer
	var buf [20]byte
	numBytes := strconv.AppendInt(buf[:0], int64(n), 10)
	padLen := f.gutterWidth - len(numBytes)
	if padLen <= 0 {
		return string(numBytes)
	}
	// Build padded string: spaces + digits
	result := make([]byte, f.gutterWidth)
	for i := 0; i < padLen; i++ {
		result[i] = ' '
	}
	copy(result[padLen:], numBytes)
	return string(result)
}

// RenderGutter renders the line number gutter for a given line.
// Returns a formatted string with line number and separator.
func (f *Formatter) RenderGutter(lineNum, cursorLine int) string {
	var b strings.Builder

	// Gutter BG: emit once at the start if set
	gutterBG := f.gutterPal.BG
	if gutterBG != "" {
		b.WriteString(hexBGEscape(string(gutterBG)))
	}

	// Determine number text and style based on mode
	var numText string
	var numFG theme.HexColor
	var numStyle theme.TextStyle

	switch f.mode {
	case ModeAbsolute:
		numText = f.formatNum(lineNum)
		numFG = f.lineNumPal.AbsoluteFG
		numStyle = f.lineNumPal.AbsoluteStyle

	case ModeRelative:
		dist := abs(lineNum - cursorLine)
		numText = f.formatNum(dist)
		numFG = f.lineNumPal.RelativeFG
		numStyle = f.lineNumPal.RelativeStyle

	case ModeHybrid:
		if lineNum == cursorLine {
			numText = f.formatNum(lineNum)
			numFG = f.lineNumPal.CursorFG
			numStyle = f.lineNumPal.CursorStyle
		} else {
			dist := abs(lineNum - cursorLine)
			numText = f.formatNum(dist)
			numFG = f.lineNumPal.RelativeFG
			numStyle = f.lineNumPal.RelativeStyle
		}
	}

	// Left margin
	b.WriteString(" ")

	// Render styled line number
	b.WriteString(styledText(numText, numFG, numStyle))

	// Re-apply gutter BG after styledText reset
	if gutterBG != "" {
		b.WriteString(hexBGEscape(string(gutterBG)))
	}

	// Render separator: " <char> "
	b.WriteString(" ")
	b.WriteString(f.renderSeparator())
	b.WriteString(" ")

	// Reset all attributes
	if gutterBG != "" || numFG != "" || hasStyle(numStyle) {
		b.WriteString("\x1b[0m")
	}

	return b.String()
}

// RenderBlankGutter returns a blank gutter of the same visual width as a
// normal line-number gutter. Used for wrap-continuation rows.
func (f *Formatter) RenderBlankGutter() string {
	var b strings.Builder
	if f.gutterPal.BG != "" {
		b.WriteString(hexBGEscape(string(f.gutterPal.BG)))
	}
	b.WriteString(" ")
	b.WriteString(strings.Repeat(" ", f.gutterWidth))
	b.WriteString(" ")
	b.WriteString(f.renderSeparator())
	b.WriteString(" ")
	if f.gutterPal.BG != "" {
		b.WriteString("\x1b[0m")
	}
	return b.String()
}

// renderSeparator renders the separator char with its own FG/BG/style.
func (f *Formatter) renderSeparator() string {
	ch := f.gutterPal.SeparatorChar
	if ch == "" {
		ch = "│"
	}
	fg := f.gutterPal.SeparatorFG
	bg := f.gutterPal.SeparatorBG
	style := f.gutterPal.SeparatorStyle

	needsEscape := fg != "" || bg != "" || hasStyle(style)
	if !needsEscape {
		return ch
	}

	var b strings.Builder
	var codes []string
	if fg != "" {
		if code := hexToFGCode(string(fg)); code != "" {
			codes = append(codes, code)
		}
	}
	if bg != "" {
		if code := hexToBGCode(string(bg)); code != "" {
			codes = append(codes, code)
		}
	}
	codes = append(codes, styleCodes(style)...)

	if len(codes) > 0 {
		b.WriteString("\x1b[")
		b.WriteString(strings.Join(codes, ";"))
		b.WriteString("m")
	}
	b.WriteString(ch)
	if len(codes) > 0 {
		b.WriteString("\x1b[0m")
		// Re-apply gutter BG after separator reset if set
		if f.gutterPal.BG != "" {
			b.WriteString(hexBGEscape(string(f.gutterPal.BG)))
		}
	}
	return b.String()
}

// styledText wraps text with optional FG color and TextStyle codes.
func styledText(text string, fg theme.HexColor, style theme.TextStyle) string {
	if fg == "" && !hasStyle(style) {
		return text
	}

	var codes []string
	if fg != "" {
		if code := hexToFGCode(string(fg)); code != "" {
			codes = append(codes, code)
		}
	}
	codes = append(codes, styleCodes(style)...)

	if len(codes) == 0 {
		return text
	}

	return "\x1b[" + strings.Join(codes, ";") + "m" + text + "\x1b[0m"
}

// hasStyle returns true if any TextStyle flag is set.
func hasStyle(s theme.TextStyle) bool {
	return s.Bold || s.Dim || s.Italic || s.Underline
}

// styleCodes returns SGR code strings for set TextStyle flags.
func styleCodes(s theme.TextStyle) []string {
	var codes []string
	if s.Bold {
		codes = append(codes, "1")
	}
	if s.Dim {
		codes = append(codes, "2")
	}
	if s.Italic {
		codes = append(codes, "3")
	}
	if s.Underline {
		codes = append(codes, "4")
	}
	return codes
}

// hexToFGCode returns a 24-bit foreground SGR code for "#rrggbb".
func hexToFGCode(hex string) string {
	r, g, b, ok := parseHex(hex)
	if !ok {
		return ""
	}
	return fmt.Sprintf("38;2;%d;%d;%d", r, g, b)
}

// hexToBGCode returns a 24-bit background SGR code for "#rrggbb".
func hexToBGCode(hex string) string {
	r, g, b, ok := parseHex(hex)
	if !ok {
		return ""
	}
	return fmt.Sprintf("48;2;%d;%d;%d", r, g, b)
}

// hexBGEscape returns a complete background escape sequence for "#rrggbb".
func hexBGEscape(hex string) string {
	code := hexToBGCode(hex)
	if code == "" {
		return ""
	}
	return "\x1b[" + code + "m"
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
	var buf [20]byte
	return len(strconv.AppendInt(buf[:0], int64(maxLine), 10))
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
