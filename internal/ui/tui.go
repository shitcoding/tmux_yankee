package ui

import (
	"fmt"
	"io"
	"os"
	"strings"

	"golang.org/x/term"
)

// TUI represents the terminal UI
type TUI struct {
	paneID      string
	content     []string
	mode        string
	cursorLine  int
	viewportTop int
	width       int
	height      int
	oldState    *term.State
}

// NewTUI creates a new TUI instance
func NewTUI(paneID string, content []string, mode string) *TUI {
	return &TUI{
		paneID:      paneID,
		content:     content,
		mode:        mode,
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
	buf := make([]byte, 3)
	for {
		n, err := os.Stdin.Read(buf)
		if err != nil {
			if err == io.EOF {
				break
			}
			return fmt.Errorf("read failed: %w", err)
		}

		key := buf[:n]

		// Handle input
		quit := t.handleInput(key)
		if quit {
			break
		}

		// Re-render
		t.render()
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
	case len(key) == 1 && key[0] == 'j':
		t.moveCursorDown()
	case len(key) == 1 && key[0] == 'k':
		t.moveCursorUp()
	case len(key) == 1 && key[0] == 3: // Ctrl-C
		return true
	}
	return false
}

// moveCursorDown moves cursor down one line
func (t *TUI) moveCursorDown() {
	if t.cursorLine < len(t.content)-1 {
		t.cursorLine++
		// Adjust viewport if cursor moves off screen
		if t.cursorLine >= t.viewportTop+t.height {
			t.viewportTop++
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

	// Render visible lines
	for i := t.viewportTop; i < endLine; i++ {
		line := t.content[i]

		// Highlight cursor line
		if i == t.cursorLine {
			b.WriteString("\x1b[7m") // Reverse video
		}

		// Truncate line if too long
		if len(line) > t.width {
			line = line[:t.width]
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
