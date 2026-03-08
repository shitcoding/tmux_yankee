package flash

import (
	"testing"
)

func TestFindMatches_Basic(t *testing.T) {
	lines := []string{
		"hello world",
		"foo bar baz",
		"hello again",
	}

	matches := FindMatches(lines, "hello", 0, 3)

	if len(matches) != 2 {
		t.Fatalf("expected 2 matches, got %d", len(matches))
	}

	if matches[0].Line != 0 || matches[0].ColStart != 0 || matches[0].ColEnd != 5 {
		t.Errorf("match[0] = {Line:%d, ColStart:%d, ColEnd:%d}, want {0, 0, 5}",
			matches[0].Line, matches[0].ColStart, matches[0].ColEnd)
	}

	if matches[1].Line != 2 || matches[1].ColStart != 0 || matches[1].ColEnd != 5 {
		t.Errorf("match[1] = {Line:%d, ColStart:%d, ColEnd:%d}, want {2, 0, 5}",
			matches[1].Line, matches[1].ColStart, matches[1].ColEnd)
	}
}

func TestFindMatches_SmartcaseLower(t *testing.T) {
	lines := []string{
		"Hello World",
		"HELLO WORLD",
		"hello world",
	}

	// All-lowercase pattern → case-insensitive
	matches := FindMatches(lines, "hello", 0, 3)

	if len(matches) != 3 {
		t.Fatalf("smartcase lowercase: expected 3 matches, got %d", len(matches))
	}
}

func TestFindMatches_SmartcaseUpper(t *testing.T) {
	lines := []string{
		"Hello World",
		"HELLO WORLD",
		"hello world",
	}

	// Pattern has uppercase → case-sensitive
	matches := FindMatches(lines, "Hello", 0, 3)

	if len(matches) != 1 {
		t.Fatalf("smartcase uppercase: expected 1 match, got %d", len(matches))
	}

	if matches[0].Line != 0 {
		t.Errorf("expected match on line 0, got line %d", matches[0].Line)
	}
}

func TestFindMatches_ViewportClipping(t *testing.T) {
	lines := []string{
		"match here", // line 0 - outside viewport
		"match here", // line 1 - outside viewport
		"match here", // line 2 - in viewport
		"match here", // line 3 - in viewport
		"match here", // line 4 - outside viewport
	}

	matches := FindMatches(lines, "match", 2, 2)

	if len(matches) != 2 {
		t.Fatalf("viewport clipping: expected 2 matches, got %d", len(matches))
	}

	if matches[0].Line != 2 || matches[1].Line != 3 {
		t.Errorf("expected lines 2,3 but got %d,%d", matches[0].Line, matches[1].Line)
	}
}

func TestFindMatches_MultiplePerLine(t *testing.T) {
	lines := []string{
		"ab ab ab",
	}

	matches := FindMatches(lines, "ab", 0, 1)

	if len(matches) != 3 {
		t.Fatalf("multiple per line: expected 3 matches, got %d", len(matches))
	}

	wantCols := [][2]int{{0, 2}, {3, 5}, {6, 8}}
	for i, want := range wantCols {
		if matches[i].ColStart != want[0] || matches[i].ColEnd != want[1] {
			t.Errorf("match[%d] cols = (%d,%d), want (%d,%d)",
				i, matches[i].ColStart, matches[i].ColEnd, want[0], want[1])
		}
	}
}

func TestFindMatches_EmptyPattern(t *testing.T) {
	lines := []string{"hello world"}

	matches := FindMatches(lines, "", 0, 1)

	if matches != nil {
		t.Errorf("empty pattern: expected nil, got %d matches", len(matches))
	}
}

func TestFindMatches_OverlappingMatches(t *testing.T) {
	lines := []string{
		"aaa",
	}

	matches := FindMatches(lines, "aa", 0, 1)

	if len(matches) != 2 {
		t.Fatalf("overlapping: expected 2 matches, got %d", len(matches))
	}

	if matches[0].ColStart != 0 || matches[0].ColEnd != 2 {
		t.Errorf("match[0] = (%d,%d), want (0,2)", matches[0].ColStart, matches[0].ColEnd)
	}
	if matches[1].ColStart != 1 || matches[1].ColEnd != 3 {
		t.Errorf("match[1] = (%d,%d), want (1,3)", matches[1].ColStart, matches[1].ColEnd)
	}
}

func TestFindMatches_Unicode(t *testing.T) {
	lines := []string{
		"cafe\u0301 hello",   // "cafe\u0301" is "cafe" + combining accent (5 runes)
		"\u4f60\u597d world", // Chinese "hello" (2 runes) + " world"
	}

	// Search for "hello" in first line
	matches := FindMatches(lines, "hello", 0, 2)

	if len(matches) != 1 {
		t.Fatalf("unicode: expected 1 match for 'hello', got %d", len(matches))
	}

	// "cafe\u0301" is 5 runes (c,a,f,e,combining-accent), then space at 5, then hello starts at 6
	if matches[0].Line != 0 || matches[0].ColStart != 6 {
		t.Errorf("unicode match = {Line:%d, ColStart:%d}, want {0, 6}",
			matches[0].Line, matches[0].ColStart)
	}
}

func TestFindMatches_UnicodeMultibyte(t *testing.T) {
	lines := []string{
		"\u4f60\u597d\u4f60\u597d", // [你好你好] = 4 runes
	}

	matches := FindMatches(lines, "\u4f60\u597d", 0, 1)

	// Pattern [你好] is 2 runes. Line is [你好你好].
	// Match at col 0: [你好] matches.
	// Col 1: [好你] does NOT match.
	// Col 2: [你好] matches.
	// So 2 matches total.
	if len(matches) != 2 {
		t.Fatalf("unicode multibyte: expected 2 matches, got %d", len(matches))
	}

	if matches[0].ColStart != 0 || matches[1].ColStart != 2 {
		t.Errorf("unicode multibyte: cols = (%d, %d), want (0, 2)",
			matches[0].ColStart, matches[1].ColStart)
	}
}

func TestFindMatches_ViewportBeyondLines(t *testing.T) {
	lines := []string{
		"hello",
		"world",
	}

	// Viewport extends beyond available lines
	matches := FindMatches(lines, "hello", 0, 100)

	if len(matches) != 1 {
		t.Fatalf("viewport beyond: expected 1 match, got %d", len(matches))
	}
}

func TestFindMatches_NoMatches(t *testing.T) {
	lines := []string{
		"hello world",
	}

	matches := FindMatches(lines, "xyz", 0, 1)

	if len(matches) != 0 {
		t.Errorf("no matches: expected empty, got %d", len(matches))
	}
}

func TestFindMatches_SortOrder(t *testing.T) {
	lines := []string{
		"bb aa bb",
		"aa bb aa",
	}

	matches := FindMatches(lines, "aa", 0, 2)

	if len(matches) != 3 {
		t.Fatalf("sort order: expected 3 matches, got %d", len(matches))
	}

	// Should be sorted by line then column
	prev := matches[0]
	for i := 1; i < len(matches); i++ {
		m := matches[i]
		if m.Line < prev.Line || (m.Line == prev.Line && m.ColStart <= prev.ColStart) {
			t.Errorf("matches not sorted: match[%d]={%d,%d} before match[%d]={%d,%d}",
				i-1, prev.Line, prev.ColStart, i, m.Line, m.ColStart)
		}
		prev = m
	}
}
