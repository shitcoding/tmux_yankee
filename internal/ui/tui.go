package ui

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/shitcoding/tmux_yankee/internal/config"
	"github.com/shitcoding/tmux_yankee/internal/input"
	"github.com/shitcoding/tmux_yankee/internal/linenums"
	vmode "github.com/shitcoding/tmux_yankee/internal/mode"
	"github.com/shitcoding/tmux_yankee/internal/motion"
	"github.com/shitcoding/tmux_yankee/internal/selection"
	"github.com/shitcoding/tmux_yankee/internal/theme"
	"github.com/shitcoding/tmux_yankee/internal/tmux"
	"golang.org/x/term"
)

// inputEvent represents an async stdin read result
type inputEvent struct {
	data []byte
	err  error
}

// tmuxClient is an interface for tmux operations (for testability)
type tmuxClient interface {
	SetBuffer(text string) error
}

// TUI represents the terminal UI
type TUI struct {
	cfg           config.Settings
	paneID        string
	doc           *Document // Document with color preservation
	lineNumMode   string    // Line number mode (absolute/relative/hybrid)
	formatter     *linenums.Formatter
	palette       theme.Palette
	modeMachine   *vmode.Machine
	client        tmuxClient
	clipboardFunc func(text string) error // injectable for testing; nil uses copyToClipboard
	parser        *input.Parser
	motionHandler motion.Handler
	cursorLine    int
	cursorCol     int
	viewportTop   int
	hOffset       int // horizontal scroll offset (0-based column index of leftmost visible char)
	width         int
	height        int
	oldState      *term.State
}

// NewTUI creates a new TUI instance from resolved settings.
func NewTUI(cfg config.Settings, content []string) *TUI {
	// Parse mode string
	lineNumMode, err := linenums.ModeFromString(string(cfg.Mode))
	if err != nil {
		lineNumMode = linenums.ModeHybrid
	}

	// Create document from raw content (with ANSI codes)
	doc := NewDocument(content)

	// Calculate max line number
	maxLine := doc.LineCount()
	if maxLine == 0 {
		maxLine = 1
	}

	// Set initial cursor position based on StartPosition setting
	var initialCursorLine int
	switch cfg.StartPosition {
	case config.StartPositionTop:
		initialCursorLine = 0
	case config.StartPositionMiddle:
		initialCursorLine = (maxLine - 1) / 2
	default: // StartPositionBottom or unset
		initialCursorLine = maxLine - 1
	}
	if initialCursorLine < 0 {
		initialCursorLine = 0
	}

	// Use configured toggle key; default to 'L' if zero
	toggleKey := cfg.ToggleModeKey
	if toggleKey == 0 {
		toggleKey = 'L'
	}

	return &TUI{
		cfg:           cfg,
		paneID:        cfg.PaneID,
		doc:           doc,
		lineNumMode:   string(cfg.Mode),
		formatter:     linenums.NewFormatterWithPalette(lineNumMode, maxLine, cfg.Palette.LineNum),
		palette:       cfg.Palette,
		modeMachine:   vmode.NewMachine(),
		client:        tmux.NewClient(),
		parser:        input.NewParserWithToggleKey(toggleKey),
		motionHandler: motion.NewVimHandler(),
		cursorLine:    initialCursorLine,
		cursorCol:     0,
		viewportTop:   0,
	}
}

// Run starts the TUI event loop
func (t *TUI) Run() error {
	// Initialize terminal
	if err := t.initTerminal(); err != nil {
		return fmt.Errorf("terminal init failed: %w", err)
	}
	defer t.restoreTerminal()

	// Track resize events from tmux pane/window changes (zoom/unzoom, client resize)
	resizeCh := make(chan os.Signal, 1)
	signal.Notify(resizeCh, syscall.SIGWINCH)
	defer signal.Stop(resizeCh)

	// Set initial terminal size
	if err := t.updateSize(); err != nil {
		return fmt.Errorf("get terminal size failed: %w", err)
	}

	// Center cursor in viewport on startup (like vim's zz).
	// clampViewportAndCursor only ensures the cursor is barely visible
	// (at the edge); we want the initial view centered on the cursor.
	if t.height > 0 && t.doc.LineCount() > 0 {
		if t.cfg.WrapMode == config.WrapModeWrap {
			t.centerViewportWrap(t.wrapContentWidth())
		} else {
			t.viewportTop = t.cursorLine - t.height/2
			if t.viewportTop < 0 {
				t.viewportTop = 0
			}
		}
	}

	// Clear screen and hide cursor
	fmt.Print("\x1b[2J\x1b[?25l")

	// Enable SGR mouse wheel reporting
	fmt.Print("\x1b[?1000h\x1b[?1006h")

	// Initial render
	t.render()

	// Read stdin in a goroutine so the main loop can select on both input and SIGWINCH
	const maxKeySequenceBytes = 64 // SGR mouse sequences can be up to ~20 bytes
	inputCh := make(chan inputEvent)
	go func() {
		buf := make([]byte, maxKeySequenceBytes)
		for {
			n, err := os.Stdin.Read(buf)
			if err != nil {
				inputCh <- inputEvent{err: err}
				return
			}
			if n == 0 {
				continue
			}

			data := make([]byte, n)
			copy(data, buf[:n])
			inputCh <- inputEvent{data: data}
		}
	}()

	for {
		select {
		case event := <-inputCh:
			if event.err != nil {
				if event.err == io.EOF {
					return nil
				}
				return fmt.Errorf("read failed: %w", event.err)
			}

			// Process each byte individually to handle rapid key presses
			needsRender := false
			for i := 0; i < len(event.data); i++ {
				key := event.data[i : i+1]

				// Handle input
				quit := t.handleInput(key)
				if quit {
					return nil
				}

				needsRender = true
			}

			// Flush any pending ESC that wasn't followed by '[' in this read.
			// Standalone ESC arrives as a single byte; mouse sequences (ESC [ <)
			// arrive as a burst. Flushing here makes ESC responsive without
			// needing a second keypress.
			if flushCmd := t.parser.Flush(); flushCmd.Type != input.CommandNone {
				if t.handleCommand(flushCmd) {
					return nil
				}
				needsRender = true
			}

			// Re-render once after processing all bytes to reduce flicker
			if needsRender {
				t.render()
			}

		case <-resizeCh:
			// Re-read terminal size on SIGWINCH and keep cursor/viewport in bounds
			if err := t.updateSize(); err != nil {
				return fmt.Errorf("get terminal size failed: %w", err)
			}
			t.render()
		}
	}
}

// initTerminal switches to raw mode
func (t *TUI) initTerminal() error {
	oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		return err
	}
	t.oldState = oldState
	return nil
}

// restoreTerminal restores terminal state
func (t *TUI) restoreTerminal() {
	if t.oldState != nil {
		// Disable mouse reporting
		fmt.Print("\x1b[?1006l\x1b[?1000l")
		// Show cursor and clear screen
		fmt.Print("\x1b[?25h\x1b[2J\x1b[H")
		term.Restore(int(os.Stdin.Fd()), t.oldState)
	}
}

// updateSize refreshes terminal dimensions and clamps viewport/cursor
func (t *TUI) updateSize() error {
	width, height, err := term.GetSize(int(os.Stdin.Fd()))
	if err != nil {
		return err
	}

	t.width = width
	t.height = height
	t.clampViewportAndCursor()
	return nil
}

// clampViewportAndCursor keeps cursor/viewport valid after resize
func (t *TUI) clampViewportAndCursor() {
	lineCount := t.doc.LineCount()
	if lineCount <= 0 {
		t.cursorLine = 0
		t.cursorCol = 0
		t.viewportTop = 0
		return
	}

	// Clamp cursor line to available content
	if t.cursorLine < 0 {
		t.cursorLine = 0
	}
	if t.cursorLine >= lineCount {
		t.cursorLine = lineCount - 1
	}

	// Clamp cursor column to current line width
	maxCol := t.doc.LineRuneCount(t.cursorLine)
	if t.cursorCol < 0 {
		t.cursorCol = 0
	}
	if t.cursorCol > maxCol {
		t.cursorCol = maxCol
	}

	if t.height <= 0 {
		t.height = 0
		t.viewportTop = 0
		return
	}

	if t.cfg.WrapMode == config.WrapModeWrap {
		// Skip maxTop clamping (assumes 1 line = 1 display row).
		// ensureCursorVisibleWrap handles wrap-aware viewport during render.
		// Only enforce basic bounds here.
		if t.viewportTop < 0 {
			t.viewportTop = 0
		}
		if t.cursorLine < t.viewportTop {
			t.viewportTop = t.cursorLine
		}
		return
	}

	// Keep viewport in valid range
	maxTop := lineCount - t.height
	if maxTop < 0 {
		maxTop = 0
	}
	if t.viewportTop < 0 {
		t.viewportTop = 0
	}
	if t.viewportTop > maxTop {
		t.viewportTop = maxTop
	}

	// Keep cursor visible after resize
	if t.cursorLine < t.viewportTop {
		t.viewportTop = t.cursorLine
	}
	if t.cursorLine >= t.viewportTop+t.height {
		t.viewportTop = t.cursorLine - t.height + 1
	}
	if t.viewportTop < 0 {
		t.viewportTop = 0
	}
	if t.viewportTop > maxTop {
		t.viewportTop = maxTop
	}
}

// ensureCursorVisibleH adjusts hOffset so the cursor column is within the
// visible horizontal viewport. Called after every cursor movement.
func (t *TUI) ensureCursorVisibleH(contentWidth int) {
	if contentWidth <= 0 {
		t.hOffset = 0
		return
	}
	if t.cursorCol < t.hOffset {
		t.hOffset = t.cursorCol
	}
	if t.cursorCol >= t.hOffset+contentWidth {
		t.hOffset = t.cursorCol - contentWidth + 1
	}
	if t.hOffset < 0 {
		t.hOffset = 0
	}
}

// wrapContentWidth returns the number of content columns available after the
// line-number gutter. Used by wrap-mode viewport helpers.
func (t *TUI) wrapContentWidth() int {
	sampleGutter := t.formatter.RenderGutter(1, 1)
	gutterWidth := len(stripANSI(sampleGutter))
	cw := t.width - gutterWidth
	if cw < 1 {
		cw = 1
	}
	return cw
}

// centerViewportWrap sets viewportTop so the cursor line starts approximately
// at the vertical middle of the screen, accounting for wrapped display rows.
// Used once on startup to give a centered initial view.
func (t *TUI) centerViewportWrap(contentWidth int) {
	if t.height <= 0 || contentWidth <= 0 {
		return
	}
	targetRowsAbove := t.height / 2
	rowsAbove := 0
	vt := t.cursorLine
	for vt > 0 {
		chunks := wordWrapChunks(t.doc.Cells(vt-1), contentWidth)
		if rowsAbove+len(chunks) > targetRowsAbove {
			break
		}
		rowsAbove += len(chunks)
		vt--
	}
	t.viewportTop = vt
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

	// Count display rows from viewportTop to the cursor line's first row.
	// If that exceeds t.height, advance viewportTop one content line at a time.
	for {
		rows := 0
		cursorFirstRow := 0
		lineCount := t.doc.LineCount()
		for i := t.viewportTop; i < lineCount; i++ {
			chunks := wordWrapChunks(t.doc.Cells(i), contentWidth)
			if i == t.cursorLine {
				cursorFirstRow = rows
			}
			rows += len(chunks)
			// Early exit: we've counted past the cursor and past the screen.
			if i > t.cursorLine && rows >= t.height {
				break
			}
		}

		if cursorFirstRow < t.height {
			break // cursor is visible
		}
		// Advance viewportTop by one line and retry.
		t.viewportTop++
		if t.viewportTop > t.cursorLine {
			t.viewportTop = t.cursorLine
			break
		}
	}
}

// handleInput processes keyboard input
// Returns true if should quit
func (t *TUI) handleInput(key []byte) bool {
	// Feed one byte at a time into the stateful parser, which accumulates
	// multi-byte sequences (SGR mouse, escape sequences) internally.
	if len(key) != 1 {
		return false
	}

	cmd := t.parser.Parse(key[0])
	if cmd.Type == input.CommandNone {
		// In visual mode, 'y' should yank the selection immediately
		// rather than waiting for a second key (like 'yy' for yank-line).
		if t.parser.HasPendingYPrefix() && t.modeMachine.Mode() != vmode.Normal {
			t.parser.ClearPending()
			return t.yank()
		}
		return false
	}
	return t.handleCommand(cmd)
}

// handleCommand executes a parsed command. Returns true if the TUI should exit.
func (t *TUI) handleCommand(cmd input.Command) bool {
	switch cmd.Type {
	case input.CommandNone:
		return false

	case input.CommandMotion:
		// Execute motion via motion handler
		cursor := motion.Cursor{Line: t.cursorLine, Col: t.cursorCol}
		viewport := motion.Viewport{Top: t.viewportTop, Height: t.height}
		result := t.motionHandler.Apply(t, cursor, viewport, cmd.Motion, cmd.Count)

		// Update cursor and viewport
		t.cursorLine = result.Cursor.Line
		t.cursorCol = result.Cursor.Col
		if t.cfg.WrapMode == config.WrapModeWrap {
			// In wrap mode, only accept viewport from motions that intentionally
			// reposition it. For cursor-only motions, keep the current viewport
			// and let ensureCursorVisibleWrap handle it during render (the motion
			// handler's adjustViewport assumes 1 line = 1 display row).
			switch cmd.Motion {
			case motion.MotionViewportCenter:
				// zz: wrap-aware centering (the motion handler assumes 1 line = 1 row)
				t.centerViewportWrap(t.wrapContentWidth())
			case motion.MotionHalfPageUp, motion.MotionHalfPageDown,
				motion.MotionViewportTop, motion.MotionViewportBottom:
				t.viewportTop = result.Viewport.Top
			}
		} else {
			t.viewportTop = result.Viewport.Top
		}

		// Notify mode machine of cursor movement
		pos := selection.Pos{Line: t.cursorLine, Col: t.cursorCol}
		t.modeMachine.OnCursorMoved(pos)

	case input.CommandVisual:
		// Toggle character-wise visual mode
		pos := selection.Pos{Line: t.cursorLine, Col: t.cursorCol}
		t.modeMachine.Handle(vmode.EventToggleVisualChar, pos)

	case input.CommandVisualLine:
		// Toggle line-wise visual mode
		pos := selection.Pos{Line: t.cursorLine, Col: t.cursorCol}
		t.modeMachine.Handle(vmode.EventToggleVisualLine, pos)

	case input.CommandEscape:
		// Exit visual mode back to normal
		pos := selection.Pos{Line: t.cursorLine, Col: t.cursorCol}
		t.modeMachine.Handle(vmode.EventEscape, pos)

	case input.CommandYank:
		return t.yank()

	case input.CommandYankLine:
		return t.yankLine()

	case input.CommandToggleLineMode:
		t.toggleMode()

	case input.CommandQuit:
		return true

	case input.CommandMouseScroll:
		return t.handleMouseScroll(cmd.ScrollDirection)

	case input.CommandCharSearch:
		if cs, ok := t.motionHandler.(motion.CharSearcher); ok {
			cursor := motion.Cursor{Line: t.cursorLine, Col: t.cursorCol}
			var newCursor motion.Cursor
			switch cmd.SearchKind {
			case input.SearchRepeat:
				newCursor = cs.RepeatCharSearch(t, cursor, cmd.Count)
			case input.SearchRepeatReverse:
				newCursor = cs.RepeatCharSearchReverse(t, cursor, cmd.Count)
			default:
				dir := searchKindToDirection(cmd.SearchKind)
				newCursor = cs.ApplyCharSearch(t, cursor, dir, cmd.SearchChar, cmd.Count)
			}
			t.cursorLine = newCursor.Line
			t.cursorCol = newCursor.Col
			pos := selection.Pos{Line: t.cursorLine, Col: t.cursorCol}
			t.modeMachine.OnCursorMoved(pos)
		}
	}

	return false
}

// searchKindToDirection converts input.SearchKind to motion.CharSearchDirection.
func searchKindToDirection(sk input.SearchKind) motion.CharSearchDirection {
	switch sk {
	case input.SearchFindForward:
		return motion.CharSearchFindForward
	case input.SearchTillForward:
		return motion.CharSearchTillForward
	case input.SearchFindBackward:
		return motion.CharSearchFindBackward
	case input.SearchTillBackward:
		return motion.CharSearchTillBackward
	default:
		return motion.CharSearchFindForward
	}
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

	// Viewport-scroll path: content is taller than the terminal window.
	if t.height > 0 && lineCount > t.height {
		maxViewportTop := lastLine - t.height + 1
		switch dir {
		case input.ScrollUp:
			t.viewportTop -= scrollStep
			if t.viewportTop < 0 {
				t.viewportTop = 0
			}
			// Cursor must stay within the (now higher) viewport window.
			if newBottom := t.viewportTop + t.height - 1; t.cursorLine > newBottom {
				t.cursorLine = newBottom
			}
			t.clampViewportAndCursor()
			return false

		case input.ScrollDown:
			if t.viewportTop >= maxViewportTop {
				return true // viewport already at bottom of content: overscroll → exit
			}
			t.viewportTop += scrollStep
			if t.viewportTop > maxViewportTop {
				t.viewportTop = maxViewportTop
			}
			// Cursor must stay within the (now lower) viewport window.
			if t.cursorLine < t.viewportTop {
				t.cursorLine = t.viewportTop
			}
			t.clampViewportAndCursor()
			return false
		}
	}

	// Cursor-only fallback: content fits in viewport or height not yet set.
	switch dir {
	case input.ScrollUp:
		if t.cursorLine > 0 {
			t.cursorLine--
			t.clampViewportAndCursor()
		}
		return false
	case input.ScrollDown:
		if t.cursorLine >= lastLine {
			return true
		}
		t.cursorLine++
		t.clampViewportAndCursor()
		return false
	}
	return false
}

// CursorLine returns the current cursor line (exported for testing).
func (t *TUI) CursorLine() int { return t.cursorLine }

// SetCursorLine sets the cursor line directly (exported for testing).
func (t *TUI) SetCursorLine(line int) { t.cursorLine = line }

// ViewportTop returns the current viewport top line index (exported for testing).
func (t *TUI) ViewportTop() int { return t.viewportTop }

// SetViewportTop sets the viewport top directly (exported for testing).
func (t *TUI) SetViewportTop(top int) { t.viewportTop = top }

// SetHeight sets the terminal height (exported for testing without a real terminal).
func (t *TUI) SetHeight(h int) {
	t.height = h
	t.clampViewportAndCursor()
}

// HandleCommand processes a Command directly (exported for testing).
func (t *TUI) HandleCommand(cmd input.Command) bool { return t.handleCommand(cmd) }

// toggleMode cycles through line number modes
func (t *TUI) toggleMode() {
	t.formatter.ToggleMode()
	// Update mode string for consistency
	switch t.formatter.CurrentMode() {
	case linenums.ModeAbsolute:
		t.lineNumMode = "absolute"
	case linenums.ModeRelative:
		t.lineNumMode = "relative"
	case linenums.ModeHybrid:
		t.lineNumMode = "hybrid"
	}
}

// lineSelection computes cursor column and selection range for a given line
// relative to the current mode machine region.
func (t *TUI) lineSelection(lineIdx int, region selection.Region) (cursorCol, selStart, selEnd int) {
	cursorCol = -1
	selStart = -1
	selEnd = -1

	if lineIdx == t.cursorLine {
		cursorCol = t.cursorCol
	}

	if region.Kind == selection.KindNone {
		return
	}

	start, end := region.Start, region.End
	if start.Line > end.Line || (start.Line == end.Line && start.Col > end.Col) {
		start, end = end, start
	}

	if region.Kind == selection.KindChar {
		if lineIdx == start.Line && lineIdx == end.Line {
			selStart = start.Col
			selEnd = end.Col
		} else if lineIdx == start.Line {
			selStart = start.Col
			selEnd = 9999
		} else if lineIdx == end.Line {
			selStart = 0
			selEnd = end.Col
		} else if lineIdx > start.Line && lineIdx < end.Line {
			selStart = 0
			selEnd = 9999
		}
	} else if region.Kind == selection.KindLine {
		if lineIdx >= start.Line && lineIdx <= end.Line {
			selStart = 0
			selEnd = 9999
		}
	}
	return
}

// bgEscape returns the ANSI escape sequence to set the background color from
// a cell's Style. Returns empty string if the style has default background.
func bgEscape(s Style) string {
	if s.BgColor == -1 {
		return fmt.Sprintf("\x1b[48;2;%d;%d;%dm", s.BgR, s.BgG, s.BgB)
	}
	if s.BgColor >= 256 {
		return fmt.Sprintf("\x1b[48;5;%dm", s.BgColor-256)
	}
	if s.BgColor > 0 {
		return fmt.Sprintf("\x1b[%dm", s.BgColor)
	}
	return ""
}

// render draws the TUI
func (t *TUI) render() {
	if t.cfg.WrapMode == config.WrapModeWrap {
		t.renderWrap()
	} else {
		t.renderScroll()
	}
}

// renderScroll renders with horizontal scrolling (default mode).
func (t *TUI) renderScroll() {
	var b strings.Builder
	b.WriteString("\x1b[H")

	endLine := t.viewportTop + t.height
	if endLine > t.doc.LineCount() {
		endLine = t.doc.LineCount()
	}

	region := t.modeMachine.Region()

	sampleGutter := t.formatter.RenderGutter(1, 1)
	gutterWidth := len(stripANSI(sampleGutter))
	contentWidth := t.width - gutterWidth
	if contentWidth < 0 {
		contentWidth = 0
	}

	t.ensureCursorVisibleH(contentWidth)

	for i := t.viewportTop; i < endLine; i++ {
		gutter := t.formatter.RenderGutter(i+1, t.cursorLine+1)
		b.WriteString(gutter)

		cursorCol, selStart, selEnd := t.lineSelection(i, region)

		renderedLine := RenderCellsWithPalette(t.doc.Cells(i), cursorCol, selStart, selEnd, t.hOffset, contentWidth, t.palette)
		b.WriteString(renderedLine)
		b.WriteString("\x1b[K")

		if i < endLine-1 {
			b.WriteString("\r\n")
		}
	}

	for i := endLine - t.viewportTop; i < t.height; i++ {
		b.WriteString("\r\n\x1b[K")
	}

	fmt.Print(b.String())
}

// wrapChunk represents a contiguous slice of cells within a single display row.
// Indices are cell positions [start, end) within the parent line's cell array.
type wrapChunk struct {
	start, end int
}

// trimTrailingSpaceCells returns the effective length of cells after stripping
// trailing ASCII spaces. Used only in wrap layout: lines captured from tmux
// are padded with spaces to the original pane width, which causes unnecessary
// wrapping when the gutter reduces available content columns.
func trimTrailingSpaceCells(cells []Cell) int {
	n := len(cells)
	for n > 0 && cells[n-1].Rune == ' ' {
		n--
	}
	return n
}

// wordWrapChunks splits a line's cells into display-row chunks that break at
// word boundaries (spaces). If no space is found within a chunk, it hard-wraps
// at contentWidth as a fallback for long unbroken words.
// Trailing spaces are trimmed before chunking to prevent unnecessary wrapping
// of lines padded by tmux capture-pane.
func wordWrapChunks(cells []Cell, contentWidth int) []wrapChunk {
	n := trimTrailingSpaceCells(cells)
	if n == 0 {
		return []wrapChunk{{0, 0}}
	}
	if contentWidth <= 0 {
		return []wrapChunk{{0, n}}
	}

	var chunks []wrapChunk
	pos := 0
	for pos < n {
		end := pos + contentWidth
		if end >= n {
			// Remaining cells fit in one row.
			chunks = append(chunks, wrapChunk{pos, n})
			break
		}
		// Search backwards from the break point for a space to wrap after.
		breakAt := -1
		for j := end - 1; j > pos; j-- {
			if cells[j].Rune == ' ' {
				breakAt = j + 1 // include the space in this chunk
				break
			}
		}
		if breakAt < 0 {
			// No space found — hard wrap at contentWidth.
			breakAt = end
		}
		chunks = append(chunks, wrapChunk{pos, breakAt})
		pos = breakAt
	}
	return chunks
}

// renderWrap renders with line wrapping — long lines span multiple display rows.
// Uses word-boundary wrapping so lines break at spaces when possible.
func (t *TUI) renderWrap() {
	var b strings.Builder
	b.WriteString("\x1b[H")

	// In wrap mode, horizontal offset is always 0.
	t.hOffset = 0

	region := t.modeMachine.Region()

	sampleGutter := t.formatter.RenderGutter(1, 1)
	gutterWidth := len(stripANSI(sampleGutter))
	contentWidth := t.width - gutterWidth
	if contentWidth < 1 {
		contentWidth = 1
	}
	blankGutter := t.formatter.RenderBlankGutter()

	// Adjust viewport so cursor is on-screen (wrap-aware).
	t.ensureCursorVisibleWrap(contentWidth)

	displayRow := 0
	lineCount := t.doc.LineCount()

	for i := t.viewportTop; i < lineCount && displayRow < t.height; i++ {
		cells := t.doc.Cells(i)
		cursorCol, selStart, selEnd := t.lineSelection(i, region)

		chunks := wordWrapChunks(cells, contentWidth)

		// Render each chunk (display row) of this content line
		for ci, chunk := range chunks {
			if displayRow >= t.height {
				break
			}
			// Gutter: line number on first chunk, blank on continuation
			if ci == 0 {
				gutter := t.formatter.RenderGutter(i+1, t.cursorLine+1)
				b.WriteString(gutter)
			} else {
				b.WriteString(blankGutter)
			}

			chunkWidth := chunk.end - chunk.start
			// For the last chunk of the cursor line, use full contentWidth
			// so the cursor can be rendered past the end of text.
			maxWidth := chunkWidth
			if i == t.cursorLine && ci == len(chunks)-1 {
				maxWidth = contentWidth
			}

			renderedLine := RenderCellsWithPalette(cells, cursorCol, selStart, selEnd, chunk.start, maxWidth, t.palette)
			b.WriteString(renderedLine)
			// Extend the last cell's background through \x1b[K so that
			// full-width highlights (e.g. git diff) don't break on wrapped rows.
			if chunkWidth < contentWidth && chunk.end > chunk.start && chunk.end-1 < len(cells) {
				if bg := bgEscape(cells[chunk.end-1].Style); bg != "" {
					b.WriteString(bg)
					b.WriteString("\x1b[K\x1b[0m")
				} else {
					b.WriteString("\x1b[K")
				}
			} else {
				b.WriteString("\x1b[K")
			}

			displayRow++
			if displayRow < t.height {
				b.WriteString("\r\n")
			}
		}
	}

	// Clear remaining display rows
	for displayRow < t.height {
		if displayRow > 0 {
			b.WriteString("\r\n")
		}
		b.WriteString("\x1b[K")
		displayRow++
	}

	fmt.Print(b.String())
}

// GetMode returns the current line number mode
func (t *TUI) GetMode() string {
	return t.lineNumMode
}

// Document interface implementation for motion.Handler
// Delegates to internal Document

// LineCount returns the total number of lines in the document.
func (t *TUI) LineCount() int {
	return t.doc.LineCount()
}

// Line returns the plain text content of the line at the given index.
func (t *TUI) Line(index int) string {
	return t.doc.Line(index)
}

// LineRuneCount returns the number of runes in the line.
func (t *TUI) LineRuneCount(index int) int {
	return t.doc.LineRuneCount(index)
}

// yank extracts selected text, copies to clipboard and tmux buffer
// Returns true to quit TUI after yank
func (t *TUI) yank() bool {
	// Get current selection region from mode machine
	region := t.modeMachine.Region()

	// Only yank if there is an active selection
	if region.Kind == selection.KindNone {
		return false
	}

	// Extract plain text lines for selection extraction
	plainLines := make([]string, t.doc.LineCount())
	for i := 0; i < t.doc.LineCount(); i++ {
		plainLines[i] = t.doc.Line(i)
	}

	// Extract selected text using region-based extraction (no gutter stripping needed)
	text, err := selection.ExtractRegion(plainLines, region)
	if err != nil {
		fmt.Fprintf(os.Stderr, "yank: ExtractRegion failed: %v\n", err)
		return false
	}

	// clipboardCopy dispatches to injected clipboardFunc (for tests) or real impl
	clipboardCopy := func(s string) error {
		if t.clipboardFunc != nil {
			return t.clipboardFunc(s)
		}
		return t.copyToClipboard(s)
	}

	// Route copy operations based on CopyTarget setting
	switch t.cfg.CopyTarget {
	case config.CopyTargetTmux:
		// tmux buffer only, skip clipboard
		if err := t.client.SetBuffer(text); err != nil {
			fmt.Fprintf(os.Stderr, "yank: SetBuffer failed: %v\n", err)
		}
	case config.CopyTargetClipboard:
		// clipboard only, skip tmux buffer
		if err := clipboardCopy(text); err != nil {
			fmt.Fprintf(os.Stderr, "yank: copyToClipboard failed: %v\n", err)
		}
	default: // CopyTargetBoth or unset
		// Set tmux buffer first (reliable fallback), then clipboard
		if err := t.client.SetBuffer(text); err != nil {
			fmt.Fprintf(os.Stderr, "yank: SetBuffer failed: %v\n", err)
		}
		if err := clipboardCopy(text); err != nil {
			fmt.Fprintf(os.Stderr, "yank: copyToClipboard failed: %v\n", err)
		}
	}

	// Exit visual mode and return to Normal mode (vim behavior)
	pos := selection.Pos{Line: t.cursorLine, Col: t.cursorCol}
	t.modeMachine.Handle(vmode.EventEscape, pos)

	// ExitOnYank=true (default): exit TUI after yank
	// ExitOnYank=false: stay in TUI in Normal mode (selection already cleared above)
	if !t.cfg.ExitOnYank {
		return false
	}
	return true
}


// yankLine yanks the full content of the current cursor line (yy binding).
// Unlike yank(), it does not require an active visual selection.
func (t *TUI) yankLine() bool {
	if t.doc.LineCount() == 0 {
		return false
	}
	text := t.doc.Line(t.cursorLine)

	clipboardCopy := func(s string) error {
		if t.clipboardFunc != nil {
			return t.clipboardFunc(s)
		}
		return t.copyToClipboard(s)
	}

	switch t.cfg.CopyTarget {
	case config.CopyTargetTmux:
		if err := t.client.SetBuffer(text); err != nil {
			fmt.Fprintf(os.Stderr, "yankLine: SetBuffer failed: %v\n", err)
		}
	case config.CopyTargetClipboard:
		if err := clipboardCopy(text); err != nil {
			fmt.Fprintf(os.Stderr, "yankLine: copyToClipboard failed: %v\n", err)
		}
	default: // CopyTargetBoth or unset
		if err := t.client.SetBuffer(text); err != nil {
			fmt.Fprintf(os.Stderr, "yankLine: SetBuffer failed: %v\n", err)
		}
		if err := clipboardCopy(text); err != nil {
			fmt.Fprintf(os.Stderr, "yankLine: copyToClipboard failed: %v\n", err)
		}
	}

	if !t.cfg.ExitOnYank {
		return false
	}
	return true
}

// copyToClipboard copies text to system clipboard via copy_stdin.sh
func (t *TUI) copyToClipboard(text string) error {
	// Get the directory where the binary is located
	execPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("cannot determine executable path: %w", err)
	}
	binDir := filepath.Dir(execPath)

	// Try multiple possible locations relative to binary
	possiblePaths := []string{
		filepath.Join(binDir, "..", "scripts", "copy_stdin.sh"), // ../scripts from bin/
		"scripts/copy_stdin.sh",                                 // From project root (if run from there)
		"/usr/local/bin/copy_stdin.sh",                          // System-wide install
	}

	var scriptPath string
	for _, path := range possiblePaths {
		absPath, _ := filepath.Abs(path)
		if _, err := os.Stat(absPath); err == nil {
			scriptPath = absPath
			break
		}
	}

	if scriptPath == "" {
		return fmt.Errorf("copy_stdin.sh not found in any of: %v", possiblePaths)
	}

	cmd := exec.Command(scriptPath)
	cmd.Stdin = bytes.NewBufferString(text)
	cmd.Stderr = os.Stderr // Show errors from script

	return cmd.Run()
}
