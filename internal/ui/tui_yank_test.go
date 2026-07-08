package ui

import (
	"testing"

	"github.com/shitcoding/tmux_yankee/internal/config"
	vmode "github.com/shitcoding/tmux_yankee/internal/mode"
	"github.com/shitcoding/tmux_yankee/internal/selection"
	"github.com/shitcoding/tmux_yankee/internal/theme"
)

// TestYankCharWiseSelection tests character-wise selection yank extraction
func TestYankCharWiseSelection(t *testing.T) {
	// Setup: TUI with multi-line content
	content := []string{
		"line 1",
		"line 2",
		"line 3",
	}

	tui := newTestTUI("test-pane", content, "absolute")

	// Activate character-wise visual mode at line 0, col 0
	pos := selection.Pos{Line: 0, Col: 0}
	tui.modeMachine.Handle(vmode.EventToggleVisualChar, pos)

	// Move cursor to line 1, col 3 (select "line 1\nline").
	// Col 3 is the 'e' in "line 2"; the endCol is inclusive, so
	// extracting from col 0 to col 3 yields "line" (4 characters).
	tui.cursorLine = 1
	tui.cursorCol = 3
	endPos := selection.Pos{Line: 1, Col: 3}
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

	// Assert: extracted text should be "line 1\nline" (char-wise from 0,0 to 1,3)
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

	tui := newTestTUI("test-pane", content, "absolute")

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
	tui := newTestTUI("test-pane", content, "absolute")

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
	bufferContent  string
	setBufferErr   error
	setBufferCalls int
}

func (m *mockTmuxClient) SetBuffer(text string) error {
	m.bufferContent = text
	m.setBufferCalls++
	return m.setBufferErr
}

// newTestTUIWithSettings creates a TUI for testing with explicit config.Settings.
func newTestTUIWithSettings(cfg config.Settings, content []string) *TUI {
	return NewTUI(cfg, content)
}

// TestYank_CopyTargetTmuxOnly verifies that CopyTarget=tmux sets tmux buffer but not clipboard.
func TestYank_CopyTargetTmuxOnly(t *testing.T) {
	content := []string{"line 1", "line 2"}
	cfg := config.Settings{
		PaneID:        "test-pane",
		Mode:          config.LineNumberModeAbsolute,
		Palette:       theme.Presets[theme.ThemeDefault],
		CopyTarget:    config.CopyTargetTmux,
		ExitOnYank:    true,
		StartPosition: config.StartPositionBottom,
		ToggleModeKey: 'L',
	}
	tui := newTestTUIWithSettings(cfg, content)

	// Activate line-wise selection on line 0
	pos := selection.Pos{Line: 0, Col: 0}
	tui.modeMachine.Handle(vmode.EventToggleVisualLine, pos)

	// Mock clients
	mockClient := &mockTmuxClient{}
	tui.client = mockClient
	clipboardCalled := false
	tui.clipboardFunc = func(text string) error {
		clipboardCalled = true
		return nil
	}

	tui.yank()

	if mockClient.setBufferCalls == 0 {
		t.Error("Expected SetBuffer to be called for CopyTargetTmux, but it was not")
	}
	if clipboardCalled {
		t.Error("Expected clipboard NOT to be called for CopyTargetTmux, but it was")
	}
}

// TestYank_CopyTargetClipboardOnly verifies that CopyTarget=clipboard calls clipboard but not tmux buffer.
func TestYank_CopyTargetClipboardOnly(t *testing.T) {
	content := []string{"line 1", "line 2"}
	cfg := config.Settings{
		PaneID:        "test-pane",
		Mode:          config.LineNumberModeAbsolute,
		Palette:       theme.Presets[theme.ThemeDefault],
		CopyTarget:    config.CopyTargetClipboard,
		ExitOnYank:    true,
		StartPosition: config.StartPositionBottom,
		ToggleModeKey: 'L',
	}
	tui := newTestTUIWithSettings(cfg, content)

	// Activate line-wise selection on line 0
	pos := selection.Pos{Line: 0, Col: 0}
	tui.modeMachine.Handle(vmode.EventToggleVisualLine, pos)

	// Mock clients
	mockClient := &mockTmuxClient{}
	tui.client = mockClient
	clipboardCalled := false
	tui.clipboardFunc = func(text string) error {
		clipboardCalled = true
		return nil
	}

	tui.yank()

	if mockClient.setBufferCalls != 0 {
		t.Errorf("Expected SetBuffer NOT to be called for CopyTargetClipboard, but it was called %d times", mockClient.setBufferCalls)
	}
	if !clipboardCalled {
		t.Error("Expected clipboard to be called for CopyTargetClipboard, but it was not")
	}
}

// TestYank_ExitOnYankFalse verifies that ExitOnYank=false keeps TUI in normal mode after yank.
func TestYank_ExitOnYankFalse(t *testing.T) {
	content := []string{"line 1", "line 2"}
	cfg := config.Settings{
		PaneID:        "test-pane",
		Mode:          config.LineNumberModeAbsolute,
		Palette:       theme.Presets[theme.ThemeDefault],
		CopyTarget:    config.CopyTargetTmux,
		ExitOnYank:    false,
		StartPosition: config.StartPositionBottom,
		ToggleModeKey: 'L',
	}
	tui := newTestTUIWithSettings(cfg, content)

	// Activate line-wise selection
	pos := selection.Pos{Line: 0, Col: 0}
	tui.modeMachine.Handle(vmode.EventToggleVisualLine, pos)

	// Mock client
	mockClient := &mockTmuxClient{}
	tui.client = mockClient
	tui.clipboardFunc = func(text string) error { return nil }

	shouldQuit := tui.yank()

	if shouldQuit {
		t.Error("Expected yank to return false (stay in TUI) when ExitOnYank=false, got true")
	}

	// Mode should be Normal after yank
	if tui.modeMachine.Mode() != vmode.Normal {
		t.Errorf("Expected mode to be Normal after yank with ExitOnYank=false, got %v", tui.modeMachine.Mode())
	}

	// Region should be cleared
	region := tui.modeMachine.Region()
	if region.Kind != selection.KindNone {
		t.Errorf("Expected region kind to be KindNone after yank, got %v", region.Kind)
	}
}

// TestTUI_StartPositionTop verifies cursorLine=0 when StartPosition=top.
func TestTUI_StartPositionTop(t *testing.T) {
	content := []string{"line 1", "line 2", "line 3", "line 4", "line 5"}
	cfg := config.Settings{
		PaneID:        "test-pane",
		Mode:          config.LineNumberModeAbsolute,
		Palette:       theme.Presets[theme.ThemeDefault],
		CopyTarget:    config.CopyTargetBoth,
		ExitOnYank:    true,
		StartPosition: config.StartPositionTop,
		ToggleModeKey: 'L',
	}
	tui := newTestTUIWithSettings(cfg, content)

	if tui.cursorLine != 0 {
		t.Errorf("Expected cursorLine=0 for StartPositionTop, got %d", tui.cursorLine)
	}
}

// TestTUI_StartPositionMiddle verifies cursorLine=len(content)/2 when StartPosition=middle.
func TestTUI_StartPositionMiddle(t *testing.T) {
	content := []string{"line 1", "line 2", "line 3", "line 4", "line 5"}
	cfg := config.Settings{
		PaneID:        "test-pane",
		Mode:          config.LineNumberModeAbsolute,
		Palette:       theme.Presets[theme.ThemeDefault],
		CopyTarget:    config.CopyTargetBoth,
		ExitOnYank:    true,
		StartPosition: config.StartPositionMiddle,
		ToggleModeKey: 'L',
	}
	tui := newTestTUIWithSettings(cfg, content)

	expected := (len(content) - 1) / 2
	if tui.cursorLine != expected {
		t.Errorf("Expected cursorLine=%d for StartPositionMiddle, got %d", expected, tui.cursorLine)
	}
}

// TestTUI_StartPositionBottom verifies cursorLine=last line when StartPosition=bottom (default).
func TestTUI_StartPositionBottom(t *testing.T) {
	content := []string{"line 1", "line 2", "line 3", "line 4", "line 5"}
	cfg := config.Settings{
		PaneID:        "test-pane",
		Mode:          config.LineNumberModeAbsolute,
		Palette:       theme.Presets[theme.ThemeDefault],
		CopyTarget:    config.CopyTargetBoth,
		ExitOnYank:    true,
		StartPosition: config.StartPositionBottom,
		ToggleModeKey: 'L',
	}
	tui := newTestTUIWithSettings(cfg, content)

	expected := len(content) - 1
	if tui.cursorLine != expected {
		t.Errorf("Expected cursorLine=%d for StartPositionBottom, got %d", expected, tui.cursorLine)
	}
}
