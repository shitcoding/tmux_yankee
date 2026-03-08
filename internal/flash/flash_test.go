package flash

import (
	"testing"
)

func TestState_EnterAndCancel(t *testing.T) {
	s := New(Options{MinChars: 1})

	if s.Active {
		t.Fatal("new state should not be active")
	}

	s.Enter(5, 10, 0)

	if !s.Active {
		t.Fatal("state should be active after Enter")
	}
	if s.SavedCursor != [2]int{5, 10} {
		t.Errorf("SavedCursor = %v, want [5 10]", s.SavedCursor)
	}
	if s.SavedViewport != 0 {
		t.Errorf("SavedViewport = %d, want 0", s.SavedViewport)
	}

	action := s.HandleKey(27, nil) // Escape
	if action.Type != ActionCancel {
		t.Errorf("Escape action = %d, want ActionCancel", action.Type)
	}
	if s.Active {
		t.Error("state should be inactive after cancel")
	}
}

func TestState_TypeAndJump(t *testing.T) {
	s := New(Options{MinChars: 1})
	lines := []string{
		"foo bar baz",
		"foo qux foo",
	}

	s.Enter(0, 0, 0)

	// Type "foo" - should find matches and assign labels
	action := s.UpdatePattern("foo", lines, 0, 2)

	// Should have 3 matches
	if len(s.Matches) != 3 {
		t.Fatalf("expected 3 matches, got %d", len(s.Matches))
	}

	if action.Type == ActionAutoJump {
		t.Fatal("should not auto-jump with 3 matches")
	}

	// Find which label was assigned to first match
	var firstLabel byte
	for _, m := range s.Matches {
		if m.Line == 0 && m.ColStart == 0 && m.Label != 0 {
			firstLabel = m.Label
			break
		}
	}

	if firstLabel == 0 {
		t.Fatal("first match should have a label")
	}

	// Press the label key — default jumpPos is JumpPosMatchEnd
	action = s.HandleKey(firstLabel, lines)
	if action.Type != ActionJump {
		t.Errorf("label key action = %d, want ActionJump", action.Type)
	}
	// "foo" at col 0 with JumpPosMatchEnd → col 2 (ColEnd-1 = 3-1 = 2)
	if action.Line != 0 || action.Col != 2 {
		t.Errorf("jump target = (%d,%d), want (0,2)", action.Line, action.Col)
	}
	if s.Active {
		t.Error("state should be inactive after jump")
	}
}

func TestState_AutoJumpOnSingleMatch(t *testing.T) {
	s := New(Options{MinChars: 1})
	lines := []string{
		"hello world",
		"foo bar baz",
	}

	s.Enter(0, 0, 0)

	// "world" appears exactly once
	action := s.UpdatePattern("world", lines, 0, 2)

	if action.Type != ActionAutoJump {
		t.Errorf("single match action = %d, want ActionAutoJump", action.Type)
	}
	// Default jumpPos is JumpPosMatchEnd: "world" ColEnd=11, so col=10
	if action.Line != 0 || action.Col != 10 {
		t.Errorf("auto-jump target = (%d,%d), want (0,10)", action.Line, action.Col)
	}
	if s.Active {
		t.Error("state should be inactive after auto-jump")
	}
}

func TestState_BackspaceToEmpty(t *testing.T) {
	s := New(Options{MinChars: 1})
	lines := []string{"hello"}

	s.Enter(0, 0, 0)
	s.UpdatePattern("h", lines, 0, 1)

	// Backspace when pattern has 1 char → cancel
	action := s.HandleKey(127, nil)
	if action.Type != ActionCancel {
		t.Errorf("backspace-to-empty action = %d, want ActionCancel", action.Type)
	}
	if s.Active {
		t.Error("state should be inactive after backspace-to-empty")
	}
}

func TestState_BackspaceWithLongerPattern(t *testing.T) {
	s := New(Options{MinChars: 1})
	// Use lines with multiple matches so UpdatePattern doesn't auto-jump
	lines := []string{
		"he said hello here",
		"he went home",
	}

	s.Enter(0, 0, 0)
	action := s.UpdatePattern("he", lines, 0, 2)

	if action.Type != ActionContinue {
		t.Fatalf("setup: expected ActionContinue after UpdatePattern, got %d", action.Type)
	}

	// Backspace when pattern has 2 chars → continue (caller handles shortening)
	action = s.HandleKey(127, nil)
	if action.Type != ActionContinue {
		t.Errorf("backspace-with-pattern action = %d, want ActionContinue", action.Type)
	}
}

func TestState_MinCharsDelay(t *testing.T) {
	s := New(Options{MinChars: 2})
	lines := []string{
		"aa bb cc",
		"aa dd aa",
	}

	s.Enter(0, 0, 0)

	// Single char pattern - matches found but no labels assigned
	action := s.UpdatePattern("a", lines, 0, 2)

	if action.Type != ActionContinue {
		t.Errorf("single char action = %d, want ActionContinue", action.Type)
	}

	// Check that no labels were assigned (pattern < minChars)
	for _, m := range s.Matches {
		if m.Label != 0 {
			t.Errorf("match at (%d,%d) has label '%c' despite minChars=2",
				m.Line, m.ColStart, m.Label)
		}
	}

	// Two char pattern - now labels should be assigned
	action = s.UpdatePattern("aa", lines, 0, 2)

	hasLabels := false
	for _, m := range s.Matches {
		if m.Label != 0 {
			hasLabels = true
			break
		}
	}
	if !hasLabels {
		t.Error("expected labels to be assigned when pattern length >= minChars")
	}
}

func TestState_OverlayAccess(t *testing.T) {
	s := New(Options{MinChars: 1})

	// Overlay should be nil when not active
	if s.Overlay() != nil {
		t.Error("overlay should be nil when inactive")
	}

	// Use lines with multiple matches so UpdatePattern doesn't auto-jump
	lines := []string{
		"hello hello hello",
	}
	s.Enter(0, 0, 0)
	action := s.UpdatePattern("hello", lines, 0, 1)

	if action.Type != ActionContinue {
		t.Fatalf("setup: expected ActionContinue, got %d (need multiple matches)", action.Type)
	}

	o := s.Overlay()
	if o == nil {
		t.Fatal("overlay should not be nil when active with matches")
	}
	if o.Prompt != "hello" {
		t.Errorf("overlay prompt = %q, want %q", o.Prompt, "hello")
	}

	// Cancel and check overlay becomes nil
	s.HandleKey(27, nil)
	if s.Overlay() != nil {
		t.Error("overlay should be nil after cancel")
	}
}

func TestState_DefaultMinChars(t *testing.T) {
	s := New(Options{}) // MinChars defaults to 0 → should become 1

	lines := []string{"abc"}
	s.Enter(0, 0, 0)
	action := s.UpdatePattern("abc", lines, 0, 1)

	// Should auto-jump since there's exactly 1 match
	if action.Type != ActionAutoJump {
		t.Errorf("default minChars: action = %d, want ActionAutoJump", action.Type)
	}
}

func TestState_HandleKeyUnknownChar(t *testing.T) {
	s := New(Options{MinChars: 1})

	// Use a pattern with multiple matches so we stay in flash mode
	s.Enter(0, 0, 0)
	lines := []string{"ab ab ab"}
	s.UpdatePattern("ab", lines, 0, 1)

	// Press a non-alphabetic char that cannot be a label or alt-jump key.
	// Labels are lowercase a-z, alt-jump uses uppercase A-Z → use '!' to avoid both.
	action := s.HandleKey('!', lines)
	if action.Type != ActionContinue {
		t.Errorf("unknown char action = %d, want ActionContinue", action.Type)
	}
}

func TestState_BackspaceKey8(t *testing.T) {
	s := New(Options{MinChars: 1})
	lines := []string{"hello"}

	s.Enter(0, 0, 0)
	s.UpdatePattern("h", lines, 0, 1)

	// Backspace via key code 8 (ctrl-H)
	action := s.HandleKey(8, nil)
	if action.Type != ActionCancel {
		t.Errorf("backspace(8) with 1-char pattern action = %d, want ActionCancel", action.Type)
	}
}

func TestState_EmptyPatternNoMatches(t *testing.T) {
	s := New(Options{MinChars: 1})
	lines := []string{"hello"}

	s.Enter(0, 0, 0)
	action := s.UpdatePattern("", lines, 0, 1)

	if action.Type != ActionContinue {
		t.Errorf("empty pattern action = %d, want ActionContinue", action.Type)
	}
	if len(s.Matches) != 0 {
		t.Errorf("empty pattern: expected 0 matches, got %d", len(s.Matches))
	}
}

func TestState_BackspaceEmptyPattern(t *testing.T) {
	s := New(Options{MinChars: 1})

	s.Enter(0, 0, 0)
	s.UpdatePattern("", []string{"hello"}, 0, 1)

	// Backspace with empty pattern → cancel (len <= 1 is true for len 0)
	action := s.HandleKey(127, nil)
	if action.Type != ActionCancel {
		t.Errorf("backspace empty action = %d, want ActionCancel", action.Type)
	}
}

func TestFlash_FTMultipleMatches(t *testing.T) {
	f := New(Options{MinChars: 1})
	// Simulate: line has multiple 'a' chars
	line := "banana"
	matches := FindMatches([]string{line}, "a", 0, 1)
	if len(matches) != 3 {
		t.Fatalf("expected 3 'a' matches, got %d", len(matches))
	}
	f.Enter(0, 0, 0)
	act := f.UpdatePattern("a", []string{line}, 0, 1)
	// With 3 matches, should continue (not auto-jump)
	if act.Type != ActionContinue {
		t.Errorf("expected ActionContinue with 3 matches, got %v", act.Type)
	}
	// All 3 matches should have labels
	for i, m := range f.Matches {
		if m.Label == 0 {
			t.Errorf("match[%d] should have label", i)
		}
	}
}

func TestFlash_FTSingleMatchAutoJumps(t *testing.T) {
	f := New(Options{MinChars: 1})
	// Line with only one 'z'
	line := "banana z fruit"
	matches := FindMatches([]string{line}, "z", 0, 1)
	if len(matches) != 1 {
		t.Fatalf("expected 1 'z' match, got %d", len(matches))
	}
	f.Enter(0, 0, 0)
	act := f.UpdatePattern("z", []string{line}, 0, 1)
	// Single match should auto-jump
	if act.Type != ActionAutoJump {
		t.Errorf("expected ActionAutoJump with 1 match, got %v", act.Type)
	}
	if act.Col != 7 {
		t.Errorf("auto-jump col = %d, want 7", act.Col)
	}
}

func TestFlash_FTScopedToSingleLine(t *testing.T) {
	f := New(Options{MinChars: 1})
	// Multiple lines, but viewport scoped to one line
	lines := []string{
		"apple apricot",
		"banana",
		"avocado",
	}
	// Scope to line 1 only (banana) — searching for 'a' should find 3 matches (b-a-n-a-n-a)
	f.Enter(1, 0, 0)
	act := f.UpdatePattern("a", lines, 1, 1)
	if act.Type != ActionContinue {
		t.Errorf("expected ActionContinue with 3 matches on line 1, got %v", act.Type)
	}
	if len(f.Matches) != 3 {
		t.Fatalf("expected 3 'a' matches on 'banana', got %d", len(f.Matches))
	}
	// Matches should have Line == 1 (document coordinates)
	for _, m := range f.Matches {
		if m.Line != 1 {
			t.Errorf("match line = %d, want 1", m.Line)
		}
	}
}

func TestFlash_LabelDisambiguation(t *testing.T) {
	// Setup: pattern "a" has matches, one may be labeled 's'.
	// Typing 's' when extending to "as" still produces matches should extend pattern,
	// not jump to the label. This test verifies the data needed for disambiguation
	// is available from the flash package.
	lines := []string{"as as as basket"}
	f := New(Options{MinChars: 1})
	f.Enter(0, 0, 0)

	// Search for "a"
	act := f.UpdatePattern("a", lines, 0, 1)
	if act.Type != ActionContinue {
		t.Fatalf("expected continue, got %v", act.Type)
	}

	// Check if 's' is assigned as a label
	hasLabelS := false
	for _, m := range f.Matches {
		if m.Label == 's' {
			hasLabelS = true
			break
		}
	}

	// Whether or not 's' is a label, extending to "as" should find matches
	extended := FindMatches(lines, "as", 0, 1)
	if len(extended) < 2 {
		t.Fatalf("expected at least 2 matches for 'as', got %d", len(extended))
	}

	// Verify we can check label membership and test extended matches independently.
	// The TUI uses this data to decide: if extending produces matches, treat as
	// pattern extension; if not, treat as label jump.
	t.Logf("label 's' assigned: %v, extended matches: %d", hasLabelS, len(extended))
}

func TestFlash_LabelDisambiguationNoExtendedMatches(t *testing.T) {
	// When extending the pattern produces zero matches, a label char should jump.
	lines := []string{"fox fox fox"}
	f := New(Options{MinChars: 1})
	f.Enter(0, 0, 0)

	// Search for "fox" -- should have 3 matches with labels
	act := f.UpdatePattern("fox", lines, 0, 1)
	if act.Type != ActionContinue {
		t.Fatalf("expected continue, got %v", act.Type)
	}
	if len(f.Matches) != 3 {
		t.Fatalf("expected 3 matches, got %d", len(f.Matches))
	}

	// Find any label that does NOT extend the pattern to produce matches
	for _, m := range f.Matches {
		if m.Label == 0 {
			continue
		}
		extended := FindMatches(lines, "fox"+string(rune(m.Label)), 0, 1)
		if len(extended) == 0 {
			// This label char doesn't extend the pattern -- disambiguation
			// should treat it as a label jump.
			jumpAct := f.HandleKey(m.Label, lines)
			if jumpAct.Type != ActionJump {
				t.Errorf("expected ActionJump for label '%c' with no extended matches, got %v",
					m.Label, jumpAct.Type)
			}
			return
		}
	}
	t.Skip("all labels also extend the pattern -- cannot test this case with this input")
}

func TestHandleKey_JumpPositions(t *testing.T) {
	// Use "the quick brown fox jumps over the lazy dog"
	// Pattern "the" appears twice (col 0 and col 31), ensuring multi-match (no auto-jump).
	lines := []string{"the quick brown fox jumps over the lazy dog"}

	t.Run("default_lowercase_match_end", func(t *testing.T) {
		s := New(Options{MinChars: 1}) // default: jumpPos=MatchEnd, altJumpPos=MatchStart
		s.Enter(0, 0, 0)
		act := s.UpdatePattern("the", lines, 0, 1)
		if act.Type != ActionContinue {
			t.Fatalf("expected ActionContinue, got %d", act.Type)
		}
		if len(s.Matches) < 2 {
			t.Fatalf("expected at least 2 matches for 'the', got %d", len(s.Matches))
		}

		// Find the match at col 0
		var target Match
		for _, m := range s.Matches {
			if m.ColStart == 0 {
				target = m
				break
			}
		}
		if target.Label == 0 {
			t.Fatal("match at col 0 should have a label")
		}

		// Lowercase label → JumpPosMatchEnd → col 2 ("the" is cols 0-2, ColEnd=3, so 3-1=2)
		act = s.HandleKey(target.Label, lines)
		if act.Type != ActionJump {
			t.Fatalf("expected ActionJump, got %d", act.Type)
		}
		if act.Col != 2 {
			t.Errorf("match_end: col = %d, want 2", act.Col)
		}
	})

	t.Run("uppercase_alt_match_start", func(t *testing.T) {
		s := New(Options{MinChars: 1}) // default altJumpPos=MatchStart
		s.Enter(0, 0, 0)
		act := s.UpdatePattern("the", lines, 0, 1)
		if act.Type != ActionContinue {
			t.Fatalf("expected ActionContinue, got %d", act.Type)
		}

		// Find the match at col 31 ("the" second occurrence)
		var target Match
		for _, m := range s.Matches {
			if m.ColStart == 31 {
				target = m
				break
			}
		}
		if target.Label == 0 {
			t.Fatal("match at col 31 should have a label")
		}

		// Uppercase of the label → AltJumpPos (MatchStart) → col 31
		upperKey := target.Label - 32 // 'a' → 'A'
		act = s.HandleKey(upperKey, lines)
		if act.Type != ActionJump {
			t.Fatalf("expected ActionJump, got %d", act.Type)
		}
		if act.Col != 31 {
			t.Errorf("alt match_start: col = %d, want 31", act.Col)
		}
	})

	t.Run("explicit_match_start_jumppos", func(t *testing.T) {
		s := New(Options{
			MinChars:   1,
			JumpPos:    JumpPosMatchStart,
			AltJumpPos: JumpPosOff,
		})
		s.Enter(0, 0, 0)
		act := s.UpdatePattern("the", lines, 0, 1)
		if act.Type != ActionContinue {
			t.Fatalf("expected ActionContinue, got %d", act.Type)
		}

		var target Match
		for _, m := range s.Matches {
			if m.ColStart == 0 {
				target = m
				break
			}
		}
		if target.Label == 0 {
			t.Fatal("match at col 0 should have a label")
		}

		act = s.HandleKey(target.Label, lines)
		if act.Type != ActionJump {
			t.Fatalf("expected ActionJump, got %d", act.Type)
		}
		if act.Col != 0 {
			t.Errorf("match_start: col = %d, want 0", act.Col)
		}
	})

	t.Run("word_start_jumppos", func(t *testing.T) {
		// "ow" in "brown" starts at col 12, word "brown" starts at col 10
		s := New(Options{
			MinChars:   1,
			JumpPos:    JumpPosWordStart,
			AltJumpPos: JumpPosOff,
		})
		// "ow" appears in "brown" (col 12) and "over" (col 31... wait let me verify)
		// "the quick brown fox jumps over the lazy dog"
		//  0123456789012345678901234567890123456789012345
		// "brown" at col 10, "ow" at col 12
		// "over" at col 26, not "ow"... let me check
		// Actually: "over" - "ov" not "ow". Let me use a different pattern.
		// Use "ow" which appears in "brown" (col 12) and... let me check:
		// b-r-o-w-n => o at 12, w at 13 => "ow" at 12
		// Actually, does "ow" appear anywhere else? "over" has "ov" not "ow".
		// So "ow" is unique → auto-jump. Need a multi-match pattern.
		// Use lines with multiple "ow" occurrences:
		testLines := []string{"the brown cow showed power"}
		s.Enter(0, 0, 0)
		act := s.UpdatePattern("ow", testLines, 0, 1)
		if act.Type != ActionContinue {
			t.Fatalf("expected ActionContinue, got %d (matches: %d)", act.Type, len(s.Matches))
		}
		if len(s.Matches) < 2 {
			t.Fatalf("expected at least 2 'ow' matches, got %d", len(s.Matches))
		}

		// "brown" starts at col 4, "ow" at col 6 → word_start of "ow" → col 4 ("brown")
		var target Match
		for _, m := range s.Matches {
			if m.ColStart == 6 { // "ow" in "brown"
				target = m
				break
			}
		}
		if target.Label == 0 {
			t.Fatal("match at col 6 should have a label")
		}

		act = s.HandleKey(target.Label, testLines)
		if act.Type != ActionJump {
			t.Fatalf("expected ActionJump, got %d", act.Type)
		}
		if act.Col != 4 {
			t.Errorf("word_start: col = %d, want 4", act.Col)
		}
	})

	t.Run("word_end_jumppos", func(t *testing.T) {
		testLines := []string{"the brown cow showed power"}
		s := New(Options{
			MinChars:   1,
			JumpPos:    JumpPosWordEnd,
			AltJumpPos: JumpPosOff,
		})
		s.Enter(0, 0, 0)
		act := s.UpdatePattern("ow", testLines, 0, 1)
		if act.Type != ActionContinue {
			t.Fatalf("expected ActionContinue, got %d", act.Type)
		}

		// "ow" in "brown": ColStart=6, ColEnd=8, last char at col 7 ('w')
		// word "brown" ends at col 8 ('n'), word_end → 8
		var target Match
		for _, m := range s.Matches {
			if m.ColStart == 6 { // "ow" in "brown"
				target = m
				break
			}
		}
		if target.Label == 0 {
			t.Fatal("match at col 6 should have a label")
		}

		act = s.HandleKey(target.Label, testLines)
		if act.Type != ActionJump {
			t.Fatalf("expected ActionJump, got %d", act.Type)
		}
		if act.Col != 8 {
			t.Errorf("word_end: col = %d, want 8", act.Col)
		}
	})

	t.Run("alt_jump_off", func(t *testing.T) {
		s := New(Options{
			MinChars:   1,
			JumpPos:    JumpPosMatchEnd,
			AltJumpPos: JumpPosOff,
		})
		s.Enter(0, 0, 0)
		act := s.UpdatePattern("the", lines, 0, 1)
		if act.Type != ActionContinue {
			t.Fatalf("expected ActionContinue, got %d", act.Type)
		}

		var target Match
		for _, m := range s.Matches {
			if m.ColStart == 0 && m.Label != 0 {
				target = m
				break
			}
		}
		if target.Label == 0 {
			t.Fatal("match at col 0 should have a label")
		}

		// Uppercase key when altJumpPos=Off → should NOT jump, should continue
		upperKey := target.Label - 32
		act = s.HandleKey(upperKey, lines)
		if act.Type != ActionContinue {
			t.Errorf("alt_jump_off: expected ActionContinue, got %d", act.Type)
		}
	})

	t.Run("uppercase_unmatched_label_continues", func(t *testing.T) {
		s := New(Options{MinChars: 1}) // default: altJumpPos=MatchStart
		s.Enter(0, 0, 0)
		act := s.UpdatePattern("the", lines, 0, 1)
		if act.Type != ActionContinue {
			t.Fatalf("expected ActionContinue, got %d", act.Type)
		}

		// Press uppercase of a char that is NOT assigned as any label
		// Use '!' (non-alphabetic) to guarantee it's not a label match
		act = s.HandleKey('!', lines)
		if act.Type != ActionContinue {
			t.Errorf("expected ActionContinue for non-label char, got %d", act.Type)
		}
	})

	t.Run("nil_lines_graceful", func(t *testing.T) {
		// HandleKey with nil lines should still resolve (using empty string for line text)
		s := New(Options{MinChars: 1})
		s.Enter(0, 0, 0)
		s.UpdatePattern("the", lines, 0, 1) // assign labels normally

		var target Match
		for _, m := range s.Matches {
			if m.Label != 0 {
				target = m
				break
			}
		}

		// With nil lines, ResolveJumpCol gets empty string, but should not panic
		act := s.HandleKey(target.Label, nil)
		if act.Type != ActionJump {
			t.Fatalf("expected ActionJump even with nil lines, got %d", act.Type)
		}
	})
}

func TestAutoJump_UsesJumpPos(t *testing.T) {
	// Verify auto-jump (single labeled match via UpdatePattern) respects jumpPos.
	// Use "xylophone" which is unique in the text → triggers auto-jump.
	lines := []string{"the xylophone rings loudly"}
	// "xylophone" starts at col 4, ends at col 13 (9 chars)

	t.Run("default_match_end", func(t *testing.T) {
		s := New(Options{MinChars: 1})
		s.Enter(0, 0, 0)
		act := s.UpdatePattern("xylophone", lines, 0, 1)
		if act.Type != ActionAutoJump {
			t.Fatalf("expected ActionAutoJump, got %d", act.Type)
		}
		// JumpPosMatchEnd: ColEnd-1 = 13-1 = 12
		if act.Col != 12 {
			t.Errorf("match_end auto-jump col = %d, want 12", act.Col)
		}
	})

	t.Run("match_start", func(t *testing.T) {
		s := New(Options{MinChars: 1, JumpPos: JumpPosMatchStart})
		s.Enter(0, 0, 0)
		act := s.UpdatePattern("xylophone", lines, 0, 1)
		if act.Type != ActionAutoJump {
			t.Fatalf("expected ActionAutoJump, got %d", act.Type)
		}
		// JumpPosMatchStart: ColStart = 4
		if act.Col != 4 {
			t.Errorf("match_start auto-jump col = %d, want 4", act.Col)
		}
	})

	t.Run("word_start", func(t *testing.T) {
		s := New(Options{MinChars: 1, JumpPos: JumpPosWordStart})
		s.Enter(0, 0, 0)
		// "xylo" is unique, within word "xylophone" starting at col 4
		act := s.UpdatePattern("xylo", lines, 0, 1)
		if act.Type != ActionAutoJump {
			t.Fatalf("expected ActionAutoJump, got %d", act.Type)
		}
		// WordStart: "xylo" starts at col 4, word "xylophone" starts at col 4
		if act.Col != 4 {
			t.Errorf("word_start auto-jump col = %d, want 4", act.Col)
		}
	})

	t.Run("word_end", func(t *testing.T) {
		s := New(Options{MinChars: 1, JumpPos: JumpPosWordEnd})
		s.Enter(0, 0, 0)
		// "xylo" is unique, within word "xylophone" ending at col 12
		act := s.UpdatePattern("xylo", lines, 0, 1)
		if act.Type != ActionAutoJump {
			t.Fatalf("expected ActionAutoJump, got %d", act.Type)
		}
		// WordEnd: "xylo" ends at col 8 (exclusive), last match char at col 7
		// Word "xylophone" ends at col 12, so word_end → 12
		if act.Col != 12 {
			t.Errorf("word_end auto-jump col = %d, want 12", act.Col)
		}
	})
}
