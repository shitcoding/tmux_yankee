package mode

import (
	"testing"

	"github.com/shitcoding/tmux_yankee/internal/selection"
)

func TestNewMachine(t *testing.T) {
	m := NewMachine()

	if m.Mode() != Normal {
		t.Errorf("NewMachine() mode = %v, want Normal", m.Mode())
	}

	region := m.Region()
	if region.Kind != selection.KindNone {
		t.Errorf("NewMachine() region.Kind = %v, want KindNone", region.Kind)
	}
}

func TestModeTransitions_NormalToVisualChar(t *testing.T) {
	m := NewMachine()
	cursor := selection.Pos{Line: 5, Col: 10}

	// Trigger visual character mode
	changed := m.Handle(EventToggleVisualChar, cursor)

	if !changed {
		t.Error("Handle(EventToggleVisualChar) should return true when transitioning from Normal")
	}

	if m.Mode() != VisualChar {
		t.Errorf("Mode() = %v, want VisualChar", m.Mode())
	}

	region := m.Region()
	if region.Kind != selection.KindChar {
		t.Errorf("Region().Kind = %v, want KindChar", region.Kind)
	}

	if region.Start != cursor {
		t.Errorf("Region().Start = %v, want %v", region.Start, cursor)
	}

	if region.End != cursor {
		t.Errorf("Region().End = %v, want %v (should start as zero-width)", region.End, cursor)
	}
}

func TestModeTransitions_VisualCharToNormal(t *testing.T) {
	m := NewMachine()
	cursor := selection.Pos{Line: 5, Col: 10}

	// Enter visual char mode
	m.Handle(EventToggleVisualChar, cursor)

	// Toggle again to return to Normal
	changed := m.Handle(EventToggleVisualChar, cursor)

	if !changed {
		t.Error("Handle(EventToggleVisualChar) should return true when transitioning to Normal")
	}

	if m.Mode() != Normal {
		t.Errorf("Mode() = %v, want Normal", m.Mode())
	}

	region := m.Region()
	if region.Kind != selection.KindNone {
		t.Errorf("Region().Kind = %v, want KindNone", region.Kind)
	}
}

func TestModeTransitions_NormalToVisualLine(t *testing.T) {
	m := NewMachine()
	cursor := selection.Pos{Line: 5, Col: 10}

	// Trigger visual line mode
	changed := m.Handle(EventToggleVisualLine, cursor)

	if !changed {
		t.Error("Handle(EventToggleVisualLine) should return true when transitioning from Normal")
	}

	if m.Mode() != VisualLine {
		t.Errorf("Mode() = %v, want VisualLine", m.Mode())
	}

	region := m.Region()
	if region.Kind != selection.KindLine {
		t.Errorf("Region().Kind = %v, want KindLine", region.Kind)
	}

	if region.Start != cursor {
		t.Errorf("Region().Start = %v, want %v", region.Start, cursor)
	}

	if region.End != cursor {
		t.Errorf("Region().End = %v, want %v (should start as zero-width)", region.End, cursor)
	}
}

func TestModeTransitions_VisualLineToNormal(t *testing.T) {
	m := NewMachine()
	cursor := selection.Pos{Line: 5, Col: 10}

	// Enter visual line mode
	m.Handle(EventToggleVisualLine, cursor)

	// Toggle again to return to Normal
	changed := m.Handle(EventToggleVisualLine, cursor)

	if !changed {
		t.Error("Handle(EventToggleVisualLine) should return true when transitioning to Normal")
	}

	if m.Mode() != Normal {
		t.Errorf("Mode() = %v, want Normal", m.Mode())
	}

	region := m.Region()
	if region.Kind != selection.KindNone {
		t.Errorf("Region().Kind = %v, want KindNone", region.Kind)
	}
}

func TestModeTransitions_VisualCharToVisualLine(t *testing.T) {
	m := NewMachine()
	cursor := selection.Pos{Line: 5, Col: 10}

	// Enter visual char mode
	m.Handle(EventToggleVisualChar, cursor)

	// Move cursor to create a region
	m.OnCursorMoved(selection.Pos{Line: 7, Col: 15})

	// Switch to visual line mode
	changed := m.Handle(EventToggleVisualLine, cursor)

	if !changed {
		t.Error("Handle(EventToggleVisualLine) should return true when switching from VisualChar")
	}

	if m.Mode() != VisualLine {
		t.Errorf("Mode() = %v, want VisualLine", m.Mode())
	}

	region := m.Region()
	if region.Kind != selection.KindLine {
		t.Errorf("Region().Kind = %v, want KindLine", region.Kind)
	}

	// Start position should be preserved
	if region.Start != cursor {
		t.Errorf("Region().Start = %v, want %v (should preserve start)", region.Start, cursor)
	}
}

func TestModeTransitions_VisualLineToVisualChar(t *testing.T) {
	m := NewMachine()
	cursor := selection.Pos{Line: 5, Col: 10}

	// Enter visual line mode
	m.Handle(EventToggleVisualLine, cursor)

	// Move cursor to create a region
	m.OnCursorMoved(selection.Pos{Line: 7, Col: 15})

	// Switch to visual char mode
	changed := m.Handle(EventToggleVisualChar, cursor)

	if !changed {
		t.Error("Handle(EventToggleVisualChar) should return true when switching from VisualLine")
	}

	if m.Mode() != VisualChar {
		t.Errorf("Mode() = %v, want VisualChar", m.Mode())
	}

	region := m.Region()
	if region.Kind != selection.KindChar {
		t.Errorf("Region().Kind = %v, want KindChar", region.Kind)
	}

	// Start position should be preserved
	if region.Start != cursor {
		t.Errorf("Region().Start = %v, want %v (should preserve start)", region.Start, cursor)
	}
}

func TestModeTransitions_EscapeFromVisualChar(t *testing.T) {
	m := NewMachine()
	cursor := selection.Pos{Line: 5, Col: 10}

	// Enter visual char mode
	m.Handle(EventToggleVisualChar, cursor)

	// Press Escape
	changed := m.Handle(EventEscape, cursor)

	if !changed {
		t.Error("Handle(EventEscape) should return true when exiting visual mode")
	}

	if m.Mode() != Normal {
		t.Errorf("Mode() = %v, want Normal", m.Mode())
	}

	region := m.Region()
	if region.Kind != selection.KindNone {
		t.Errorf("Region().Kind = %v, want KindNone", region.Kind)
	}
}

func TestModeTransitions_EscapeFromVisualLine(t *testing.T) {
	m := NewMachine()
	cursor := selection.Pos{Line: 5, Col: 10}

	// Enter visual line mode
	m.Handle(EventToggleVisualLine, cursor)

	// Press Escape
	changed := m.Handle(EventEscape, cursor)

	if !changed {
		t.Error("Handle(EventEscape) should return true when exiting visual mode")
	}

	if m.Mode() != Normal {
		t.Errorf("Mode() = %v, want Normal", m.Mode())
	}

	region := m.Region()
	if region.Kind != selection.KindNone {
		t.Errorf("Region().Kind = %v, want KindNone", region.Kind)
	}
}

func TestModeTransitions_EscapeFromNormalNoChange(t *testing.T) {
	m := NewMachine()
	cursor := selection.Pos{Line: 5, Col: 10}

	// Press Escape in Normal mode (should be no-op)
	changed := m.Handle(EventEscape, cursor)

	if changed {
		t.Error("Handle(EventEscape) should return false when already in Normal mode")
	}

	if m.Mode() != Normal {
		t.Errorf("Mode() = %v, want Normal", m.Mode())
	}
}

func TestOnCursorMoved_UpdatesRegionInVisualChar(t *testing.T) {
	m := NewMachine()
	start := selection.Pos{Line: 5, Col: 10}

	// Enter visual char mode
	m.Handle(EventToggleVisualChar, start)

	// Move cursor
	newCursor := selection.Pos{Line: 7, Col: 20}
	m.OnCursorMoved(newCursor)

	region := m.Region()
	if region.Start != start {
		t.Errorf("Region().Start = %v, want %v (should not change)", region.Start, start)
	}

	if region.End != newCursor {
		t.Errorf("Region().End = %v, want %v", region.End, newCursor)
	}
}

func TestOnCursorMoved_UpdatesRegionInVisualLine(t *testing.T) {
	m := NewMachine()
	start := selection.Pos{Line: 5, Col: 10}

	// Enter visual line mode
	m.Handle(EventToggleVisualLine, start)

	// Move cursor
	newCursor := selection.Pos{Line: 7, Col: 20}
	m.OnCursorMoved(newCursor)

	region := m.Region()
	if region.Start != start {
		t.Errorf("Region().Start = %v, want %v (should not change)", region.Start, start)
	}

	if region.End != newCursor {
		t.Errorf("Region().End = %v, want %v", region.End, newCursor)
	}
}

func TestOnCursorMoved_NoOpInNormalMode(t *testing.T) {
	m := NewMachine()
	cursor := selection.Pos{Line: 5, Col: 10}

	// Move cursor in Normal mode (should be no-op)
	m.OnCursorMoved(cursor)

	if m.Mode() != Normal {
		t.Errorf("Mode() = %v, want Normal (should not change)", m.Mode())
	}

	region := m.Region()
	if region.Kind != selection.KindNone {
		t.Errorf("Region().Kind = %v, want KindNone (should not change)", region.Kind)
	}
}

// Edge case tests

func TestModeTransitions_MultipleToggles(t *testing.T) {
	m := NewMachine()
	cursor := selection.Pos{Line: 5, Col: 10}

	// Normal -> VisualChar
	m.Handle(EventToggleVisualChar, cursor)
	if m.Mode() != VisualChar {
		t.Errorf("Mode() = %v, want VisualChar after first toggle", m.Mode())
	}

	// VisualChar -> Normal
	m.Handle(EventToggleVisualChar, cursor)
	if m.Mode() != Normal {
		t.Errorf("Mode() = %v, want Normal after second toggle", m.Mode())
	}

	// Normal -> VisualChar again
	m.Handle(EventToggleVisualChar, cursor)
	if m.Mode() != VisualChar {
		t.Errorf("Mode() = %v, want VisualChar after third toggle", m.Mode())
	}
}

func TestModeTransitions_EscapeIdempotent(t *testing.T) {
	m := NewMachine()
	cursor := selection.Pos{Line: 5, Col: 10}

	// Multiple Escape presses in Normal mode should be no-op
	changed1 := m.Handle(EventEscape, cursor)
	changed2 := m.Handle(EventEscape, cursor)
	changed3 := m.Handle(EventEscape, cursor)

	if changed1 || changed2 || changed3 {
		t.Error("Handle(EventEscape) should always return false in Normal mode")
	}

	if m.Mode() != Normal {
		t.Errorf("Mode() = %v, want Normal", m.Mode())
	}
}

func TestModeTransitions_TogglePreservesStartPosition(t *testing.T) {
	m := NewMachine()
	start := selection.Pos{Line: 5, Col: 10}

	// Enter visual char mode
	m.Handle(EventToggleVisualChar, start)

	// Move cursor multiple times
	m.OnCursorMoved(selection.Pos{Line: 6, Col: 15})
	m.OnCursorMoved(selection.Pos{Line: 7, Col: 20})
	m.OnCursorMoved(selection.Pos{Line: 8, Col: 25})

	region := m.Region()
	if region.Start != start {
		t.Errorf("Region().Start = %v, want %v (should remain at initial position)", region.Start, start)
	}

	if region.End != (selection.Pos{Line: 8, Col: 25}) {
		t.Errorf("Region().End = %v, want {8, 25}", region.End)
	}
}

func TestModeTransitions_SwitchModesPreservesPositions(t *testing.T) {
	m := NewMachine()
	start := selection.Pos{Line: 5, Col: 10}
	end := selection.Pos{Line: 8, Col: 20}

	// Enter visual char mode and create a region
	m.Handle(EventToggleVisualChar, start)
	m.OnCursorMoved(end)

	// Switch to visual line mode
	m.Handle(EventToggleVisualLine, start)

	region := m.Region()
	if region.Kind != selection.KindLine {
		t.Errorf("Region().Kind = %v, want KindLine", region.Kind)
	}

	if region.Start != start {
		t.Errorf("Region().Start = %v, want %v (should preserve after mode switch)", region.Start, start)
	}

	if region.End != end {
		t.Errorf("Region().End = %v, want %v (should preserve after mode switch)", region.End, end)
	}
}

func TestModeTransitions_BackAndForthModeSwitch(t *testing.T) {
	m := NewMachine()
	start := selection.Pos{Line: 5, Col: 10}
	end := selection.Pos{Line: 8, Col: 20}

	// VisualChar -> VisualLine -> VisualChar
	m.Handle(EventToggleVisualChar, start)
	m.OnCursorMoved(end)

	m.Handle(EventToggleVisualLine, start)
	if m.Mode() != VisualLine {
		t.Errorf("Mode() = %v, want VisualLine", m.Mode())
	}

	m.Handle(EventToggleVisualChar, start)
	if m.Mode() != VisualChar {
		t.Errorf("Mode() = %v, want VisualChar", m.Mode())
	}

	region := m.Region()
	if region.Start != start || region.End != end {
		t.Errorf("Region positions changed during mode switches: Start=%v, End=%v", region.Start, region.End)
	}
}

func TestModeTransitions_CursorMovementDuringModeSwitch(t *testing.T) {
	m := NewMachine()
	start := selection.Pos{Line: 5, Col: 10}

	// Enter visual char mode
	m.Handle(EventToggleVisualChar, start)

	// Move cursor
	pos1 := selection.Pos{Line: 6, Col: 15}
	m.OnCursorMoved(pos1)

	// Switch to visual line mode
	m.Handle(EventToggleVisualLine, start)

	// Continue moving cursor in visual line mode
	pos2 := selection.Pos{Line: 8, Col: 25}
	m.OnCursorMoved(pos2)

	region := m.Region()
	if region.Kind != selection.KindLine {
		t.Errorf("Region().Kind = %v, want KindLine", region.Kind)
	}

	if region.Start != start {
		t.Errorf("Region().Start = %v, want %v", region.Start, start)
	}

	if region.End != pos2 {
		t.Errorf("Region().End = %v, want %v (should update to latest cursor)", region.End, pos2)
	}
}

func TestRegionCleanupOnExitToNormal(t *testing.T) {
	m := NewMachine()
	start := selection.Pos{Line: 5, Col: 10}
	end := selection.Pos{Line: 8, Col: 20}

	// Create a visual selection
	m.Handle(EventToggleVisualChar, start)
	m.OnCursorMoved(end)

	// Verify region exists
	region := m.Region()
	if region.Kind == selection.KindNone {
		t.Error("Region should exist in VisualChar mode")
	}

	// Exit to Normal mode
	m.Handle(EventEscape, start)

	// Verify region is cleaned up
	region = m.Region()
	if region.Kind != selection.KindNone {
		t.Errorf("Region().Kind = %v, want KindNone after exiting to Normal", region.Kind)
	}
}
