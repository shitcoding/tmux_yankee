package textobj

import (
	"testing"

	"github.com/shitcoding/tmux_yankee/internal/keymap"
	"github.com/shitcoding/tmux_yankee/internal/motion"
)

// testDoc implements motion.Document for testing.
type testDoc struct {
	lines []string
}

func (d *testDoc) LineCount() int          { return len(d.lines) }
func (d *testDoc) Line(i int) string       { return d.lines[i] }
func (d *testDoc) LineRuneCount(i int) int { return len([]rune(d.lines[i])) }

func TestInnerBracket_CursorInsideParen(t *testing.T) {
	doc := &testDoc{lines: []string{"foo (bar) baz"}}
	// Cursor on 'b' inside parens (col 5)
	r := Resolve(doc, motion.Cursor{Line: 0, Col: 5}, keymap.ActionTextObjectInnerParen)
	if !r.OK {
		t.Fatal("expected OK range for cursor inside parens")
	}
	if r.StartCol != 5 || r.EndCol != 7 {
		t.Errorf("inner paren: got cols [%d,%d], want [5,7]", r.StartCol, r.EndCol)
	}
}

func TestInnerBracket_CursorBeforeParenSameLine(t *testing.T) {
	doc := &testDoc{lines: []string{"foo (bar) baz"}}
	// Cursor on 'f' before the parens (col 0)
	r := Resolve(doc, motion.Cursor{Line: 0, Col: 0}, keymap.ActionTextObjectInnerParen)
	if !r.OK {
		t.Fatal("expected OK range for cursor before parens on same line")
	}
	if r.StartCol != 5 || r.EndCol != 7 {
		t.Errorf("inner paren: got cols [%d,%d], want [5,7]", r.StartCol, r.EndCol)
	}
}

func TestInnerBracket_CursorOnOpenParen(t *testing.T) {
	doc := &testDoc{lines: []string{"foo (bar) baz"}}
	// Cursor on '(' (col 4)
	r := Resolve(doc, motion.Cursor{Line: 0, Col: 4}, keymap.ActionTextObjectInnerParen)
	if !r.OK {
		t.Fatal("expected OK range for cursor on open paren")
	}
	if r.StartCol != 5 || r.EndCol != 7 {
		t.Errorf("inner paren: got cols [%d,%d], want [5,7]", r.StartCol, r.EndCol)
	}
}

func TestInnerBracket_CursorOnCloseParen(t *testing.T) {
	doc := &testDoc{lines: []string{"foo (bar) baz"}}
	// Cursor on ')' (col 8)
	r := Resolve(doc, motion.Cursor{Line: 0, Col: 8}, keymap.ActionTextObjectInnerParen)
	if !r.OK {
		t.Fatal("expected OK range for cursor on close paren")
	}
	if r.StartCol != 5 || r.EndCol != 7 {
		t.Errorf("inner paren: got cols [%d,%d], want [5,7]", r.StartCol, r.EndCol)
	}
}

func TestInnerBracket_CursorBeforeParenDifferentLine(t *testing.T) {
	doc := &testDoc{lines: []string{
		"cursor here",
		"(content)",
	}}
	// Cursor on line 0 before the brackets entirely
	r := Resolve(doc, motion.Cursor{Line: 0, Col: 0}, keymap.ActionTextObjectInnerParen)
	if !r.OK {
		t.Fatal("expected OK range for cursor on line before parens")
	}
	if r.StartLine != 1 || r.StartCol != 1 || r.EndLine != 1 || r.EndCol != 7 {
		t.Errorf("inner paren: got [%d:%d - %d:%d], want [1:1 - 1:7]",
			r.StartLine, r.StartCol, r.EndLine, r.EndCol)
	}
}

func TestInnerBracket_MultiLineBlock(t *testing.T) {
	doc := &testDoc{lines: []string{
		"before",
		"(",
		"  content",
		")",
	}}
	// Cursor on "before" (line 0)
	r := Resolve(doc, motion.Cursor{Line: 0, Col: 0}, keymap.ActionTextObjectInnerParen)
	if !r.OK {
		t.Fatal("expected OK range for cursor before multiline block")
	}
	// Inner should be line 2 "  content"
	if r.StartLine != 2 || r.StartCol != 0 || r.EndLine != 2 {
		t.Errorf("inner paren: got [%d:%d - %d:%d], want start at [2:0]",
			r.StartLine, r.StartCol, r.EndLine, r.EndCol)
	}
}

func TestInnerBracket_CursorInsideMultiLine(t *testing.T) {
	doc := &testDoc{lines: []string{
		"(",
		"  content",
		")",
	}}
	// Cursor on "content" (line 1, col 2)
	r := Resolve(doc, motion.Cursor{Line: 1, Col: 2}, keymap.ActionTextObjectInnerParen)
	if !r.OK {
		t.Fatal("expected OK range for cursor inside multiline block")
	}
	if r.StartLine != 1 || r.StartCol != 0 {
		t.Errorf("inner paren: got start [%d:%d], want [1:0]",
			r.StartLine, r.StartCol)
	}
}

func TestInnerBracket_SquareBracket(t *testing.T) {
	doc := &testDoc{lines: []string{"foo [bar] baz"}}
	// Cursor before brackets (col 0)
	r := Resolve(doc, motion.Cursor{Line: 0, Col: 0}, keymap.ActionTextObjectInnerBracket)
	if !r.OK {
		t.Fatal("expected OK range for cursor before square brackets")
	}
	if r.StartCol != 5 || r.EndCol != 7 {
		t.Errorf("inner bracket: got cols [%d,%d], want [5,7]", r.StartCol, r.EndCol)
	}
}

func TestInnerBracket_AngleBracket(t *testing.T) {
	doc := &testDoc{lines: []string{"foo <bar> baz"}}
	// Cursor before brackets (col 0)
	r := Resolve(doc, motion.Cursor{Line: 0, Col: 0}, keymap.ActionTextObjectInnerAngle)
	if !r.OK {
		t.Fatal("expected OK range for cursor before angle brackets")
	}
	if r.StartCol != 5 || r.EndCol != 7 {
		t.Errorf("inner angle: got cols [%d,%d], want [5,7]", r.StartCol, r.EndCol)
	}
}

func TestABracket_CursorBeforeParen(t *testing.T) {
	doc := &testDoc{lines: []string{"foo (bar) baz"}}
	// Cursor on 'f' (col 0) — before parens
	r := Resolve(doc, motion.Cursor{Line: 0, Col: 0}, keymap.ActionTextObjectAParen)
	if !r.OK {
		t.Fatal("expected OK range for cursor before parens with 'a' text object")
	}
	// 'a' includes the brackets themselves
	if r.StartCol != 4 || r.EndCol != 8 {
		t.Errorf("a paren: got cols [%d,%d], want [4,8]", r.StartCol, r.EndCol)
	}
}

func TestInnerBracket_CursorAfterParen(t *testing.T) {
	doc := &testDoc{lines: []string{"foo (bar) baz"}}
	// Cursor after close paren (col 10) — backward fallback finds the pair behind cursor
	r := Resolve(doc, motion.Cursor{Line: 0, Col: 10}, keymap.ActionTextObjectInnerParen)
	if !r.OK {
		t.Fatal("expected match for cursor after closing paren (backward fallback)")
	}
	// Inner content "bar" is at cols 5-7
	if r.StartCol != 5 || r.EndCol != 7 {
		t.Errorf("expected cols [5,7], got [%d,%d]", r.StartCol, r.EndCol)
	}
}

func TestInnerBracket_CursorAfterParenDifferentLine(t *testing.T) {
	doc := &testDoc{lines: []string{
		"first (content) here",
		"second line",
		"cursor here",
	}}
	// Cursor on line 2, parens on line 0 — backward fallback finds them
	r := Resolve(doc, motion.Cursor{Line: 2, Col: 3}, keymap.ActionTextObjectInnerParen)
	if !r.OK {
		t.Fatal("expected match from backward fallback across lines")
	}
	// Inner content "content" is at line 0, cols 7-13
	if r.StartLine != 0 || r.StartCol != 7 || r.EndLine != 0 || r.EndCol != 13 {
		t.Errorf("expected (0,7)-(0,13), got (%d,%d)-(%d,%d)",
			r.StartLine, r.StartCol, r.EndLine, r.EndCol)
	}
}

func TestInnerBracket_UnmatchedOpenBeforeMatchedPair(t *testing.T) {
	// Bug: findOpenBracket finds an unmatched '(' above cursor, but there's no
	// matching ')'. Meanwhile a complete (matched) pair exists further away.
	// The fix: if findCloseBracket fails, fall through to the next strategy.
	doc := &testDoc{lines: []string{
		"unmatched open ( here with no close",
		"some text",
		"matched pair (content) on this line",
		"cursor here",
	}}
	r := Resolve(doc, motion.Cursor{Line: 3, Col: 0}, keymap.ActionTextObjectInnerParen)
	if !r.OK {
		t.Fatal("expected match: should skip unmatched bracket and find matched pair via fallback")
	}
	// Should find the matched pair on line 2: inner "content" at cols 14-20
	if r.StartLine != 2 || r.StartCol != 14 || r.EndLine != 2 || r.EndCol != 20 {
		t.Errorf("expected (2,14)-(2,20), got (%d,%d)-(%d,%d)",
			r.StartLine, r.StartCol, r.EndLine, r.EndCol)
	}
}

func TestABracket_UnmatchedOpenBeforeMatchedPair(t *testing.T) {
	doc := &testDoc{lines: []string{
		"unmatched ( here",
		"matched (pair) below",
		"cursor",
	}}
	r := Resolve(doc, motion.Cursor{Line: 2, Col: 0}, keymap.ActionTextObjectAParen)
	if !r.OK {
		t.Fatal("expected match: should skip unmatched bracket and find matched pair via fallback")
	}
	// aBracket includes the brackets: (pair) at line 1, cols 8-13
	if r.StartLine != 1 || r.StartCol != 8 || r.EndLine != 1 || r.EndCol != 13 {
		t.Errorf("expected (1,8)-(1,13), got (%d,%d)-(%d,%d)",
			r.StartLine, r.StartCol, r.EndLine, r.EndCol)
	}
}
