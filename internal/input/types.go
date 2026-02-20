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
	CommandMouseScroll // set when a mouse wheel event is received
	CommandYankLine    // yy: yank current line without requiring a visual selection
	CommandCharSearch  // f/t/F/T/;/,: character search on current line
	CommandDemoNext   // Tab: next demo page
	CommandDemoPrev   // Shift+Tab: previous demo page
)

// ScrollDirection indicates mouse wheel direction for CommandMouseScroll.
type ScrollDirection int

const (
	ScrollNone ScrollDirection = iota
	ScrollUp
	ScrollDown
)

// SearchKind indicates the type of character search for CommandCharSearch.
type SearchKind int

const (
	SearchFindForward   SearchKind = iota // f — cursor ON char
	SearchTillForward                     // t — cursor BEFORE char
	SearchFindBackward                    // F — cursor ON char (backward)
	SearchTillBackward                    // T — cursor AFTER char (backward)
	SearchRepeat                          // ; — repeat last
	SearchRepeatReverse                   // , — repeat last, reversed
)

// Command represents a parsed input command.
type Command struct {
	Type            CommandType
	Motion          motion.Motion   // Only valid if Type == CommandMotion
	Count           int             // Repeat count (0 if not specified)
	ScrollDirection ScrollDirection  // set when Type == CommandMouseScroll
	SearchKind      SearchKind      // valid when Type == CommandCharSearch
	SearchChar      byte            // target character (0 for ;/,)
}

// Pending represents the parser's pending state for multi-key sequences.
type Pending struct {
	Count    int  // Accumulated count digits
	HasCount bool // Whether any count digits were entered
	Prefix   byte // Pending prefix key ('g' for gg/G sequences)
}
