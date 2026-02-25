package mode

// Value represents the current vim-like mode
type Value int

const (
	// Normal is the default mode (no selection)
	Normal Value = iota
	// VisualChar is character-wise visual mode (vim 'v')
	VisualChar
	// VisualLine is line-wise visual mode (vim 'V')
	VisualLine
	// VisualBlock is block-wise visual mode (vim Ctrl-V)
	VisualBlock
)

// String returns the string representation of the mode
func (v Value) String() string {
	switch v {
	case Normal:
		return "Normal"
	case VisualChar:
		return "VisualChar"
	case VisualLine:
		return "VisualLine"
	case VisualBlock:
		return "VisualBlock"
	default:
		return "Unknown"
	}
}

// Event represents a mode transition event
type Event int

const (
	// EventToggleVisualChar toggles character-wise visual mode (vim 'v')
	EventToggleVisualChar Event = iota
	// EventToggleVisualLine toggles line-wise visual mode (vim 'V')
	EventToggleVisualLine
	// EventToggleVisualBlock toggles block-wise visual mode (vim Ctrl-V)
	EventToggleVisualBlock
	// EventEscape exits any visual mode back to Normal (vim 'Esc')
	EventEscape
)

// String returns the string representation of the event
func (e Event) String() string {
	switch e {
	case EventToggleVisualChar:
		return "ToggleVisualChar"
	case EventToggleVisualLine:
		return "ToggleVisualLine"
	case EventToggleVisualBlock:
		return "ToggleVisualBlock"
	case EventEscape:
		return "Escape"
	default:
		return "Unknown"
	}
}
