package input

import "github.com/shitcoding/tmux_yankee/internal/motion"

// Parser parses keyboard input into commands, handling multi-key sequences
// and count prefixes.
type Parser struct {
	pending Pending
}

// NewParser creates a new input parser.
func NewParser() *Parser {
	return &Parser{}
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
//   - Invalid sequences clear pending state
func (p *Parser) Parse(b byte) Command {
	// Handle pending 'g' prefix first
	if p.pending.Prefix == 'g' {
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

	// All other keys complete the command
	cmd := p.parseCommand(b)
	p.clearPending()
	return cmd
}

// parsePrefixedKey handles keys following a 'g' prefix.
func (p *Parser) parsePrefixedKey(b byte) Command {
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

// parseCommand parses a complete command from a single key.
func (p *Parser) parseCommand(b byte) Command {
	count := p.pending.Count

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
		return Command{
			Type:   CommandMotion,
			Motion: motion.MotionLastLine,
			Count:  count,
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
	case 'L':
		return Command{Type: CommandToggleLineMode}

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
