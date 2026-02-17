package ui_test

import (
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

func TestMouseScroll_WheelUpMovesCursorUp(t *testing.T) {
	content := []string{"line0", "line1", "line2", "line3", "line4"}
	tui := newMouseTestTUI(content)
	// Start at bottom (line 4)
	if tui.CursorLine() != 4 {
		t.Fatalf("expected start at 4, got %d", tui.CursorLine())
	}
	// Wheel-up should move cursor up (like k)
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
	// Move up from bottom first
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
	// Start at bottom (line 2)
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
