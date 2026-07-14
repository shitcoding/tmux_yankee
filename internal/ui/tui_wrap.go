package ui

import vmode "github.com/shitcoding/tmux_yankee/internal/mode"

// Wrap-mode viewport math: helpers that translate between source lines and
// wrapped display rows. Extracted from tui.go (see internal/ui/tui_wrap_scroll_test.go).

// wrapContentWidth returns the number of content columns available after the
// line-number gutter. Used by wrap-mode viewport helpers.
func (t *TUI) wrapContentWidth() int {
	return max(t.width-t.gutterWidth(), 1)
}

// maxViewportTopWrap returns the largest viewportTop value for which the
// document's last source line is still visible in the viewport, accounting
// for wrapped display rows. Walks backward from the last line, summing each
// line's wrap chunk count, until adding the next line would overflow `rows`.
//
// Returns 0 when the entire wrapped document fits in `rows`. Returns
// `lineCount-1` (the last source line) when even the last line alone wraps
// past `rows` — viewportTop is source-line indexed so we cannot point
// inside a single overheight line. Otherwise returns the largest valid
// viewportTop — symmetric with non-wrap mode's `lineCount - ch`.
func (t *TUI) maxViewportTopWrap(contentWidth, rows int) int {
	lineCount := t.doc.LineCount()
	if lineCount <= 0 || contentWidth <= 0 || rows <= 0 {
		return 0
	}
	used := 0
	for i := lineCount - 1; i >= 0; i-- {
		chunks := t.cachedWrapChunks(i, t.doc.Cells(i), contentWidth)
		n := len(chunks)
		if n == 0 {
			n = 1
		}
		if used+n > rows {
			// Adding this line would overflow — it becomes the line just
			// above the viewport top. Clamp so we never return a value past
			// the last source-line index (architectural limit: viewportTop
			// is source-line indexed, no intra-line row offset). If the
			// last line alone wraps taller than `rows`, the largest sane
			// viewport top is `lineCount-1` so that last line is at least
			// the only visible source line.
			top := min(i+1, lineCount-1)
			return top
		}
		used += n
	}
	return 0
}

// contentNeedsScrollWrap reports whether wrap-mode content extends beyond
// the current viewport in either direction (so a wheel/scroll command
// should move the viewport rather than only the cursor).
func (t *TUI) contentNeedsScrollWrap(contentWidth, rows int) bool {
	if t.viewportTop > 0 {
		return true
	}
	return t.maxViewportTopWrap(contentWidth, rows) > 0
}

// firstNonBlankRuneCol returns the rune column of the first non-blank
// rune on `line`. Used by wrap-mode H/M/L cursor overrides so the new
// cursor column matches vim semantics (cursor lands on first non-blank
// of the target line). Returns 0 for empty or all-blank lines.
func firstNonBlankRuneCol(line string) int {
	rcol := 0
	for _, r := range line {
		if r != ' ' && r != '\t' {
			return rcol
		}
		rcol++
	}
	return 0
}

// middleVisibleLineWrap returns the source line at approximately the
// middle display row of the current viewport, accounting for wrapped
// lines. Used by M (MotionScreenMiddle) in wrap mode. Walks forward
// from viewportTop summing wrap chunks until the cumulative row count
// reaches ch/2; that source line is the return value.
func (t *TUI) middleVisibleLineWrap(contentWidth int) int {
	if contentWidth <= 0 {
		return t.viewportTop
	}
	ch := t.contentHeight()
	if ch <= 0 {
		return t.viewportTop
	}
	target := ch / 2
	used := 0
	lineCount := t.doc.LineCount()
	line := t.viewportTop
	for line < lineCount {
		chunks := t.cachedWrapChunks(line, t.doc.Cells(line), contentWidth)
		n := len(chunks)
		if n == 0 {
			n = 1
		}
		if used+n > target {
			return line
		}
		used += n
		line++
	}
	if line >= lineCount {
		line = lineCount - 1
	}
	if line < 0 {
		line = 0
	}
	return line
}

// maxViewportTopWrapToFit returns the largest viewportTop value for which
// `targetLine` fits in the viewport with as much preceding content visible
// as the row budget allows. Used by zb (cursor at bottom of viewport) in
// wrap mode. Walks backward from `targetLine` summing wrap chunks until
// adding the next line would overflow.
//
// Architectural limit: viewportTop is source-line indexed (no intra-line
// row offset). In mixed-height cases — e.g. preceding line is very tall
// but `targetLine` is short — the returned top can leave spare rows below
// `targetLine`, and the renderer will fill them with the next source line.
// So `targetLine` may not be EXACTLY the last visible line; it's the
// closest source-line-indexed approximation. The cursor is always visible
// (callers should follow up with clampCursorIntoWrapViewport).
func (t *TUI) maxViewportTopWrapToFit(targetLine, contentWidth, rows int) int {
	if contentWidth <= 0 || rows <= 0 {
		return 0
	}
	lineCount := t.doc.LineCount()
	if lineCount <= 0 {
		return 0
	}
	if targetLine < 0 {
		targetLine = 0
	}
	if targetLine >= lineCount {
		targetLine = lineCount - 1
	}
	used := 0
	for i := targetLine; i >= 0; i-- {
		chunks := t.cachedWrapChunks(i, t.doc.Cells(i), contentWidth)
		n := len(chunks)
		if n == 0 {
			n = 1
		}
		if used+n > rows {
			// Adding this line would overflow. The viewport top is one line
			// below i. Cap at targetLine so the helper never returns a top
			// past the target itself.
			top := min(i+1, targetLine)
			return top
		}
		used += n
	}
	return 0
}

// advanceViewportByDisplayRows returns a new viewportTop after scrolling
// forward by `rows` display rows in wrap mode. Walks lines forward from
// the current viewportTop, summing wrap chunks, until `rows` have been
// consumed or the wrap-aware bottom is reached. Used by Ctrl-D/F-style
// page motions where the canonical vim distance is "N display rows", not
// "N source lines".
func (t *TUI) advanceViewportByDisplayRows(rows, contentWidth int) int {
	if rows <= 0 || contentWidth <= 0 {
		return t.viewportTop
	}
	maxTop := t.maxViewportTopWrap(contentWidth, t.contentHeight())
	lineCount := t.doc.LineCount()
	used := 0
	top := t.viewportTop
	for top < lineCount-1 && top < maxTop && used < rows {
		chunks := t.cachedWrapChunks(top, t.doc.Cells(top), contentWidth)
		n := len(chunks)
		if n == 0 {
			n = 1
		}
		used += n
		top++
	}
	if top > maxTop {
		top = maxTop
	}
	return top
}

// clampCursorIntoWrapViewport pulls cursorLine into the wrap-aware visible
// window: not above viewportTop, not past lastVisibleLineWrap. Called after
// a wrap-aware viewport reposition (page motions, etc.) so the render-time
// ensureCursorVisibleWrap pass cannot tug viewportTop forward/back to keep
// a now-out-of-view cursor visible, which would undo the deliberate move.
// If cursorLine changes the cursorCol is re-clamped to the new line's
// rune count so callers don't see a stale-out-of-range column (except in
// VisualBlock mode, where post-EOL cursors are intentional for selection).
func (t *TUI) clampCursorIntoWrapViewport(contentWidth int) {
	if contentWidth <= 0 {
		return
	}
	startLine := t.cursorLine
	if t.cursorLine < t.viewportTop {
		t.cursorLine = t.viewportTop
	}
	if lastVis := t.lastVisibleLineWrap(contentWidth); t.cursorLine > lastVis {
		t.cursorLine = lastVis
	}
	if t.cursorLine != startLine && t.modeMachine != nil && t.modeMachine.Mode() != vmode.VisualBlock {
		maxCol := t.doc.LineRuneCount(t.cursorLine)
		if t.cursorCol > maxCol {
			t.cursorCol = maxCol
		}
	}
}

// retreatViewportByDisplayRows returns a new viewportTop after scrolling
// backward by `rows` display rows in wrap mode. Symmetric counterpart to
// advanceViewportByDisplayRows.
func (t *TUI) retreatViewportByDisplayRows(rows, contentWidth int) int {
	if rows <= 0 || contentWidth <= 0 {
		return t.viewportTop
	}
	used := 0
	top := t.viewportTop
	for top > 0 && used < rows {
		top--
		chunks := t.cachedWrapChunks(top, t.doc.Cells(top), contentWidth)
		n := len(chunks)
		if n == 0 {
			n = 1
		}
		used += n
	}
	return top
}

// lastVisibleLineWrap returns the index of the last logical line whose
// wrapped rows entirely fit within the current viewport (from viewportTop
// downward, summing each line's wrap chunk count up to ch). Used to clamp
// the cursor to a fully-visible source line after wrap-mode scroll-up
// operations. Note: a line whose first row IS visible but later wrapped
// rows are clipped is NOT included; use that semantic if cursor needs to
// remain on a partially-visible line.
func (t *TUI) lastVisibleLineWrap(contentWidth int) int {
	ch := t.contentHeight()
	if ch <= 0 || contentWidth <= 0 {
		return t.viewportTop
	}
	rowsUsed := 0
	lastFit := t.viewportTop
	lineCount := t.doc.LineCount()
	for i := t.viewportTop; i < lineCount; i++ {
		chunks := t.cachedWrapChunks(i, t.doc.Cells(i), contentWidth)
		rowsUsed += len(chunks)
		if rowsUsed > ch {
			break
		}
		lastFit = i
	}
	return lastFit
}

// visibleDocLineRange returns the (top, height) of document lines that have
// at least one display row visible in the current viewport. Used for flash
// matching in wrap mode where document lines may span multiple display rows.
func (t *TUI) visibleDocLineRange() (top, height int) {
	contentWidth := t.wrapContentWidth()
	if contentWidth <= 0 {
		return t.viewportTop, t.contentHeight()
	}

	contentH := t.contentHeight()
	displayRow := 0
	firstLine := -1
	lastLine := -1

	for i := t.viewportTop; i < t.doc.LineCount() && displayRow < contentH; i++ {
		cells := t.doc.Cells(i)
		chunks := t.cachedWrapChunks(i, cells, contentWidth)
		nRows := len(chunks)
		if nRows == 0 {
			nRows = 1
		}

		if firstLine == -1 {
			firstLine = i
		}
		lastLine = i
		displayRow += nRows
	}

	if firstLine == -1 {
		return t.viewportTop, t.contentHeight()
	}
	return firstLine, lastLine - firstLine + 1
}

// viewportTopForRowsAbove walks source lines backward from the cursor line,
// summing wrapped display rows, and returns the highest line whose chunks still
// fit within targetRowsAbove display rows above the cursor.
func (t *TUI) viewportTopForRowsAbove(targetRowsAbove, contentWidth int) int {
	rowsAbove := 0
	vt := t.cursorLine
	for vt > 0 {
		chunks := t.cachedWrapChunks(vt-1, t.doc.Cells(vt-1), contentWidth)
		if rowsAbove+len(chunks) > targetRowsAbove {
			break
		}
		rowsAbove += len(chunks)
		vt--
	}
	return vt
}

// centerViewportWrap sets viewportTop so the cursor line starts approximately
// at the vertical middle of the screen, accounting for wrapped display rows.
// Used once on startup to give a centered initial view.
func (t *TUI) centerViewportWrap(contentWidth int) {
	if t.height <= 0 || contentWidth <= 0 {
		return
	}
	t.viewportTop = t.viewportTopForRowsAbove(t.height/2, contentWidth)
}

// bottomViewportWrap sets viewportTop so the cursor line appears at the bottom
// of the viewport, accounting for wrapped display rows. Used on startup when
// StartPosition=bottom to match the actual pane appearance.
// visibleRows is the number of content rows (excluding status bar).
func (t *TUI) bottomViewportWrap(contentWidth, visibleRows int) {
	if visibleRows <= 0 || contentWidth <= 0 {
		return
	}
	// Count rows for the cursor line itself
	cursorChunks := t.cachedWrapChunks(t.cursorLine, t.doc.Cells(t.cursorLine), contentWidth)
	targetRowsAbove := visibleRows - len(cursorChunks)
	if targetRowsAbove <= 0 {
		t.viewportTop = t.cursorLine
		return
	}
	t.viewportTop = t.viewportTopForRowsAbove(targetRowsAbove, contentWidth)
}

// ensureCursorVisibleWrap adjusts viewportTop so the cursor line is visible
// when lines may occupy more than one display row due to word wrapping.
// Called at the start of renderWrap, after clampViewportAndCursor has done
// rough (1-line = 1-row) clamping.
func (t *TUI) ensureCursorVisibleWrap(contentWidth int) {
	if t.height <= 0 || contentWidth <= 0 {
		return
	}

	// If cursor is above viewport, snap viewport to cursor.
	if t.cursorLine < t.viewportTop {
		t.viewportTop = t.cursorLine
	}
	if t.viewportTop == t.cursorLine {
		return // cursor line IS the viewport top; always visible
	}

	// Single forward pass: count display rows from viewportTop to cursorLine.
	totalRows := 0
	for i := t.viewportTop; i < t.cursorLine; i++ {
		chunks := t.cachedWrapChunks(i, t.doc.Cells(i), contentWidth)
		totalRows += len(chunks)
	}

	// If cursor's first row fits in the viewport, nothing to do.
	if totalRows < t.height {
		return
	}

	// Cursor is below viewport. Walk viewportTop forward, subtracting each
	// line's wrapped row count, until the cursor fits. O(n) total.
	for totalRows >= t.height && t.viewportTop < t.cursorLine {
		chunks := t.cachedWrapChunks(t.viewportTop, t.doc.Cells(t.viewportTop), contentWidth)
		totalRows -= len(chunks)
		t.viewportTop++
	}
}
