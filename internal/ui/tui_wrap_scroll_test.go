package ui

import (
	"strings"
	"testing"

	"github.com/shitcoding/tmux_yankee/internal/config"
	"github.com/shitcoding/tmux_yankee/internal/input"
	"github.com/shitcoding/tmux_yankee/internal/linenums"
	vmode "github.com/shitcoding/tmux_yankee/internal/mode"
	"github.com/shitcoding/tmux_yankee/internal/motion"
)

// makeWrapScrollTUI builds a TUI for wrap-mode scroll tests: short source
// lineCount, long lines that wrap into many display rows, and a viewport
// shorter than the total wrapped display rows.
//
// linesText controls the document; width/height set the viewport dimensions.
// Status bar is off so contentHeight() == height.
func makeWrapScrollTUI(t *testing.T, linesText []string, width, height int) *TUI {
	t.Helper()
	cfg := config.Settings{
		Mode:     config.LineNumberModeAbsolute,
		WrapMode: config.WrapModeOn,
		// StatusBar default (off, since not set explicitly to On).
	}
	doc := NewDocument(linesText)
	maxLine := doc.LineCount()
	if maxLine == 0 {
		maxLine = 1
	}
	return &TUI{
		cfg:            cfg,
		doc:            doc,
		lineNumMode:    string(cfg.Mode),
		formatter:      linenums.NewFormatter(linenums.ModeAbsolute, maxLine),
		modeMachine:    vmode.NewMachine(),
		motionHandler:  motion.NewVimHandler(),
		parser:         input.NewParser(),
		width:          width,
		height:         height,
		searchMatchIdx: -1,
	}
}

func TestHandleMouseScroll_WrapMode_ShortDocLongLines_AdvancesViewport(t *testing.T) {
	// 5 source lines, each ~120 chars. With width=20 (and gutter taking some
	// columns), each line wraps to several display rows. Total wrapped
	// display rows easily exceed the viewport height. lineCount=5 < ch=10.
	//
	// Pre-fix: handleMouseScroll's `lineCount > ch` is FALSE, so it falls
	// into the cursor-only path and viewportTop never moves — the user sees
	// the same content over and over.
	long := strings.Repeat("abcdefghij ", 12) // 132 chars
	lines := []string{long, long, long, long, long}
	ti := makeWrapScrollTUI(t, lines, 20, 10)

	// Start with viewport scrolled down so there's something to scroll up to.
	ti.viewportTop = 3
	ti.cursorLine = 4

	before := ti.viewportTop
	ti.handleCommand(input.Command{Type: input.CommandMouseScroll, ScrollDirection: input.ScrollUp})

	if ti.viewportTop >= before {
		t.Errorf("wheel-up in wrap mode should decrease viewportTop; before=%d after=%d", before, ti.viewportTop)
	}
}

func TestHandleMouseScroll_WrapMode_DownScrollRespectsWrapAwareMax(t *testing.T) {
	// Same setup; scroll DOWN from the top should advance viewportTop toward
	// a wrap-aware max — not the bogus `lineCount - ch + 1` value (which
	// would be negative here and clamped to 0, effectively forbidding any
	// down-scroll).
	long := strings.Repeat("abcdefghij ", 12)
	lines := []string{long, long, long, long, long}
	ti := makeWrapScrollTUI(t, lines, 20, 10)

	ti.viewportTop = 0
	ti.cursorLine = 0

	// Scroll down a few times.
	for i := 0; i < 3; i++ {
		ti.handleCommand(input.Command{Type: input.CommandMouseScroll, ScrollDirection: input.ScrollDown})
	}

	if ti.viewportTop == 0 {
		t.Errorf("wheel-down in wrap mode should advance viewportTop past 0 when wrapped display rows exceed ch; viewportTop still 0")
	}
}

func TestHandleMouseScroll_NonWrapMode_StillUsesSourceLineCheck(t *testing.T) {
	// Regression guard: non-wrap mode keeps the existing semantics.
	// 5 source lines, height=10 → lineCount <= ch → cursor-only fallback.
	lines := []string{"a", "b", "c", "d", "e"}
	cfg := config.Settings{
		Mode:     config.LineNumberModeAbsolute,
		WrapMode: config.WrapModeOff,
	}
	doc := NewDocument(lines)
	ti := &TUI{
		cfg:            cfg,
		doc:            doc,
		lineNumMode:    string(cfg.Mode),
		formatter:      linenums.NewFormatter(linenums.ModeAbsolute, 5),
		modeMachine:    vmode.NewMachine(),
		motionHandler:  motion.NewVimHandler(),
		parser:         input.NewParser(),
		width:          40,
		height:         10,
		viewportTop:    0,
		cursorLine:     2,
		searchMatchIdx: -1,
	}

	ti.handleCommand(input.Command{Type: input.CommandMouseScroll, ScrollDirection: input.ScrollUp})

	// Non-wrap, doc fits: cursor moves, viewport stays.
	if ti.viewportTop != 0 {
		t.Errorf("non-wrap small doc: viewportTop should stay 0, got %d", ti.viewportTop)
	}
	if ti.cursorLine != 1 {
		t.Errorf("non-wrap small doc: cursor should move up by 1 (2→1); got %d", ti.cursorLine)
	}
}

func TestMaxViewportTopWrap_LastLineTallerThanViewport_NeverReturnsPastEOF(t *testing.T) {
	// Architectural limit: viewportTop is source-line-indexed. If the last
	// source line alone wraps to MORE display rows than the viewport, the
	// helper must NOT return lineCount (past EOF). Largest sane value is
	// lineCount-1 so the last line is the only visible source line.
	short := "x"
	huge := strings.Repeat("abcdefghij ", 30) // ~330 chars
	lines := []string{short, short, huge}
	ti := makeWrapScrollTUI(t, lines, 20, 5)

	top := ti.maxViewportTopWrap(ti.wrapContentWidth(), ti.contentHeight())
	wantTop := ti.doc.LineCount() - 1
	if top != wantTop {
		t.Errorf("maxViewportTopWrap = %d, want %d (last source line index — viewport pinned to the overheight tail line)", top, wantTop)
	}
}

func TestHandleMouseScroll_NonWrapMode_LongDoc_ScrollsViewport(t *testing.T) {
	// Regression guard: non-wrap mode with lineCount > ch still scrolls
	// the viewport (existing behavior).
	lines := make([]string, 100)
	for i := range lines {
		lines[i] = "line"
	}
	cfg := config.Settings{
		Mode:     config.LineNumberModeAbsolute,
		WrapMode: config.WrapModeOff,
	}
	doc := NewDocument(lines)
	ti := &TUI{
		cfg:            cfg,
		doc:            doc,
		lineNumMode:    string(cfg.Mode),
		formatter:      linenums.NewFormatter(linenums.ModeAbsolute, 100),
		modeMachine:    vmode.NewMachine(),
		motionHandler:  motion.NewVimHandler(),
		parser:         input.NewParser(),
		width:          40,
		height:         20,
		viewportTop:    50,
		cursorLine:     60,
		searchMatchIdx: -1,
	}

	before := ti.viewportTop
	ti.handleCommand(input.Command{Type: input.CommandMouseScroll, ScrollDirection: input.ScrollUp})

	if ti.viewportTop >= before {
		t.Errorf("non-wrap long doc wheel-up: viewportTop should decrease; before=%d after=%d", before, ti.viewportTop)
	}
}
