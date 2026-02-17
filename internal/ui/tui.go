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
		t.viewportTop = result.Viewport.Top

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

	case input.CommandToggleLineMode:
		t.toggleMode()

	case input.CommandQuit:
		return true

	case input.CommandMouseScroll:
		return t.handleMouseScroll(cmd.ScrollDirection)
	}

	return false
}

// handleMouseScroll handles wheel-up/down events.
// Wheel-up moves cursor up (like k). Wheel-down moves cursor down (like j),
// but exits if already at the last line (overscroll to exit).
func (t *TUI) handleMouseScroll(dir input.ScrollDirection) bool {
	switch dir {
	case input.ScrollUp:
		if t.cursorLine > 0 {
			t.cursorLine--
			t.clampViewportAndCursor()
		}
		return false
	case input.ScrollDown:
		lastLine := t.doc.LineCount() - 1
		if t.cursorLine >= lastLine {
			return true // overscroll at bottom: exit
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

// render draws the TUI
func (t *TUI) render() {
	var b strings.Builder

	// Move cursor to top-left
	b.WriteString("\x1b[H")

	// Calculate visible range
	endLine := t.viewportTop + t.height
	if endLine > t.doc.LineCount() {
		endLine = t.doc.LineCount()
	}

	// Get selection region from mode machine
	region := t.modeMachine.Region()

	// Render visible lines
	for i := t.viewportTop; i < endLine; i++ {
		// Render line number gutter (1-indexed for display)
		gutter := t.formatter.RenderGutter(i+1, t.cursorLine+1)
		b.WriteString(gutter)

		// Get raw ANSI line
		rawLine := t.doc.RawLine(i)

		// Calculate cursor and selection for this line
		cursorCol := -1
		selStart := -1
		selEnd := -1

		if i == t.cursorLine {
			cursorCol = t.cursorCol
		}

		// Determine selection range for this line based on region kind
		if region.Kind != selection.KindNone {
			// Normalize region (swap if backwards)
			start, end := region.Start, region.End
			if start.Line > end.Line || (start.Line == end.Line && start.Col > end.Col) {
				start, end = end, start
			}

			if region.Kind == selection.KindChar {
				// Character-wise selection - column precision
				if i == start.Line && i == end.Line {
					// Single line selection
					selStart = start.Col
					selEnd = end.Col
				} else if i == start.Line {
					// First line of multi-line selection
					selStart = start.Col
					selEnd = 9999 // To end of line
				} else if i == end.Line {
					// Last line of multi-line selection
					selStart = 0
					selEnd = end.Col
				} else if i > start.Line && i < end.Line {
					// Middle line - select entire line
					selStart = 0
					selEnd = 9999
				}
			} else if region.Kind == selection.KindLine {
				// Line-wise selection - entire lines
				if i >= start.Line && i <= end.Line {
					selStart = 0
					selEnd = 9999
				}
			}
		}

		// Calculate available width (account for gutter)
		gutterWidth := len(stripANSI(gutter))
		availableWidth := t.width - gutterWidth

		// Render line with ANSI color preservation and cursor/selection overlay
		renderedLine := RenderLineWithPalette(rawLine, cursorCol, selStart, selEnd, availableWidth, t.palette)
		b.WriteString(renderedLine)

		// Clear to end of line
		b.WriteString("\x1b[K")

		// Newline if not last line
		if i < endLine-1 {
			b.WriteString("\r\n")
		}
	}

	// Clear remaining lines
	for i := endLine - t.viewportTop; i < t.height; i++ {
		b.WriteString("\r\n\x1b[K")
	}

	// Write to stdout
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
