package input

import "github.com/shitcoding/tmux_yankee/internal/motion"

// CommandType represents the type of command parsed from input.
type CommandType int

const (
	CommandNone CommandType = iota
	CommandMotion
	CommandYank
	CommandVisual
	CommandVisualLine
	CommandEscape
	CommandQuit
	CommandToggleLineMode
)

// Command represents a parsed input command.
type Command struct {
	Type   CommandType
	Motion motion.Motion // Only valid if Type == CommandMotion
	Count  int           // Repeat count (0 if not specified)
}

// Pending represents the parser's pending state for multi-key sequences.
type Pending struct {
	Count    int  // Accumulated count digits
	HasCount bool // Whether any count digits were entered
	Prefix   byte // Pending prefix key ('g' for gg/G sequences)
}
