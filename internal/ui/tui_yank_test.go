package ui

import (
	"testing"

	vmode "github.com/shitcoding/tmux_yankee/internal/mode"
	"github.com/shitcoding/tmux_yankee/internal/selection"
)

// TestYankCharWiseSelection tests character-wise selection yank extraction
func TestYankCharWiseSelection(t *testing.T) {
	// Setup: TUI with multi-line content
	content := []string{
		"line 1",
		"line 2",
		"line 3",
	}

	tui := NewTUI("test-pane", content, "absolute")

	// Activate character-wise visual mode at line 0, col 0
	pos := selection.Pos{Line: 0, Col: 0}
	tui.modeMachine.Handle(vmode.EventToggleVisualChar, pos)

	// Move cursor to line 1, col 4 (select "line 1\nline")
	tui.cursorLine = 1
	tui.cursorCol = 4
	endPos := selection.Pos{Line: 1, Col: 4}
	tui.modeMachine.OnCursorMoved(endPos)

	// Mock tmux client to capture buffer content
	mockClient := &mockTmuxClient{}
	tui.client = mockClient

	// Execute yank
	shouldQuit := tui.yank()

	// Assert: should quit after yank
	if !shouldQuit {
		t.Errorf("Expected yank to return true (quit), got false")
	}

	// Assert: mode machine should return to Normal mode
	if tui.modeMachine.Mode() != vmode.Normal {
		t.Errorf("Expected mode to be Normal after yank, got %v", tui.modeMachine.Mode())
	}

	// Assert: region should be empty (KindNone)
	region := tui.modeMachine.Region()
	if region.Kind != selection.KindNone {
		t.Errorf("Expected region kind to be KindNone after yank, got %v", region.Kind)
	}

	// Assert: extracted text should be "line 1\nline" (char-wise from 0,0 to 1,4)
	expectedText := "line 1\nline"
	if mockClient.bufferContent != expectedText {
		t.Errorf("Expected buffer content %q, got %q", expectedText, mockClient.bufferContent)
	}
}

// TestYankLineWiseSelection tests line-wise selection yank extraction
func TestYankLineWiseSelection(t *testing.T) {
	// Setup: TUI with multi-line content
	content := []string{
		"line 1",
		"line 2",
		"line 3",
	}

	tui := NewTUI("test-pane", content, "absolute")

	// Activate line-wise visual mode at line 0
	pos := selection.Pos{Line: 0, Col: 0}
	tui.modeMachine.Handle(vmode.EventToggleVisualLine, pos)

	// Move cursor to line 2 (select lines 0-2)
	tui.cursorLine = 2
	endPos := selection.Pos{Line: 2, Col: 0}
	tui.modeMachine.OnCursorMoved(endPos)

	// Mock tmux client to capture buffer content
	mockClient := &mockTmuxClient{}
	tui.client = mockClient

	// Execute yank
	shouldQuit := tui.yank()

	// Assert: should quit after yank
	if !shouldQuit {
		t.Errorf("Expected yank to return true (quit), got false")
	}

	// Assert: mode machine should return to Normal mode
	if tui.modeMachine.Mode() != vmode.Normal {
		t.Errorf("Expected mode to be Normal after yank, got %v", tui.modeMachine.Mode())
	}

	// Assert: extracted text should be all three lines (line-wise)
	expectedText := "line 1\nline 2\nline 3"
	if mockClient.bufferContent != expectedText {
		t.Errorf("Expected buffer content %q, got %q", expectedText, mockClient.bufferContent)
	}
}

// TestYankNoSelection tests that yank without selection returns false
func TestYankNoSelection(t *testing.T) {
	content := []string{"line 1"}
	tui := NewTUI("test-pane", content, "absolute")

	// Mock tmux client
	mockClient := &mockTmuxClient{}
	tui.client = mockClient

	// Execute yank without activating selection
	shouldQuit := tui.yank()

	// Assert: should NOT quit (no selection)
	if shouldQuit {
		t.Errorf("Expected yank to return false (no selection), got true")
	}

	// Assert: buffer should be empty (no yank occurred)
	if mockClient.bufferContent != "" {
		t.Errorf("Expected empty buffer, got %q", mockClient.bufferContent)
	}
}

// mockTmuxClient is a mock implementation of tmux.Client for testing
type mockTmuxClient struct {
	bufferContent string
	setBufferErr  error
}

func (m *mockTmuxClient) SetBuffer(text string) error {
	m.bufferContent = text
	return m.setBufferErr
}

func (m *mockTmuxClient) CapturePane(paneID string, start, end int) ([]string, error) {
	return nil, nil
}

func (m *mockTmuxClient) GetFormatVar(paneID, formatVar string) (string, error) {
	return "", nil
}

func (m *mockTmuxClient) GetHistorySize(paneID string) (int, error) {
	return 0, nil
}

func (m *mockTmuxClient) GetScrollPosition(paneID string) (int, error) {
	return 0, nil
}
