package selection

import (
	"fmt"
	"regexp"
	"strings"
)

// Selection manages visual selection state
type Selection struct {
	active    bool
	startLine int
	startCol  int
	endLine   int
	endCol    int
}

// New creates a new Selection
func New() *Selection {
	return &Selection{
		active: false,
	}
}

// Toggle toggles selection mode on/off
func (s *Selection) Toggle() {
	s.active = !s.active
}

// UpdateEnd updates the selection end position as cursor moves
func (s *Selection) UpdateEnd(line, col int) {
	s.endLine = line
	s.endCol = col
}

// SetStart sets the selection start position
func (s *Selection) SetStart(line, col int) {
	s.startLine = line
	s.startCol = col
}

// Extract extracts selected text from content, stripping line number gutters
// Content should be the rendered lines including gutters
func (s *Selection) Extract(content []string) (string, error) {
	if !s.active {
		return "", fmt.Errorf("selection is not active")
	}

	if len(content) == 0 {
		return "", fmt.Errorf("content is empty")
	}

	// Get normalized range
	start, end := s.Range()

	// Validate bounds
	if start < 0 || end >= len(content) {
		return "", fmt.Errorf("selection range [%d,%d] out of bounds for content length %d", start, end, len(content))
	}

	// Extract lines
	var lines []string
	for i := start; i <= end; i++ {
		line := content[i]
		// Strip gutter from line
		stripped := stripGutter(line)
		lines = append(lines, stripped)
	}

	return strings.Join(lines, "\n"), nil
}

// stripGutter removes the line number gutter from a line
// Gutter format: [ANSI codes] <spaces> <number> [ANSI codes] <spaces> │ <content>
func stripGutter(line string) string {
	// Pattern matches: optional ANSI codes, whitespace, digits, optional ANSI codes, whitespace, │, space, then content
	// We want to keep everything after "│ " (separator + single space)

	// First, find the separator │
	sepIdx := strings.Index(line, "│")
	if sepIdx == -1 {
		// No separator found, return as-is (shouldn't happen in normal operation)
		return line
	}

	// Return everything after "│ " (skip the separator and exactly ONE space after it)
	// The gutter format is always: <gutter> │ <space> <content>
	// We want to return <content>, preserving ALL whitespace in it
	afterSep := line[sepIdx+len("│"):]

	// Remove exactly one leading space if present (the gutter separator space)
	// DO NOT remove tabs or other whitespace - those are part of the content
	if len(afterSep) > 0 && afterSep[0] == ' ' {
		return afterSep[1:]
	}

	return afterSep
}

// Range returns the normalized selection range (start <= end)
func (s *Selection) Range() (start, end int) {
	if s.startLine <= s.endLine {
		return s.startLine, s.endLine
	}
	return s.endLine, s.startLine
}

// IsActive returns whether selection is currently active
func (s *Selection) IsActive() bool {
	return s.active
}

// Clear deactivates the selection
func (s *Selection) Clear() {
	s.active = false
}

// stripANSI removes ANSI escape codes from a string
func stripANSI(s string) string {
	re := regexp.MustCompile(`\x1b\[[0-9;]*m`)
	return re.ReplaceAllString(s, "")
}
