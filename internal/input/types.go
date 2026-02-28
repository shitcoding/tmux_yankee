package input

import (
	"github.com/shitcoding/tmux_yankee/internal/motion"
)

// CommandType represents the type of command parsed from input.
type CommandType int

const (
	CommandNone CommandType = iota
	CommandMotion
	CommandYank
	CommandVisual
	CommandVisualLine
	CommandVisualBlock
	CommandSwapEnd    // o: swap cursor to opposite end of selection
	CommandSwapCorner // O: swap cursor to other corner (block: same line, else: same as o)
	CommandEscape
	CommandQuit
	CommandToggleLineMode
	CommandMouseScroll // set when a mouse wheel event is received
	CommandYankLine    // yy: yank current line without requiring a visual selection
	CommandCharSearch  // f/t/F/T/;/,: character search on current line
	CommandDemoNext      // Tab: next demo page
	CommandDemoPrev      // Shift+Tab: previous demo page
	CommandDemoThemeNext    // ]: next demo theme
	CommandDemoThemePrev    // [: previous demo theme
	CommandToggleWrapMode    // gw: toggle wrap on/off
	CommandDisplayLineDown  // gj: move down one display row (wrap mode)
	CommandDisplayLineUp    // gk: move up one display row (wrap mode)
	CommandMouseLeftPress       // left mouse button pressed
	CommandMouseLeftDrag        // left mouse button drag (motion with button held)
	CommandMouseRelease         // mouse button released
	CommandSearchForward        // '/' — enter search input mode (forward)
	CommandSearchBackward       // '?' — enter search input mode (backward)
	CommandSearchConfirm        // Enter during search input
	CommandSearchCancel         // Escape during search input
	CommandSearchUpdate         // char typed/deleted during search — incremental update
	CommandSearchNext           // 'n' — next match
	CommandSearchPrev           // 'N' — previous match
	CommandSearchWordForward    // '*' — search word under cursor forward
	CommandSearchWordBackward   // '#' — search word under cursor backward
	CommandScrollLineUp         // Ctrl-Y — viewport scroll up without cursor move
	CommandScrollLineDown       // Ctrl-E — viewport scroll down without cursor move
	CommandJumpBack             // `` or '' — jump to previous position
	CommandJumpListBack         // Ctrl-O — jump list backward
	CommandJumpListForward      // Ctrl-I — jump list forward
	CommandSetMark              // m{char} — set mark at cursor
	CommandGoToMark             // `{char} or '{char} — go to mark
	CommandTextObject           // iw, aw, etc. — text object (visual mode)
	CommandSearchSelect         // gn — search and select next match
	CommandSearchSelectBack     // gN — search and select previous match
	CommandClearSearch          // \ — clear search highlights
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
	MouseRow        int             // 0-based terminal row (mouse events)
	MouseCol        int             // 0-based terminal column (mouse events)
	SearchPattern   string          // search text (for SearchConfirm/SearchUpdate)
	MarkChar        byte            // mark character for set/goto mark (m{char}/`{char})
	TextObject      string          // text object action name (for CommandTextObject)
}

// Pending represents the parser's pending state for multi-key sequences.
type Pending struct {
	Count    int  // Accumulated count digits
	HasCount bool // Whether any count digits were entered
	Prefix   byte // Pending prefix key ('g' for gg/G sequences)
}
