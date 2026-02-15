package ui

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/shitcoding/tmux_yankee/internal/linenums"
	"github.com/shitcoding/tmux_yankee/internal/selection"
	"github.com/shitcoding/tmux_yankee/internal/tmux"
	"golang.org/x/term"
)

// TUI represents the terminal UI
type TUI struct {
	paneID      string
	content     []string
	mode        string
	formatter   *linenums.Formatter
	selection   *selection.Selection
	client      *tmux.Client
	cursorLine  int
	viewportTop int
	width       int
	height      int
	oldState    *term.State
}

// NewTUI creates a new TUI instance
func NewTUI(paneID string, content []string, mode string) *TUI {
	// Parse mode string
	lineNumMode, err := linenums.ModeFromString(mode)
	if err != nil {
		lineNumMode = linenums.ModeHybrid
	}

	// Calculate max line number
	maxLine := len(content)
	if maxLine == 0 {
		maxLine = 1
	}

	return &TUI{
		paneID:      paneID,
		content:     content,
		mode:        mode,
		formatter:   linenums.NewFormatter(lineNumMode, maxLine),
		selection:   selection.New(),
		client:      tmux.NewClient(),
		cursorLine:  0,
		viewportTop: 0,
	}
}

// Run starts the TUI event loop
func (t *TUI) Run() error {
	// Initialize terminal
	if err := t.initTerminal(); err != nil {
		return fmt.Errorf("terminal init failed: %w", err)
	}
	defer t.restoreTerminal()

	// Get terminal size
	width, height, err := term.GetSize(int(os.Stdin.Fd()))
	if err != nil {
		return fmt.Errorf("get terminal size failed: %w", err)
	}
	t.width = width
	t.height = height

	// Clear screen and hide cursor
	fmt.Print("\x1b[2J\x1b[?25l")

	// Initial render
	t.render()

	// Event loop
	const maxKeySequenceBytes = 3 // Escape sequences like \x1b[A
	buf := make([]byte, maxKeySequenceBytes)
	for {
		n, err := os.Stdin.Read(buf)
		if err != nil {
			if err == io.EOF {
				break
			}
			return fmt.Errorf("read failed: %w", err)
		}

		// Process each byte individually to handle rapid key presses
		needsRender := false
		for i := 0; i < n; i++ {
			key := buf[i : i+1]

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
	}

	return nil
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
		// Show cursor and clear screen
		fmt.Print("\x1b[?25h\x1b[2J\x1b[H")
		term.Restore(int(os.Stdin.Fd()), t.oldState)
	}
}

// handleInput processes keyboard input
// Returns true if should quit
func (t *TUI) handleInput(key []byte) bool {
	switch {
	case len(key) == 1 && key[0] == 'q':
		return true
	case len(key) == 1 && key[0] == 'v':
		t.toggleSelection()
	case len(key) == 1 && (key[0] == 'y' || key[0] == 13): // 'y' or Enter
		return t.yank()
	case len(key) == 1 && key[0] == 'j':
		t.moveCursorDown()
	case len(key) == 1 && key[0] == 'k':
		t.moveCursorUp()
	case len(key) == 1 && key[0] == 'L':
		t.toggleMode()
	case len(key) == 1 && key[0] == 3: // Ctrl-C
		return true
	}
	return false
}

// toggleMode cycles through line number modes
func (t *TUI) toggleMode() {
	t.formatter.ToggleMode()
	// Update mode string for consistency
	switch t.formatter.CurrentMode() {
	case linenums.ModeAbsolute:
		t.mode = "absolute"
	case linenums.ModeRelative:
		t.mode = "relative"
	case linenums.ModeHybrid:
		t.mode = "hybrid"
	}
}

// toggleSelection toggles visual selection mode
func (t *TUI) toggleSelection() {
	if !t.selection.IsActive() {
		// Activate selection at current cursor position
		t.selection.SetStart(t.cursorLine, 0)
		t.selection.UpdateEnd(t.cursorLine, 0)
		t.selection.Toggle()
	} else {
		// Deactivate selection
		t.selection.Toggle()
	}
}

// moveCursorDown moves cursor down one line
func (t *TUI) moveCursorDown() {
	if t.cursorLine < len(t.content)-1 {
		t.cursorLine++
		// Adjust viewport if cursor moves off screen
		if t.cursorLine >= t.viewportTop+t.height {
			t.viewportTop++
		}
		// Update selection end if active
		if t.selection.IsActive() {
			t.selection.UpdateEnd(t.cursorLine, 0)
		}
	}
}

// moveCursorUp moves cursor up one line
func (t *TUI) moveCursorUp() {
	if t.cursorLine > 0 {
		t.cursorLine--
		// Adjust viewport if cursor moves off screen
		if t.cursorLine < t.viewportTop {
			t.viewportTop--
		}
		// Update selection end if active
		if t.selection.IsActive() {
			t.selection.UpdateEnd(t.cursorLine, 0)
		}
	}
}

// render draws the TUI
func (t *TUI) render() {
	var b strings.Builder

	// Move cursor to top-left
	b.WriteString("\x1b[H")

	// Calculate visible range
	endLine := t.viewportTop + t.height
	if endLine > len(t.content) {
		endLine = len(t.content)
	}

	// Get selection range if active
	var selStart, selEnd int
	if t.selection.IsActive() {
		selStart, selEnd = t.selection.Range()
	}

	// Render visible lines
	for i := t.viewportTop; i < endLine; i++ {
		line := t.content[i]

		// Render line number gutter (1-indexed for display)
		gutter := t.formatter.RenderGutter(i+1, t.cursorLine+1)
		b.WriteString(gutter)

		// Determine if this line is selected
		isSelected := t.selection.IsActive() && i >= selStart && i <= selEnd

		// Highlight cursor line or selected line
		if isSelected {
			b.WriteString("\x1b[7m") // Reverse video for selection
		} else if i == t.cursorLine {
			b.WriteString("\x1b[7m") // Reverse video for cursor
		}

		// Truncate line if too long (account for gutter width)
		gutterWidth := len(stripANSI(gutter))
		availableWidth := t.width - gutterWidth
		runes := []rune(line)
		if len(runes) > availableWidth {
			line = string(runes[:availableWidth])
		}

		b.WriteString(line)

		// Reset style and clear to end of line
		b.WriteString("\x1b[0m\x1b[K")

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

// stripANSI removes ANSI escape codes from a string
func stripANSI(s string) string {
	result := ""
	inEscape := false
	for _, r := range s {
		if r == '\x1b' {
			inEscape = true
			continue
		}
		if inEscape {
			if r == 'm' {
				inEscape = false
			}
			continue
		}
		result += string(r)
	}
	return result
}

// GetMode returns the current line number mode
func (t *TUI) GetMode() string {
	return t.mode
}

// yank extracts selected text, copies to clipboard and tmux buffer
// Returns true to quit TUI after yank
func (t *TUI) yank() bool {
	// Only yank if selection is active
	if !t.selection.IsActive() {
		return false
	}

	// Build rendered content with gutters for extraction
	renderedContent := t.buildRenderedContent()

	// Extract selected text (gutter will be stripped)
	text, err := t.selection.Extract(renderedContent)
	if err != nil {
		// Silently fail - could log error in production
		return false
	}

	// ALWAYS set tmux buffer first (reliable fallback)
	if err := t.client.SetBuffer(text); err != nil {
		// Silently fail - could log error in production
	}

	// THEN try clipboard copy (optional, may fail gracefully)
	if err := t.copyToClipboard(text); err != nil {
		// Silently fail - clipboard copy is optional, buffer is already set
	}

	// Clear selection
	t.selection.Clear()

	// Exit TUI after yank (yank-and-cancel behavior)
	return true
}

// buildRenderedContent builds the full content array with rendered gutters
// This is needed for gutter stripping during extraction
func (t *TUI) buildRenderedContent() []string {
	rendered := make([]string, len(t.content))
	for i, line := range t.content {
		// Check if line already has gutter (for tests)
		if strings.Contains(line, "│") {
			// Content already has gutter, use as-is
			rendered[i] = line
		} else {
			// Content is plain text, add gutter
			gutter := t.formatter.RenderGutter(i+1, t.cursorLine+1)
			rendered[i] = gutter + line
		}
	}
	return rendered
}

// copyToClipboard copies text to system clipboard via copy_stdin.sh
func (t *TUI) copyToClipboard(text string) error {
	// Find copy_stdin.sh script
	// Try multiple possible locations
	possiblePaths := []string{
		"scripts/copy_stdin.sh",        // From project root
		"../../scripts/copy_stdin.sh",  // From internal/ui directory (for tests)
		"/usr/local/bin/copy_stdin.sh", // System-wide install
	}

	var scriptPath string
	for _, path := range possiblePaths {
		if _, err := os.Stat(path); err == nil {
			scriptPath = path
			break
		}
	}

	if scriptPath == "" {
		return fmt.Errorf("copy_stdin.sh not found")
	}

	cmd := exec.Command(scriptPath)
	cmd.Stdin = bytes.NewBufferString(text)

	return cmd.Run()
}
