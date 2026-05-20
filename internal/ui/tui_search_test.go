package ui

import "testing"

// makeSearchTUI builds a minimal TUI carrying just a document, enough for
// computeSearchMatches to be exercised without a full TUI setup.
func makeSearchTUI(lines []string) *TUI {
	return &TUI{
		doc:            NewDocument(lines),
		searchMatchIdx: -1,
	}
}

func TestComputeSearchMatches_ZeroWidthBoundary(t *testing.T) {
	// Word-boundary regex matches between non-word and word chars at every
	// boundary; without the zero-width guard each boundary would surface as a
	// 1-cell phantom match.
	ti := makeSearchTUI([]string{"foo bar baz"})
	ti.computeSearchMatches(`\b`)
	if got := len(ti.searchMatches); got != 0 {
		t.Errorf("\\b should produce 0 matches; got %d", got)
	}
}

func TestComputeSearchMatches_OptionalEmpty(t *testing.T) {
	// "a?" matches zero or one 'a' at every position; on "bbb" it matches
	// zero-width at every gap. All should be filtered out.
	ti := makeSearchTUI([]string{"bbb"})
	ti.computeSearchMatches("a?")
	if got := len(ti.searchMatches); got != 0 {
		t.Errorf("a? on bbb should produce 0 matches; got %d", got)
	}
}

func TestComputeSearchMatches_RealLengthZeroPattern(t *testing.T) {
	// Anchor-only patterns ^ and $ are zero-width per line — should be
	// filtered.
	ti := makeSearchTUI([]string{"hello", "world"})
	ti.computeSearchMatches(`^`)
	if got := len(ti.searchMatches); got != 0 {
		t.Errorf("^ should produce 0 matches; got %d", got)
	}
	ti.computeSearchMatches(`$`)
	if got := len(ti.searchMatches); got != 0 {
		t.Errorf("$ should produce 0 matches; got %d", got)
	}
}

func TestComputeSearchMatches_MixedZeroAndReal(t *testing.T) {
	// "a?" on "abba" yields 3 matches from FindAllStringIndex: real "a" at
	// [0,1], zero-width [2,2] between the two 'b's, and real "a" at [3,4].
	// (Go's regexp suppresses zero-width matches adjacent to non-empty ones,
	// so this fixture is chosen to expose one real zero-width match.) With
	// the guard, only the two real-length matches should survive.
	ti := makeSearchTUI([]string{"abba"})
	ti.computeSearchMatches("a?")
	if got := len(ti.searchMatches); got != 2 {
		t.Fatalf("expected 2 real matches for a? on \"abba\", got %d", got)
	}
	got := ti.searchMatches
	if got[0].Line != 0 || got[0].ColStart != 0 || got[0].ColEnd != 0 {
		t.Errorf("match 0: got %+v, want {Line:0 ColStart:0 ColEnd:0}", got[0])
	}
	if got[1].Line != 0 || got[1].ColStart != 3 || got[1].ColEnd != 3 {
		t.Errorf("match 1: got %+v, want {Line:0 ColStart:3 ColEnd:3}", got[1])
	}
}

func TestComputeSearchMatches_RegularPatternStillWorks(t *testing.T) {
	// Sanity check that the new guard does not regress a regular literal
	// pattern.
	ti := makeSearchTUI([]string{"foo bar foo"})
	ti.computeSearchMatches("foo")
	if got := len(ti.searchMatches); got != 2 {
		t.Fatalf("foo on \"foo bar foo\" should have 2 matches; got %d", got)
	}
}
