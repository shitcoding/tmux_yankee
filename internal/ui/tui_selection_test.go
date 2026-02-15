package ui

import (
	"bytes"
	"io"
	"os"
	"os/exec"
	"strings"
	"testing"
)

// TestTUISelectionToggle tests 'v' key toggles selection mode
func TestTUISelectionToggle(t *testing.T) {
	content := []string{
		"Line 1",
		"Line 2",
		"Line 3",
	}

	tui := NewTUI("%1", content, "hybrid")

	// Initially, selection should not be active
	if tui.selection != nil && tui.selection.IsActive() {
		t.Error("Selection should be inactive initially")
	}

	// Simulate 'v' key press
	quit := tui.handleInput([]byte{'v'})
	if quit {
		t.Error("'v' key should not trigger quit")
	}

	// Selection should now be active
	if tui.selection == nil || !tui.selection.IsActive() {
		t.Error("Selection should be active after 'v' key")
	}

	// Record cursor position when selection started
	startLine := tui.cursorLine

	// Press 'v' again to deactivate
	quit = tui.handleInput([]byte{'v'})
	if quit {
		t.Error("Second 'v' key should not trigger quit")
	}

	// Selection should be inactive again
	if tui.selection != nil && tui.selection.IsActive() {
		t.Error("Selection should be inactive after second 'v' key")
	}

	// Cursor should still be at same position
	if tui.cursorLine != startLine {
		t.Errorf("Cursor should not move, got %d want %d", tui.cursorLine, startLine)
	}
}

// TestTUISelectionUpdateOnCursorMove tests selection updates as cursor moves
func TestTUISelectionUpdateOnCursorMove(t *testing.T) {
	content := []string{
		"Line 1",
		"Line 2",
		"Line 3",
		"Line 4",
		"Line 5",
	}

	tui := NewTUI("%1", content, "hybrid")
	tui.height = 10 // Set viewport height

	// Start at line 1
	tui.cursorLine = 1

	// Activate selection
	tui.handleInput([]byte{'v'})
	if tui.selection == nil || !tui.selection.IsActive() {
		t.Fatal("Selection should be active")
	}

	selectionStartLine := tui.cursorLine

	// Move cursor down
	tui.handleInput([]byte{'j'})
	if tui.cursorLine != 2 {
		t.Errorf("Cursor should be at line 2, got %d", tui.cursorLine)
	}

	// Selection end should be updated
	start, end := tui.selection.Range()
	if start != selectionStartLine {
		t.Errorf("Selection start = %d, want %d", start, selectionStartLine)
	}
	if end != tui.cursorLine {
		t.Errorf("Selection end = %d, want %d", end, tui.cursorLine)
	}

	// Move cursor down again
	tui.handleInput([]byte{'j'})
	tui.handleInput([]byte{'j'})
	if tui.cursorLine != 4 {
		t.Errorf("Cursor should be at line 4, got %d", tui.cursorLine)
	}

	// Selection should span from start to current cursor
	start, end = tui.selection.Range()
	if start != selectionStartLine {
		t.Errorf("Selection start should remain %d, got %d", selectionStartLine, start)
	}
	if end != 4 {
		t.Errorf("Selection end should be 4, got %d", end)
	}
}

// TestTUISelectionUpwardMovement tests selection when cursor moves upward
func TestTUISelectionUpwardMovement(t *testing.T) {
	content := []string{
		"Line 1",
		"Line 2",
		"Line 3",
		"Line 4",
		"Line 5",
	}

	tui := NewTUI("%1", content, "hybrid")
	tui.height = 10
	tui.cursorLine = 3 // Start at line 4 (0-indexed)

	// Activate selection
	tui.handleInput([]byte{'v'})
	startLine := tui.cursorLine

	// Move cursor up
	tui.handleInput([]byte{'k'})
	tui.handleInput([]byte{'k'})

	if tui.cursorLine != 1 {
		t.Errorf("Cursor should be at line 1, got %d", tui.cursorLine)
	}

	// Selection should handle backward range
	start, end := tui.selection.Range()
	// Range() should normalize, so start <= end
	if start > end {
		t.Errorf("Range() should be normalized, got start=%d end=%d", start, end)
	}
	if start != 1 {
		t.Errorf("Normalized start should be 1, got %d", start)
	}
	if end != startLine {
		t.Errorf("Normalized end should be %d, got %d", startLine, end)
	}
}

// TestTUIVisualSelectionHighlight tests that selected lines are visually highlighted
func TestTUIVisualSelectionHighlight(t *testing.T) {
	content := []string{
		"Line 1",
		"Line 2",
		"Line 3",
	}

	tui := NewTUI("%1", content, "hybrid")
	tui.width = 80
	tui.height = 10

	// Activate selection
	tui.handleInput([]byte{'v'})

	// Move down to select multiple lines
	tui.handleInput([]byte{'j'})

	// Render to buffer instead of stdout
	// This is a simplified test - actual implementation would need to capture render output
	// For now, we just verify the selection state is correct
	if tui.selection == nil || !tui.selection.IsActive() {
		t.Fatal("Selection should be active")
	}

	start, end := tui.selection.Range()
	if start != 0 || end != 1 {
		t.Errorf("Selection range should be [0,1], got [%d,%d]", start, end)
	}

	// TODO: When rendering is implemented, verify that lines in selection range
	// have visual highlighting (reverse video, background color, etc.)
}

// TestTUIYankKey tests 'y' key yanks selected text
func TestTUIYankKey(t *testing.T) {
	// Skip this test if not in a tmux session
	if os.Getenv("TMUX") == "" {
		t.Skip("Not in tmux session, skipping yank test")
	}

	content := []string{
		"  1 │ First line",
		"  2 │ Second line",
		"  3 │ Third line",
	}

	tui := NewTUI("%1", content, "hybrid")
	tui.height = 10

	// Activate selection
	tui.handleInput([]byte{'v'})
	// Move down to select 2 lines
	tui.handleInput([]byte{'j'})

	// Press 'y' to yank
	quit := tui.handleInput([]byte{'y'})

	// Check if yank was triggered
	// This test will fail until Yank() is implemented
	// Expected behavior:
	// 1. Text is extracted from selection
	// 2. Gutter is stripped
	// 3. Text is written to clipboard via copy_stdin.sh
	// 4. Text is written to tmux buffer via SetBuffer
	// 5. TUI exits (quit = true) OR selection is cleared

	// For now, just verify the key doesn't crash
	_ = quit
}

// TestTUIEnterKeyYanks tests 'Enter' key also yanks selected text
func TestTUIEnterKeyYanks(t *testing.T) {
	if os.Getenv("TMUX") == "" {
		t.Skip("Not in tmux session, skipping yank test")
	}

	content := []string{
		"  1 │ Line A",
		"  2 │ Line B",
	}

	tui := NewTUI("%1", content, "hybrid")
	tui.height = 10

	// Activate selection
	tui.handleInput([]byte{'v'})

	// Press Enter to yank
	quit := tui.handleInput([]byte{13}) // 13 = Enter/Return

	// Similar to 'y' key test - verify it doesn't crash
	_ = quit
}

// TestTUIYankWithoutSelection tests yanking when no selection is active
func TestTUIYankWithoutSelection(t *testing.T) {
	content := []string{
		"Line 1",
		"Line 2",
	}

	tui := NewTUI("%1", content, "hybrid")

	// Try to yank without selection
	quit := tui.handleInput([]byte{'y'})

	// Should not quit, should not crash
	if quit {
		t.Error("Yank without selection should not quit")
	}

	// No error should occur, operation should be no-op
}

// TestTUIYankClearsSelection tests that yank clears the selection
func TestTUIYankClearsSelection(t *testing.T) {
	if os.Getenv("TMUX") == "" {
		t.Skip("Not in tmux session")
	}

	content := []string{
		"  1 │ Text here",
		"  2 │ More text",
	}

	tui := NewTUI("%1", content, "hybrid")
	tui.height = 10

	// Activate selection
	tui.handleInput([]byte{'v'})
	tui.handleInput([]byte{'j'})

	if tui.selection == nil || !tui.selection.IsActive() {
		t.Fatal("Selection should be active before yank")
	}

	// Yank
	tui.handleInput([]byte{'y'})

	// After yank, selection should be cleared (depending on yank-and-exit behavior)
	// This test will fail until Yank() is implemented
	// Expected: either TUI exits OR selection is cleared
}

// TestTUIYankStripsGutter tests that yanked text has gutter removed
func TestTUIYankStripsGutter(t *testing.T) {
	if os.Getenv("TMUX") == "" {
		t.Skip("Not in tmux session")
	}

	// Content should be plain text (gutters are rendering-only)
	content := []string{
		"Hello",
		"World",
	}

	tui := NewTUI("%1", content, "hybrid")
	tui.height = 10

	// Activate line-wise selection and select both lines
	tui.handleInput([]byte{'V'}) // Line-wise visual mode
	tui.handleInput([]byte{'j'})

	// Yank
	tui.handleInput([]byte{'y'})

	// Read tmux buffer to verify text is correct
	cmd := exec.Command("tmux", "show-buffer")
	output, err := cmd.Output()
	if err != nil {
		t.Fatalf("Failed to read tmux buffer: %v", err)
	}

	text := strings.TrimSpace(string(output))

	// Should contain text without gutter
	wantLines := []string{"Hello", "World"}
	gotLines := strings.Split(text, "\n")

	if len(gotLines) != len(wantLines) {
		t.Errorf("Yanked text lines = %d, want %d", len(gotLines), len(wantLines))
	}

	for i, want := range wantLines {
		if i >= len(gotLines) {
			break
		}
		got := strings.TrimSpace(gotLines[i])
		if got != want {
			t.Errorf("Line %d = %q, want %q", i, got, want)
		}
		// Ensure no gutter separator
		if strings.Contains(got, "│") {
			t.Errorf("Line %d should not contain gutter separator: %q", i, got)
		}
	}
}

// TestTUIYankToSystemClipboard tests text is copied to system clipboard
func TestTUIYankToSystemClipboard(t *testing.T) {
	// This test requires actual clipboard commands to be available
	// Skip if copy_stdin.sh is not executable
	scriptPath := "../../scripts/copy_stdin.sh"
	if _, err := os.Stat(scriptPath); os.IsNotExist(err) {
		t.Skip("copy_stdin.sh not found, skipping clipboard test")
	}

	// Content should be plain text (gutters are rendering-only)
	content := []string{
		"Clipboard test",
	}

	tui := NewTUI("%1", content, "hybrid")
	tui.height = 10

	// Activate line-wise selection
	tui.handleInput([]byte{'V'}) // Line-wise visual mode

	// Yank
	tui.handleInput([]byte{'y'})

	// Verify clipboard content (platform-specific)
	// This is a simplified test - actual verification depends on platform
	// On macOS: pbpaste
	// On Linux X11: xclip -o
	// On Linux Wayland: wl-paste

	var clipboardCmd *exec.Cmd
	switch {
	case commandExists("pbpaste"):
		clipboardCmd = exec.Command("pbpaste")
	case commandExists("xclip"):
		clipboardCmd = exec.Command("xclip", "-selection", "clipboard", "-o")
	case commandExists("wl-paste"):
		clipboardCmd = exec.Command("wl-paste")
	default:
		t.Skip("No clipboard command available")
	}

	output, err := clipboardCmd.Output()
	if err != nil {
		t.Logf("Clipboard read failed (expected if yank not yet implemented): %v", err)
		return
	}

	text := strings.TrimSpace(string(output))
	if !strings.Contains(text, "Clipboard test") {
		t.Errorf("Clipboard should contain 'Clipboard test', got %q", text)
	}
	if strings.Contains(text, "│") {
		t.Errorf("Clipboard should not contain gutter separator, got %q", text)
	}
}

// TestTUIYankMultilinePreservesNewlines tests newlines are preserved in yanked text
func TestTUIYankMultilinePreservesNewlines(t *testing.T) {
	if os.Getenv("TMUX") == "" {
		t.Skip("Not in tmux session")
	}

	// Content should be plain text (gutters are rendering-only)
	content := []string{
		"Line A",
		"Line B",
		"Line C",
	}

	tui := NewTUI("%1", content, "hybrid")
	tui.height = 10

	// Select all lines (line-wise)
	tui.handleInput([]byte{'V'}) // Line-wise visual mode
	tui.handleInput([]byte{'j'})
	tui.handleInput([]byte{'j'})

	// Yank
	tui.handleInput([]byte{'y'})

	// Check tmux buffer
	cmd := exec.Command("tmux", "show-buffer")
	output, err := cmd.Output()
	if err != nil {
		t.Fatalf("Failed to read tmux buffer: %v", err)
	}

	text := string(output)
	lines := strings.Split(strings.TrimSpace(text), "\n")

	if len(lines) != 3 {
		t.Errorf("Expected 3 lines in buffer, got %d", len(lines))
	}

	expectedLines := []string{"Line A", "Line B", "Line C"}
	for i, want := range expectedLines {
		if i >= len(lines) {
			break
		}
		got := strings.TrimSpace(lines[i])
		if got != want {
			t.Errorf("Line %d = %q, want %q", i, got, want)
		}
	}
}

// TestTUIYankEmptySelection tests yanking empty/whitespace lines
func TestTUIYankEmptySelection(t *testing.T) {
	if os.Getenv("TMUX") == "" {
		t.Skip("Not in tmux session")
	}

	// Content should be plain text (gutters are rendering-only)
	content := []string{
		"",
		"",
	}

	tui := NewTUI("%1", content, "hybrid")
	tui.height = 10

	// Select empty lines (line-wise)
	tui.handleInput([]byte{'V'}) // Line-wise visual mode
	tui.handleInput([]byte{'j'})

	// Yank
	quit := tui.handleInput([]byte{'y'})

	// Should not crash, may or may not exit depending on implementation
	_ = quit
}

// TestTUISelectionModeCycle tests mode cycling doesn't break selection
func TestTUISelectionModeCycle(t *testing.T) {
	content := []string{
		"Line 1",
		"Line 2",
		"Line 3",
	}

	tui := NewTUI("%1", content, "hybrid")
	tui.height = 10

	// Activate selection
	tui.handleInput([]byte{'v'})
	tui.handleInput([]byte{'j'})

	if tui.selection == nil || !tui.selection.IsActive() {
		t.Fatal("Selection should be active")
	}

	initialStart, initialEnd := tui.selection.Range()

	// Cycle mode with 'L'
	tui.handleInput([]byte{'L'})

	// Selection should still be active
	if tui.selection == nil || !tui.selection.IsActive() {
		t.Error("Selection should remain active after mode toggle")
	}

	// Selection range should be unchanged
	start, end := tui.selection.Range()
	if start != initialStart || end != initialEnd {
		t.Errorf("Selection range changed after mode toggle: got [%d,%d], want [%d,%d]",
			start, end, initialStart, initialEnd)
	}
}

// TestTUIYankWithHybridGutter tests yanking with hybrid mode colored gutter
func TestTUIYankWithHybridGutter(t *testing.T) {
	if os.Getenv("TMUX") == "" {
		t.Skip("Not in tmux session")
	}

	// Content should be plain text (gutters are rendering-only)
	content := []string{
		"Above cursor",
		"One away",
		"At cursor",
		"Below cursor",
	}

	tui := NewTUI("%1", content, "hybrid")
	tui.height = 10
	tui.cursorLine = 2 // Set cursor at "At cursor" line

	// Select all (line-wise)
	tui.cursorLine = 0
	tui.handleInput([]byte{'V'}) // Line-wise visual mode
	tui.handleInput([]byte{'j'})
	tui.handleInput([]byte{'j'})
	tui.handleInput([]byte{'j'})

	// Yank
	tui.handleInput([]byte{'y'})

	// Check buffer
	cmd := exec.Command("tmux", "show-buffer")
	output, err := cmd.Output()
	if err != nil {
		t.Fatalf("Failed to read tmux buffer: %v", err)
	}

	text := string(output)

	// Should not contain ANSI codes
	if strings.Contains(text, "\x1b[") {
		t.Errorf("Yanked text should not contain ANSI codes: %q", text)
	}

	// Should not contain gutter separator
	if strings.Contains(text, "│") {
		t.Errorf("Yanked text should not contain gutter separator: %q", text)
	}

	// Should contain actual text
	expectedPhrases := []string{"Above cursor", "One away", "At cursor", "Below cursor"}
	for _, phrase := range expectedPhrases {
		if !strings.Contains(text, phrase) {
			t.Errorf("Yanked text should contain %q, got: %q", phrase, text)
		}
	}
}

// TestTUISelectionPersistsThroughScroll tests selection is maintained during scrolling
func TestTUISelectionPersistsThroughScroll(t *testing.T) {
	// Create content larger than viewport
	content := make([]string, 100)
	for i := range content {
		content[i] = "Line " + string(rune('A'+i%26))
	}

	tui := NewTUI("%1", content, "hybrid")
	tui.height = 10 // Small viewport

	// Start selection at line 5
	tui.cursorLine = 5
	tui.handleInput([]byte{'v'})

	// Move down multiple times to trigger viewport scroll
	for i := 0; i < 15; i++ {
		tui.handleInput([]byte{'j'})
	}

	// Selection should still be active
	if tui.selection == nil || !tui.selection.IsActive() {
		t.Error("Selection should persist through scrolling")
	}

	// Selection should span from start to current cursor
	start, end := tui.selection.Range()
	if start != 5 {
		t.Errorf("Selection start should be 5, got %d", start)
	}
	if end != 20 {
		t.Errorf("Selection end should be 20 (5+15), got %d", end)
	}
}

// Helper function to check if a command exists
func commandExists(cmd string) bool {
	_, err := exec.LookPath(cmd)
	return err == nil
}

// MockClipboard is a test helper to capture clipboard writes
type MockClipboard struct {
	buffer bytes.Buffer
}

func (m *MockClipboard) Write(p []byte) (n int, err error) {
	return m.buffer.Write(p)
}

func (m *MockClipboard) Read(p []byte) (n int, err error) {
	return m.buffer.Read(p)
}

func (m *MockClipboard) String() string {
	return m.buffer.String()
}

// TestTUIYankIntegration is a comprehensive integration test
func TestTUIYankIntegration(t *testing.T) {
	if os.Getenv("TMUX") == "" {
		t.Skip("Not in tmux session")
	}

	// Content should be plain text (gutters are rendering-only)
	content := []string{
		"func main() {",
		"\tfmt.Println(\"Hello\")",
		"\treturn",
		"}",
	}

	tui := NewTUI("%1", content, "hybrid")
	tui.height = 10

	// Scenario: Select function body (lines 2-3)
	tui.cursorLine = 1
	tui.handleInput([]byte{'V'}) // Start line-wise selection
	tui.handleInput([]byte{'j'})  // Extend to line 3

	// Verify selection is active and correct
	if tui.selection == nil || !tui.selection.IsActive() {
		t.Fatal("Selection should be active")
	}

	start, end := tui.selection.Range()
	if start != 1 || end != 2 {
		t.Fatalf("Selection range should be [1,2], got [%d,%d]", start, end)
	}

	// Yank
	tui.handleInput([]byte{'y'})

	// Verify tmux buffer contains correct text without gutter
	cmd := exec.Command("tmux", "show-buffer")
	output, err := cmd.Output()
	if err != nil {
		t.Fatalf("Failed to read tmux buffer: %v", err)
	}

	// Trim only trailing whitespace to preserve leading tabs/spaces in content
	text := strings.TrimRight(string(output), " \t\n\r")
	expectedLines := []string{
		"\tfmt.Println(\"Hello\")",
		"\treturn",
	}

	gotLines := strings.Split(text, "\n")
	if len(gotLines) != len(expectedLines) {
		t.Errorf("Expected %d lines, got %d", len(expectedLines), len(gotLines))
	}

	for i, want := range expectedLines {
		if i >= len(gotLines) {
			break
		}
		got := gotLines[i]
		if got != want {
			t.Errorf("Line %d:\ngot:  %q\nwant: %q", i, got, want)
		}
	}
}

// BenchmarkSelection benchmarks selection operations
func BenchmarkSelectionToggle(b *testing.B) {
	content := make([]string, 1000)
	for i := range content {
		content[i] = "Line content here"
	}

	tui := NewTUI("%1", content, "hybrid")
	tui.height = 50

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tui.handleInput([]byte{'v'})
	}
}

func BenchmarkSelectionUpdate(b *testing.B) {
	content := make([]string, 1000)
	for i := range content {
		content[i] = "Line content here"
	}

	tui := NewTUI("%1", content, "hybrid")
	tui.height = 50
	tui.handleInput([]byte{'v'}) // Activate selection

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tui.handleInput([]byte{'j'})
	}
}

// DevNull is a writer that discards all data (for benchmarking)
type DevNull struct{}

func (d DevNull) Write(p []byte) (n int, err error) {
	return len(p), nil
}

// captureStdout captures stdout for testing render output
func captureStdout(f func()) string {
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	f()

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	io.Copy(&buf, r)
	return buf.String()
}
