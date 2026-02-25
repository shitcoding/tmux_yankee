package input

import (
	"strconv"
	"strings"

	"github.com/shitcoding/tmux_yankee/internal/motion"
)

// Parser parses keyboard input into commands, handling multi-key sequences
// and count prefixes.
type Parser struct {
	pending      Pending
	toggleKey    byte
	wrapKey      byte
	mouseBuf     []byte  // accumulates bytes of an in-progress SGR mouse or CSI sequence
	inMouse      bool    // true while accumulating \x1b[< ... M/m
	inCSI        bool    // true while accumulating a non-mouse CSI param sequence
	deferredCmd  Command // command to emit on next Parse call (used when ESC is held)
	hasDeferred  bool    // true if a deferred command is waiting
}

// NewParser creates a new input parser with the default toggle key ('L').
func NewParser() *Parser {
	return &Parser{toggleKey: 'L', wrapKey: 'w'}
}

// NewParserWithToggleKey creates a new input parser with a configurable toggle key.
func NewParserWithToggleKey(toggleKey byte) *Parser {
	return &Parser{toggleKey: toggleKey, wrapKey: 'w'}
}

// NewParserWithKeys creates a parser with custom toggle and wrap keys.
func NewParserWithKeys(toggleKey, wrapKey byte) *Parser {
	return &Parser{toggleKey: toggleKey, wrapKey: wrapKey}
}

// Parse processes a single byte of input and returns a command if complete.
// Returns CommandNone if the input is part of an incomplete sequence.
//
// Count accumulation:
//   - Digits accumulate into pending count (e.g., "5" then "j" → count=5)
//   - "0" is treated as motion (MotionLineStart) if no count pending
//   - "0" is treated as digit if count already started (e.g., "10" → count=10)
//
// Prefix handling:
//   - "g" sets pending prefix, waits for second key
//   - "gg" → MotionFirstLine
//   - "z" sets pending prefix, waits for second key
//   - "zt/zz/zb" → viewport positioning
//   - Invalid sequences clear pending state
func (p *Parser) Parse(b byte) Command {
	// Return any deferred command first (e.g., ESC held from previous call).
	if p.hasDeferred {
		deferred := p.deferredCmd
		p.hasDeferred = false
		p.deferredCmd = Command{}
		// Process current byte after returning deferred; store as next deferred if needed.
		// Re-invoke parse logic for b, storing any result as new deferred.
		result := p.parseInner(b)
		if result.Type != CommandNone {
			// Can only hold one deferred; return deferred now, next call returns result.
			// Store result as new deferred.
			p.deferredCmd = result
			p.hasDeferred = true
		}
		return deferred
	}
	return p.parseInner(b)
}

// parseInner contains the actual parse logic.
func (p *Parser) parseInner(b byte) Command {
	// SGR mouse sequence accumulation (priority over normal parsing).
	// Sequences: ESC [ < Btn ; Cx ; Cy M/m
	if p.inMouse {
		p.mouseBuf = append(p.mouseBuf, b)
		if len(p.mouseBuf) > 32 {
			// Malformed or truncated sequence: reset and recover.
			p.inMouse = false
			p.mouseBuf = nil
			return Command{Type: CommandNone}
		}
		if b == 'M' || b == 'm' {
			return p.finalizeMouse()
		}
		return Command{Type: CommandNone}
	}

	// Generic CSI sequence accumulation (ESC [ params... final).
	// Entered when ESC [ is followed by a parameter/intermediate byte.
	if p.inCSI {
		p.mouseBuf = append(p.mouseBuf, b)
		if len(p.mouseBuf) > 32 {
			// Too long — discard.
			p.inCSI = false
			p.mouseBuf = nil
			return Command{Type: CommandNone}
		}
		// Final byte (0x40-0x7E) terminates the CSI sequence.
		if b >= 0x40 && b <= 0x7E {
			return p.finalizeCSI()
		}
		return Command{Type: CommandNone}
	}

	// Detect 3-byte prefix: ESC [ <
	// We speculatively buffer ESC and ESC+[ to detect SGR mouse sequences.
	// ESC is held (deferred) until we know whether '[' follows.
	// If '[' follows, we hold ESC+[ until we know whether '<' follows.
	// If '<' follows ESC+[, we enter inMouse mode.
	// If the sequence is not confirmed, we emit the deferred ESC and process normally.
	switch len(p.mouseBuf) {
	case 0:
		if b == 0x1b {
			p.mouseBuf = []byte{0x1b}
			// Hold ESC — wait to see if '[' follows. Emit nothing yet.
			return Command{Type: CommandNone}
		}
	case 1: // have ESC, check for '['
		if b == '[' {
			p.mouseBuf = append(p.mouseBuf, b)
			// Hold ESC [ — wait to see if '<' follows. Emit nothing yet.
			return Command{Type: CommandNone}
		}
		// Not '[' — ESC was standalone. Clear pending state and emit ESC.
		p.mouseBuf = nil
		p.clearPending()
		p.deferredCmd = p.parseNormalByte(b)
		p.hasDeferred = true
		return Command{Type: CommandEscape}
	case 2: // have ESC [
		if b == '<' {
			p.mouseBuf = append(p.mouseBuf, b)
			p.inMouse = true
			return Command{Type: CommandNone} // start accumulating mouse sequence
		}
		if b == 'Z' {
			// ESC [ Z = Shift+Tab → demo page prev
			p.mouseBuf = nil
			return Command{Type: CommandDemoPrev}
		}
		// CSI final byte without params (e.g., ESC[A for arrow up).
		if b >= 0x40 && b <= 0x7E {
			p.mouseBuf = nil
			return p.csiCommand("", b)
		}
		// CSI parameter/intermediate byte — start accumulating.
		p.mouseBuf = append(p.mouseBuf, b)
		p.inCSI = true
		return Command{Type: CommandNone}
	}

	// Handle pending 'g', 'z', 'y', or char-search (f/t/F/T) prefix first
	if p.pending.Prefix == 'g' || p.pending.Prefix == 'z' || p.pending.Prefix == 'y' ||
		p.pending.Prefix == 'f' || p.pending.Prefix == 't' || p.pending.Prefix == 'F' || p.pending.Prefix == 'T' {
		cmd := p.parsePrefixedKey(b)
		p.clearPending()
		return cmd
	}

	// Handle digits for count accumulation
	if b >= '1' && b <= '9' {
		p.pending.Count = p.pending.Count*10 + int(b-'0')
		p.pending.HasCount = true
		return Command{Type: CommandNone}
	}

	// Special case: '0' is motion (line start) if no count, else digit
	if b == '0' {
		if p.pending.HasCount {
			// Digit: part of count (e.g., "10" → count=10)
			p.pending.Count = p.pending.Count * 10
			return Command{Type: CommandNone}
		}
		// Motion: go to line start
		cmd := Command{
			Type:   CommandMotion,
			Motion: motion.MotionLineStart,
			Count:  0,
		}
		p.clearPending()
		return cmd
	}

	// Handle 'g' prefix (wait for next key)
	if b == 'g' {
		p.pending.Prefix = 'g'
		return Command{Type: CommandNone}
	}

	// Handle 'z' prefix (wait for next key)
	if b == 'z' {
		p.pending.Prefix = 'z'
		return Command{Type: CommandNone}
	}

	// Handle 'y' prefix: 'yy' = yank current line (wait for second key)
	if b == 'y' {
		p.pending.Prefix = 'y'
		return Command{Type: CommandNone}
	}

	// Handle f/t/F/T prefix: character search (wait for target char)
	if b == 'f' || b == 'F' || b == 't' || b == 'T' {
		p.pending.Prefix = b
		return Command{Type: CommandNone}
	}

	// All other keys complete the command
	cmd := p.parseCommand(b)
	p.clearPending()
	return cmd
}

// parsePrefixedKey handles keys following a 'g' or 'z' prefix.
func (p *Parser) parsePrefixedKey(b byte) Command {
	if p.pending.Prefix == 'g' {
		switch b {
		case 'g':
			// gg → first line
			return Command{
				Type:   CommandMotion,
				Motion: motion.MotionFirstLine,
				Count:  p.pending.Count,
			}
		case 'j':
			// gj → display line down
			return Command{Type: CommandDisplayLineDown, Count: p.pending.Count}
		case 'k':
			// gk → display line up
			return Command{Type: CommandDisplayLineUp, Count: p.pending.Count}
		case p.wrapKey:
			// gw → toggle wrap mode
			return Command{Type: CommandToggleWrapMode}
		default:
			// Invalid sequence, clear pending
			return Command{Type: CommandNone}
		}
	}

	if p.pending.Prefix == 'z' {
		switch b {
		case 't':
			// zt → position cursor line at top of viewport
			return Command{
				Type:   CommandMotion,
				Motion: motion.MotionViewportTop,
				Count:  0,
			}
		case 'z':
			// zz → position cursor line at center of viewport
			return Command{
				Type:   CommandMotion,
				Motion: motion.MotionViewportCenter,
				Count:  0,
			}
		case 'b':
			// zb → position cursor line at bottom of viewport
			return Command{
				Type:   CommandMotion,
				Motion: motion.MotionViewportBottom,
				Count:  0,
			}
		default:
			// Invalid sequence, clear pending
			return Command{Type: CommandNone}
		}
	}

	if p.pending.Prefix == 'y' {
		switch b {
		case 'y':
			// yy → yank current line
			return Command{Type: CommandYankLine}
		default:
			// Invalid sequence, clear pending
			return Command{Type: CommandNone}
		}
	}

	if p.pending.Prefix == 'f' || p.pending.Prefix == 't' || p.pending.Prefix == 'F' || p.pending.Prefix == 'T' {
		var kind SearchKind
		switch p.pending.Prefix {
		case 'f':
			kind = SearchFindForward
		case 't':
			kind = SearchTillForward
		case 'F':
			kind = SearchFindBackward
		case 'T':
			kind = SearchTillBackward
		}
		return Command{
			Type:       CommandCharSearch,
			SearchKind: kind,
			SearchChar: b,
			Count:      p.pending.Count,
		}
	}

	// No valid prefix
	return Command{Type: CommandNone}
}

// parseCommand parses a complete command from a single key.
func (p *Parser) parseCommand(b byte) Command {
	// Pass count as-is: 0 means "no count typed", >0 means explicit count.
	// Motion handlers are responsible for treating count=0 as 1 repetition
	// (or as special meaning for G/gg where 0 = last/first line).
	count := p.pending.Count

	// Check configurable toggle key before entering the switch
	if b == p.toggleKey {
		return Command{Type: CommandToggleLineMode}
	}

	switch b {
	// Motion commands
	case 'j':
		return Command{
			Type:   CommandMotion,
			Motion: motion.MotionDown,
			Count:  count,
		}
	case 'k':
		return Command{
			Type:   CommandMotion,
			Motion: motion.MotionUp,
			Count:  count,
		}
	case 'h':
		return Command{
			Type:   CommandMotion,
			Motion: motion.MotionLeft,
			Count:  count,
		}
	case 'l':
		return Command{
			Type:   CommandMotion,
			Motion: motion.MotionRight,
			Count:  count,
		}
	case '$':
		return Command{
			Type:   CommandMotion,
			Motion: motion.MotionLineEnd,
			Count:  count,
		}
	case 'G':
		// G needs raw count: 0 = last line, N = line N
		return Command{
			Type:   CommandMotion,
			Motion: motion.MotionLastLine,
			Count:  p.pending.Count,
		}
	case 4: // Ctrl-D
		return Command{
			Type:   CommandMotion,
			Motion: motion.MotionHalfPageDown,
			Count:  count,
		}
	case 21: // Ctrl-U
		return Command{
			Type:   CommandMotion,
			Motion: motion.MotionHalfPageUp,
			Count:  count,
		}
	case 'w':
		return Command{
			Type:   CommandMotion,
			Motion: motion.MotionWordForward,
			Count:  count,
		}
	case 'b':
		return Command{
			Type:   CommandMotion,
			Motion: motion.MotionWordBackward,
			Count:  count,
		}
	case 'e':
		return Command{
			Type:   CommandMotion,
			Motion: motion.MotionWordEnd,
			Count:  count,
		}
	case '^':
		return Command{
			Type:   CommandMotion,
			Motion: motion.MotionFirstNonBlank,
			Count:  count,
		}
	case 'W':
		return Command{
			Type:   CommandMotion,
			Motion: motion.MotionWORDForward,
			Count:  count,
		}
	case 'E':
		return Command{
			Type:   CommandMotion,
			Motion: motion.MotionWORDEnd,
			Count:  count,
		}
	case 'B':
		return Command{
			Type:   CommandMotion,
			Motion: motion.MotionWORDBackward,
			Count:  count,
		}

	// Visual mode commands
	case 'v':
		return Command{Type: CommandVisual}
	case 'V':
		return Command{Type: CommandVisualLine}

	// Yank command (Enter confirms selection yank; 'y' is handled as a prefix above)
	case 13: // Enter
		return Command{Type: CommandYank}

	// Mode control
	case 27: // Escape
		return Command{Type: CommandEscape}

	// Character search repeat
	case ';':
		return Command{
			Type:       CommandCharSearch,
			SearchKind: SearchRepeat,
			Count:      count,
		}
	case ',':
		return Command{
			Type:       CommandCharSearch,
			SearchKind: SearchRepeatReverse,
			Count:      count,
		}

	// Tab: next demo page
	case 9: // Tab
		return Command{Type: CommandDemoNext}

	// Bracket keys: demo theme cycling
	case ']':
		return Command{Type: CommandDemoThemeNext}
	case '[':
		return Command{Type: CommandDemoThemePrev}

	// Quit
	case 'q', 3: // 'q' or Ctrl-C
		return Command{Type: CommandQuit}

	default:
		return Command{Type: CommandNone}
	}
}

// clearPending resets the pending state.
func (p *Parser) clearPending() {
	p.pending = Pending{}
}

// Pending returns the current pending state (for testing/debugging).
func (p *Parser) PendingState() Pending {
	return p.pending
}

// HasPendingYPrefix returns true if 'y' prefix is pending (waiting for second key).
// The TUI uses this to short-circuit to yank-selection when a visual selection is active.
func (p *Parser) HasPendingYPrefix() bool {
	return p.pending.Prefix == 'y'
}

// ClearPending resets the pending state. Used by the TUI to cancel a pending
// prefix when it handles the command directly (e.g. 'y' in visual mode).
func (p *Parser) ClearPending() {
	p.clearPending()
}

// Flush resolves any buffered state that is waiting for the next byte.
// Call this after processing all bytes from a single read to avoid requiring
// a second keypress for standalone ESC or deferred commands.
// May return a deferred command, CommandEscape (pending ESC flush), or CommandNone.
func (p *Parser) Flush() Command {
	// Return any deferred command that was waiting for the next Parse() call.
	if p.hasDeferred {
		deferred := p.deferredCmd
		p.hasDeferred = false
		p.deferredCmd = Command{}
		return deferred
	}
	if p.inCSI {
		// Incomplete CSI parameter sequence — discard silently.
		p.inCSI = false
		p.mouseBuf = nil
		return Command{Type: CommandNone}
	}
	if !p.inMouse && len(p.mouseBuf) > 0 {
		// Pending ESC (or ESC [) that didn't complete a mouse sequence.
		// Treat it as a standalone ESC. Clear any accumulated count/prefix
		// so stale state doesn't leak into subsequent commands.
		p.mouseBuf = nil
		p.clearPending()
		return Command{Type: CommandEscape}
	}
	return Command{Type: CommandNone}
}

// parseNormalByte processes a single byte through the normal (non-mouse) parse path.
// This is used to handle a byte that follows an abandoned mouse prefix.
func (p *Parser) parseNormalByte(b byte) Command {
	// Handle pending 'g', 'z', 'y', or char-search (f/t/F/T) prefix first
	if p.pending.Prefix == 'g' || p.pending.Prefix == 'z' || p.pending.Prefix == 'y' ||
		p.pending.Prefix == 'f' || p.pending.Prefix == 't' || p.pending.Prefix == 'F' || p.pending.Prefix == 'T' {
		cmd := p.parsePrefixedKey(b)
		p.clearPending()
		return cmd
	}

	// Handle digits for count accumulation
	if b >= '1' && b <= '9' {
		p.pending.Count = p.pending.Count*10 + int(b-'0')
		p.pending.HasCount = true
		return Command{Type: CommandNone}
	}

	// Special case: '0' is motion (line start) if no count, else digit
	if b == '0' {
		if p.pending.HasCount {
			p.pending.Count = p.pending.Count * 10
			return Command{Type: CommandNone}
		}
		cmd := Command{Type: CommandMotion, Motion: motion.MotionLineStart, Count: 0}
		p.clearPending()
		return cmd
	}

	// Handle 'g' prefix (wait for next key)
	if b == 'g' {
		p.pending.Prefix = 'g'
		return Command{Type: CommandNone}
	}

	// Handle 'z' prefix (wait for next key)
	if b == 'z' {
		p.pending.Prefix = 'z'
		return Command{Type: CommandNone}
	}

	// Handle 'y' prefix: 'yy' = yank current line (wait for second key)
	if b == 'y' {
		p.pending.Prefix = 'y'
		return Command{Type: CommandNone}
	}

	// Handle f/t/F/T prefix: character search (wait for target char)
	if b == 'f' || b == 'F' || b == 't' || b == 'T' {
		p.pending.Prefix = b
		return Command{Type: CommandNone}
	}

	// All other keys complete the command
	cmd := p.parseCommand(b)
	p.clearPending()
	return cmd
}

// csiCommand maps a parsed CSI sequence (params + final byte) to a Command.
// Handles arrow keys, Home/End, Page Up/Down, and silently discards unknown sequences.
func (p *Parser) csiCommand(params string, final byte) Command {
	// CSI sequences (arrow keys, etc.) are terminal-generated, not vim counts.
	// Clear any accumulated count/prefix so it doesn't leak to the next key.
	p.clearPending()
	switch final {
	case 'A':
		return Command{Type: CommandMotion, Motion: motion.MotionUp, Count: 1}
	case 'B':
		return Command{Type: CommandMotion, Motion: motion.MotionDown, Count: 1}
	case 'C':
		return Command{Type: CommandMotion, Motion: motion.MotionRight, Count: 1}
	case 'D':
		return Command{Type: CommandMotion, Motion: motion.MotionLeft, Count: 1}
	case 'H':
		return Command{Type: CommandMotion, Motion: motion.MotionLineStart, Count: 0}
	case 'F':
		return Command{Type: CommandMotion, Motion: motion.MotionLineEnd, Count: 0}
	case '~':
		switch params {
		case "5":
			return Command{Type: CommandMotion, Motion: motion.MotionHalfPageUp, Count: 0}
		case "6":
			return Command{Type: CommandMotion, Motion: motion.MotionHalfPageDown, Count: 0}
		}
	}
	return Command{Type: CommandNone}
}

// finalizeCSI parses a complete non-mouse CSI sequence from p.mouseBuf.
func (p *Parser) finalizeCSI() Command {
	p.inCSI = false
	buf := p.mouseBuf
	p.mouseBuf = nil

	// buf: ESC [ params... final (at least 3 bytes: ESC [ final)
	if len(buf) < 3 {
		return Command{Type: CommandNone}
	}
	final := buf[len(buf)-1]
	params := string(buf[2 : len(buf)-1]) // skip ESC [, exclude final
	return p.csiCommand(params, final)
}

// finalizeMouse parses a complete SGR mouse sequence from p.mouseBuf.
// Format: ESC [ < Btn ; Cx ; Cy M  (M=press, m=release)
//
// Button bitmask:
//   - bits 0-1: base button (0=left, 1=middle, 2=right, 3=release/none)
//   - bit 5 (32): motion/drag flag
//   - bit 6 (64): wheel flag
//
// Coordinates are 1-based in SGR; returned as 0-based in Command.
func (p *Parser) finalizeMouse() Command {
	p.inMouse = false
	buf := p.mouseBuf
	p.mouseBuf = nil

	// buf: e.g. []byte{0x1b,'[','<','6','4',';','1',';','1','M'}
	// Strip leading ESC[< (3 bytes) and trailing M/m (1 byte)
	if len(buf) < 6 {
		return Command{Type: CommandNone}
	}
	final := buf[len(buf)-1] // 'M' = press/drag, 'm' = release
	inner := string(buf[3 : len(buf)-1])
	parts := strings.SplitN(inner, ";", 3)
	if len(parts) != 3 {
		return Command{Type: CommandNone}
	}
	btn, err := strconv.Atoi(parts[0])
	if err != nil {
		return Command{Type: CommandNone}
	}
	col1, err := strconv.Atoi(parts[1])
	if err != nil {
		return Command{Type: CommandNone}
	}
	row1, err := strconv.Atoi(parts[2])
	if err != nil {
		return Command{Type: CommandNone}
	}
	if col1 < 1 || row1 < 1 {
		return Command{Type: CommandNone}
	}
	col, row := col1-1, row1-1 // Convert to 0-based

	// Clear any pending count/prefix so stale state doesn't leak.
	p.clearPending()

	isWheel := (btn & 64) != 0
	isDrag := (btn & 32) != 0
	base := btn & 3

	// Wheel events (scroll up/down)
	if isWheel && final == 'M' {
		if base == 0 {
			return Command{Type: CommandMouseScroll, ScrollDirection: ScrollUp}
		}
		if base == 1 {
			return Command{Type: CommandMouseScroll, ScrollDirection: ScrollDown}
		}
		return Command{Type: CommandNone}
	}

	// Left button press (no drag, no wheel, base=0, final=M)
	if final == 'M' && !isWheel && !isDrag && base == 0 {
		return Command{Type: CommandMouseLeftPress, MouseRow: row, MouseCol: col}
	}

	// Left button drag (motion with button held, base=0, final=M)
	if final == 'M' && !isWheel && isDrag && base == 0 {
		return Command{Type: CommandMouseLeftDrag, MouseRow: row, MouseCol: col}
	}

	// Button release (final=m)
	if final == 'm' {
		return Command{Type: CommandMouseRelease, MouseRow: row, MouseCol: col}
	}

	return Command{Type: CommandNone}
}
