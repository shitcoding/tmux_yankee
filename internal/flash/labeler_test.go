package flash

import (
	"testing"
)

func TestLabeler_BasicAssignment(t *testing.T) {
	lab := NewLabeler()
	matches := []Match{
		{Line: 0, ColStart: 0, ColEnd: 3},
		{Line: 0, ColStart: 5, ColEnd: 8},
		{Line: 1, ColStart: 0, ColEnd: 3},
	}

	lab.Assign(matches, 0, 0)

	for i, m := range matches {
		if m.Label == 0 {
			t.Errorf("match[%d] has no label", i)
		}
	}

	// All labels should be unique
	seen := make(map[byte]bool)
	for i, m := range matches {
		if seen[m.Label] {
			t.Errorf("match[%d] has duplicate label '%c'", i, m.Label)
		}
		seen[m.Label] = true
	}
}

func TestLabeler_DistanceSorting(t *testing.T) {
	lab := NewLabeler()
	// Match far from cursor, match near cursor
	matches := []Match{
		{Line: 10, ColStart: 0, ColEnd: 3}, // far
		{Line: 0, ColStart: 0, ColEnd: 3},  // near cursor
	}

	lab.Assign(matches, 0, 0)

	// The nearest match (line 0) should get first label ('a')
	if matches[1].Label != 'a' {
		t.Errorf("nearest match got label '%c', want 'a'", matches[1].Label)
	}
}

func TestLabeler_PositionMemory(t *testing.T) {
	lab := NewLabeler()

	// First call
	matches1 := []Match{
		{Line: 0, ColStart: 0, ColEnd: 3},
		{Line: 1, ColStart: 0, ColEnd: 3},
	}
	lab.Assign(matches1, 0, 0)

	label0 := matches1[0].Label
	label1 := matches1[1].Label

	// Second call with same positions
	matches2 := []Match{
		{Line: 0, ColStart: 0, ColEnd: 3},
		{Line: 1, ColStart: 0, ColEnd: 3},
	}
	lab.Assign(matches2, 0, 0)

	if matches2[0].Label != label0 {
		t.Errorf("position memory: line 0 label changed from '%c' to '%c'",
			label0, matches2[0].Label)
	}
	if matches2[1].Label != label1 {
		t.Errorf("position memory: line 1 label changed from '%c' to '%c'",
			label1, matches2[1].Label)
	}
}

func TestLabeler_Exhaustion(t *testing.T) {
	lab := NewLabeler()

	// Create 30 matches, only 26 labels available
	matches := make([]Match, 30)
	for i := range matches {
		matches[i] = Match{Line: i, ColStart: 0, ColEnd: 3}
	}

	lab.Assign(matches, 0, 0)

	labeled := 0
	unlabeled := 0
	for _, m := range matches {
		if m.Label != 0 {
			labeled++
		} else {
			unlabeled++
		}
	}

	if labeled != 26 {
		t.Errorf("exhaustion: expected 26 labeled, got %d", labeled)
	}
	if unlabeled != 4 {
		t.Errorf("exhaustion: expected 4 unlabeled, got %d", unlabeled)
	}
}

func TestLabeler_CollisionAvoidance(t *testing.T) {
	lab := NewLabeler()

	// "as" is the match. The character after the match is 'd'.
	// The default pool starts with "asdfg..."
	// Label 'a' should be tried first (nearest), but...
	// Actually, let's create a scenario where the next char would collide.
	lines := []string{
		"xxs rest", // match "xx" at col 0, next char is 's' - collides with label 's'
	}
	matches := []Match{
		{Line: 0, ColStart: 0, ColEnd: 2},
	}

	lab.AssignWithContext(matches, 0, 0, lines)

	// The first pool label 'a' should be assigned since next char is 's', not 'a'
	if matches[0].Label != 'a' {
		t.Errorf("collision: expected label 'a', got '%c'", matches[0].Label)
	}
}

func TestLabeler_CollisionSkipsLabel(t *testing.T) {
	lab := NewLabelerWithPool("sa")

	// Match "xx", next char is 's' - should skip 's' label, assign 'a'
	lines := []string{
		"xxs rest",
	}
	matches := []Match{
		{Line: 0, ColStart: 0, ColEnd: 2},
	}

	lab.AssignWithContext(matches, 0, 0, lines)

	if matches[0].Label == 's' {
		t.Error("collision avoidance: label 's' was assigned despite collision with next char 's'")
	}
	if matches[0].Label != 'a' {
		t.Errorf("collision avoidance: expected 'a' (next available), got '%c'", matches[0].Label)
	}
}

func TestLabeler_CollisionEndOfLine(t *testing.T) {
	lab := NewLabeler()

	// Match at end of line - no next character, no collision possible
	lines := []string{
		"hello",
	}
	matches := []Match{
		{Line: 0, ColStart: 3, ColEnd: 5}, // "lo" at end
	}

	lab.AssignWithContext(matches, 0, 3, lines)

	if matches[0].Label == 0 {
		t.Error("end of line: should still get a label when no next char to collide with")
	}
}

func TestLabeler_NoCollisionWithoutContext(t *testing.T) {
	lab := NewLabelerWithPool("s")

	// Without context (Assign, not AssignWithContext), no collision check
	matches := []Match{
		{Line: 0, ColStart: 0, ColEnd: 2},
	}

	lab.Assign(matches, 0, 0)

	if matches[0].Label != 's' {
		t.Errorf("no context: expected label 's', got '%c'", matches[0].Label)
	}
}

func TestLabeler_UniqueLabels(t *testing.T) {
	lab := NewLabeler()

	matches := make([]Match, 10)
	for i := range matches {
		matches[i] = Match{Line: i, ColStart: 0, ColEnd: 3}
	}

	lab.Assign(matches, 5, 0)

	seen := make(map[byte]bool)
	for i, m := range matches {
		if m.Label == 0 {
			continue
		}
		if seen[m.Label] {
			t.Errorf("duplicate label '%c' on match[%d]", m.Label, i)
		}
		seen[m.Label] = true
	}
}
