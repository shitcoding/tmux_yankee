package ui

import "strings"

// Line represents a single line in the document with both raw ANSI and plain text.
type Line struct {
	RawANSI string // Original line with ANSI escape codes
	Plain   string // Line with ANSI codes stripped (for motion calculations)
}

// Document holds pane content with color preservation.
type Document struct {
	lines []Line
}

// NewDocument creates a document from lines with ANSI codes.
func NewDocument(rawLines []string) *Document {
	lines := make([]Line, len(rawLines))
	for i, raw := range rawLines {
		lines[i] = Line{
			RawANSI: raw,
			Plain:   stripANSI(raw),
		}
	}
	return &Document{lines: lines}
}

// LineCount returns the total number of lines.
func (d *Document) LineCount() int {
	return len(d.lines)
}

// Line returns the plain text content of the line at the given index.
func (d *Document) Line(index int) string {
	if index < 0 || index >= len(d.lines) {
		return ""
	}
	return d.lines[index].Plain
}

// RawLine returns the raw ANSI content of the line at the given index.
func (d *Document) RawLine(index int) string {
	if index < 0 || index >= len(d.lines) {
		return ""
	}
	return d.lines[index].RawANSI
}

// LineRuneCount returns the number of runes in the plain text line.
func (d *Document) LineRuneCount(index int) int {
	if index < 0 || index >= len(d.lines) {
		return 0
	}
	return len([]rune(d.lines[index].Plain))
}

// stripANSI removes ANSI escape codes from a string.
// This is a simple implementation that removes CSI sequences (ESC [ ... m).
func stripANSI(s string) string {
	var result strings.Builder
	inEscape := false
	inCSI := false

	for _, r := range s {
		if r == '\x1b' { // ESC
			inEscape = true
			continue
		}

		if inEscape {
			if r == '[' {
				inCSI = true
				inEscape = false
				continue
			}
			// Unknown escape sequence, skip this character
			inEscape = false
			continue
		}

		if inCSI {
			// CSI sequence ends with a letter (A-Z, a-z)
			if (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') {
				inCSI = false
			}
			continue
		}

		result.WriteRune(r)
	}

	return result.String()
}
