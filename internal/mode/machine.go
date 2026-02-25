package mode

import "github.com/shitcoding/tmux_yankee/internal/selection"

// Machine tracks the current mode state and manages transitions
type Machine struct {
	value  Value
	region selection.Region
}

// NewMachine creates a new mode machine in Normal mode
func NewMachine() *Machine {
	return &Machine{
		value:  Normal,
		region: selection.EmptyRegion(),
	}
}

// Mode returns the current mode value
func (m *Machine) Mode() Value {
	return m.value
}

// Region returns the current selection region
// Returns an empty region (KindNone) when in Normal mode
func (m *Machine) Region() selection.Region {
	return m.region
}

// Handle processes a mode event and updates state accordingly
// Returns true if the mode changed, false otherwise
func (m *Machine) Handle(event Event, cursor selection.Pos) bool {
	switch event {
	case EventToggleVisualChar:
		return m.handleToggleVisualChar(cursor)
	case EventToggleVisualLine:
		return m.handleToggleVisualLine(cursor)
	case EventToggleVisualBlock:
		return m.handleToggleVisualBlock(cursor)
	case EventEscape:
		return m.handleEscape()
	default:
		return false
	}
}

// OnCursorMoved updates the selection region end when cursor moves
// Only has effect in visual modes; no-op in Normal mode
func (m *Machine) OnCursorMoved(cursor selection.Pos) {
	if m.value == Normal {
		return
	}

	// Update the end position of the region
	m.region.End = cursor
}

// SwapEnd swaps the cursor to the opposite end of the selection.
// In all visual modes, Start and End are exchanged so the cursor
// (always at End) jumps to what was previously the anchor.
// Returns the new cursor position and true if a swap occurred.
func (m *Machine) SwapEnd() (selection.Pos, bool) {
	if m.value == Normal {
		return selection.Pos{}, false
	}
	m.region.Start, m.region.End = m.region.End, m.region.Start
	return m.region.End, true
}

// SwapCorner swaps the cursor to the other corner on the same line.
// In block mode: only column components are swapped (cursor stays on same line).
// In char/line modes: behaves like SwapEnd (full swap).
// Returns the new cursor position and true if a swap occurred.
func (m *Machine) SwapCorner() (selection.Pos, bool) {
	if m.value == Normal {
		return selection.Pos{}, false
	}
	if m.value == VisualBlock {
		// Swap columns only — cursor stays on same line
		m.region.Start.Col, m.region.End.Col = m.region.End.Col, m.region.Start.Col
		return m.region.End, true
	}
	// Non-block: same as SwapEnd
	m.region.Start, m.region.End = m.region.End, m.region.Start
	return m.region.End, true
}

// handleToggleVisualChar handles the 'v' key press
func (m *Machine) handleToggleVisualChar(cursor selection.Pos) bool {
	switch m.value {
	case Normal:
		// Normal -> VisualChar: start character-wise selection
		m.value = VisualChar
		m.region = selection.Region{
			Kind:  selection.KindChar,
			Start: cursor,
			End:   cursor,
		}
		return true

	case VisualChar:
		// VisualChar -> Normal: cancel selection
		m.value = Normal
		m.region = selection.EmptyRegion()
		return true

	case VisualLine:
		// VisualLine -> VisualChar: switch to character-wise
		m.value = VisualChar
		m.region.Kind = selection.KindChar
		// Preserve Start and End positions
		return true

	case VisualBlock:
		// VisualBlock -> VisualChar: switch to character-wise
		m.value = VisualChar
		m.region.Kind = selection.KindChar
		// Preserve Start and End positions
		return true

	default:
		return false
	}
}

// handleToggleVisualLine handles the 'V' key press
func (m *Machine) handleToggleVisualLine(cursor selection.Pos) bool {
	switch m.value {
	case Normal:
		// Normal -> VisualLine: start line-wise selection
		m.value = VisualLine
		m.region = selection.Region{
			Kind:  selection.KindLine,
			Start: cursor,
			End:   cursor,
		}
		return true

	case VisualLine:
		// VisualLine -> Normal: cancel selection
		m.value = Normal
		m.region = selection.EmptyRegion()
		return true

	case VisualChar:
		// VisualChar -> VisualLine: switch to line-wise
		m.value = VisualLine
		m.region.Kind = selection.KindLine
		// Preserve Start and End positions
		return true

	case VisualBlock:
		// VisualBlock -> VisualLine: switch to line-wise
		m.value = VisualLine
		m.region.Kind = selection.KindLine
		// Preserve Start and End positions
		return true

	default:
		return false
	}
}

// handleToggleVisualBlock handles the Ctrl-V key press
func (m *Machine) handleToggleVisualBlock(cursor selection.Pos) bool {
	switch m.value {
	case Normal:
		// Normal -> VisualBlock: start block-wise selection
		m.value = VisualBlock
		m.region = selection.Region{
			Kind:  selection.KindBlock,
			Start: cursor,
			End:   cursor,
		}
		return true

	case VisualBlock:
		// VisualBlock -> Normal: cancel selection
		m.value = Normal
		m.region = selection.EmptyRegion()
		return true

	case VisualChar:
		// VisualChar -> VisualBlock: switch to block-wise
		m.value = VisualBlock
		m.region.Kind = selection.KindBlock
		// Preserve Start and End positions
		return true

	case VisualLine:
		// VisualLine -> VisualBlock: switch to block-wise
		m.value = VisualBlock
		m.region.Kind = selection.KindBlock
		// Preserve Start and End positions
		return true

	default:
		return false
	}
}

// handleEscape handles the Escape key press
func (m *Machine) handleEscape() bool {
	if m.value == Normal {
		// Already in Normal mode, no change
		return false
	}

	// Any visual mode -> Normal
	m.value = Normal
	m.region = selection.EmptyRegion()
	return true
}
