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
	// KindBlock represents block-wise (rectangular) selection (vim Ctrl-V)
	KindBlock
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
	case KindBlock:
		return extractBlockWise(lines, start, end)
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

// extractBlockWise extracts a rectangular column selection
// For each line in [start.Line, end.Line]: extract runes [minCol, maxCol] (inclusive)
func extractBlockWise(lines []string, start, end Pos) (string, error) {
	minCol := start.Col
	maxCol := end.Col
	if minCol > maxCol {
		minCol, maxCol = maxCol, minCol
	}

	var result []string
	for i := start.Line; i <= end.Line; i++ {
		runes := []rune(lines[i])
		if minCol >= len(runes) {
			// Line is shorter than minCol: emit empty string
			result = append(result, "")
		} else if maxCol >= len(runes) {
			// Line is shorter than maxCol: extract up to end of line
			result = append(result, string(runes[minCol:]))
		} else {
			// maxCol is inclusive
			result = append(result, string(runes[minCol:maxCol+1]))
		}
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
// endCol is INCLUSIVE (vim visual mode semantics: cursor ON the character)
func extractSingleLine(line string, startCol, endCol int) (string, error) {
	runes := []rune(line)

	// Empty line: always return empty string regardless of column values.
	// Mouse selection and visual mode can reference column 0 on empty lines.
	if len(runes) == 0 {
		return "", nil
	}

	if startCol < 0 || startCol > len(runes) {
		return "", fmt.Errorf("start column out of bounds: %d (line length: %d)", startCol, len(runes))
	}

	if endCol == -1 {
		endCol = len(runes)
	} else {
		// endCol is inclusive, so add 1 for Go slice (which is exclusive)
		endCol = endCol + 1
	}

	if endCol < 0 || endCol > len(runes) {
		return "", fmt.Errorf("end column out of bounds: %d (line length: %d)", endCol, len(runes))
	}

	if startCol > endCol {
		return "", fmt.Errorf("start column > end column: %d > %d", startCol, endCol)
	}

	return string(runes[startCol:endCol]), nil
}

// LineProvider provides access to individual lines by index without
// requiring a full []string copy. Used for zero-copy yank extraction.
type LineProvider interface {
	Line(index int) string
	LineCount() int
}

// ExtractRegionFromProvider extracts text using a LineProvider instead of []string.
// Only accesses lines within the selection region (avoids O(N) copy for large scrollback).
func ExtractRegionFromProvider(lp LineProvider, region Region) (string, error) {
	if lp.LineCount() == 0 {
		return "", fmt.Errorf("empty content")
	}

	// Normalize positions (ensure start <= end)
	start, end := region.Start, region.End
	if start.Line > end.Line || (start.Line == end.Line && start.Col > end.Col) {
		start, end = end, start
	}

	// Validate line bounds
	if start.Line < 0 || end.Line >= lp.LineCount() {
		return "", fmt.Errorf("line out of bounds: start=%d, end=%d, len=%d", start.Line, end.Line, lp.LineCount())
	}

	// Build a minimal slice containing only the needed lines
	needed := make([]string, end.Line-start.Line+1)
	for i := start.Line; i <= end.Line; i++ {
		needed[i-start.Line] = lp.Line(i)
	}

	// Adjust positions to be relative to the slice
	adjStart := Pos{Line: 0, Col: start.Col}
	adjEnd := Pos{Line: end.Line - start.Line, Col: end.Col}

	adjRegion := Region{Kind: region.Kind, Start: adjStart, End: adjEnd, Active: region.Active}
	return ExtractRegion(needed, adjRegion)
}
