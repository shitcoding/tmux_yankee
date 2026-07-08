package flash

import (
	"fmt"
	"sort"
	"strings"
)

// DefaultLabelPool defines the label characters in home-row priority order (QWERTY).
const DefaultLabelPool = "asdfghjklqwertyuiopzxcvbnm"

// Labeler assigns single-character labels to matches, prioritizing matches
// closest to the cursor and reusing previously assigned labels for stable positions.
type Labeler struct {
	pool []byte
	used map[string]byte // "line:col" → previously assigned label
}

// NewLabeler creates a Labeler with the default label pool.
func NewLabeler() *Labeler {
	return &Labeler{
		pool: []byte(DefaultLabelPool),
		used: make(map[string]byte),
	}
}

// posKey builds the map key for position memory.
func posKey(line, col int) string {
	return fmt.Sprintf("%d:%d", line, col)
}

// matchDistance computes the priority distance from cursor. Lower = closer = higher priority.
func matchDistance(m Match, cursorLine, cursorCol int) int {
	dl := m.Line - cursorLine
	if dl < 0 {
		dl = -dl
	}
	dc := m.ColStart - cursorCol
	if dc < 0 {
		dc = -dc
	}
	return dl*1000 + dc
}

// AssignWithForbidden assigns labels to matches sorted by distance from the
// cursor, avoiding collisions with the character following each match and never
// using a forbidden character (typically chars that extend the current pattern).
func (l *Labeler) AssignWithForbidden(matches []Match, cursorLine, cursorCol int, lines []string, forbidden map[byte]bool) {
	if len(matches) == 0 {
		return
	}

	// Build indices sorted by distance from cursor
	indices := make([]int, len(matches))
	for i := range indices {
		indices[i] = i
	}
	sort.SliceStable(indices, func(a, b int) bool {
		da := matchDistance(matches[indices[a]], cursorLine, cursorCol)
		db := matchDistance(matches[indices[b]], cursorLine, cursorCol)
		if da != db {
			return da < db
		}
		// Ties broken by line then col
		if matches[indices[a]].Line != matches[indices[b]].Line {
			return matches[indices[a]].Line < matches[indices[b]].Line
		}
		return matches[indices[a]].ColStart < matches[indices[b]].ColStart
	})

	// Track which labels from the pool have been consumed
	poolUsed := make(map[byte]bool)

	// Pass 1: Reuse previously assigned labels for known positions
	for _, idx := range indices {
		m := &matches[idx]
		key := posKey(m.Line, m.ColStart)
		if label, ok := l.used[key]; ok {
			if !poolUsed[label] && !forbidden[label] && !l.labelCollides(label, m, lines) {
				m.Label = label
				poolUsed[label] = true
			}
		}
	}

	// Pass 2: Assign fresh labels from pool to remaining matches
	poolIdx := 0
	for _, idx := range indices {
		m := &matches[idx]
		if m.Label != 0 {
			continue // already labeled in pass 1
		}

		for poolIdx < len(l.pool) {
			candidate := l.pool[poolIdx]
			poolIdx++
			if poolUsed[candidate] || forbidden[candidate] {
				continue
			}
			if l.labelCollides(candidate, m, lines) {
				continue
			}
			m.Label = candidate
			poolUsed[candidate] = true
			key := posKey(m.Line, m.ColStart)
			l.used[key] = candidate
			break
		}
		// If poolIdx exhausted, Label remains 0 (no label)
	}

	// Update position memory for pass-1 matches too
	for _, idx := range indices {
		m := &matches[idx]
		if m.Label != 0 {
			l.used[posKey(m.Line, m.ColStart)] = m.Label
		}
	}
}

// labelCollides checks if a label character case-insensitively matches the
// character immediately following the match end in the source text.
func (l *Labeler) labelCollides(label byte, m *Match, lines []string) bool {
	if lines == nil {
		return false
	}
	if m.Line < 0 || m.Line >= len(lines) {
		return false
	}
	line := lines[m.Line]
	lineRunes := []rune(line)
	if m.ColEnd >= len(lineRunes) {
		return false // no character after match
	}
	nextRune := lineRunes[m.ColEnd]
	nextLower := strings.ToLower(string(nextRune))
	labelLower := strings.ToLower(string(rune(label)))
	return nextLower == labelLower
}
