package ui

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"unicode"
	"syscall"
	"unicode/utf8"

	"github.com/shitcoding/tmux_yankee/internal/config"
	"github.com/shitcoding/tmux_yankee/internal/flash"
	"github.com/shitcoding/tmux_yankee/internal/input"
	"github.com/shitcoding/tmux_yankee/internal/keymap"
	"github.com/shitcoding/tmux_yankee/internal/linenums"
	vmode "github.com/shitcoding/tmux_yankee/internal/mode"
	"github.com/shitcoding/tmux_yankee/internal/motion"
	"github.com/shitcoding/tmux_yankee/internal/selection"
	"github.com/shitcoding/tmux_yankee/internal/textobj"
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
	cfg            config.Settings
	paneID         string
	doc            *Document // Document with color preservation
	lineNumMode    string    // Line number mode (absolute/relative/hybrid)
	formatter      *linenums.Formatter
	palette        theme.Palette
	modeMachine    *vmode.Machine
	client         tmuxClient
	clipboardFunc  func(text string) error // injectable for testing; nil uses copyToClipboard
	parser         *input.Parser
	motionHandler  motion.Handler
	cursorLine     int
	cursorCol      int
	viewportTop    int
	hOffset        int // horizontal scroll offset (0-based column index of leftmost visible char)
	width          int
	height         int
	oldState       *term.State
	displayGoalCol int  // desired display column within wrapped row (for gj/gk)
	hasDisplayGoal bool // whether displayGoalCol is set
	dirty          bool // true when visible state changed and render is needed

	// Mouse drag state
	mouseDragActive bool          // true while left button is held and dragging
	mouseDragAnchor selection.Pos // document position where drag started
	mouseDragEnd    selection.Pos // latest drag position (updated on drag events)

	// Wrap chunk cache: avoids recomputing wordWrapChunks for the same line+width
	wrapCache      map[int][]wrapChunk // line index → chunks
	wrapCacheWidth int                 // contentWidth used to populate wrapCache

	// Cached gutter values (invalidated on resize/theme/mode change)
	cachedGutterWidth int    // cached gutter width (visual columns, strip ANSI)
	cachedBlankGutter string // cached blank gutter string

	// Search state
	searchPattern   string         // confirmed search pattern
	searchRegex     *regexp.Regexp // compiled (nil if empty/invalid)
	searchMatches   []searchMatch  // all matches, sorted by (Line, ColStart)
	searchMatchIdx  int            // current match index (-1 = none)
	searchDirection int            // 1=forward, -1=backward
	searchActive    bool           // highlights shown (persists until cleared)
	searchSavedLine int            // cursor line before search started
	searchSavedCol  int            // cursor col before search started

	// Jump back state (`` / '' — saved position before a jump)
	prevCursorLine int
	prevCursorCol  int
	hasPrevCursor  bool // false until first jump saves a position

	// Jump list (Ctrl-O/Ctrl-I)
	jumpList    [][2]int // circular list of [line, col] positions
	jumpListIdx int      // current position in jump list (== len(jumpList) means at tip)

	// Marks (m{a-z} / `{a-z})
	marks map[byte][2]int

	// Flash navigation state (nil when disabled by config)
	flash *flash.State

	// Demo mode fields
	isDemo         bool
	demoPages      [][]string
	demoPageIndex  int
	demoPageNames  []string
	demoThemeIndex int
	demoThemeName  theme.ThemeName
}

// gutterWidth returns the cached gutter visual width.
// It's recomputed when zero (after resize/theme/mode change).
func (t *TUI) gutterWidth() int {
	if t.cachedGutterWidth == 0 {
		sample := t.formatter.RenderGutter(1, 1)
		t.cachedGutterWidth = utf8.RuneCountInString(stripANSI(sample))
	}
	return t.cachedGutterWidth
}

// blankGutter returns the cached blank gutter string.
func (t *TUI) blankGutter() string {
	if t.cachedBlankGutter == "" {
		t.cachedBlankGutter = t.formatter.RenderBlankGutter()
	}
	return t.cachedBlankGutter
}

// invalidateGutterCache clears cached gutter values (call on resize/theme/mode change).
func (t *TUI) invalidateGutterCache() {
	t.cachedGutterWidth = 0
	t.cachedBlankGutter = ""
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
	wrapKey := cfg.WrapKey
	if wrapKey == 0 {
		wrapKey = 'w'
	}

	// Use normal-mode keymap from ModeKeymap for initial parser.
	// Fall back to default parser if ModeKeymap is zero-valued (test convenience).
	var parser *input.Parser
	normalKm := cfg.ModeKeymap.ForMode(false)
	if normalKm.Direct != nil {
		parser = input.NewParserWithKeymap(toggleKey, wrapKey, normalKm)
	} else {
		parser = input.NewParserWithKeys(toggleKey, wrapKey)
	}

	var flashState *flash.State
	if cfg.FlashEnabled {
		flashState = flash.New(flash.Options{
			MinChars:   cfg.FlashMinChars,
			JumpPos:    flash.JumpPos(cfg.FlashJumpPos),
			AltJumpPos: flash.JumpPos(cfg.FlashAltJumpPos),
		})
	}

	return &TUI{
		cfg:           cfg,
		paneID:        cfg.PaneID,
		doc:           doc,
		lineNumMode:   string(cfg.Mode),
		formatter:     linenums.NewFormatterWithFullPalette(lineNumMode, maxLine, cfg.Palette.Gutter, cfg.Palette.LineNum),
		palette:       cfg.Palette,
		modeMachine:   vmode.NewMachine(),
		client:        tmux.NewClient(),
		parser:        parser,
		motionHandler: motion.NewVimHandler(),
		cursorLine:    initialCursorLine,
		cursorCol:     0,
		viewportTop:   0,
		flash:         flashState,
	}
}

// NewDemoTUI creates a TUI in demo mode with the given demo pages.
func NewDemoTUI(cfg config.Settings, pages [][]string, pageNames []string) *TUI {
	if len(pages) == 0 {
		pages = [][]string{{""}}
		pageNames = []string{"Empty"}
	}

	content := pages[0]
	doc := NewDocument(content)
	maxLine := doc.LineCount()
	if maxLine == 0 {
		maxLine = 1
	}

	lineNumMode, err := linenums.ModeFromString(string(cfg.Mode))
	if err != nil {
		lineNumMode = linenums.ModeHybrid
	}

	// Start at middle of content
	initialCursorLine := (maxLine - 1) / 2
	if initialCursorLine < 0 {
		initialCursorLine = 0
	}

	toggleKey := cfg.ToggleModeKey
	if toggleKey == 0 {
		toggleKey = 'L'
	}
	wrapKey := cfg.WrapKey
	if wrapKey == 0 {
		wrapKey = 'w'
	}

	// Find starting theme index from config
	startThemeName := theme.ThemeName(cfg.ThemeName)
	themeIndex := 0
	for i, tn := range theme.ThemeOrder {
		if tn == startThemeName {
			themeIndex = i
			break
		}
	}

	// Use normal-mode keymap from ModeKeymap for initial parser.
	var parser *input.Parser
	normalKm := cfg.ModeKeymap.ForMode(false)
	if normalKm.Direct != nil {
		parser = input.NewParserWithKeymap(toggleKey, wrapKey, normalKm)
	} else {
		parser = input.NewParserWithKeys(toggleKey, wrapKey)
	}

	return &TUI{
		cfg:            cfg,
		doc:            doc,
		lineNumMode:    string(cfg.Mode),
		formatter:      linenums.NewFormatterWithFullPalette(lineNumMode, maxLine, cfg.Palette.Gutter, cfg.Palette.LineNum),
		palette:        cfg.Palette,
		modeMachine:    vmode.NewMachine(),
		parser:         parser,
		motionHandler:  motion.NewVimHandler(),
		cursorLine:     initialCursorLine,
		cursorCol:      0,
		viewportTop:    0,
		isDemo:         true,
		demoPages:      pages,
		demoPageIndex:  0,
		demoPageNames:  pageNames,
		demoThemeIndex: themeIndex,
		demoThemeName:  startThemeName,
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

	// Position viewport based on StartPosition:
	//   bottom → cursor at bottom of viewport (like zb) — matches actual pane
	//   middle → cursor centered (like zz)
	//   top    → cursor at top (viewportTop = 0, already default)
	// Account for the status bar which steals a row from content area.
	if t.height > 0 && t.doc.LineCount() > 0 {
		visibleRows := t.height
		if t.shouldShowStatusBar() {
			visibleRows--
		}
		switch t.cfg.StartPosition {
		case config.StartPositionMiddle:
			if t.cfg.WrapMode == config.WrapModeOn {
				t.centerViewportWrap(t.wrapContentWidth())
			} else {
				t.viewportTop = t.cursorLine - visibleRows/2
				if t.viewportTop < 0 {
					t.viewportTop = 0
				}
			}
		case config.StartPositionBottom:
			if t.cfg.WrapMode == config.WrapModeOn {
				t.bottomViewportWrap(t.wrapContentWidth(), visibleRows)
			} else {
				t.viewportTop = t.cursorLine - visibleRows + 1
				if t.viewportTop < 0 {
					t.viewportTop = 0
				}
			}
		// StartPositionTop: viewportTop = 0 (already default)
		}
	}

	// Clear screen and hide cursor
	fmt.Print("\x1b[2J\x1b[?25l")

	// Enable mouse reporting: basic tracking + button-event (drag) + SGR extended format
	fmt.Print("\x1b[?1000h\x1b[?1002h\x1b[?1006h")

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

			// Process each byte individually to handle rapid key presses.
			// Only render when visible state actually changes (dirty flag).
			for i := 0; i < len(event.data); i++ {
				key := event.data[i : i+1]

				// Handle input
				quit := t.handleInput(key)
				if quit {
					return nil
				}
			}

			// Flush any pending ESC that wasn't followed by '[' in this read.
			// Standalone ESC arrives as a single byte; mouse sequences (ESC [ <)
			// arrive as a burst. Flushing here makes ESC responsive without
			// needing a second keypress.
			if flushCmd := t.parser.Flush(); flushCmd.Type != input.CommandNone {
				if t.handleCommand(flushCmd) {
					return nil
				}
			}

			// Re-render once after processing all bytes, only if state changed
			if t.dirty {
				t.render()
				t.dirty = false
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

// initTerminal switches to raw mode and enables the alternate screen buffer.
// The alternate screen is critical: tmux sets #{alternate_on}=1 for the pane,
// which prevents the WheelUpPane binding from re-launching yankee while the
// TUI is starting up (before mouse reporting is enabled).
func (t *TUI) initTerminal() error {
	oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		return err
	}
	t.oldState = oldState
	// Enter alternate screen buffer immediately — before any rendering or
	// mouse setup — so tmux sees alternate_on=1 as early as possible.
	fmt.Print("\x1b[?1049h")
	return nil
}

// restoreTerminal restores terminal state
func (t *TUI) restoreTerminal() {
	if t.oldState != nil {
		// Disable mouse reporting
		fmt.Print("\x1b[?1006l\x1b[?1002l\x1b[?1000l")
		// Show cursor and clear screen
		fmt.Print("\x1b[?25h\x1b[2J\x1b[H")
		// Leave alternate screen buffer
		fmt.Print("\x1b[?1049l")
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
	t.invalidateGutterCache()
	t.clampViewportAndCursor()
	return nil
}

// contentHeight returns the number of rows available for content, accounting
// for the status bar which occupies the last row when visible.
func (t *TUI) contentHeight() int {
	if t.shouldShowStatusBar() {
		return t.height - 1
	}
	return t.height
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

	// Clamp cursor column to current line width.
	// In VisualBlock mode, allow the cursor past end-of-line so the
	// rectangular selection retains its right edge on shorter lines.
	if t.cursorCol < 0 {
		t.cursorCol = 0
	}
	if t.modeMachine.Mode() != vmode.VisualBlock {
		maxCol := t.doc.LineRuneCount(t.cursorLine)
		if t.cursorCol > maxCol {
			t.cursorCol = maxCol
		}
	}

	if t.height <= 0 {
		t.height = 0
		t.viewportTop = 0
		return
	}

	if t.cfg.WrapMode == config.WrapModeOn {
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

	// Use content height (excludes status bar) for viewport calculations.
	ch := t.contentHeight()

	// Keep viewport in valid range
	maxTop := lineCount - ch
	if maxTop < 0 {
		maxTop = 0
	}
	if t.viewportTop < 0 {
		t.viewportTop = 0
	}
	if t.viewportTop > maxTop {
		t.viewportTop = maxTop
	}

	// Keep cursor visible within content area (not behind status bar)
	if t.cursorLine < t.viewportTop {
		t.viewportTop = t.cursorLine
	}
	if t.cursorLine >= t.viewportTop+ch {
		t.viewportTop = t.cursorLine - ch + 1
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
	gutterWidth := utf8.RuneCountInString(stripANSI(sampleGutter))
	cw := t.width - gutterWidth
	if cw < 1 {
		cw = 1
	}
	return cw
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
		chunks := t.cachedWrapChunks(vt-1, t.doc.Cells(vt-1), contentWidth)
		if rowsAbove+len(chunks) > targetRowsAbove {
			break
		}
		rowsAbove += len(chunks)
		vt--
	}
	t.viewportTop = vt
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

// handleInput processes keyboard input
// Returns true if should quit
func (t *TUI) handleInput(key []byte) bool {
	// Feed one byte at a time into the stateful parser, which accumulates
	// multi-byte sequences (SGR mouse, escape sequences) internally.
	if len(key) != 1 {
		return false
	}

	// Flash mode intercepts all input when active
	if t.flash != nil && t.flash.Active {
		return t.handleFlashInput(key[0])
	}

	cmd := t.parser.Parse(key[0])

	if cmd.Type == input.CommandNone {
		// In visual mode, 'y' should yank the selection immediately
		// rather than waiting for a second key (like 'yy' for yank-line).
		if t.parser.HasPendingYPrefix() && t.modeMachine.Mode() != vmode.Normal {
			t.parser.ClearPending()
			t.dirty = true
			return t.yank()
		}
		return false
	}
	return t.handleCommand(cmd)
}

// handleFlashInput processes a single byte during flash mode.
func (t *TUI) handleFlashInput(b byte) bool {
	// flashViewportBounds returns the viewport top and height for flash matching,
	// accounting for wrap mode where long lines consume multiple display rows.
	flashViewportBounds := func() (vpTop, vpHeight int) {
		vpTop = t.viewportTop
		vpHeight = t.contentHeight()
		if t.cfg.WrapMode == config.WrapModeOn {
			vpTop, vpHeight = t.visibleDocLineRange()
		}
		return
	}

	// Escape: cancel and restore position
	if b == 27 {
		t.flash.HandleKey(b, nil)
		t.cursorLine = t.flash.SavedCursor[0]
		t.cursorCol = t.flash.SavedCursor[1]
		t.viewportTop = t.flash.SavedViewport
		t.dirty = true
		return false
	}

	// Backspace: shorten pattern or cancel if empty
	if b == 127 || b == 8 {
		action := t.flash.HandleKey(b, nil)
		if action.Type == flash.ActionCancel {
			t.cursorLine = t.flash.SavedCursor[0]
			t.cursorCol = t.flash.SavedCursor[1]
			t.viewportTop = t.flash.SavedViewport
			t.dirty = true
			return false
		}
		// Shorten pattern and re-run matching
		if len(t.flash.Pattern) > 0 {
			t.flash.Pattern = t.flash.Pattern[:len(t.flash.Pattern)-1]
		}
		lines := t.getPlainLines()
		vpTop, vpHeight := flashViewportBounds()
		t.flash.UpdatePattern(t.flash.Pattern, lines, vpTop, vpHeight)
		t.dirty = true
		return false
	}

	// Non-printable: ignore
	if b < 32 || b >= 127 {
		return false
	}

	// Printable character: try extending pattern first, then check labels.
	// Labels are assigned from chars that DON'T extend the pattern, so if the
	// char extends the pattern (produces matches), it's unambiguously a pattern char.
	// If extending produces zero matches, check if it's a label and jump.
	extendedPattern := t.flash.Pattern + string(rune(b))
	lines := t.getPlainLines()
	vpTop, vpHeight := flashViewportBounds()
	extendedMatches := flash.FindMatches(lines, extendedPattern, vpTop, vpHeight)

	if len(extendedMatches) > 0 {
		// Extending produces matches — treat as pattern extension
		t.flash.Pattern = extendedPattern
		act := t.flash.UpdatePattern(t.flash.Pattern, lines, vpTop, vpHeight)
		if act.Type == flash.ActionAutoJump {
			t.savePrevCursor()
			t.cursorLine = act.Line
			t.cursorCol = act.Col
			t.modeMachine.OnCursorMoved(selection.Pos{Line: t.cursorLine, Col: t.cursorCol})
			t.clampViewportAndCursor()
		}
		t.dirty = true
		return false
	}

	// Extending produces zero matches — check if it's a label
	action := t.flash.HandleKey(b, lines)
	if action.Type == flash.ActionJump {
		t.savePrevCursor()
		t.cursorLine = action.Line
		t.cursorCol = action.Col
		t.modeMachine.OnCursorMoved(selection.Pos{Line: t.cursorLine, Col: t.cursorCol})
		t.clampViewportAndCursor()
		t.dirty = true
		return false
	}

	// Not a label either — ignore the keystroke (pattern stays unchanged)
	t.dirty = true
	return false
}

// getPlainLines returns the plain text content of all document lines.
func (t *TUI) getPlainLines() []string {
	n := t.doc.LineCount()
	lines := make([]string, n)
	for i := 0; i < n; i++ {
		lines[i] = t.doc.Line(i)
	}
	return lines
}

// moveDisplayLine moves the cursor by `delta` display rows within wrapped content.
// Positive delta moves down, negative moves up. Uses displayGoalCol to maintain
// a sticky column position across successive gj/gk movements.
func (t *TUI) moveDisplayLine(delta int) {
	contentWidth := t.wrapContentWidth()
	if contentWidth <= 0 {
		return
	}
	lineCount := t.doc.LineCount()
	if lineCount == 0 {
		return
	}

	// Find which display row within the current line we're on.
	cells := t.doc.Cells(t.cursorLine)
	chunks := t.cachedWrapChunks(t.cursorLine, cells, contentWidth)

	// Determine current chunk index and display column within it.
	chunkIdx := 0
	for i, ch := range chunks {
		if t.cursorCol < ch.end || i == len(chunks)-1 {
			chunkIdx = i
			break
		}
	}
	// Calculate display column (terminal columns) from chunk start to cursor,
	// accounting for wide characters (CJK, emoji).
	displayCol := 0
	for ci := chunks[chunkIdx].start; ci < t.cursorCol && ci < chunks[chunkIdx].end; ci++ {
		displayCol += runeDisplayWidth(cells[ci].Rune)
	}

	// Set goal column on first gj/gk; reuse on subsequent ones.
	if !t.hasDisplayGoal {
		t.displayGoalCol = displayCol
		t.hasDisplayGoal = true
	}

	// Walk delta display rows.
	line := t.cursorLine
	ci := chunkIdx
	remaining := delta
	if remaining > 0 {
		for remaining > 0 {
			if ci < len(chunks)-1 {
				// Move to next chunk within same line.
				ci++
			} else {
				// Cross to next line.
				line++
				if line >= lineCount {
					line = lineCount - 1
					// Stay on last chunk of last line.
					cells = t.doc.Cells(line)
					chunks = t.cachedWrapChunks(line, cells, contentWidth)
					ci = len(chunks) - 1
					break
				}
				cells = t.doc.Cells(line)
				chunks = t.cachedWrapChunks(line, cells, contentWidth)
				ci = 0
			}
			remaining--
		}
	} else {
		remaining = -remaining
		for remaining > 0 {
			if ci > 0 {
				// Move to previous chunk within same line.
				ci--
			} else {
				// Cross to previous line.
				line--
				if line < 0 {
					line = 0
					cells = t.doc.Cells(line)
					chunks = t.cachedWrapChunks(line, cells, contentWidth)
					ci = 0
					break
				}
				cells = t.doc.Cells(line)
				chunks = t.cachedWrapChunks(line, cells, contentWidth)
				ci = len(chunks) - 1
			}
			remaining--
		}
	}

	// Apply goal column in target chunk using display-column-aware positioning.
	// Walk display columns from chunk start until we reach the goal column,
	// correctly handling wide characters (CJK, emoji).
	ch := chunks[ci]
	col := ch.start
	displayCols := 0
	for col < ch.end && col < len(cells) {
		w := runeDisplayWidth(cells[col].Rune)
		if displayCols+w > t.displayGoalCol {
			break
		}
		displayCols += w
		col++
	}
	if col >= ch.end {
		col = ch.end - 1
	}
	if col < ch.start {
		col = ch.start
	}
	t.cursorLine = line
	t.cursorCol = col
}

// handleCommand executes a parsed command. Returns true if the TUI should exit.
func (t *TUI) handleCommand(cmd input.Command) bool {
	// Snapshot visible state before executing the command.
	prevCursorLine := t.cursorLine
	prevCursorCol := t.cursorCol
	prevViewportTop := t.viewportTop
	prevHOffset := t.hOffset
	prevMode := t.modeMachine.Mode()
	prevRegion := t.modeMachine.Region()
	prevWrapMode := t.cfg.WrapMode

	// While in search input mode, only allow search-related commands.
	if t.parser.InSearchMode() {
		switch cmd.Type {
		case input.CommandSearchForward, input.CommandSearchBackward,
			input.CommandSearchUpdate, input.CommandSearchConfirm, input.CommandSearchCancel:
			// Allow these through.
		default:
			// Ignore non-search commands during search input.
			return false
		}
	}

	// While in colon input mode, only allow colon-related commands.
	if t.parser.InColonMode() {
		switch cmd.Type {
		case input.CommandColonEnter, input.CommandColonUpdate,
			input.CommandColonExecute, input.CommandColonCancel:
			// Allow these through.
		default:
			// Ignore non-colon commands during colon input.
			return false
		}
	}

	switch cmd.Type {
	case input.CommandNone:
		return false

	case input.CommandMotion:
		// Save cursor for jump-back on jump motions
		switch cmd.Motion {
		case motion.MotionFirstLine, motion.MotionLastLine,
			motion.MotionScreenTop, motion.MotionScreenMiddle, motion.MotionScreenBottom,
			motion.MotionMatchBracket, motion.MotionPercentage,
			motion.MotionHalfPageUp, motion.MotionHalfPageDown,
			motion.MotionPageUp, motion.MotionPageDown,
			motion.MotionParagraphForward, motion.MotionParagraphBackward:
			t.savePrevCursor()
		}

		// Execute motion via motion handler
		cursor := motion.Cursor{Line: t.cursorLine, Col: t.cursorCol}
		viewport := motion.Viewport{Top: t.viewportTop, Height: t.contentHeight()}
		result := t.motionHandler.Apply(t, cursor, viewport, cmd.Motion, cmd.Count)

		// Update cursor and viewport
		t.cursorLine = result.Cursor.Line
		t.cursorCol = result.Cursor.Col

		// In VisualBlock mode, allow cursor past end-of-line for horizontal
		// motions so the rectangular selection can extend beyond shorter lines.
		if t.modeMachine.Mode() == vmode.VisualBlock {
			switch cmd.Motion {
			case motion.MotionRight:
				t.cursorCol = cursor.Col + cmd.Count
				if cmd.Count == 0 {
					t.cursorCol = cursor.Col + 1
				}
			case motion.MotionLineEnd:
				// $ in block mode: allow the full line length (not lineLen-1)
				lineLen := t.doc.LineRuneCount(t.cursorLine)
				if t.cursorCol < lineLen {
					t.cursorCol = lineLen
				}
			}
		}
		if t.cfg.WrapMode == config.WrapModeOn {
			// In wrap mode, only accept viewport from motions that intentionally
			// reposition it. For cursor-only motions, keep the current viewport
			// and let ensureCursorVisibleWrap handle it during render (the motion
			// handler's adjustViewport assumes 1 line = 1 display row).
			switch cmd.Motion {
			case motion.MotionViewportCenter:
				// zz: wrap-aware centering (the motion handler assumes 1 line = 1 row)
				t.centerViewportWrap(t.wrapContentWidth())
			case motion.MotionHalfPageUp, motion.MotionHalfPageDown,
				motion.MotionPageUp, motion.MotionPageDown,
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

	case input.CommandVisualBlock:
		// Toggle block-wise visual mode
		pos := selection.Pos{Line: t.cursorLine, Col: t.cursorCol}
		t.modeMachine.Handle(vmode.EventToggleVisualBlock, pos)

	case input.CommandSwapEnd:
		// Swap cursor to opposite end of selection (o)
		if newPos, ok := t.modeMachine.SwapEnd(); ok {
			t.cursorLine = newPos.Line
			t.cursorCol = newPos.Col
		}

	case input.CommandSwapCorner:
		// Swap cursor to other corner (O — column-only in block mode)
		if newPos, ok := t.modeMachine.SwapCorner(); ok {
			t.cursorLine = newPos.Line
			t.cursorCol = newPos.Col
		}

	case input.CommandEscape:
		if t.modeMachine.Mode() != vmode.Normal {
			// First ESC: exit visual mode back to normal.
			pos := selection.Pos{Line: t.cursorLine, Col: t.cursorCol}
			t.modeMachine.Handle(vmode.EventEscape, pos)
		} else if t.searchActive {
			// Second ESC (already normal mode): clear search highlights.
			t.searchActive = false
			t.searchMatches = t.searchMatches[:0]
			t.searchMatchIdx = -1
			t.dirty = true
		}

	case input.CommandYank:
		return t.yank()

	case input.CommandYankLine:
		return t.yankLine()

	case input.CommandToggleLineMode:
		t.toggleMode()

	case input.CommandToggleWrapMode:
		t.toggleWrapMode()

	case input.CommandDemoNext:
		if t.isDemo {
			t.cycleDemoPage(1)
		}

	case input.CommandDemoPrev:
		if t.isDemo {
			t.cycleDemoPage(-1)
		}

	case input.CommandDemoThemeNext:
		if t.isDemo {
			t.cycleTheme(1)
		}

	case input.CommandDemoThemePrev:
		if t.isDemo {
			t.cycleTheme(-1)
		}

	case input.CommandThemeNext:
		t.cycleTheme(1)

	case input.CommandThemePrev:
		t.cycleTheme(-1)

	case input.CommandQuit:
		return true

	case input.CommandMouseScroll:
		return t.handleMouseScroll(cmd.ScrollDirection)

	case input.CommandMouseLeftPress:
		t.handleMousePress(cmd.MouseRow, cmd.MouseCol)

	case input.CommandMouseLeftDrag:
		t.handleMouseDrag(cmd.MouseRow, cmd.MouseCol)

	case input.CommandMouseRelease:
		t.handleMouseRelease(cmd.MouseRow, cmd.MouseCol)

	case input.CommandDisplayLineDown:
		if t.cfg.WrapMode != config.WrapModeOn {
			// Wrap off → delegate to regular j
			return t.handleCommand(input.Command{Type: input.CommandMotion, Motion: motion.MotionDown, Count: cmd.Count})
		}
		count := cmd.Count
		if count == 0 {
			count = 1
		}
		t.moveDisplayLine(count)
		t.modeMachine.OnCursorMoved(selection.Pos{Line: t.cursorLine, Col: t.cursorCol})

	case input.CommandDisplayLineUp:
		if t.cfg.WrapMode != config.WrapModeOn {
			// Wrap off → delegate to regular k
			return t.handleCommand(input.Command{Type: input.CommandMotion, Motion: motion.MotionUp, Count: cmd.Count})
		}
		count := cmd.Count
		if count == 0 {
			count = 1
		}
		t.moveDisplayLine(-count)
		t.modeMachine.OnCursorMoved(selection.Pos{Line: t.cursorLine, Col: t.cursorCol})

	case input.CommandSearchForward:
		t.searchSavedLine = t.cursorLine
		t.searchSavedCol = t.cursorCol
		t.searchDirection = 1
		t.dirty = true

	case input.CommandSearchBackward:
		t.searchSavedLine = t.cursorLine
		t.searchSavedCol = t.cursorCol
		t.searchDirection = -1
		t.dirty = true

	case input.CommandSearchUpdate:
		t.incrementalSearch(cmd.SearchPattern)
		t.dirty = true

	case input.CommandSearchConfirm:
		t.searchPattern = cmd.SearchPattern
		t.searchActive = true
		if len(t.searchMatches) > 0 && t.searchMatchIdx >= 0 {
			t.jumpToMatch(t.searchMatchIdx)
		}
		t.dirty = true

	case input.CommandSearchCancel:
		// Restore cursor to saved position.
		t.cursorLine = t.searchSavedLine
		t.cursorCol = t.searchSavedCol
		// Keep searchActive for n/N if a previous confirmed pattern exists.
		t.dirty = true

	case input.CommandClearSearch:
		t.searchActive = false
		t.searchMatches = nil
		t.searchMatchIdx = -1
		t.dirty = true

	case input.CommandColonEnter:
		t.dirty = true

	case input.CommandColonUpdate:
		t.dirty = true

	case input.CommandColonExecute:
		lineNum := cmd.Count
		t.savePrevCursor()
		result := t.motionHandler.Apply(t.doc, motion.Cursor{Line: t.cursorLine, Col: t.cursorCol},
			motion.Viewport{Top: t.viewportTop, Height: t.height - 1},
			motion.MotionFirstLine, lineNum)
		t.cursorLine = result.Cursor.Line
		t.cursorCol = result.Cursor.Col
		t.viewportTop = result.Viewport.Top
		t.modeMachine.OnCursorMoved(selection.Pos{Line: t.cursorLine, Col: t.cursorCol})
		t.dirty = true

	case input.CommandColonCancel:
		t.dirty = true

	case input.CommandSearchNext:
		if t.searchActive && len(t.searchMatches) > 0 {
			t.savePrevCursor()
			// Vim behavior: search from cursor position in the original search direction.
			var idx int
			if t.searchDirection > 0 {
				idx = t.nearestMatch(t.cursorLine, t.cursorCol+1, 1)
			} else {
				idx = t.nearestMatch(t.cursorLine, t.cursorCol, -1)
			}
			t.jumpToMatch(idx)
		}

	case input.CommandSearchPrev:
		if t.searchActive && len(t.searchMatches) > 0 {
			t.savePrevCursor()
			// Vim behavior: search from cursor position in the opposite direction.
			if t.searchDirection > 0 {
				idx := t.nearestMatch(t.cursorLine, t.cursorCol, -1)
				t.jumpToMatch(idx)
			} else {
				idx := t.nearestMatch(t.cursorLine, t.cursorCol+1, 1)
				t.jumpToMatch(idx)
			}
		}

	case input.CommandSearchWordForward:
		word := t.wordAtCursor()
		if word != "" {
			t.savePrevCursor()
			t.searchPattern = `\b` + regexp.QuoteMeta(word) + `\b`
			t.searchDirection = 1
			t.searchActive = true
			t.computeSearchMatches(t.searchPattern)
			// Jump to next match after cursor.
			idx := t.nearestMatch(t.cursorLine, t.cursorCol+1, 1)
			t.jumpToMatch(idx)
		}

	case input.CommandSearchWordBackward:
		word := t.wordAtCursor()
		if word != "" {
			t.savePrevCursor()
			t.searchPattern = `\b` + regexp.QuoteMeta(word) + `\b`
			t.searchDirection = -1
			t.searchActive = true
			t.computeSearchMatches(t.searchPattern)
			// Jump to previous match before cursor.
			idx := t.nearestMatch(t.cursorLine, t.cursorCol, -1)
			t.jumpToMatch(idx)
		}

	case input.CommandScrollLineUp:
		// Ctrl-Y: scroll viewport up one line, cursor stays (clamp if off-screen)
		if t.viewportTop > 0 {
			t.viewportTop--
			// If cursor is below viewport, pull it up
			maxVisible := t.viewportTop + t.contentHeight() - 1
			if t.cursorLine > maxVisible {
				t.cursorLine = maxVisible
				t.modeMachine.OnCursorMoved(selection.Pos{Line: t.cursorLine, Col: t.cursorCol})
			}
		}

	case input.CommandScrollLineDown:
		// Ctrl-E: scroll viewport down one line, cursor stays (clamp if off-screen)
		lineCount := t.doc.LineCount()
		maxTop := lineCount - t.contentHeight()
		if maxTop < 0 {
			maxTop = 0
		}
		if t.viewportTop < maxTop {
			t.viewportTop++
			// If cursor is above viewport, pull it down
			if t.cursorLine < t.viewportTop {
				t.cursorLine = t.viewportTop
				t.modeMachine.OnCursorMoved(selection.Pos{Line: t.cursorLine, Col: t.cursorCol})
			}
		}

	case input.CommandJumpBack:
		// `` or '': swap cursor with previous position
		if t.hasPrevCursor {
			oldLine, oldCol := t.cursorLine, t.cursorCol
			t.cursorLine = t.prevCursorLine
			t.cursorCol = t.prevCursorCol
			t.prevCursorLine = oldLine
			t.prevCursorCol = oldCol
			t.clampViewportAndCursor()
			t.modeMachine.OnCursorMoved(selection.Pos{Line: t.cursorLine, Col: t.cursorCol})
		}

	case input.CommandJumpListBack:
		// Ctrl-O: jump backward in jump list
		if len(t.jumpList) > 0 {
			// If at tip (not navigating), save current position first so
			// Ctrl-I can return here.
			if t.jumpListIdx >= len(t.jumpList) {
				t.jumpList = append(t.jumpList, [2]int{t.cursorLine, t.cursorCol})
				t.jumpListIdx = len(t.jumpList) - 1
			}
			if t.jumpListIdx > 0 {
				count := cmd.Count
				if count == 0 {
					count = 1
				}
				t.jumpListIdx -= count
				if t.jumpListIdx < 0 {
					t.jumpListIdx = 0
				}
				pos := t.jumpList[t.jumpListIdx]
				t.cursorLine = pos[0]
				t.cursorCol = pos[1]
				if t.cursorLine >= t.doc.LineCount() {
					t.cursorLine = t.doc.LineCount() - 1
				}
				if t.cursorLine < 0 {
					t.cursorLine = 0
				}
				t.clampViewportAndCursor()
				t.modeMachine.OnCursorMoved(selection.Pos{Line: t.cursorLine, Col: t.cursorCol})
			}
		}

	case input.CommandJumpListForward:
		// Ctrl-I: jump forward in jump list
		if len(t.jumpList) > 0 && t.jumpListIdx < len(t.jumpList)-1 {
			count := cmd.Count
			if count == 0 {
				count = 1
			}
			t.jumpListIdx += count
			if t.jumpListIdx >= len(t.jumpList) {
				t.jumpListIdx = len(t.jumpList) - 1
			}
			pos := t.jumpList[t.jumpListIdx]
			t.cursorLine = pos[0]
			t.cursorCol = pos[1]
			if t.cursorLine >= t.doc.LineCount() {
				t.cursorLine = t.doc.LineCount() - 1
			}
			if t.cursorLine < 0 {
				t.cursorLine = 0
			}
			t.clampViewportAndCursor()
			t.modeMachine.OnCursorMoved(selection.Pos{Line: t.cursorLine, Col: t.cursorCol})
		}

	case input.CommandSetMark:
		// m{char}: save cursor position as mark
		if cmd.MarkChar >= 'a' && cmd.MarkChar <= 'z' {
			if t.marks == nil {
				t.marks = make(map[byte][2]int)
			}
			t.marks[cmd.MarkChar] = [2]int{t.cursorLine, t.cursorCol}
		}

	case input.CommandGoToMark:
		if cmd.MarkChar == '`' || cmd.MarkChar == '\'' {
			// `` or '': jump back to previous position (same as CommandJumpBack)
			if t.hasPrevCursor {
				oldLine, oldCol := t.cursorLine, t.cursorCol
				t.cursorLine = t.prevCursorLine
				t.cursorCol = t.prevCursorCol
				t.prevCursorLine = oldLine
				t.prevCursorCol = oldCol
				t.clampViewportAndCursor()
				t.modeMachine.OnCursorMoved(selection.Pos{Line: t.cursorLine, Col: t.cursorCol})
			}
		} else if pos, ok := t.marks[cmd.MarkChar]; ok {
			// `{a-z} or '{a-z}: jump to named mark
			t.savePrevCursor()
			t.cursorLine = pos[0]
			t.cursorCol = pos[1]
			if t.cursorLine >= t.doc.LineCount() {
				t.cursorLine = t.doc.LineCount() - 1
			}
			if t.cursorLine < 0 {
				t.cursorLine = 0
			}
			lineLen := t.doc.LineRuneCount(t.cursorLine)
			if t.cursorCol > lineLen {
				t.cursorCol = lineLen
			}
			t.clampViewportAndCursor()
			t.modeMachine.OnCursorMoved(selection.Pos{Line: t.cursorLine, Col: t.cursorCol})
		}

	case input.CommandCharSearch:
		// Flash f/t: when enabled, no count, and not a repeat, check for multiple matches
		if t.flash != nil && t.cfg.FlashFTEnabled && !t.flash.Active && cmd.Count == 0 &&
			cmd.SearchKind != input.SearchRepeat && cmd.SearchKind != input.SearchRepeatReverse {
			charStr := string(rune(cmd.SearchChar))
			lines := t.getPlainLines()
			// Scope FindMatches to the current line only
			matches := flash.FindMatches(lines, charStr, t.cursorLine, 1)
			if len(matches) > 1 {
				// Multiple matches on this line — enter flash mode
				t.flash.Enter(t.cursorLine, t.cursorCol, t.viewportTop)
				act := t.flash.UpdatePattern(charStr, lines, t.cursorLine, 1)
				if act.Type == flash.ActionAutoJump {
					// Shouldn't happen since len > 1, but handle gracefully
					t.savePrevCursor()
					t.cursorLine = act.Line
					t.cursorCol = act.Col
					t.modeMachine.OnCursorMoved(selection.Pos{Line: t.cursorLine, Col: t.cursorCol})
					t.clampViewportAndCursor()
				}
				t.dirty = true
				break
			}
			// 0 or 1 match — fall through to normal char search
		}
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

	case input.CommandSearchSelect:
		// gn: find next search match, select it in visual char mode
		if t.searchActive && len(t.searchMatches) > 0 {
			idx := t.nearestMatch(t.cursorLine, t.cursorCol, 1)
			m := t.searchMatches[idx]
			// Enter visual mode at match start, move cursor to match end
			if t.modeMachine.Mode() == vmode.Normal {
				t.modeMachine.Handle(vmode.EventToggleVisualChar, selection.Pos{Line: m.Line, Col: m.ColStart})
			} else {
				// Already visual — reset to match start
				t.modeMachine.Handle(vmode.EventEscape, selection.Pos{Line: t.cursorLine, Col: t.cursorCol})
				t.modeMachine.Handle(vmode.EventToggleVisualChar, selection.Pos{Line: m.Line, Col: m.ColStart})
			}
			t.cursorLine = m.Line
			t.cursorCol = m.ColEnd
			t.modeMachine.OnCursorMoved(selection.Pos{Line: m.Line, Col: m.ColEnd})
			t.clampViewportAndCursor()
		}

	case input.CommandSearchSelectBack:
		// gN: find previous search match, select it in visual char mode
		if t.searchActive && len(t.searchMatches) > 0 {
			idx := t.nearestMatch(t.cursorLine, t.cursorCol, -1)
			m := t.searchMatches[idx]
			if t.modeMachine.Mode() == vmode.Normal {
				t.modeMachine.Handle(vmode.EventToggleVisualChar, selection.Pos{Line: m.Line, Col: m.ColStart})
			} else {
				t.modeMachine.Handle(vmode.EventEscape, selection.Pos{Line: t.cursorLine, Col: t.cursorCol})
				t.modeMachine.Handle(vmode.EventToggleVisualChar, selection.Pos{Line: m.Line, Col: m.ColStart})
			}
			t.cursorLine = m.Line
			t.cursorCol = m.ColEnd
			t.modeMachine.OnCursorMoved(selection.Pos{Line: m.Line, Col: m.ColEnd})
			t.clampViewportAndCursor()
		}

	case input.CommandTextObject:
		// Text objects only work in visual mode (read-only viewer has no operators)
		if t.modeMachine.Mode() != vmode.Normal {
			cursor := motion.Cursor{Line: t.cursorLine, Col: t.cursorCol}
			action := keymap.Action(cmd.TextObject)
			r := textobj.Resolve(t, cursor, action)
			if r.OK {
				// Check if text object range is within current selection (for expansion)
				region := t.modeMachine.Region()
				selStart := region.Start
				selEnd := region.End
				if selStart.Line > selEnd.Line || (selStart.Line == selEnd.Line && selStart.Col > selEnd.Col) {
					selStart, selEnd = selEnd, selStart
				}
				contained := (r.StartLine > selStart.Line || (r.StartLine == selStart.Line && r.StartCol >= selStart.Col)) &&
					(r.EndLine < selEnd.Line || (r.EndLine == selEnd.Line && r.EndCol <= selEnd.Col))

				if contained {
					// Expansion: resolve from one line beyond selection end
					expandLine := selEnd.Line + 1
					if expandLine < t.doc.LineCount() {
						expanded := textobj.Resolve(t, motion.Cursor{Line: expandLine, Col: 0}, action)
						if expanded.OK {
							// Extend selection end to include the expanded range
							r.StartLine = selStart.Line
							r.StartCol = selStart.Col
							r.EndLine = expanded.EndLine
							r.EndCol = expanded.EndCol
						}
					}
				}

				// Set selection to the (possibly expanded) text object range
				t.modeMachine.Handle(vmode.EventEscape, selection.Pos{Line: t.cursorLine, Col: t.cursorCol})
				t.modeMachine.Handle(vmode.EventToggleVisualChar, selection.Pos{Line: r.StartLine, Col: r.StartCol})
				t.cursorLine = r.EndLine
				t.cursorCol = r.EndCol
				t.modeMachine.OnCursorMoved(selection.Pos{Line: r.EndLine, Col: r.EndCol})
				t.clampViewportAndCursor()
			}
		}

	case input.CommandFlashEnter:
		if t.flash != nil && !t.flash.Active {
			t.flash.Enter(t.cursorLine, t.cursorCol, t.viewportTop)
			t.dirty = true
		}
	}

	// Sync parser keymap after any mode transition (covers keyboard, mouse,
	// gn/gN, text objects — all paths uniformly).
	t.syncKeymapToMode()

	// Only mark dirty if visible state actually changed.
	curMode := t.modeMachine.Mode()
	curRegion := t.modeMachine.Region()
	if t.cursorLine != prevCursorLine ||
		t.cursorCol != prevCursorCol ||
		t.viewportTop != prevViewportTop ||
		t.hOffset != prevHOffset ||
		curMode != prevMode ||
		curRegion != prevRegion ||
		t.cfg.WrapMode != prevWrapMode {
		t.dirty = true
	}

	// Reset display goal column on any command that isn't gj/gk.
	if cmd.Type != input.CommandDisplayLineDown && cmd.Type != input.CommandDisplayLineUp {
		t.hasDisplayGoal = false
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
	sampleGutter := t.formatter.RenderGutter(1, 1)
	gutterWidth := utf8.RuneCountInString(stripANSI(sampleGutter))

	// Content column: subtract gutter, clamp to 0
	contentCol := termCol - gutterWidth
	if contentCol < 0 {
		contentCol = 0 // clicking on gutter → column 0
	}

	if t.cfg.WrapMode == config.WrapModeOn {
		return t.mouseToDocPosWrap(termRow, contentCol, gutterWidth)
	}
	return t.mouseToDocPosScroll(termRow, contentCol)
}

// mouseToDocPosScroll maps terminal position to document position in non-wrap mode.
func (t *TUI) mouseToDocPosScroll(termRow, contentCol int) (selection.Pos, bool) {
	lineCount := t.doc.LineCount()
	docLine := t.viewportTop + termRow

	// Clamp to valid line range
	if docLine < 0 {
		docLine = 0
	}
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
	contentWidth := t.width - gutterWidth
	if contentWidth < 1 {
		contentWidth = 1
	}

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
				maxCol := t.doc.LineRuneCount(lineIdx) - 1
				if maxCol < 0 {
					maxCol = 0
				}
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
	maxCol := t.doc.LineRuneCount(lastLine) - 1
	if maxCol < 0 {
		maxCol = 0
	}
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
	maxCol := len(runes) - 1
	if maxCol < 0 {
		maxCol = 0
	}
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

	// Viewport-scroll path: content is taller than the terminal window.
	ch := t.contentHeight()
	if ch > 0 && lineCount > ch {
		maxViewportTop := lastLine - ch + 1
		switch dir {
		case input.ScrollUp:
			t.viewportTop -= step
			if t.viewportTop < 0 {
				t.viewportTop = 0
			}
			// Cursor must stay within the (now higher) viewport window.
			if newBottom := t.viewportTop + ch - 1; t.cursorLine > newBottom {
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
	t.invalidateGutterCache()
	t.clampViewportAndCursor()
}

// HandleCommand processes a Command directly (exported for testing).
func (t *TUI) HandleCommand(cmd input.Command) bool { return t.handleCommand(cmd) }

// toggleMode cycles through line number modes
func (t *TUI) toggleMode() {
	t.formatter.ToggleMode()
	t.invalidateGutterCache()
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

// syncKeymapToMode swaps the parser's keymap to match the current mode machine state.
// No-op if ModeKeymap was not configured (zero-valued, e.g. in tests).
func (t *TUI) syncKeymapToMode() {
	km := t.cfg.ModeKeymap.ForMode(t.modeMachine.Mode() != vmode.Normal)
	if km.Direct == nil {
		return
	}
	t.parser.SetKeymap(km)
}

// toggleWrapMode switches between wrap and scroll mode at runtime.
func (t *TUI) toggleWrapMode() {
	if t.cfg.WrapMode == config.WrapModeOn {
		t.cfg.WrapMode = config.WrapModeOff
	} else {
		t.cfg.WrapMode = config.WrapModeOn
	}
	t.hOffset = 0
	t.clampViewportAndCursor()
}

// cycleDemoPage cycles the demo page by delta (+1 or -1) with wrapping.
func (t *TUI) cycleDemoPage(delta int) {
	if len(t.demoPages) == 0 {
		return
	}
	t.demoPageIndex = (t.demoPageIndex + delta + len(t.demoPages)) % len(t.demoPages)

	// Rebuild document from new page
	content := t.demoPages[t.demoPageIndex]
	t.doc = NewDocument(content)
	t.wrapCache = nil

	maxLine := t.doc.LineCount()
	if maxLine == 0 {
		maxLine = 1
	}

	// Recalculate formatter gutter width
	lineNumMode, err := linenums.ModeFromString(t.lineNumMode)
	if err != nil {
		lineNumMode = linenums.ModeHybrid
	}
	t.formatter = linenums.NewFormatterWithFullPalette(lineNumMode, maxLine, t.palette.Gutter, t.palette.LineNum)
	t.invalidateGutterCache()

	// Reset cursor to middle
	t.cursorLine = (maxLine - 1) / 2
	if t.cursorLine < 0 {
		t.cursorLine = 0
	}
	t.cursorCol = 0
	t.viewportTop = 0
	t.hOffset = 0

	// Reset selection mode machine state
	t.modeMachine = vmode.NewMachine()
	t.dirty = true
}

// cycleTheme cycles the demo theme by delta (+1 or -1) with wrapping.
func (t *TUI) cycleTheme(delta int) {
	n := len(theme.ThemeOrder)
	if n == 0 {
		return
	}
	t.demoThemeIndex = (t.demoThemeIndex + delta + n) % n
	t.demoThemeName = theme.ThemeOrder[t.demoThemeIndex]

	// Resolve the new theme palette (pure preset, no overrides)
	palette, err := theme.Resolve(t.demoThemeName, theme.ThemeOverrides{})
	if err != nil {
		return
	}
	t.palette = palette

	// Recreate formatter with new palette colors
	lineNumMode, modeErr := linenums.ModeFromString(t.lineNumMode)
	if modeErr != nil {
		lineNumMode = linenums.ModeHybrid
	}
	maxLine := t.doc.LineCount()
	if maxLine == 0 {
		maxLine = 1
	}
	t.formatter = linenums.NewFormatterWithFullPalette(lineNumMode, maxLine, palette.Gutter, palette.LineNum)
	t.invalidateGutterCache()
	t.dirty = true
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

	lastCol := t.doc.LineRuneCount(lineIdx) - 1
	if lastCol < 0 {
		lastCol = 0
	}

	if region.Kind == selection.KindChar {
		if lineIdx == start.Line && lineIdx == end.Line {
			selStart = start.Col
			selEnd = end.Col
		} else if lineIdx == start.Line {
			selStart = start.Col
			selEnd = lastCol
		} else if lineIdx == end.Line {
			selStart = 0
			selEnd = end.Col
		} else if lineIdx > start.Line && lineIdx < end.Line {
			selStart = 0
			selEnd = lastCol
		}
	} else if region.Kind == selection.KindLine {
		if lineIdx >= start.Line && lineIdx <= end.Line {
			selStart = 0
			selEnd = lastCol
		}
	} else if region.Kind == selection.KindBlock {
		if lineIdx >= start.Line && lineIdx <= end.Line {
			// Block mode: use column range from raw (un-normalized) positions
			minCol := region.Start.Col
			maxCol := region.End.Col
			if minCol > maxCol {
				minCol, maxCol = maxCol, minCol
			}
			selStart = minCol
			selEnd = maxCol
		}
	}
	return
}

// computeSearchMatches compiles the pattern and finds all matches in the document.
func (t *TUI) computeSearchMatches(pattern string) {
	t.searchMatches = t.searchMatches[:0]
	t.searchMatchIdx = -1
	if pattern == "" {
		t.searchRegex = nil
		return
	}

	// Smart case: if pattern has any uppercase letter, case-sensitive; else case-insensitive.
	hasUpper := false
	for _, r := range pattern {
		if unicode.IsUpper(r) {
			hasUpper = true
			break
		}
	}
	regexPattern := pattern
	if !hasUpper {
		regexPattern = "(?i)" + pattern
	}

	re, err := regexp.Compile(regexPattern)
	if err != nil {
		// Invalid regex — fall back to literal
		re, err = regexp.Compile(regexp.QuoteMeta(pattern))
		if err != nil {
			t.searchRegex = nil
			return
		}
	}
	t.searchRegex = re

	lineCount := t.doc.LineCount()
	for i := 0; i < lineCount; i++ {
		line := t.doc.Line(i)
		locs := re.FindAllStringIndex(line, -1)
		for _, loc := range locs {
			// Convert byte offsets to rune offsets.
			colStart := byteOffsetToRuneOffset(line, loc[0])
			colEnd := byteOffsetToRuneOffset(line, loc[1]) - 1
			if colEnd < colStart {
				colEnd = colStart
			}
			t.searchMatches = append(t.searchMatches, searchMatch{
				Line:     i,
				ColStart: colStart,
				ColEnd:   colEnd,
			})
		}
	}
}

// byteOffsetToRuneOffset converts a byte offset in a string to a rune offset.
func byteOffsetToRuneOffset(s string, byteOff int) int {
	runeIdx := 0
	for i := range s {
		if i >= byteOff {
			return runeIdx
		}
		runeIdx++
	}
	return runeIdx
}

// nearestMatch finds the nearest match index from position in the given direction.
// Returns -1 if no matches exist. Wraps around.
func (t *TUI) nearestMatch(fromLine, fromCol, direction int) int {
	n := len(t.searchMatches)
	if n == 0 {
		return -1
	}
	// Binary search for the first match at or after (fromLine, fromCol).
	idx := sort.Search(n, func(i int) bool {
		m := t.searchMatches[i]
		return m.Line > fromLine || (m.Line == fromLine && m.ColStart >= fromCol)
	})
	if direction > 0 {
		// Forward: use idx, wrapping to 0 if past end.
		if idx >= n {
			return 0
		}
		return idx
	}
	// Backward: use idx-1, wrapping to n-1 if before start.
	idx--
	if idx < 0 {
		return n - 1
	}
	return idx
}

// jumpToMatch moves the cursor to the specified match and updates the viewport.
func (t *TUI) jumpToMatch(idx int) {
	if idx < 0 || idx >= len(t.searchMatches) {
		return
	}
	m := t.searchMatches[idx]
	t.searchMatchIdx = idx
	t.cursorLine = m.Line
	t.cursorCol = m.ColStart

	// Scroll viewport to make the match visible.
	t.clampViewportAndCursor()

	// Notify mode machine of cursor movement (extends selection if in visual mode).
	pos := selection.Pos{Line: t.cursorLine, Col: t.cursorCol}
	t.modeMachine.OnCursorMoved(pos)
}

// incrementalSearch recomputes matches and jumps to the nearest match from the saved position.
func (t *TUI) incrementalSearch(pattern string) {
	t.computeSearchMatches(pattern)
	t.searchActive = len(t.searchMatches) > 0
	if t.searchActive {
		idx := t.nearestMatch(t.searchSavedLine, t.searchSavedCol, t.searchDirection)
		t.jumpToMatch(idx)
	}
}

// savePrevCursor saves the current cursor position for jump-back (``/'')
// and pushes it onto the jump list.
// After this call, jumpListIdx == len(jumpList), meaning "at tip" (not
// navigating backward). The first Ctrl-O will save the current position
// and then jump to the entry before it.
func (t *TUI) savePrevCursor() {
	t.prevCursorLine = t.cursorLine
	t.prevCursorCol = t.cursorCol
	t.hasPrevCursor = true

	// Push to jump list: truncate any forward entries (if we navigated back)
	if t.jumpListIdx < len(t.jumpList) {
		t.jumpList = t.jumpList[:t.jumpListIdx]
	}
	t.jumpList = append(t.jumpList, [2]int{t.cursorLine, t.cursorCol})
	const maxJumpList = 100
	if len(t.jumpList) > maxJumpList {
		t.jumpList = t.jumpList[len(t.jumpList)-maxJumpList:]
	}
	// "at tip" — past the last entry
	t.jumpListIdx = len(t.jumpList)
}

// wordAtCursor extracts the word under the cursor (similar to vim's * word boundary).
func (t *TUI) wordAtCursor() string {
	if t.doc.LineCount() == 0 {
		return ""
	}
	cells := t.doc.Cells(t.cursorLine)
	if len(cells) == 0 {
		return ""
	}
	col := t.cursorCol
	if col >= len(cells) {
		col = len(cells) - 1
	}

	// Check if cursor is on a word character.
	isWord := func(r rune) bool {
		return (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') ||
			(r >= '0' && r <= '9') || r == '_'
	}

	if !isWord(cells[col].Rune) {
		return ""
	}

	// Expand left.
	start := col
	for start > 0 && isWord(cells[start-1].Rune) {
		start--
	}
	// Expand right.
	end := col
	for end < len(cells)-1 && isWord(cells[end+1].Rune) {
		end++
	}

	var buf strings.Builder
	for i := start; i <= end; i++ {
		buf.WriteRune(cells[i].Rune)
	}
	return buf.String()
}

// lineSearchRanges returns all search match ranges on a given line,
// plus the current match range (or {-1,-1} if not on this line).
func (t *TUI) lineSearchRanges(lineIdx int) (ranges [][2]int, currentRange [2]int) {
	currentRange = [2]int{-1, -1}
	if !t.searchActive || len(t.searchMatches) == 0 {
		return
	}

	// Binary search for the first match on this line.
	n := len(t.searchMatches)
	lo := sort.Search(n, func(i int) bool {
		return t.searchMatches[i].Line >= lineIdx
	})

	for i := lo; i < n && t.searchMatches[i].Line == lineIdx; i++ {
		m := t.searchMatches[i]
		ranges = append(ranges, [2]int{m.ColStart, m.ColEnd})
		if i == t.searchMatchIdx {
			currentRange = [2]int{m.ColStart, m.ColEnd}
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

// shouldShowStatusBar returns true if the status bar should be displayed.
func (t *TUI) shouldShowStatusBar() bool {
	if t.height <= 1 {
		return false
	}
	if t.isDemo {
		return true // demo always shows status bar
	}
	return t.cfg.StatusBar == config.StatusBarOn
}

// render draws the TUI
func (t *TUI) render() {
	showStatus := t.shouldShowStatusBar()
	contentHeight := t.height
	if showStatus {
		contentHeight = t.height - 1 // reserve last row for status bar
		t.height = contentHeight
	}

	if t.cfg.WrapMode == config.WrapModeOn {
		t.renderWrap()
	} else {
		t.renderScroll()
	}

	if showStatus {
		t.height = contentHeight + 1 // restore full height
		t.renderStatusBar()
	} else if t.flash != nil && t.flash.Active {
		// No status bar but flash is active -- render inline prompt on last row
		t.renderInlineFlashPrompt()
	}
}

// renderInlineFlashPrompt renders a minimal flash prompt on the last terminal row
// when the status bar is disabled. This gives the user visual feedback about their
// flash search pattern and match count.
func (t *TUI) renderInlineFlashPrompt() {
	if t.width <= 0 {
		return
	}

	labeled := 0
	for _, m := range t.flash.Matches {
		if m.Label != 0 {
			labeled++
		}
	}
	total := len(t.flash.Matches)

	prompt := fmt.Sprintf("FLASH /%s [%d/%d]", t.flash.Pattern, labeled, total)

	// Truncate if wider than terminal
	promptRunes := []rune(prompt)
	if len(promptRunes) > t.width {
		promptRunes = promptRunes[:t.width]
	}

	// Render on last row with FlashLabel palette for visibility
	pal := t.palette.FlashLabel
	fmt.Printf("\x1b[%d;1H%s%s\x1b[K\x1b[0m", t.height, cellPaletteSGR(pal), string(promptRunes))
}

// renderScroll renders with horizontal scrolling (default mode).
func (t *TUI) renderScroll() {
	var b strings.Builder
	b.Grow(t.width * t.height * 2)
	b.WriteString("\x1b[H")

	endLine := t.viewportTop + t.height
	if endLine > t.doc.LineCount() {
		endLine = t.doc.LineCount()
	}

	region := t.modeMachine.Region()

	gutterWidth := t.gutterWidth()
	contentWidth := t.width - gutterWidth
	if contentWidth < 0 {
		contentWidth = 0
	}

	t.ensureCursorVisibleH(contentWidth)

	var flashOverlay *flash.Overlay
	if t.flash != nil {
		flashOverlay = t.flash.Overlay()
	}

	for i := t.viewportTop; i < endLine; i++ {
		gutter := t.formatter.RenderGutter(i+1, t.cursorLine+1)
		b.WriteString(gutter)

		cursorCol, selStart, selEnd := t.lineSelection(i, region)

		searchRanges, currentMatch := t.lineSearchRanges(i)
		renderedLine := RenderCellsWithPalette(t.doc.Cells(i), cursorCol, selStart, selEnd, searchRanges, currentMatch, t.hOffset, contentWidth, t.palette, flashOverlay, i)
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

// searchMatch represents a single search result in the document.
type searchMatch struct {
	Line     int
	ColStart int // rune index, inclusive
	ColEnd   int // rune index, inclusive
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

// cachedWrapChunks returns wordWrapChunks for a line, using the TUI's cache.
// The cache is invalidated when contentWidth changes (e.g., on resize).
// When the cache exceeds a size threshold, entries far from the viewport are evicted.
func (t *TUI) cachedWrapChunks(lineIdx int, cells []Cell, contentWidth int) []wrapChunk {
	if t.wrapCache != nil && t.wrapCacheWidth == contentWidth {
		if chunks, ok := t.wrapCache[lineIdx]; ok {
			return chunks
		}
	} else {
		// Width changed — invalidate entire cache
		t.wrapCache = make(map[int][]wrapChunk)
		t.wrapCacheWidth = contentWidth
	}
	chunks := wordWrapChunks(cells, contentWidth)
	t.wrapCache[lineIdx] = chunks

	// Evict entries far from the viewport when cache grows too large.
	maxEntries := t.height * 20
	if maxEntries < 100 {
		maxEntries = 100
	}
	if len(t.wrapCache) > maxEntries {
		margin := t.height * 5
		lo := t.viewportTop - margin
		hi := t.viewportTop + t.height + margin
		for k := range t.wrapCache {
			if k < lo || k > hi {
				delete(t.wrapCache, k)
			}
		}
	}

	return chunks
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
		// Find the furthest cell index that fits within contentWidth display columns.
		end := pos
		displayCols := 0
		for end < n {
			w := runeDisplayWidth(cells[end].Rune)
			if displayCols+w > contentWidth {
				break
			}
			displayCols += w
			end++
		}
		if end == pos {
			// Single cell wider than contentWidth — include it to avoid infinite loop.
			end = pos + 1
		}
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
			// No space found — hard wrap at end.
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
	b.Grow(t.width * t.height * 2)
	b.WriteString("\x1b[H")

	// In wrap mode, horizontal offset is always 0.
	t.hOffset = 0

	region := t.modeMachine.Region()

	gutterWidth := t.gutterWidth()
	contentWidth := t.width - gutterWidth
	if contentWidth < 1 {
		contentWidth = 1
	}
	blankGutter := t.blankGutter()

	// Adjust viewport so cursor is on-screen (wrap-aware).
	t.ensureCursorVisibleWrap(contentWidth)

	var flashOverlay *flash.Overlay
	if t.flash != nil {
		flashOverlay = t.flash.Overlay()
	}

	displayRow := 0
	lineCount := t.doc.LineCount()

	for i := t.viewportTop; i < lineCount && displayRow < t.height; i++ {
		cells := t.doc.Cells(i)
		cursorCol, selStart, selEnd := t.lineSelection(i, region)

		chunks := t.cachedWrapChunks(i, cells, contentWidth)

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

			// Calculate chunk display width in terminal columns (not rune count)
			// so wide characters (CJK, emoji) are accounted for correctly.
			chunkDW := 0
			for ci2 := chunk.start; ci2 < chunk.end && ci2 < len(cells); ci2++ {
				chunkDW += runeDisplayWidth(cells[ci2].Rune)
			}
			// For the last chunk of the cursor line, use full contentWidth
			// so the cursor can be rendered past the end of text.
			maxWidth := chunkDW
			if i == t.cursorLine && ci == len(chunks)-1 {
				maxWidth = contentWidth
			}

			searchRanges, currentMatch := t.lineSearchRanges(i)
			renderedLine := RenderCellsWithPalette(cells, cursorCol, selStart, selEnd, searchRanges, currentMatch, chunk.start, maxWidth, t.palette, flashOverlay, i)
			b.WriteString(renderedLine)
			// Extend the last cell's background through \x1b[K so that
			// full-width highlights (e.g. git diff) don't break on wrapped rows.
			if chunkDW < contentWidth && chunk.end > chunk.start && chunk.end-1 < len(cells) {
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

// parseStatusHex parses a "#rrggbb" string into RGB components (for status bar).
func parseStatusHex(hex string) (r, g, b int, ok bool) {
	hex = strings.TrimPrefix(hex, "#")
	if len(hex) != 6 {
		return 0, 0, 0, false
	}
	rv, err := fmt.Sscanf(hex, "%02x%02x%02x", &r, &g, &b)
	if err != nil || rv != 3 {
		return 0, 0, 0, false
	}
	return r, g, b, true
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

	// Extract selected text lazily (only accesses lines within the selection region).
	text, err := selection.ExtractRegionFromProvider(t.doc, region)
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
	t.syncKeymapToMode()

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
