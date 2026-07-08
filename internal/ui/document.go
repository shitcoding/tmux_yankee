package ui

import (
	"strings"
	"unicode/utf8"
)

// Line represents a single line in the document with plain text and parsed cells.
type Line struct {
	Plain     string // Line with ANSI codes stripped (for motion calculations)
	Cells     []Cell // Pre-parsed ANSI cells (cached at load time to avoid per-frame reparse)
	RuneCount int    // Number of runes in Plain (cached to avoid repeated allocation)
}

// Document holds pane content with color preservation.
type Document struct {
	lines []Line
}

// NewDocument creates a document from lines with ANSI codes.
func NewDocument(rawLines []string) *Document {
	lines := make([]Line, len(rawLines))
	for i, raw := range rawLines {
		plain := stripANSI(raw)
		lines[i] = Line{
			Plain:     plain,
			Cells:     ParseANSILine(raw),
			RuneCount: utf8.RuneCountInString(plain),
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

// Cells returns the pre-parsed ANSI cells for the line at the given index.
// Returns nil for out-of-bounds indices.
func (d *Document) Cells(index int) []Cell {
	if index < 0 || index >= len(d.lines) {
		return nil
	}
	return d.lines[index].Cells
}

// LineRuneCount returns the number of runes in the plain text line.
func (d *Document) LineRuneCount(index int) int {
	if index < 0 || index >= len(d.lines) {
		return 0
	}
	return d.lines[index].RuneCount
}

// stripANSI removes ANSI escape sequences from s, returning the plain text
// that downstream code (search, motion, yank) consumes via Document.Plain.
//
// The scanner in escape_scanner.go is the source of truth for which byte
// ranges constitute an escape sequence. By routing every ESC introducer
// through scanEscape we ensure stripANSI and ParseANSILine agree exactly on
// what's "escape" vs "text" — closing escape-injection paths (CSI with
// non-letter terminators, OSC titles/hyperlinks, DCS/APC/PM/SOS payloads,
// SS2/SS3 single shifts, charset designations, DEC screen-alignment test,
// UTF-8 mode switches, and so on) at the parser boundary.
func stripANSI(s string) string {
	// Fast path: no escape characters -> return as-is (no allocation).
	if strings.IndexByte(s, 0x1b) < 0 {
		return s
	}

	runes := []rune(s)
	var b strings.Builder
	b.Grow(len(s))
	i := 0
	for i < len(runes) {
		if runes[i] == '\x1b' {
			i = scanEscape(runes, i)
			continue
		}
		b.WriteRune(runes[i])
		i++
	}
	return b.String()
}
