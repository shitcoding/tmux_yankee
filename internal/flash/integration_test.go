package flash

import (
	"strings"
	"testing"
)

// makeLines is a test helper that splits a multi-line string into a []string
// document, trimming the leading newline if present.
func makeLines(t *testing.T, doc string) []string {
	t.Helper()
	doc = strings.TrimPrefix(doc, "\n")
	return strings.Split(doc, "\n")
}

func TestIntegration_FullSearchAndJump(t *testing.T) {
	lines := makeLines(t, `
the quick brown fox jumps over the lazy dog
a fox ran past another fox in the forest
the end of the story`)

	s := New(Options{MinChars: 1})
	s.Enter(0, 0, 0)

	// Type "f" -- should match "fox" at multiple locations plus "forest"
	act := s.UpdatePattern("f", lines, 0, 3)
	if act.Type != ActionContinue {
		t.Fatalf("after 'f': expected ActionContinue, got %d", act.Type)
	}
	if len(s.Matches) == 0 {
		t.Fatal("after 'f': expected matches, got none")
	}

	// Type "fo" -- should narrow to "fox" and "forest" occurrences
	act = s.UpdatePattern("fo", lines, 0, 3)
	if act.Type != ActionContinue {
		t.Fatalf("after 'fo': expected ActionContinue, got %d", act.Type)
	}
	matchesAfterFO := len(s.Matches)
	if matchesAfterFO == 0 {
		t.Fatal("after 'fo': expected matches, got none")
	}

	// Type "fox" -- should narrow to exactly "fox" occurrences
	act = s.UpdatePattern("fox", lines, 0, 3)
	if act.Type != ActionContinue {
		t.Fatalf("after 'fox': expected ActionContinue, got %d", act.Type)
	}
	if len(s.Matches) == 0 {
		t.Fatal("after 'fox': expected matches, got none")
	}
	if len(s.Matches) >= matchesAfterFO {
		// "fox" should match fewer or equal positions than "fo"
		// (equal if "fo" only appeared in "fox" contexts)
	}

	// Verify all matches have labels
	for i, m := range s.Matches {
		if m.Label == 0 {
			t.Errorf("match[%d] at (%d,%d) has no label", i, m.Line, m.ColStart)
		}
	}

	// Pick a specific match and press its label
	target := s.Matches[0]
	if target.Label == 0 {
		t.Fatal("first match has no label to press")
	}

	act = s.HandleKey(target.Label, lines)
	if act.Type != ActionJump {
		t.Errorf("label press: expected ActionJump, got %d", act.Type)
	}
	// Default jumpPos is JumpPosMatchEnd: col = ColEnd - 1
	wantCol := target.ColEnd - 1
	if wantCol < target.ColStart {
		wantCol = target.ColStart
	}
	if act.Line != target.Line || act.Col != wantCol {
		t.Errorf("jump target = (%d,%d), want (%d,%d)",
			act.Line, act.Col, target.Line, wantCol)
	}

	// State should be deactivated after jump
	if s.Active {
		t.Error("state should be inactive after jump")
	}
	if s.Overlay() != nil {
		t.Error("overlay should be nil after jump")
	}
}

func TestIntegration_SearchCancelRestore(t *testing.T) {
	lines := makeLines(t, `
first line of text
second line with words
third line here`)

	s := New(Options{MinChars: 1})

	// Save a specific cursor position
	origLine, origCol, origViewport := 1, 7, 0
	s.Enter(origLine, origCol, origViewport)

	if s.SavedCursor != [2]int{origLine, origCol} {
		t.Fatalf("SavedCursor = %v, want [%d %d]", s.SavedCursor, origLine, origCol)
	}
	if s.SavedViewport != origViewport {
		t.Fatalf("SavedViewport = %d, want %d", s.SavedViewport, origViewport)
	}

	// Type some pattern characters
	s.UpdatePattern("l", lines, 0, 3)
	s.UpdatePattern("li", lines, 0, 3)
	s.UpdatePattern("lin", lines, 0, 3)

	// Flash is active and has state
	if !s.Active {
		t.Fatal("state should be active during search")
	}
	if s.Pattern != "lin" {
		t.Errorf("pattern = %q, want %q", s.Pattern, "lin")
	}

	// Cancel with Escape
	act := s.HandleKey(27, nil)
	if act.Type != ActionCancel {
		t.Errorf("escape: expected ActionCancel, got %d", act.Type)
	}

	// Verify saved position is still accessible for the caller to restore
	if s.SavedCursor != [2]int{origLine, origCol} {
		t.Errorf("after cancel: SavedCursor = %v, want [%d %d]",
			s.SavedCursor, origLine, origCol)
	}
	// Note: State.exit() clears Active but the caller reads SavedCursor/SavedViewport
	// before the cancel. The test verifies these values survive the Enter→cancel cycle
	// by checking them before exit is called (they are set at Enter time and not mutated).
}

func TestIntegration_BackspaceNarrowsPattern(t *testing.T) {
	lines := makeLines(t, `
food for thought
foolish football fan
fantastic finish`)

	s := New(Options{MinChars: 1})
	s.Enter(0, 0, 0)

	// Type "f" -- broad match
	s.UpdatePattern("f", lines, 0, 3)
	matchesF := len(s.Matches)

	// Type "fo" -- narrower
	s.UpdatePattern("fo", lines, 0, 3)
	matchesFO := len(s.Matches)

	// Type "foo" -- even narrower
	s.UpdatePattern("foo", lines, 0, 3)
	matchesFOO := len(s.Matches)

	// Verify progressive narrowing
	if matchesFO > matchesF {
		t.Errorf("'fo' (%d matches) should not exceed 'f' (%d matches)", matchesFO, matchesF)
	}
	if matchesFOO > matchesFO {
		t.Errorf("'foo' (%d matches) should not exceed 'fo' (%d matches)", matchesFOO, matchesFO)
	}
	if matchesFOO == 0 {
		t.Fatal("'foo' should have at least 1 match (food, foolish, football)")
	}

	// Backspace from "foo" -- caller handles shortening
	act := s.HandleKey(127, nil)
	if act.Type != ActionContinue {
		t.Fatalf("backspace from 'foo': expected ActionContinue, got %d", act.Type)
	}

	// Caller shortens pattern to "fo" and calls UpdatePattern
	s.UpdatePattern("fo", lines, 0, 3)
	matchesBackToFO := len(s.Matches)

	// Should have same matches as when we first typed "fo"
	if matchesBackToFO != matchesFO {
		t.Errorf("after backspace to 'fo': %d matches, want %d", matchesBackToFO, matchesFO)
	}

	// Backspace again to "f"
	act = s.HandleKey(127, nil)
	if act.Type != ActionContinue {
		t.Fatalf("backspace from 'fo': expected ActionContinue, got %d", act.Type)
	}

	s.UpdatePattern("f", lines, 0, 3)
	matchesBackToF := len(s.Matches)

	if matchesBackToF != matchesF {
		t.Errorf("after backspace to 'f': %d matches, want %d", matchesBackToF, matchesF)
	}

	// Verify more matches reappear after broadening
	if matchesBackToF <= matchesFOO {
		t.Errorf("broadening from 'foo' to 'f' should yield more matches: got %d vs %d",
			matchesBackToF, matchesFOO)
	}
}

func TestIntegration_AutoJumpSingleMatch(t *testing.T) {
	lines := makeLines(t, `
common words appear here
but xylophone is unique in this text
more common words below`)

	s := New(Options{MinChars: 1})
	s.Enter(0, 0, 0)

	// Type a pattern that uniquely matches exactly once
	// "xylophone" only appears on line 1
	act := s.UpdatePattern("xyl", lines, 0, 3)

	if act.Type != ActionAutoJump {
		t.Errorf("unique match: expected ActionAutoJump, got %d", act.Type)
	}
	if act.Line != 1 {
		t.Errorf("auto-jump line = %d, want 1", act.Line)
	}
	// Default jumpPos is JumpPosMatchEnd: "xyl" ColEnd=7, so col=6
	if act.Col != 6 {
		t.Errorf("auto-jump col = %d, want 6", act.Col)
	}

	// State should be deactivated after auto-jump
	if s.Active {
		t.Error("state should be inactive after auto-jump")
	}
}

func TestIntegration_LabelStability(t *testing.T) {
	// The labeler has position memory: once a label is assigned to a position,
	// subsequent calls should reuse that label for the same position.
	lines := makeLines(t, `
apple and apricot and avocado
another animal arrived
all around are ants`)

	s := New(Options{MinChars: 1})
	s.Enter(0, 0, 0)

	// Type "a" -- many matches across all lines
	s.UpdatePattern("a", lines, 0, 3)

	if len(s.Matches) < 5 {
		t.Fatalf("expected at least 5 matches for 'a', got %d", len(s.Matches))
	}

	// Record labels for all current match positions
	type posLabel struct {
		line, col int
		label     byte
	}
	firstLabels := make(map[[2]int]byte)
	for _, m := range s.Matches {
		if m.Label != 0 {
			firstLabels[[2]int{m.Line, m.ColStart}] = m.Label
		}
	}

	// Type "an" -- fewer matches, subset of "a" positions
	s.UpdatePattern("an", lines, 0, 3)

	if len(s.Matches) == 0 {
		t.Fatal("expected matches for 'an'")
	}

	// Verify surviving matches kept their labels (position memory)
	stableCount := 0
	for _, m := range s.Matches {
		if m.Label == 0 {
			continue
		}
		prevLabel, existed := firstLabels[[2]int{m.Line, m.ColStart}]
		if existed && prevLabel == m.Label {
			stableCount++
		}
	}

	if stableCount == 0 && len(s.Matches) > 0 {
		t.Error("no labels were stable across pattern refinement -- position memory may be broken")
	}

	t.Logf("label stability: %d/%d surviving matches kept their labels",
		stableCount, len(s.Matches))
}

func TestIntegration_OverlayConsistency(t *testing.T) {
	lines := makeLines(t, `
the cat sat on the mat
the bat ate the hat
the rat ran past`)

	s := New(Options{MinChars: 1})
	s.Enter(0, 0, 0)

	// Before any pattern: no overlay (Enter doesn't build one)
	// Overlay() is nil when Active is true but no UpdatePattern called yet
	// (overlay field is nil after Enter)

	// Type "at" -- multiple matches
	act := s.UpdatePattern("at", lines, 0, 3)
	if act.Type != ActionContinue {
		t.Fatalf("expected ActionContinue, got %d", act.Type)
	}

	overlay := s.Overlay()
	if overlay == nil {
		t.Fatal("overlay should not be nil after UpdatePattern with matches")
	}

	// Overlay should have backdrop enabled
	if !overlay.Backdrop {
		t.Error("overlay backdrop should be true")
	}

	// Overlay prompt should match current pattern
	if overlay.Prompt != "at" {
		t.Errorf("overlay prompt = %q, want %q", overlay.Prompt, "at")
	}

	// Overlay matches should correspond to state matches
	if len(overlay.Matches) != len(s.Matches) {
		t.Errorf("overlay has %d matches, state has %d",
			len(overlay.Matches), len(s.Matches))
	}

	// Verify overlay labels correspond to labeled matches
	labeledMatches := 0
	for _, m := range s.Matches {
		if m.Label != 0 {
			labeledMatches++
			label := overlay.HasLabel(m.Line, m.ColEnd)
			if label != m.Label {
				t.Errorf("overlay label at (%d,%d) = '%c', state label = '%c'",
					m.Line, m.ColEnd, label, m.Label)
			}
		}
	}
	if len(overlay.Labels) != labeledMatches {
		t.Errorf("overlay has %d labels, expected %d", len(overlay.Labels), labeledMatches)
	}

	// Verify InMatch works for known match positions
	for _, m := range s.Matches {
		for col := m.ColStart; col < m.ColEnd; col++ {
			if !overlay.InMatch(m.Line, col) {
				t.Errorf("InMatch(%d,%d) = false, expected true", m.Line, col)
			}
		}
	}

	// Type more to narrow: "at " (with trailing space)
	s.UpdatePattern("at ", lines, 0, 3)
	overlay2 := s.Overlay()
	if overlay2 == nil {
		t.Fatal("overlay should not be nil after narrowed pattern")
	}
	if overlay2.Prompt != "at " {
		t.Errorf("overlay2 prompt = %q, want %q", overlay2.Prompt, "at ")
	}

	// Fewer or equal matches expected
	if len(overlay2.Matches) > len(overlay.Matches) {
		t.Errorf("narrowing should not increase matches: %d > %d",
			len(overlay2.Matches), len(overlay.Matches))
	}

	// Backspace back to "at" (simulate caller shortening)
	s.UpdatePattern("at", lines, 0, 3)
	overlay3 := s.Overlay()
	if overlay3 == nil {
		t.Fatal("overlay should not be nil after backspace")
	}
	if overlay3.Prompt != "at" {
		t.Errorf("overlay3 prompt = %q, want %q", overlay3.Prompt, "at")
	}

	// Should be back to same match count as original "at" search
	if len(overlay3.Matches) != len(overlay.Matches) {
		t.Errorf("after backspace: %d matches, expected %d",
			len(overlay3.Matches), len(overlay.Matches))
	}
}

func TestIntegration_SmartcaseFlow(t *testing.T) {
	lines := makeLines(t, `
Hello world hello
HELLO again
hello Hello HELLO`)

	t.Run("lowercase_matches_all", func(t *testing.T) {
		s := New(Options{MinChars: 1})
		s.Enter(0, 0, 0)

		// Lowercase pattern "hello" -- smartcase: case-insensitive
		act := s.UpdatePattern("hello", lines, 0, 3)
		if act.Type == ActionAutoJump {
			t.Fatal("expected multiple matches with case-insensitive 'hello'")
		}

		// Count total matches: Hello, hello, HELLO, hello, Hello, HELLO = 6
		if len(s.Matches) != 6 {
			t.Errorf("lowercase 'hello': expected 6 matches, got %d", len(s.Matches))
		}

		// All should have labels (we have 26 labels, well under that)
		for i, m := range s.Matches {
			if m.Label == 0 {
				t.Errorf("match[%d] at (%d,%d) has no label", i, m.Line, m.ColStart)
			}
		}
	})

	t.Run("uppercase_matches_exact", func(t *testing.T) {
		s := New(Options{MinChars: 1})
		s.Enter(0, 0, 0)

		// Mixed-case pattern "Hello" -- smartcase: case-sensitive
		act := s.UpdatePattern("Hello", lines, 0, 3)

		// "Hello" appears at line 0 col 0 and line 2 col 6
		if len(s.Matches) != 2 {
			t.Errorf("case-sensitive 'Hello': expected 2 matches, got %d", len(s.Matches))
			for i, m := range s.Matches {
				t.Logf("  match[%d]: line=%d col=%d", i, m.Line, m.ColStart)
			}
		}

		// With exactly 2 matches, should not auto-jump
		if act.Type == ActionAutoJump {
			t.Error("should not auto-jump with 2 matches")
		}
	})

	t.Run("all_caps_matches_exact", func(t *testing.T) {
		s := New(Options{MinChars: 1})
		s.Enter(0, 0, 0)

		// All-caps pattern "HELLO" -- has uppercase chars, case-sensitive
		act := s.UpdatePattern("HELLO", lines, 0, 3)

		// "HELLO" appears at line 1 col 0 and line 2 col 12
		if len(s.Matches) != 2 {
			t.Errorf("case-sensitive 'HELLO': expected 2 matches, got %d", len(s.Matches))
		}
		if act.Type == ActionAutoJump {
			t.Error("should not auto-jump with 2 matches")
		}
	})
}

func TestIntegration_ViewportRestriction(t *testing.T) {
	// Build a 50-line document with "target" appearing on every 5th line
	lines := make([]string, 50)
	for i := range lines {
		if i%5 == 0 {
			lines[i] = "this line has a target word"
		} else {
			lines[i] = "this line is just filler content"
		}
	}

	s := New(Options{MinChars: 1})

	// Viewport shows lines 10-19 (10 lines visible)
	viewportTop := 10
	viewportHeight := 10
	s.Enter(15, 0, viewportTop)

	act := s.UpdatePattern("target", lines, viewportTop, viewportHeight)

	// "target" appears on lines 0, 5, 10, 15, 20, 25, 30, 35, 40, 45
	// Within viewport [10, 20): lines 10 and 15
	if len(s.Matches) != 2 {
		t.Errorf("viewport-restricted: expected 2 matches, got %d", len(s.Matches))
		for i, m := range s.Matches {
			t.Logf("  match[%d]: line=%d col=%d", i, m.Line, m.ColStart)
		}
	}

	// Verify all matches are within viewport bounds
	for _, m := range s.Matches {
		if m.Line < viewportTop || m.Line >= viewportTop+viewportHeight {
			t.Errorf("match on line %d is outside viewport [%d, %d)",
				m.Line, viewportTop, viewportTop+viewportHeight)
		}
	}

	// With only 2 matches, should not auto-jump (labels assigned to both)
	if act.Type == ActionAutoJump {
		t.Error("should not auto-jump with 2 matches")
	}

	// Verify we can jump to one of the viewport matches
	if len(s.Matches) > 0 {
		target := s.Matches[0]
		if target.Label == 0 {
			t.Fatal("viewport match should have a label")
		}
		act = s.HandleKey(target.Label, lines)
		if act.Type != ActionJump {
			t.Errorf("jump: expected ActionJump, got %d", act.Type)
		}
		if act.Line != target.Line {
			t.Errorf("jump line = %d, want %d", act.Line, target.Line)
		}
	}
}

func TestIntegration_ReenterFlashMode(t *testing.T) {
	// Verify the state machine can be reused for multiple flash sessions.
	lines := makeLines(t, `
alpha beta gamma alpha
delta epsilon zeta`)

	s := New(Options{MinChars: 1})

	// First session: search and cancel
	s.Enter(0, 0, 0)
	act := s.UpdatePattern("alpha", lines, 0, 2)
	// "alpha" appears twice on line 0, so should not auto-jump
	if act.Type != ActionContinue {
		t.Fatalf("first session: expected ActionContinue, got %d", act.Type)
	}
	if !s.Active {
		t.Fatal("first session: should be active")
	}
	s.HandleKey(27, nil) // Escape
	if s.Active {
		t.Fatal("first session: should be inactive after cancel")
	}

	// Second session: search and jump
	s.Enter(1, 5, 0)
	act = s.UpdatePattern("zeta", lines, 0, 2)
	if act.Type != ActionAutoJump {
		t.Fatalf("second session: expected AutoJump for unique 'zeta', got %d", act.Type)
	}
	if act.Line != 1 {
		t.Errorf("second session: jump line = %d, want 1", act.Line)
	}

	// Third session: verify clean state
	s.Enter(0, 3, 0)
	if s.Pattern != "" {
		t.Errorf("third session: pattern = %q, want empty", s.Pattern)
	}
	if len(s.Matches) != 0 {
		t.Errorf("third session: matches = %d, want 0", len(s.Matches))
	}
	if s.SavedCursor != [2]int{0, 3} {
		t.Errorf("third session: SavedCursor = %v, want [0 3]", s.SavedCursor)
	}
}

func TestIntegration_MinCharsDelaysLabels(t *testing.T) {
	// With MinChars=2, typing a single character should find matches but
	// not assign labels. Labels appear only after 2+ characters.
	lines := makeLines(t, `
map cap tap gap
nap sap lap rap`)

	s := New(Options{MinChars: 2})
	s.Enter(0, 0, 0)

	// Type "a" -- matches exist but no labels yet
	act := s.UpdatePattern("a", lines, 0, 2)
	if act.Type != ActionContinue {
		t.Fatalf("minChars=2, 1 char: expected ActionContinue, got %d", act.Type)
	}
	if len(s.Matches) == 0 {
		t.Fatal("minChars=2, 1 char: should still find matches")
	}
	for _, m := range s.Matches {
		if m.Label != 0 {
			t.Errorf("minChars=2, 1 char: match at (%d,%d) should not have label '%c'",
				m.Line, m.ColStart, m.Label)
		}
	}

	// Overlay should exist but have no labels
	overlay := s.Overlay()
	if overlay == nil {
		t.Fatal("overlay should exist even without labels")
	}
	if len(overlay.Labels) != 0 {
		t.Errorf("overlay should have 0 labels with 1-char pattern, got %d", len(overlay.Labels))
	}

	// Type "ap" -- now labels should appear
	act = s.UpdatePattern("ap", lines, 0, 2)
	if act.Type != ActionContinue {
		t.Fatalf("minChars=2, 2 chars: expected ActionContinue, got %d", act.Type)
	}

	hasLabel := false
	for _, m := range s.Matches {
		if m.Label != 0 {
			hasLabel = true
			break
		}
	}
	if !hasLabel {
		t.Error("minChars=2, 2 chars: expected labels to be assigned")
	}
}

func TestIntegration_NoMatchesPattern(t *testing.T) {
	// Typing a pattern with no matches should keep flash active with
	// empty matches and a valid overlay.
	lines := makeLines(t, `
hello world
goodbye earth`)

	s := New(Options{MinChars: 1})
	s.Enter(0, 0, 0)

	// Type a pattern that matches nothing
	act := s.UpdatePattern("zzzzz", lines, 0, 2)
	if act.Type != ActionContinue {
		t.Fatalf("no matches: expected ActionContinue, got %d", act.Type)
	}
	if len(s.Matches) != 0 {
		t.Errorf("no matches: expected 0 matches, got %d", len(s.Matches))
	}
	if !s.Active {
		t.Error("flash should remain active with no matches")
	}

	// Overlay should still be valid (empty but present)
	overlay := s.Overlay()
	if overlay == nil {
		t.Fatal("overlay should not be nil even with no matches")
	}
	if len(overlay.Matches) != 0 {
		t.Errorf("overlay should have 0 matches, got %d", len(overlay.Matches))
	}
	if len(overlay.Labels) != 0 {
		t.Errorf("overlay should have 0 labels, got %d", len(overlay.Labels))
	}

	// Backspace to shorten pattern should allow re-matching
	act = s.HandleKey(127, nil)
	if act.Type != ActionContinue {
		t.Fatalf("backspace from nomatch: expected ActionContinue, got %d", act.Type)
	}

	// Caller shortens to "zzzz" -- still no matches
	s.UpdatePattern("zzzz", lines, 0, 2)
	if len(s.Matches) != 0 {
		t.Errorf("still no matches expected, got %d", len(s.Matches))
	}
}

func TestIntegration_ProgressiveCharByCharTyping(t *testing.T) {
	// Simulates a user typing a word one character at a time, verifying
	// the match count monotonically decreases (or stays equal) with each char.
	lines := makeLines(t, `
function fetchData() { return fetch("/api/data"); }
function filterResults(data) { return data.filter(x => x.active); }
const formatted = format(raw);`)

	s := New(Options{MinChars: 1})
	s.Enter(0, 0, 0)

	word := "filter"
	prevCount := -1

	for i := 1; i <= len(word); i++ {
		pattern := word[:i]
		act := s.UpdatePattern(pattern, lines, 0, 3)

		if act.Type == ActionAutoJump {
			// Single match found, which is fine -- just verify it's valid
			if act.Line < 0 || act.Line >= len(lines) {
				t.Errorf("auto-jump line %d out of range", act.Line)
			}
			return
		}

		if act.Type != ActionContinue {
			t.Fatalf("pattern %q: unexpected action %d", pattern, act.Type)
		}

		currentCount := len(s.Matches)
		if prevCount >= 0 && currentCount > prevCount {
			t.Errorf("pattern %q: match count %d > previous %d (should not increase)",
				pattern, currentCount, prevCount)
		}
		prevCount = currentCount
	}
}

func TestIntegration_JumpToSecondMatch(t *testing.T) {
	// Verify we can jump to a match that is NOT the first one in document order.
	lines := makeLines(t, `
error: file not found
warning: something happened
error: connection refused`)

	s := New(Options{MinChars: 1})
	s.Enter(0, 0, 0)

	act := s.UpdatePattern("error", lines, 0, 3)
	if act.Type != ActionContinue {
		t.Fatalf("expected ActionContinue, got %d", act.Type)
	}
	if len(s.Matches) != 2 {
		t.Fatalf("expected 2 'error' matches, got %d", len(s.Matches))
	}

	// Find the match on line 2 (the second occurrence)
	var secondMatch Match
	found := false
	for _, m := range s.Matches {
		if m.Line == 2 {
			secondMatch = m
			found = true
			break
		}
	}
	if !found {
		t.Fatal("could not find 'error' match on line 2")
	}
	if secondMatch.Label == 0 {
		t.Fatal("second match should have a label")
	}

	act = s.HandleKey(secondMatch.Label, lines)
	if act.Type != ActionJump {
		t.Errorf("expected ActionJump, got %d", act.Type)
	}
	if act.Line != 2 {
		t.Errorf("jump line = %d, want 2", act.Line)
	}
	// Default jumpPos is JumpPosMatchEnd: "error" at col 0, ColEnd=5, so col=4
	if act.Col != 4 {
		t.Errorf("jump col = %d, want 4", act.Col)
	}
}
