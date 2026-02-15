package selection

import (
	"fmt"
	"strings"
)

// Kind represents the type of selection
type Kind int

const (
	// KindNone represents no selection
	KindNone Kind = iota
	// KindChar represents character-wise selection (vim 'v')
	KindChar
	// KindLine represents line-wise selection (vim 'V')
	KindLine
)

// Pos represents a position in the buffer (0-based line and column)
// Column is measured in runes (UTF-8 code points), not bytes
type Pos struct {
	Line int // 0-based line number
	Col  int // 0-based column (rune index)
}

// Region represents a selected region with start and end positions
type Region struct {
	Kind  Kind // Selection kind (character-wise or line-wise)
	Start Pos  // Start position
	End   Pos  // End position (inclusive)
	// Active indicates which end is the "cursor" (true = End, false = Start)
	// Used for vim-style selection expansion
	Active bool
}

// EmptyRegion returns a region with KindNone and zero positions
// Used to represent "no selection" state
func EmptyRegion() Region {
	return Region{
		Kind:  KindNone,
		Start: Pos{Line: 0, Col: 0},
		End:   Pos{Line: 0, Col: 0},
	}
}

// ExtractRegion extracts text from lines based on the region
// Handles both character-wise and line-wise selections
// Automatically normalizes reversed selections (where End < Start)
func ExtractRegion(lines []string, region Region) (string, error) {
	if len(lines) == 0 {
		return "", fmt.Errorf("empty content")
	}

	// Normalize positions (ensure start <= end)
	start, end := region.Start, region.End
	if start.Line > end.Line || (start.Line == end.Line && start.Col > end.Col) {
		start, end = end, start
	}

	// Validate line bounds
	if start.Line < 0 || end.Line >= len(lines) {
		return "", fmt.Errorf("line out of bounds: start=%d, end=%d, len=%d", start.Line, end.Line, len(lines))
	}

	switch region.Kind {
	case KindLine:
		return extractLineWise(lines, start, end)
	case KindChar:
		return extractCharWise(lines, start, end)
	default:
		return "", fmt.Errorf("invalid selection kind: %v", region.Kind)
	}
}

// extractLineWise extracts complete lines from start.Line to end.Line (inclusive)
func extractLineWise(lines []string, start, end Pos) (string, error) {
	var result []string
	for i := start.Line; i <= end.Line; i++ {
		result = append(result, lines[i])
	}
	return strings.Join(result, "\n"), nil
}

// extractCharWise extracts characters from start to end position
// Supports single-line and multi-line selections
func extractCharWise(lines []string, start, end Pos) (string, error) {
	if start.Line == end.Line {
		// Single-line selection
		return extractSingleLine(lines[start.Line], start.Col, end.Col)
	}

	// Multi-line selection
	var result []string

	// First line: from start.Col to end of line
	firstLine, err := extractSingleLine(lines[start.Line], start.Col, -1)
	if err != nil {
		return "", fmt.Errorf("extracting first line: %w", err)
	}
	result = append(result, firstLine)

	// Middle lines: complete lines
	for i := start.Line + 1; i < end.Line; i++ {
		result = append(result, lines[i])
	}

	// Last line: from beginning to end.Col
	lastLine, err := extractSingleLine(lines[end.Line], 0, end.Col)
	if err != nil {
		return "", fmt.Errorf("extracting last line: %w", err)
	}
	result = append(result, lastLine)

	return strings.Join(result, "\n"), nil
}

// extractSingleLine extracts a substring from a line using rune indices
// If endCol is -1, extracts to end of line
func extractSingleLine(line string, startCol, endCol int) (string, error) {
	runes := []rune(line)

	if startCol < 0 || startCol > len(runes) {
		return "", fmt.Errorf("start column out of bounds: %d (line length: %d)", startCol, len(runes))
	}

	if endCol == -1 {
		endCol = len(runes)
	}

	if endCol < 0 || endCol > len(runes) {
		return "", fmt.Errorf("end column out of bounds: %d (line length: %d)", endCol, len(runes))
	}

	if startCol > endCol {
		return "", fmt.Errorf("start column > end column: %d > %d", startCol, endCol)
	}

	return string(runes[startCol:endCol]), nil
}
