package ui_test

import (
	"fmt"
	"testing"

	"github.com/shitcoding/tmux_yankee/internal/config"
	"github.com/shitcoding/tmux_yankee/internal/input"
	"github.com/shitcoding/tmux_yankee/internal/theme"
	"github.com/shitcoding/tmux_yankee/internal/ui"
)

func newMouseTestTUI(content []string) *ui.TUI {
	cfg := config.Settings{
		PaneID:          "%0",
		Mode:            config.LineNumberModeHybrid,
		ScrollbackLines: 2000,
		Palette:         theme.Presets[theme.ThemeDefault],
		ToggleModeKey:   'L',
		CopyTarget:      config.CopyTargetBoth,
		ExitOnYank:      true,
		StartPosition:   config.StartPositionBottom,
	}
	return ui.NewTUI(cfg, content)
}

// makeLines returns n lines of content for testing.
func makeLines(n int) []string {
	lines := make([]string, n)
	for i := range lines {
		lines[i] = fmt.Sprintf("line%d", i)
	}
	return lines
}

// --- Cursor-only fallback path (content fits in viewport / height not set) ---

func TestMouseScroll_WheelUpMovesCursorUp(t *testing.T) {
	content := []string{"line0", "line1", "line2", "line3", "line4"}
	tui := newMouseTestTUI(content)
	// Start at bottom (line 4), height not set → cursor-only fallback
	if tui.CursorLine() != 4 {
		t.Fatalf("expected start at 4, got %d", tui.CursorLine())
	}
	cmd := input.Command{Type: input.CommandMouseScroll, ScrollDirection: input.ScrollUp}
	quit := tui.HandleCommand(cmd)
	if quit {
		t.Fatal("wheel-up should not quit")
	}
	if tui.CursorLine() != 3 {
		t.Fatalf("expected cursor at 3 after wheel-up, got %d", tui.CursorLine())
	}
}

func TestMouseScroll_WheelDownMovesCursorDown(t *testing.T) {
	content := []string{"line0", "line1", "line2", "line3", "line4"}
	tui := newMouseTestTUI(content)
	// height not set → cursor-only fallback
	tui.SetCursorLine(2)
	cmd := input.Command{Type: input.CommandMouseScroll, ScrollDirection: input.ScrollDown}
	quit := tui.HandleCommand(cmd)
	if quit {
		t.Fatal("wheel-down in middle should not quit")
	}
	if tui.CursorLine() != 3 {
		t.Fatalf("expected cursor at 3, got %d", tui.CursorLine())
	}
}

func TestMouseScroll_WheelDownAtBottomExits(t *testing.T) {
	content := []string{"line0", "line1", "line2"}
	tui := newMouseTestTUI(content)
	// Start at bottom (line 2), height not set → cursor-only fallback
	cmd := input.Command{Type: input.CommandMouseScroll, ScrollDirection: input.ScrollDown}
	quit := tui.HandleCommand(cmd)
	if !quit {
		t.Fatal("wheel-down at bottom line should return quit=true")
	}
}

func TestMouseScroll_WheelUpAtTopDoesNotQuit(t *testing.T) {
	content := []string{"line0", "line1"}
	tui := newMouseTestTUI(content)
	tui.SetCursorLine(0)
	cmd := input.Command{Type: input.CommandMouseScroll, ScrollDirection: input.ScrollUp}
	quit := tui.HandleCommand(cmd)
	if quit {
		t.Fatal("wheel-up at top should not quit")
	}
	if tui.CursorLine() != 0 {
		t.Fatalf("cursor should stay at 0, got %d", tui.CursorLine())
	}
}

// --- Viewport-scroll path (content taller than terminal, height set) ---

func TestMouseScroll_ViewportScrollUp_ShiftsViewport(t *testing.T) {
	// 100 lines, height=20 → viewport-scroll path
	tui := newMouseTestTUI(makeLines(100))
	tui.SetHeight(20)
	// After SetHeight, clampViewportAndCursor places viewportTop at 80 (cursor=99, lastLine=99)
	if tui.ViewportTop() != 80 {
		t.Fatalf("expected initial viewportTop=80, got %d", tui.ViewportTop())
	}

	cmd := input.Command{Type: input.CommandMouseScroll, ScrollDirection: input.ScrollUp}
	quit := tui.HandleCommand(cmd)
	if quit {
		t.Fatal("scroll-up should not quit")
	}
	// Viewport shifted up by scrollStep=3: 80 → 77
	if tui.ViewportTop() != 77 {
		t.Errorf("expected viewportTop=77, got %d", tui.ViewportTop())
	}
	// Cursor pinned to bottom of new viewport: 77+20-1=96
	if tui.CursorLine() != 96 {
		t.Errorf("expected cursor=96, got %d", tui.CursorLine())
	}
}

func TestMouseScroll_ViewportScrollDown_ShiftsViewport(t *testing.T) {
	// 100 lines, height=20, viewport at 50
	tui := newMouseTestTUI(makeLines(100))
	tui.SetHeight(20)
	tui.SetCursorLine(50)
	tui.SetViewportTop(50)

	cmd := input.Command{Type: input.CommandMouseScroll, ScrollDirection: input.ScrollDown}
	quit := tui.HandleCommand(cmd)
	if quit {
		t.Fatal("scroll-down in middle should not quit")
	}
	// Viewport shifted down by scrollStep=3: 50 → 53
	if tui.ViewportTop() != 53 {
		t.Errorf("expected viewportTop=53, got %d", tui.ViewportTop())
	}
	// Cursor was at 50 which is now below new viewportTop (53) → pinned to 53
	if tui.CursorLine() != 53 {
		t.Errorf("expected cursor=53, got %d", tui.CursorLine())
	}
}

func TestMouseScroll_ViewportScrollDown_AtBottom_Exits(t *testing.T) {
	// 100 lines, height=20 → maxViewportTop=80; cursor and viewport start at bottom
	tui := newMouseTestTUI(makeLines(100))
	tui.SetHeight(20)
	// After SetHeight with cursor at lastLine=99: viewportTop=80=maxViewportTop
	if tui.ViewportTop() != 80 {
		t.Fatalf("expected viewportTop=80 (at bottom), got %d", tui.ViewportTop())
	}
	cmd := input.Command{Type: input.CommandMouseScroll, ScrollDirection: input.ScrollDown}
	quit := tui.HandleCommand(cmd)
	if !quit {
		t.Fatal("scroll-down at bottom of content should return quit=true")
	}
}

func TestMouseScroll_ViewportScrollUp_AtTop_Clamps(t *testing.T) {
	// 100 lines, height=20, viewport already at top
	tui := newMouseTestTUI(makeLines(100))
	tui.SetHeight(20)
	tui.SetCursorLine(0)
	tui.SetViewportTop(0)

	cmd := input.Command{Type: input.CommandMouseScroll, ScrollDirection: input.ScrollUp}
	quit := tui.HandleCommand(cmd)
	if quit {
		t.Fatal("scroll-up at top should not quit")
	}
	// Viewport stays at 0 (clamped)
	if tui.ViewportTop() != 0 {
		t.Errorf("expected viewportTop=0, got %d", tui.ViewportTop())
	}
	// Cursor stays at 0
	if tui.CursorLine() != 0 {
		t.Errorf("expected cursor=0, got %d", tui.CursorLine())
	}
}

func TestMouseScroll_ViewportScrollDown_CursorInMiddle_NoPin(t *testing.T) {
	// Cursor in middle of viewport, scroll down: viewport shifts but cursor doesn't need pinning
	tui := newMouseTestTUI(makeLines(100))
	tui.SetHeight(20)
	// Place viewport at 20, cursor at 30 (comfortably in viewport 20-39)
	tui.SetViewportTop(20)
	tui.SetCursorLine(30)

	cmd := input.Command{Type: input.CommandMouseScroll, ScrollDirection: input.ScrollDown}
	quit := tui.HandleCommand(cmd)
	if quit {
		t.Fatal("should not quit")
	}
	// viewport: 20 → 23
	if tui.ViewportTop() != 23 {
		t.Errorf("expected viewportTop=23, got %d", tui.ViewportTop())
	}
	// cursor (30) >= viewportTop (23) → no pin needed
	if tui.CursorLine() != 30 {
		t.Errorf("expected cursor unchanged at 30, got %d", tui.CursorLine())
	}
}
