package ui

import (
	"testing"

	vmode "github.com/shitcoding/tmux_yankee/internal/mode"
)

func TestTUI_TextObject_InnerParen_CursorOutside(t *testing.T) {
	// Content with parens — cursor will start before them
	content := []string{"cursor here (content) after"}
	tui := newTestTUI("test-pane", content, "absolute")
	tui.width = 80
	tui.height = 24
	tui.cursorLine = 0
	tui.cursorCol = 0 // cursor at start, before the parens

	// Feed "vi(" — enter visual mode, then inner paren text object
	tui.handleInput([]byte{'v'})

	// Verify we're in visual mode
	if tui.modeMachine.Mode() == vmode.Normal {
		t.Fatal("expected visual mode after 'v', got Normal")
	}

	tui.handleInput([]byte{'i'})
	tui.handleInput([]byte{'('})

	// After vi(, we should have a selection covering "content"
	// The word "content" starts at col 13 and ends at col 19
	region := tui.modeMachine.Region()
	t.Logf("Mode: %v", tui.modeMachine.Mode())
	t.Logf("Region: start=(%d,%d) end=(%d,%d)", region.Start.Line, region.Start.Col, region.End.Line, region.End.Col)
	t.Logf("Cursor: line=%d col=%d", tui.cursorLine, tui.cursorCol)

	if tui.modeMachine.Mode() == vmode.Normal {
		t.Fatal("expected visual mode after vi(, got Normal — text object didn't fire")
	}

	// The paren content "content" is at cols 13-19
	// Selection should cover this range
	start, end := region.Start, region.End
	if start.Col > end.Col {
		start, end = end, start
	}
	if start.Col != 13 || end.Col != 19 {
		t.Errorf("expected selection cols [13,19], got [%d,%d]", start.Col, end.Col)
	}
}

func TestTUI_TextObject_InnerParen_CursorInside(t *testing.T) {
	content := []string{"foo (bar baz) end"}
	tui := newTestTUI("test-pane", content, "absolute")
	tui.width = 80
	tui.height = 24
	tui.cursorLine = 0
	tui.cursorCol = 6 // cursor on 'a' in "bar"

	tui.handleInput([]byte{'v'})
	tui.handleInput([]byte{'i'})
	tui.handleInput([]byte{'('})

	region := tui.modeMachine.Region()
	t.Logf("Region: start=(%d,%d) end=(%d,%d)", region.Start.Line, region.Start.Col, region.End.Line, region.End.Col)

	if tui.modeMachine.Mode() == vmode.Normal {
		t.Fatal("expected visual mode after vi( inside parens, got Normal")
	}

	// "bar baz" is at cols 5-11
	start, end := region.Start, region.End
	if start.Col > end.Col {
		start, end = end, start
	}
	if start.Col != 5 || end.Col != 11 {
		t.Errorf("expected selection cols [5,11], got [%d,%d]", start.Col, end.Col)
	}
}

func TestTUI_TextObject_InnerBracket_CursorOutside(t *testing.T) {
	content := []string{"cursor [content] after"}
	tui := newTestTUI("test-pane", content, "absolute")
	tui.width = 80
	tui.height = 24
	tui.cursorLine = 0
	tui.cursorCol = 0

	tui.handleInput([]byte{'v'})
	tui.handleInput([]byte{'i'})
	tui.handleInput([]byte{'['})

	region := tui.modeMachine.Region()
	t.Logf("Region: start=(%d,%d) end=(%d,%d)", region.Start.Line, region.Start.Col, region.End.Line, region.End.Col)

	if tui.modeMachine.Mode() == vmode.Normal {
		t.Fatal("expected visual mode after vi[ outside brackets, got Normal")
	}

	// "content" is at cols 8-14
	start, end := region.Start, region.End
	if start.Col > end.Col {
		start, end = end, start
	}
	if start.Col != 8 || end.Col != 14 {
		t.Errorf("expected selection cols [8,14], got [%d,%d]", start.Col, end.Col)
	}
}

func TestTUI_TextObject_InnerParen_MultiLine_CursorOutside(t *testing.T) {
	content := []string{
		"cursor here",
		"(",
		"  inner content",
		")",
	}
	tui := newTestTUI("test-pane", content, "absolute")
	tui.width = 80
	tui.height = 24
	tui.cursorLine = 0
	tui.cursorCol = 0

	tui.handleInput([]byte{'v'})
	tui.handleInput([]byte{'i'})
	tui.handleInput([]byte{'('})

	region := tui.modeMachine.Region()
	t.Logf("Mode: %v", tui.modeMachine.Mode())
	t.Logf("Region: start=(%d,%d) end=(%d,%d)", region.Start.Line, region.Start.Col, region.End.Line, region.End.Col)

	if tui.modeMachine.Mode() == vmode.Normal {
		t.Fatal("expected visual mode after vi( with cursor before multiline block, got Normal")
	}

	// Inner content should be on line 2
	if region.Start.Line != 2 && region.End.Line != 2 {
		t.Errorf("expected selection to include line 2, got start=%d end=%d",
			region.Start.Line, region.End.Line)
	}
}
