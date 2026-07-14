package ui

import (
	"github.com/shitcoding/tmux_yankee/internal/config"
	"github.com/shitcoding/tmux_yankee/internal/input"
	vmode "github.com/shitcoding/tmux_yankee/internal/mode"
	"github.com/shitcoding/tmux_yankee/internal/selection"
)

// Mouse handling: coordinate mapping, click/drag/release selection, and wheel
// scroll. Extracted from tui.go (see internal/ui/tui_mouse_test.go).

// mouseToDocPos maps terminal coordinates (0-based row, col) to document position.
// Accounts for gutter width, viewport offset, horizontal scroll offset, and wrap mode.
// Returns the clamped document position and whether the click was valid.
func (t *TUI) mouseToDocPos(termRow, termCol int) (selection.Pos, bool) {
	lineCount := t.doc.LineCount()
	if lineCount == 0 {
		return selection.Pos{}, false
	}

	// Ignore clicks on the status bar row
	if t.shouldShowStatusBar() && termRow >= t.height-1 {
		return selection.Pos{}, false
	}

	// Compute gutter width (same as render path)
	gutterWidth := t.gutterWidth()

	// Content column: subtract gutter, clamp to 0 (clicking on gutter → column 0)
	contentCol := max(termCol-gutterWidth, 0)

	if t.cfg.WrapMode == config.WrapModeOn {
		return t.mouseToDocPosWrap(termRow, contentCol, gutterWidth)
	}
	return t.mouseToDocPosScroll(termRow, contentCol)
}

// mouseToDocPosScroll maps terminal position to document position in non-wrap mode.
func (t *TUI) mouseToDocPosScroll(termRow, contentCol int) (selection.Pos, bool) {
	lineCount := t.doc.LineCount()
	// Clamp to valid line range
	docLine := max(t.viewportTop+termRow, 0)
	if docLine >= lineCount {
		docLine = lineCount - 1
	}

	// Map content column to rune index, accounting for horizontal scroll and wide chars
	runeCol := t.displayColToRune(docLine, contentCol+t.hOffset)

	return selection.Pos{Line: docLine, Col: runeCol}, true
}

// mouseToDocPosWrap maps terminal position to document position in wrap mode.
func (t *TUI) mouseToDocPosWrap(termRow, contentCol, gutterWidth int) (selection.Pos, bool) {
	lineCount := t.doc.LineCount()
	contentWidth := max(t.width-gutterWidth, 1)

	// Walk display rows from viewportTop to find which logical line + chunk
	// corresponds to termRow.
	displayRow := 0
	for lineIdx := t.viewportTop; lineIdx < lineCount; lineIdx++ {
		cells := t.doc.Cells(lineIdx)
		chunks := t.cachedWrapChunks(lineIdx, cells, contentWidth)
		for chunkIdx, ch := range chunks {
			if displayRow == termRow {
				// Found the target display row — map contentCol to rune within chunk
				runeCol := ch.start + t.displayColToRuneInChunk(lineIdx, ch, contentCol)
				maxCol := max(t.doc.LineRuneCount(lineIdx)-1, 0)
				if runeCol > maxCol {
					runeCol = maxCol
				}
				return selection.Pos{Line: lineIdx, Col: runeCol}, true
			}
			displayRow++
			_ = chunkIdx
		}
		if displayRow > termRow {
			break
		}
	}

	// Click below rendered content — clamp to last line, last column
	lastLine := lineCount - 1
	maxCol := max(t.doc.LineRuneCount(lastLine)-1, 0)
	return selection.Pos{Line: lastLine, Col: maxCol}, true
}

// displayColToRune converts a 0-based display column to a rune index on a line,
// accounting for wide characters (CJK, emoji).
func (t *TUI) displayColToRune(lineIdx, displayCol int) int {
	plain := t.doc.Line(lineIdx)
	runes := []rune(plain)
	col := 0
	runeIdx := 0
	for runeIdx < len(runes) {
		w := runeDisplayWidth(runes[runeIdx])
		if w == 0 {
			w = 1
		}
		if col+w > displayCol {
			break
		}
		col += w
		runeIdx++
	}
	// Clamp to last valid position
	maxCol := max(len(runes)-1, 0)
	if runeIdx > maxCol {
		runeIdx = maxCol
	}
	return runeIdx
}

// displayColToRuneInChunk converts a display column to a rune index within a wrap chunk.
func (t *TUI) displayColToRuneInChunk(lineIdx int, ch wrapChunk, displayCol int) int {
	plain := t.doc.Line(lineIdx)
	runes := []rune(plain)
	col := 0
	offset := 0
	for i := ch.start; i < ch.end && i < len(runes); i++ {
		w := runeDisplayWidth(runes[i])
		if w == 0 {
			w = 1
		}
		if col+w > displayCol {
			break
		}
		col += w
		offset++
	}
	return offset
}

// handleMousePress handles a left mouse button press.
func (t *TUI) handleMousePress(row, col int) {
	pos, ok := t.mouseToDocPos(row, col)
	if !ok {
		return
	}

	// Cancel any existing visual selection
	if t.modeMachine.Mode() != vmode.Normal {
		t.modeMachine.Handle(vmode.EventEscape, pos)
	}

	// Move cursor to clicked position
	t.cursorLine = pos.Line
	t.cursorCol = pos.Col

	// Start tracking drag
	t.mouseDragActive = true
	t.mouseDragAnchor = pos
	t.mouseDragEnd = pos
}

// handleMouseDrag handles mouse motion with left button held.
func (t *TUI) handleMouseDrag(row, col int) {
	if !t.mouseDragActive {
		return
	}

	pos, ok := t.mouseToDocPos(row, col)
	if !ok {
		return
	}

	t.mouseDragEnd = pos

	// Update cursor to drag end for visual feedback
	t.cursorLine = pos.Line
	t.cursorCol = pos.Col

	// Create/update live visual selection during drag
	anchor := t.mouseDragAnchor
	if anchor.Line != pos.Line || anchor.Col != pos.Col {
		// Ensure we're in visual char mode with anchor as start
		if t.modeMachine.Mode() == vmode.Normal {
			t.modeMachine.Handle(vmode.EventToggleVisualChar, anchor)
		}
		t.modeMachine.OnCursorMoved(pos)
	} else {
		// Dragged back to anchor — cancel selection
		if t.modeMachine.Mode() != vmode.Normal {
			t.modeMachine.Handle(vmode.EventEscape, pos)
		}
	}
}

// handleMouseRelease handles mouse button release.
func (t *TUI) handleMouseRelease(row, col int) {
	if !t.mouseDragActive {
		return
	}
	t.mouseDragActive = false

	pos, ok := t.mouseToDocPos(row, col)
	if !ok {
		return
	}

	anchor := t.mouseDragAnchor

	// If released at the same position as press (click, no drag) — just position cursor
	if anchor.Line == pos.Line && anchor.Col == pos.Col {
		// Cancel any selection that might have been started
		if t.modeMachine.Mode() != vmode.Normal {
			t.modeMachine.Handle(vmode.EventEscape, pos)
		}
		return
	}

	// Drag completed — finalize the visual selection
	// The selection should already be active from handleMouseDrag,
	// just update the final end position.
	t.cursorLine = pos.Line
	t.cursorCol = pos.Col
	t.mouseDragEnd = pos
	t.modeMachine.OnCursorMoved(pos)
}

// scrollStep is the number of viewport lines moved per mouse wheel event.
const scrollStep = 3

// handleMouseScroll handles wheel-up/down events using viewport-based scrolling.
//
// When content exceeds the terminal height the entire viewport shifts by scrollStep
// lines so the content visibly scrolls (like tmux copy-mode / vim). The cursor is
// pinned to stay inside the new viewport window.
//
// When content fits entirely on screen (or height is unset in tests) the function
// falls back to single-line cursor movement to keep the existing test coverage valid.
//
// Overscroll-down at the bottom of content returns true (signals TUI exit).
func (t *TUI) handleMouseScroll(dir input.ScrollDirection) bool {
	lineCount := t.doc.LineCount()
	if lineCount <= 0 {
		return false
	}
	lastLine := lineCount - 1

	// In wrap mode, scroll by 1 logical line instead of scrollStep because
	// a single logical line can span many display rows when wrapped.
	step := scrollStep
	if t.cfg.WrapMode == config.WrapModeOn {
		step = 1
	}

	// Decide whether the viewport-scroll path applies. In wrap mode the
	// source-line check (`lineCount > ch`) is misleading because long lines
	// wrap into many display rows — the document can need scrolling even
	// when `lineCount <= ch`. Use a wrap-aware check that asks "is there
	// content outside the viewport in either direction?"
	ch := t.contentHeight()
	wrapOn := t.cfg.WrapMode == config.WrapModeOn
	contentWidth := 0
	maxViewportTop := 0
	scroll := false
	if ch > 0 {
		if wrapOn {
			contentWidth = t.wrapContentWidth()
			if contentWidth > 0 {
				maxViewportTop = t.maxViewportTopWrap(contentWidth, ch)
				scroll = t.contentNeedsScrollWrap(contentWidth, ch)
			}
		} else {
			if lineCount > ch {
				maxViewportTop = lastLine - ch + 1
				scroll = true
			}
		}
	}

	if scroll {
		switch dir {
		case input.ScrollUp:
			t.viewportTop -= step
			if t.viewportTop < 0 {
				t.viewportTop = 0
			}
			// Cursor must stay within the (now higher) viewport window.
			if wrapOn {
				if lastVis := t.lastVisibleLineWrap(contentWidth); t.cursorLine > lastVis {
					t.cursorLine = lastVis
				}
			} else if newBottom := t.viewportTop + ch - 1; t.cursorLine > newBottom {
				t.cursorLine = newBottom
			}
			t.clampViewportAndCursor()
			t.dirty = true
			return false

		case input.ScrollDown:
			if t.viewportTop >= maxViewportTop {
				return true // viewport already at bottom of content: overscroll → exit
			}
			t.viewportTop += step
			if t.viewportTop > maxViewportTop {
				t.viewportTop = maxViewportTop
			}
			// Cursor must stay within the (now lower) viewport window.
			if t.cursorLine < t.viewportTop {
				t.cursorLine = t.viewportTop
			}
			t.clampViewportAndCursor()
			t.dirty = true
			return false
		}
	}

	// Cursor-only fallback: content fits in viewport or height not yet set.
	switch dir {
	case input.ScrollUp:
		if t.cursorLine > 0 {
			t.cursorLine--
			t.clampViewportAndCursor()
			t.dirty = true
		}
		return false
	case input.ScrollDown:
		if t.cursorLine >= lastLine {
			return true
		}
		t.cursorLine++
		t.clampViewportAndCursor()
		t.dirty = true
		return false
	}
	return false
}
