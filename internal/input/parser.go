package input

import (
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/shitcoding/tmux_yankee/internal/keymap"
	"github.com/shitcoding/tmux_yankee/internal/motion"
)

// Parser parses keyboard input into commands, handling multi-key sequences
// and count prefixes.
type Parser struct {
	pending     Pending
	toggleKey   byte
	wrapKey     byte
	km          keymap.Keymap
	mouseBuf    []byte  // accumulates bytes of an in-progress SGR mouse or CSI sequence
	inMouse     bool    // true while accumulating \x1b[< ... M/m
	inCSI       bool    // true while accumulating a non-mouse CSI param sequence
	deferredCmd Command // command to emit on next Parse call (used when ESC is held)
	hasDeferred bool    // true if a deferred command is waiting
	searchBuf   []rune  // accumulates search pattern text (rune-aware for UTF-8)
	inSearch    bool    // true while collecting search input
	searchDir   byte    // '/' or '?'
	colonBuf    []rune  // accumulates colon command digits
	inColon     bool    // true while collecting colon input
	searchUTF8  []byte  // accumulates bytes of an in-progress multi-byte UTF-8 rune
}

// NewParser creates a new input parser with the default toggle key ('L') and default keymap.
func NewParser() *Parser {
	return &Parser{toggleKey: 'L', wrapKey: 'w', km: keymap.DefaultKeymap()}
}

// NewParserWithKeys creates a parser with custom toggle and wrap keys.
func NewParserWithKeys(toggleKey, wrapKey byte) *Parser {
	return &Parser{toggleKey: toggleKey, wrapKey: wrapKey, km: keymap.DefaultKeymap()}
}

// NewParserWithKeymap creates a parser with custom keys and a user-configured keymap.
func NewParserWithKeymap(toggleKey, wrapKey byte, km keymap.Keymap) *Parser {
	return &Parser{toggleKey: toggleKey, wrapKey: wrapKey, km: km}
}

// SetKeymap replaces the parser's keymap. Used by TUI to swap keymaps on mode transitions.
// Not thread-safe — caller must ensure no concurrent Parse calls (guaranteed by single-threaded TUI loop).
func (p *Parser) SetKeymap(km keymap.Keymap) {
	p.km = km
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

	// Search input mode: intercept resolved bytes before normal key processing.
	// ESC (0x1b) falls through to the mouse/CSI state machine so that
	// ESC sequences (arrow keys, CSI) are properly detected. Bytes that are
	// part of an in-progress escape sequence also fall through.
	if p.inSearch && !p.inMouse && !p.inCSI && len(p.mouseBuf) == 0 && b != 0x1b {
		return p.parseSearchByte(b)
	}

	// Colon input mode: intercept resolved bytes before normal key processing.
	if p.inColon && !p.inMouse && !p.inCSI && len(p.mouseBuf) == 0 && b != 0x1b {
		return p.parseColonByte(b)
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
		// Not '[' — ESC was followed by another byte.
		p.mouseBuf = nil
		p.clearPending()

		// Alt+key detection: tmux sends Alt+key as ESC followed by printable byte.
		// Check keymap for Alt binding. If unbound, discard both bytes (CommandNone)
		// so the bare key doesn't accidentally trigger a normal binding.
		if b >= 0x20 && b <= 0x7e {
			altKey := keymap.Alt(b)
			if action, ok := p.km.Lookup(altKey); ok {
				if p.inSearch {
					p.inSearch = false
					p.searchBuf = p.searchBuf[:0]
					return Command{Type: CommandSearchCancel}
				}
				if p.inColon {
					p.inColon = false
					p.colonBuf = p.colonBuf[:0]
					return Command{Type: CommandColonCancel}
				}
				return ActionToCommand(action, 0, 0)
			}
			// Unbound Alt+key — discard silently.
			return Command{Type: CommandNone}
		}

		// Non-printable byte after ESC: treat ESC as standalone escape.
		if p.inSearch {
			// Cancel search on standalone ESC; defer the follow-up byte as search input.
			p.inSearch = false
			p.searchBuf = p.searchBuf[:0]
			p.deferredCmd = p.parseNormalByte(b)
			p.hasDeferred = true
			return Command{Type: CommandSearchCancel}
		}
		if p.inColon {
			p.inColon = false
			p.colonBuf = p.colonBuf[:0]
			p.deferredCmd = p.parseNormalByte(b)
			p.hasDeferred = true
			return Command{Type: CommandColonCancel}
		}
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

	// Handle pending prefix (keymap-driven)
	if p.pending.Prefix != 0 {
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

	// Check if this key starts a prefix sequence (keymap-driven)
	if p.km.IsPrefix(b) {
		p.pending.Prefix = b
		return Command{Type: CommandNone}
	}

	// Check if this key starts a char-capture sequence (keymap-driven)
	if p.km.IsCharCapture(b) {
		p.pending.Prefix = b
		return Command{Type: CommandNone}
	}

	// Check if this key starts a text object prefix (i/a in visual mode)
	if p.km.HasTextObjectPrefix(b) {
		p.pending.Prefix = b
		return Command{Type: CommandNone}
	}

	// All other keys complete the command
	cmd := p.parseCommand(b)
	p.clearPending()
	return cmd
}

// parsePrefixedKey handles keys following a prefix key (g, z, y, f/t/F/T, m, etc.).
// Uses the keymap for lookup.
func (p *Parser) parsePrefixedKey(b byte) Command {
	prefix := p.pending.Prefix
	count := p.pending.Count

	// Char-capture prefix: the second byte is captured as a parameter
	if action, ok := p.km.LookupCharCapture(prefix); ok {
		return ActionToCommand(action, count, b)
	}

	// Text object lookup (i/a prefix)
	if action, ok := p.km.LookupTextObject(prefix, b); ok {
		return ActionToCommand(action, count, 0)
	}

	// Standard prefix lookup (user bindings take priority over defaults)
	if action, ok := p.km.LookupPrefix(prefix, b); ok {
		return ActionToCommand(action, count, 0)
	}

	// Fallback: wrap key after 'g' prefix when no explicit binding exists
	if prefix == 'g' && b == p.wrapKey {
		return Command{Type: CommandToggleWrapMode}
	}

	// Invalid sequence
	return Command{Type: CommandNone}
}

// parseCommand parses a complete command from a single key using keymap lookup.
func (p *Parser) parseCommand(b byte) Command {
	count := p.pending.Count

	// Build KeySpec from byte.
	// For bytes 1-26 (Ctrl+A through Ctrl+Z), try plain Key(b) first
	// (handles Enter=13, Tab=9 etc.), then fall back to Ctrl notation.
	key := keymap.Key(b)
	action, ok := p.km.Lookup(key)
	if !ok && b >= 1 && b <= 26 {
		key = keymap.Ctrl(byte('a' + b - 1))
		action, ok = p.km.Lookup(key)
	}

	// If key is not in keymap, check configurable toggle key as fallback.
	// This allows keymap bindings (e.g. L→screen_bottom) to take priority
	// over the default toggle key.
	if !ok {
		if b == p.toggleKey {
			return Command{Type: CommandToggleLineMode}
		}
		return Command{Type: CommandNone}
	}

	// Special handling for search/colon commands that need to set parser state
	switch action {
	case keymap.ActionSearchForward:
		p.inSearch = true
		p.searchDir = '/'
		p.searchBuf = p.searchBuf[:0]
		return Command{Type: CommandSearchForward}
	case keymap.ActionSearchBackward:
		p.inSearch = true
		p.searchDir = '?'
		p.searchBuf = p.searchBuf[:0]
		return Command{Type: CommandSearchBackward}
	case keymap.ActionColonMode:
		p.inColon = true
		p.colonBuf = p.colonBuf[:0]
		return Command{Type: CommandColonEnter}
	}

	return ActionToCommand(action, count, 0)
}

// clearPending resets the pending state.
func (p *Parser) clearPending() {
	p.pending = Pending{}
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

// PendingEscape reports whether the parser is part-way through an escape
// sequence that Flush would otherwise resolve at a read boundary: a bare ESC or
// ESC[ held awaiting the next byte, or an in-progress non-mouse CSI. In-progress
// mouse sequences are excluded because Flush already lets them persist across
// reads (see the !inMouse guard in Flush).
//
// The TUI event loop uses this to briefly wait for the rest of a sequence that
// was fragmented across reads (e.g. an SGR mouse sequence split by a TCP segment
// boundary over SSH) before flushing a lone ESC — otherwise the trailing bytes
// are misparsed as literal keys.
func (p *Parser) PendingEscape() bool {
	return p.inCSI || (!p.inMouse && len(p.mouseBuf) > 0)
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
		// Clear pending count/prefix so stale state doesn't leak into
		// subsequent keystrokes.
		p.inCSI = false
		p.mouseBuf = nil
		p.clearPending()
		return Command{Type: CommandNone}
	}
	if !p.inMouse && len(p.mouseBuf) > 0 {
		// Pending ESC (or ESC [) that didn't complete a mouse sequence.
		// Treat it as a standalone ESC. Clear any accumulated count/prefix
		// so stale state doesn't leak into subsequent commands.
		p.mouseBuf = nil
		p.clearPending()
		if p.inSearch {
			p.inSearch = false
			p.searchBuf = p.searchBuf[:0]
			return Command{Type: CommandSearchCancel}
		}
		if p.inColon {
			p.inColon = false
			p.colonBuf = p.colonBuf[:0]
			return Command{Type: CommandColonCancel}
		}
		return Command{Type: CommandEscape}
	}
	return Command{Type: CommandNone}
}

// parseSearchByte handles a single byte while in search input mode.
func (p *Parser) parseSearchByte(b byte) Command {
	// If we're accumulating a multi-byte UTF-8 sequence, continue it.
	if len(p.searchUTF8) > 0 {
		if b&0xC0 != 0x80 {
			// Not a continuation byte — discard partial sequence and
			// fall through to process this byte normally.
			p.searchUTF8 = p.searchUTF8[:0]
		} else {
			p.searchUTF8 = append(p.searchUTF8, b)
			if utf8.FullRune(p.searchUTF8) {
				r, _ := utf8.DecodeRune(p.searchUTF8)
				p.searchUTF8 = p.searchUTF8[:0]
				if r != utf8.RuneError {
					p.searchBuf = append(p.searchBuf, r)
					return Command{Type: CommandSearchUpdate, SearchPattern: string(p.searchBuf)}
				}
			}
			// Still accumulating — need more bytes.
			return Command{Type: CommandNone}
		}
	}

	switch b {
	case 13: // Enter → confirm search
		pattern := string(p.searchBuf)
		p.inSearch = false
		return Command{Type: CommandSearchConfirm, SearchPattern: pattern}
	case 127, 8: // Backspace (DEL or BS) → pop last rune
		if len(p.searchBuf) > 0 {
			p.searchBuf = p.searchBuf[:len(p.searchBuf)-1]
		}
		return Command{Type: CommandSearchUpdate, SearchPattern: string(p.searchBuf)}
	default:
		if b >= 32 && b < 127 {
			// Printable ASCII → append to search buffer
			p.searchBuf = append(p.searchBuf, rune(b))
			return Command{Type: CommandSearchUpdate, SearchPattern: string(p.searchBuf)}
		}
		if b&0xC0 == 0xC0 {
			// Start of a multi-byte UTF-8 sequence (leading byte)
			p.searchUTF8 = append(p.searchUTF8[:0], b)
			return Command{Type: CommandNone}
		}
		// Non-printable (Ctrl-codes, lone continuation bytes, etc.) → ignore
		return Command{Type: CommandNone}
	}
}

// SearchBuffer returns the current search input buffer as a string.
func (p *Parser) SearchBuffer() string {
	return string(p.searchBuf)
}

// InSearchMode returns true if the parser is currently collecting search input.
func (p *Parser) InSearchMode() bool {
	return p.inSearch
}

// SearchDir returns the search direction character ('/' or '?').
func (p *Parser) SearchDir() byte {
	return p.searchDir
}

// parseColonByte handles a single byte while in colon input mode.
func (p *Parser) parseColonByte(b byte) Command {
	switch {
	case b == 13: // Enter → execute colon command
		buf := string(p.colonBuf)
		p.inColon = false
		lineNum, err := strconv.Atoi(buf)
		if err != nil || lineNum < 1 {
			return Command{Type: CommandColonCancel}
		}
		return Command{Type: CommandColonExecute, Count: lineNum}
	case b == 127 || b == 8: // Backspace
		if len(p.colonBuf) > 0 {
			p.colonBuf = p.colonBuf[:len(p.colonBuf)-1]
		} else {
			// Empty buffer + backspace → cancel
			p.inColon = false
			return Command{Type: CommandColonCancel}
		}
		return Command{Type: CommandColonUpdate}
	case b >= '0' && b <= '9':
		p.colonBuf = append(p.colonBuf, rune(b))
		return Command{Type: CommandColonUpdate}
	default:
		// Non-digit → cancel colon mode
		p.inColon = false
		p.colonBuf = p.colonBuf[:0]
		return Command{Type: CommandColonCancel}
	}
}

// ColonBuffer returns the current colon input buffer as a string.
func (p *Parser) ColonBuffer() string {
	return string(p.colonBuf)
}

// InColonMode returns true if the parser is currently collecting colon input.
func (p *Parser) InColonMode() bool {
	return p.inColon
}

// parseNormalByte processes a single byte through the normal (non-mouse) parse path.
// This is used to handle a byte that follows an abandoned mouse prefix.
func (p *Parser) parseNormalByte(b byte) Command {
	// Handle pending prefix first
	if p.pending.Prefix != 0 {
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

	// Check if this key starts a prefix sequence
	if p.km.IsPrefix(b) {
		p.pending.Prefix = b
		return Command{Type: CommandNone}
	}

	// Check if this key starts a char-capture sequence
	if p.km.IsCharCapture(b) {
		p.pending.Prefix = b
		return Command{Type: CommandNone}
	}

	// Check if this key starts a text object prefix (i/a in visual mode)
	if p.km.HasTextObjectPrefix(b) {
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
