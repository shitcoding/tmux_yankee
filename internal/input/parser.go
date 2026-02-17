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
	mouseBuf     []byte  // accumulates bytes of an in-progress SGR mouse sequence
	inMouse      bool    // true while accumulating \x1b[< ... M/m
	deferredCmd  Command // command to emit on next Parse call (used when ESC is held)
	hasDeferred  bool    // true if a deferred command is waiting
}

// NewParser creates a new input parser with the default toggle key ('L').
func NewParser() *Parser {
	return &Parser{toggleKey: 'L'}
}

// NewParserWithToggleKey creates a new input parser with a configurable toggle key.
func NewParserWithToggleKey(toggleKey byte) *Parser {
	return &Parser{toggleKey: toggleKey}
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
		// Not '[' — ESC was standalone. Emit ESC now, process b normally in next call.
		p.mouseBuf = nil
		p.deferredCmd = p.parseNormalByte(b)
		p.hasDeferred = true
		return Command{Type: CommandEscape}
	case 2: // have ESC [
		if b == '<' {
			p.mouseBuf = append(p.mouseBuf, b)
			p.inMouse = true
			return Command{Type: CommandNone} // start accumulating mouse sequence
		}
		// Not '<' — ESC [ X is not a mouse sequence.
		// Emit deferred ESC, then process b normally.
		p.mouseBuf = nil
		p.deferredCmd = p.parseNormalByte(b)
		p.hasDeferred = true
		return Command{Type: CommandEscape}
	}

	// Handle pending 'g' or 'z' prefix first
	if p.pending.Prefix == 'g' || p.pending.Prefix == 'z' {
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

	// Yank command
	case 'y', 13: // 'y' or Enter
		return Command{Type: CommandYank}

	// Mode control
	case 27: // Escape
		return Command{Type: CommandEscape}

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

// parseNormalByte processes a single byte through the normal (non-mouse) parse path.
// This is used to handle a byte that follows an abandoned mouse prefix.
func (p *Parser) parseNormalByte(b byte) Command {
	// Handle pending 'g' or 'z' prefix first
	if p.pending.Prefix == 'g' || p.pending.Prefix == 'z' {
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

	// All other keys complete the command
	cmd := p.parseCommand(b)
	p.clearPending()
	return cmd
}

// finalizeMouse parses a complete SGR mouse sequence from p.mouseBuf.
// Format: ESC [ < Btn ; Cx ; Cy M  (M=press, m=release)
// Wheel-up = button 64, wheel-down = button 65.
func (p *Parser) finalizeMouse() Command {
	p.inMouse = false
	buf := p.mouseBuf
	p.mouseBuf = nil

	// buf: e.g. []byte{0x1b,'[','<','6','4',';','1',';','1','M'}
	// Strip leading ESC[< (3 bytes) and trailing M/m (1 byte)
	if len(buf) < 6 {
		return Command{Type: CommandNone}
	}
	inner := string(buf[3 : len(buf)-1]) // "64;1;1"
	parts := strings.SplitN(inner, ";", 3)
	if len(parts) < 1 {
		return Command{Type: CommandNone}
	}
	btn, err := strconv.Atoi(parts[0])
	if err != nil {
		return Command{Type: CommandNone}
	}
	switch btn {
	case 64:
		return Command{Type: CommandMouseScroll, ScrollDirection: ScrollUp}
	case 65:
		return Command{Type: CommandMouseScroll, ScrollDirection: ScrollDown}
	default:
		return Command{Type: CommandNone} // clicks, drags, etc.
	}
}
