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

func TestCtrlE_WrapMode_ShortDocLongLines_AdvancesViewport(t *testing.T) {
	// Ctrl-E (scroll-line-down) uses `lineCount - contentHeight` for maxTop
	// which is wrong in wrap mode. With 5 short source lines that each wrap
	// past ch, scrolling down with Ctrl-E should still advance viewportTop.
	long := strings.Repeat("abcdefghij ", 12)
	lines := []string{long, long, long, long, long}
	ti := makeWrapScrollTUI(t, lines, 20, 10)

	ti.viewportTop = 0
	ti.cursorLine = 0

	ti.handleCommand(input.Command{Type: input.CommandScrollLineDown})

	if ti.viewportTop == 0 {
		t.Errorf("Ctrl-E in wrap mode with wrapped-display-rows > ch should advance viewportTop; still 0")
	}
}

func TestCtrlY_WrapMode_ClampsCursorToWrapAwareBottom(t *testing.T) {
	// Pre-W2: Ctrl-Y clamped the cursor to `viewportTop + ch - 1` (source-
	// line arithmetic). In wrap mode that overshoots — it can leave the
	// cursor on a source line whose wrapped rows lie BELOW the visible
	// display rows. Post-W2 the clamp uses lastVisibleLineWrap so the
	// cursor lands on the last fully-visible source line.
	//
	// Setup: 10 source lines, each wrapping to ~3 display rows on a 20-col
	// terminal, ch=10. From viewportTop=5 only lines 5..~8 are visible.
	long := strings.Repeat("abcd ", 10) // 50 chars → wraps to ~3 rows on 20 cols
	lines := make([]string, 10)
	for i := range lines {
		lines[i] = long
	}
	ti := makeWrapScrollTUI(t, lines, 20, 10)

	ti.viewportTop = 5
	ti.cursorLine = 8 // sits inside the visible window at viewportTop=5

	ti.handleCommand(input.Command{Type: input.CommandScrollLineUp})

	if ti.viewportTop != 4 {
		t.Fatalf("Ctrl-Y should decrement viewportTop 5→4, got %d", ti.viewportTop)
	}
	// After viewportTop=4, the wrap-aware last fully-visible source line
	// must NOT exclude the cursor. Either cursor is unchanged (if still
	// fully visible) or clamped to lastVisibleLineWrap (if pushed off).
	wantCap := ti.lastVisibleLineWrap(ti.wrapContentWidth())
	if ti.cursorLine > wantCap {
		t.Errorf("Ctrl-Y left cursor (%d) past wrap-aware visible bottom (%d)", ti.cursorLine, wantCap)
	}
}

func TestCtrlE_NonWrapMode_LongDoc_AdvancesViewport(t *testing.T) {
	// Regression guard.
	lines := make([]string, 50)
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
		formatter:      linenums.NewFormatter(linenums.ModeAbsolute, 50),
		modeMachine:    vmode.NewMachine(),
		motionHandler:  motion.NewVimHandler(),
		parser:         input.NewParser(),
		width:          40,
		height:         20,
		viewportTop:    0,
		cursorLine:     0,
		searchMatchIdx: -1,
	}

	ti.handleCommand(input.Command{Type: input.CommandScrollLineDown})

	if ti.viewportTop != 1 {
		t.Errorf("non-wrap Ctrl-E should advance viewportTop 0→1, got %d", ti.viewportTop)
	}
}

func TestHalfPageDown_WrapMode_AdvancesByDisplayRows(t *testing.T) {
	// Ctrl-D in wrap mode should scroll by ~half the VISIBLE display rows,
	// not by half the source-line count. With 5 source lines each wrapping
	// past ch, the motion handler returns viewportTop = current + ch/2
	// (source-line), which overshoots — viewportTop lands past the wrap-
	// aware bottom.
	long := strings.Repeat("abcdefghij ", 12) // ~130 chars → wraps to many rows on 20 wide
	lines := []string{long, long, long, long, long}
	ti := makeWrapScrollTUI(t, lines, 20, 10)

	ti.viewportTop = 0
	ti.cursorLine = 0

	ti.handleCommand(input.Command{Type: input.CommandMotion, Motion: motion.MotionHalfPageDown})

	maxTop := ti.maxViewportTopWrap(ti.wrapContentWidth(), ti.contentHeight())
	if ti.viewportTop > maxTop {
		t.Errorf("Ctrl-D in wrap mode: viewportTop=%d, exceeds wrap-aware maxTop=%d", ti.viewportTop, maxTop)
	}
	if ti.viewportTop == 0 {
		t.Errorf("Ctrl-D in wrap mode: viewportTop unchanged (still 0); should advance by display rows")
	}

	// Simulate the render-time ensureCursorVisibleWrap pass and assert it
	// does NOT pull viewportTop forward to chase a stale source-line cursor.
	postDispatchTop := ti.viewportTop
	ti.ensureCursorVisibleWrap(ti.wrapContentWidth())
	if ti.viewportTop != postDispatchTop {
		t.Errorf("ensureCursorVisibleWrap re-overshot the wrap-aware viewport: before=%d after=%d", postDispatchTop, ti.viewportTop)
	}
}

func TestHalfPageUp_WrapMode_RetreatsByDisplayRows(t *testing.T) {
	// Symmetric to HalfPageDown. From viewportTop=4 (last line), Ctrl-U
	// should retreat by ~ch/2 display rows.
	long := strings.Repeat("abcdefghij ", 12)
	lines := []string{long, long, long, long, long}
	ti := makeWrapScrollTUI(t, lines, 20, 10)

	ti.viewportTop = 4
	ti.cursorLine = 4

	ti.handleCommand(input.Command{Type: input.CommandMotion, Motion: motion.MotionHalfPageUp})

	if ti.viewportTop >= 4 {
		t.Errorf("Ctrl-U in wrap mode: viewportTop=%d, expected to decrease from 4", ti.viewportTop)
	}
	if ti.viewportTop < 0 {
		t.Errorf("Ctrl-U in wrap mode: viewportTop=%d, must not go negative", ti.viewportTop)
	}
	postDispatchTop := ti.viewportTop
	ti.ensureCursorVisibleWrap(ti.wrapContentWidth())
	if ti.viewportTop != postDispatchTop {
		t.Errorf("Ctrl-U: ensureCursorVisibleWrap pulled viewportTop back: before=%d after=%d", postDispatchTop, ti.viewportTop)
	}
}

func TestPageDown_WrapMode_AdvancesByFullScreenDisplayRows(t *testing.T) {
	long := strings.Repeat("abcdefghij ", 12)
	lines := []string{long, long, long, long, long, long}
	ti := makeWrapScrollTUI(t, lines, 20, 10)

	ti.viewportTop = 0
	ti.cursorLine = 0

	ti.handleCommand(input.Command{Type: input.CommandMotion, Motion: motion.MotionPageDown})

	if ti.viewportTop == 0 {
		t.Errorf("Ctrl-F in wrap mode: viewportTop still 0; should advance by one screenful of display rows")
	}
	maxTop := ti.maxViewportTopWrap(ti.wrapContentWidth(), ti.contentHeight())
	if ti.viewportTop > maxTop {
		t.Errorf("Ctrl-F overshot wrap-aware maxTop: viewportTop=%d, maxTop=%d", ti.viewportTop, maxTop)
	}
	postDispatchTop := ti.viewportTop
	ti.ensureCursorVisibleWrap(ti.wrapContentWidth())
	if ti.viewportTop != postDispatchTop {
		t.Errorf("Ctrl-F: ensureCursorVisibleWrap re-overshot: before=%d after=%d", postDispatchTop, ti.viewportTop)
	}
}

func TestPageUp_WrapMode_RetreatsByFullScreenDisplayRows(t *testing.T) {
	long := strings.Repeat("abcdefghij ", 12)
	lines := []string{long, long, long, long, long, long}
	ti := makeWrapScrollTUI(t, lines, 20, 10)

	ti.viewportTop = 5
	ti.cursorLine = 5

	ti.handleCommand(input.Command{Type: input.CommandMotion, Motion: motion.MotionPageUp})

	if ti.viewportTop >= 5 {
		t.Errorf("Ctrl-B in wrap mode: viewportTop=%d, expected to decrease from 5", ti.viewportTop)
	}
	postDispatchTop := ti.viewportTop
	ti.ensureCursorVisibleWrap(ti.wrapContentWidth())
	if ti.viewportTop != postDispatchTop {
		t.Errorf("Ctrl-B: ensureCursorVisibleWrap pulled viewportTop back: before=%d after=%d", postDispatchTop, ti.viewportTop)
	}
}

func TestHalfPageDown_WrapMode_ClampsCursorColToShorterVisibleLine(t *testing.T) {
	// Goal: motion handler picks a LONG target line (so its cursorCol clamp
	// keeps cursorCol large), and our wrap-aware helper then pulls cursor
	// BACK onto a SHORTER visible line. The helper must re-clamp cursorCol.
	//
	// Setup: 6 lines = long, x, x, x, x, long. ch=10, width=20. Halfpage
	// down moves cursor by 5 source lines → lands on the trailing long
	// (index 5). But viewportTop only advances a few display rows because
	// the FIRST long wraps to several rows. The wrap-aware visible window
	// from the new viewportTop excludes index 5 (out of view), so the
	// helper clamps cursor down to a short line (~index 4). Without the
	// col clamp, cursorCol would remain at the long line's value (e.g. 50),
	// past the short line's EOL (rune count 1).
	long := strings.Repeat("abcdefghij ", 12) // 132 chars, wraps to many rows
	short := "x"
	// First line long (so viewportTop moves only ~1 source line per Ctrl-D),
	// trailing long line as the motion handler's target so cursorCol stays
	// large, intervening short lines so the wrap-clamp lands on a short line.
	lines := []string{long, short, short, short, short, long}
	ti := makeWrapScrollTUI(t, lines, 20, 10)

	ti.viewportTop = 0
	ti.cursorLine = 0
	ti.cursorCol = 50 // valid for `long`, invalid for `short`

	ti.handleCommand(input.Command{Type: input.CommandMotion, Motion: motion.MotionHalfPageDown})

	maxCol := ti.doc.LineRuneCount(ti.cursorLine)
	if ti.cursorCol > maxCol {
		t.Errorf("after Ctrl-D wrap-clamp to line %d (rune-len=%d), cursorCol=%d (must be ≤ %d)",
			ti.cursorLine, maxCol, ti.cursorCol, maxCol)
	}
}

func TestZb_WrapMode_PlacesCursorLineAtVisibleBottom(t *testing.T) {
	// zb (MotionViewportBottom): cursor line should end up as the last
	// FULLY-visible source line. With wrap on, motion handler computes
	// newTop = cursorLine - ch + 1 (source-line arithmetic) and clamps to
	// lineCount - ch. Both are wrong for wrap mode.
	long := strings.Repeat("abcdefghij ", 12)
	lines := []string{long, long, long, long, long, long}
	ti := makeWrapScrollTUI(t, lines, 20, 10)

	ti.viewportTop = 0
	ti.cursorLine = 3 // somewhere in the middle

	ti.handleCommand(input.Command{Type: input.CommandMotion, Motion: motion.MotionViewportBottom})

	// After zb, the cursorLine should be the last fully-visible line of
	// the new viewport.
	lastVis := ti.lastVisibleLineWrap(ti.wrapContentWidth())
	if lastVis != ti.cursorLine {
		t.Errorf("zb wrap mode: cursorLine=%d should equal lastVisibleLineWrap=%d (cursor on visible-bottom row)",
			ti.cursorLine, lastVis)
	}
}

func TestZb_WrapMode_MixedHeight_CursorVisibleArchitecturalLimit(t *testing.T) {
	// Architectural-limit doc test (source-line viewportTop). Mixed setup:
	// the line BEFORE target is very tall; target itself is short. The
	// helper cannot place target at the absolute visible bottom (would
	// require intra-line offset). It DOES guarantee:
	//   * cursor is in the visible window;
	//   * viewportTop is the closest source-line approximation.
	long := strings.Repeat("abcdefghij ", 12) // ~7 wrap rows on 20-wide
	short := "x"
	lines := []string{short, long, short, short}
	ti := makeWrapScrollTUI(t, lines, 20, 10)

	ti.viewportTop = 0
	ti.cursorLine = 2 // short, preceded by tall

	ti.handleCommand(input.Command{Type: input.CommandMotion, Motion: motion.MotionViewportBottom})

	if ti.cursorLine < ti.viewportTop {
		t.Errorf("zb mixed-height: cursor (%d) above viewportTop (%d)", ti.cursorLine, ti.viewportTop)
	}
	lastVis := ti.lastVisibleLineWrap(ti.wrapContentWidth())
	if ti.cursorLine > lastVis {
		t.Errorf("zb mixed-height: cursor (%d) past lastVisibleLineWrap (%d)", ti.cursorLine, lastVis)
	}
}

func TestZt_WrapMode_PlacesCursorLineAtViewportTop(t *testing.T) {
	// zt (MotionViewportTop): cursor line should equal viewportTop. The
	// motion handler clamps to maxTop = lineCount - ch which can prevent
	// putting cursor at top when near end of doc (wrong for wrap mode where
	// the wrap-aware max is different).
	long := strings.Repeat("abcdefghij ", 12)
	lines := []string{long, long, long, long, long, long}
	ti := makeWrapScrollTUI(t, lines, 20, 10)

	ti.viewportTop = 0
	ti.cursorLine = 2

	ti.handleCommand(input.Command{Type: input.CommandMotion, Motion: motion.MotionViewportTop})

	if ti.cursorLine != ti.viewportTop {
		t.Errorf("zt wrap mode: cursorLine=%d should equal viewportTop=%d", ti.cursorLine, ti.viewportTop)
	}
}

func TestL_WrapMode_CursorLandsOnLastVisibleSourceLine(t *testing.T) {
	// L (MotionScreenBottom) should put cursor on the last fully-visible
	// source line. Motion handler uses viewportTop+height-1 which in wrap
	// mode overshoots into invisible source lines.
	long := strings.Repeat("abcdefghij ", 12)
	lines := []string{long, long, long, long, long, long}
	ti := makeWrapScrollTUI(t, lines, 20, 10)

	ti.viewportTop = 0
	ti.cursorLine = 0

	ti.handleCommand(input.Command{Type: input.CommandMotion, Motion: motion.MotionScreenBottom})

	lastVis := ti.lastVisibleLineWrap(ti.wrapContentWidth())
	if ti.cursorLine != lastVis {
		t.Errorf("L wrap mode: cursorLine=%d, want lastVisibleLineWrap=%d", ti.cursorLine, lastVis)
	}
}

func TestM_WrapMode_CursorLandsOnSourceLineAtMiddleDisplayRow(t *testing.T) {
	// M (MotionScreenMiddle) should put cursor on the source line at
	// ~middle display row. Motion handler uses viewportTop+height/2 which
	// is wrong in wrap mode.
	long := strings.Repeat("abcdefghij ", 12)
	lines := []string{long, long, long, long, long, long}
	ti := makeWrapScrollTUI(t, lines, 20, 10)

	ti.viewportTop = 0
	ti.cursorLine = 0

	ti.handleCommand(input.Command{Type: input.CommandMotion, Motion: motion.MotionScreenMiddle})

	want := ti.middleVisibleLineWrap(ti.wrapContentWidth())
	if ti.cursorLine != want {
		t.Errorf("M wrap mode: cursorLine=%d, want middleVisibleLineWrap=%d", ti.cursorLine, want)
	}
}

func TestCountedL_WrapMode_CountsFromWrapAwareBottom(t *testing.T) {
	// L 2 should land on the 2nd source line FROM the wrap-aware visible
	// bottom (1-indexed). With 6 long lines wrapping past ch, lastVis is
	// some small index N near viewportTop; L 2 → cursor = N - 1.
	long := strings.Repeat("abcdefghij ", 12)
	lines := []string{long, long, long, long, long, long}
	ti := makeWrapScrollTUI(t, lines, 20, 10)

	ti.viewportTop = 0
	ti.cursorLine = 0

	ti.handleCommand(input.Command{Type: input.CommandMotion, Motion: motion.MotionScreenBottom, Count: 2})

	lastVis := ti.lastVisibleLineWrap(ti.wrapContentWidth())
	want := lastVis - 1
	if want < ti.viewportTop {
		want = ti.viewportTop
	}
	if ti.cursorLine != want {
		t.Errorf("L 2 wrap mode: cursorLine=%d, want %d (lastVis=%d - count(2) + 1, clamped to viewportTop)",
			ti.cursorLine, want, lastVis)
	}
}

func TestL_WrapMode_CursorColAtFirstNonBlankOfOverriddenLine(t *testing.T) {
	// L should set cursorCol to the first non-blank rune of THE LINE WE
	// LAND ON. To prove the override recomputes first-nonblank for the
	// override-chosen line (not the motion handler's target line), pick
	// a doc where:
	//   - Motion handler targets the LAST line (long, no indent, col=0).
	//   - Wrap override clamps cursor to a MIDDLE line that IS indented.
	// The cursor must land on the indented line at its first non-blank.
	long := strings.Repeat("abcdefghij ", 12) // ~7 wrap rows on 20 cols
	indented := "  hello world"               // first non-blank at rune col 2
	lines := []string{long, indented, long, long}
	ti := makeWrapScrollTUI(t, lines, 20, 10)

	ti.viewportTop = 0
	ti.cursorLine = 0

	ti.handleCommand(input.Command{Type: input.CommandMotion, Motion: motion.MotionScreenBottom})

	lastVis := ti.lastVisibleLineWrap(ti.wrapContentWidth())
	if ti.cursorLine != lastVis {
		t.Fatalf("L wrap mode: cursorLine=%d, want lastVisibleLineWrap=%d", ti.cursorLine, lastVis)
	}
	// Whatever line wrap-clamping picked, cursorCol must be first non-blank
	// of THAT line, not the motion handler's target (line 3, col 0).
	wantCol := firstNonBlankRuneCol(ti.doc.Line(ti.cursorLine))
	if ti.cursorCol != wantCol {
		t.Errorf("L wrap mode: cursorCol=%d, want first-non-blank of line %d (%q) = %d",
			ti.cursorCol, ti.cursorLine, ti.doc.Line(ti.cursorLine), wantCol)
	}
}

func TestH_WrapMode_CursorClampedToLastVisible(t *testing.T) {
	// H N (MotionScreenTop with count) should jump N source lines from top
	// but never past the wrap-aware visible bottom.
	long := strings.Repeat("abcdefghij ", 12)
	lines := []string{long, long, long, long, long, long}
	ti := makeWrapScrollTUI(t, lines, 20, 10)

	ti.viewportTop = 0
	ti.cursorLine = 0

	// H 5: motion handler says viewportTop + 5 - 1 = 4. Wrap-aware visible
	// bottom is much lower (only ~1-2 lines visible at ch=10 with ~7-row
	// wrap). Override must compute min(viewportTop+5-1, lastVisibleLineWrap).
	ti.handleCommand(input.Command{Type: input.CommandMotion, Motion: motion.MotionScreenTop, Count: 5})

	lastVis := ti.lastVisibleLineWrap(ti.wrapContentWidth())
	want := ti.viewportTop + 5 - 1
	if want > lastVis {
		want = lastVis
	}
	if want < ti.viewportTop {
		want = ti.viewportTop
	}
	if ti.cursorLine != want {
		t.Errorf("H 5 wrap mode: cursorLine=%d, want min(viewportTop+count-1, lastVis)=%d (lastVis=%d)",
			ti.cursorLine, want, lastVis)
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
