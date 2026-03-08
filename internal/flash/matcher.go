package flash

import (
	"sort"
	"strings"
)

// Match represents a single pattern match within the document.
type Match struct {
	Line     int  // Document line index (0-based)
	ColStart int  // Start column (rune index, 0-based)
	ColEnd   int  // End column (exclusive)
	Label    byte // Assigned label char (0 if none)
}

// FindMatches performs literal substring matching within the visible viewport.
//
// Smartcase rules: if the pattern is all-lowercase, matching is case-insensitive;
// if any character is uppercase, matching is case-sensitive.
//
// Only lines in [viewportTop, viewportTop+viewportHeight) are searched.
// Overlapping matches are found by advancing one rune at a time.
// Results are sorted by line then column.
func FindMatches(lines []string, pattern string, viewportTop, viewportHeight int) []Match {
	if pattern == "" {
		return nil
	}

	caseSensitive := pattern != strings.ToLower(pattern)

	searchPattern := pattern
	if !caseSensitive {
		searchPattern = strings.ToLower(pattern)
	}

	patternRunes := []rune(searchPattern)
	patternLen := len(patternRunes)

	var matches []Match

	end := viewportTop + viewportHeight
	if end > len(lines) {
		end = len(lines)
	}

	for lineIdx := viewportTop; lineIdx < end; lineIdx++ {
		if lineIdx < 0 {
			continue
		}

		line := lines[lineIdx]
		searchLine := line
		if !caseSensitive {
			searchLine = strings.ToLower(line)
		}

		lineRunes := []rune(searchLine)
		lineLen := len(lineRunes)

		for col := 0; col <= lineLen-patternLen; col++ {
			if runesEqual(lineRunes[col:col+patternLen], patternRunes) {
				matches = append(matches, Match{
					Line:     lineIdx,
					ColStart: col,
					ColEnd:   col + patternLen,
				})
			}
		}
	}

	sort.Slice(matches, func(i, j int) bool {
		if matches[i].Line != matches[j].Line {
			return matches[i].Line < matches[j].Line
		}
		return matches[i].ColStart < matches[j].ColStart
	})

	return matches
}

func runesEqual(a, b []rune) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
