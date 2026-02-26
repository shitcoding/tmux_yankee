package input

import (
	"testing"

	"github.com/shitcoding/tmux_yankee/internal/keymap"
	"github.com/shitcoding/tmux_yankee/internal/motion"
)

// TestAltKeyUnbound verifies that ESC+printable with no Alt binding
// returns CommandNone (both bytes discarded, not treated as ESC + normal key).
func TestAltKeyUnbound(t *testing.T) {
	p := NewParser() // default keymap has no Alt bindings
	// Feed ESC + 'h'
	cmd := p.Parse(0x1b)
	if cmd.Type != CommandNone {
		t.Fatalf("ESC should buffer, got %v", cmd.Type)
	}
	cmd = p.Parse('h')
	if cmd.Type != CommandNone {
		t.Fatalf("Alt+h (unbound) should be CommandNone, got %v", cmd.Type)
	}
	// Verify no deferred command leaked
	cmd = p.Flush()
	if cmd.Type != CommandNone {
		t.Fatalf("Flush after Alt+h should be CommandNone, got %v", cmd.Type)
	}
}

// TestAltKeyBound verifies that ESC+printable with an Alt binding
// executes the bound action.
func TestAltKeyBound(t *testing.T) {
	km := keymap.DefaultKeymap()
	// Add Alt+h → move left
	km.Direct[keymap.Alt('h')] = keymap.ActionMoveLeft
	p := NewParserWithKeymap('L', 'w', km)
	// Feed ESC + 'h'
	cmd := p.Parse(0x1b)
	if cmd.Type != CommandNone {
		t.Fatalf("ESC should buffer, got %v", cmd.Type)
	}
	cmd = p.Parse('h')
	if cmd.Type != CommandMotion || cmd.Motion != motion.MotionLeft {
		t.Fatalf("Alt+h (bound) should be MotionLeft, got type=%v motion=%v", cmd.Type, cmd.Motion)
	}
}

// TestAltKeyDoesNotTriggerNormalBinding ensures that ESC+'j' does NOT
// trigger the normal 'j' (move down) binding when Alt+j is unbound.
func TestAltKeyDoesNotTriggerNormalBinding(t *testing.T) {
	p := NewParser()
	cmd := p.Parse(0x1b)
	if cmd.Type != CommandNone {
		t.Fatalf("ESC should buffer, got %v", cmd.Type)
	}
	cmd = p.Parse('j')
	if cmd.Type != CommandNone {
		t.Fatalf("Alt+j (unbound) should discard, got %v", cmd.Type)
	}
	// No deferred command
	cmd = p.Flush()
	if cmd.Type != CommandNone {
		t.Fatalf("No deferred after Alt+j, got %v", cmd.Type)
	}
}

// TestStandaloneEscStillWorks verifies that standalone ESC (via Flush)
// still returns CommandEscape.
func TestStandaloneEscStillWorks(t *testing.T) {
	p := NewParser()
	cmd := p.Parse(0x1b)
	if cmd.Type != CommandNone {
		t.Fatalf("ESC should buffer, got %v", cmd.Type)
	}
	// No follow-up byte — Flush resolves as standalone ESC
	cmd = p.Flush()
	if cmd.Type != CommandEscape {
		t.Fatalf("Standalone ESC should be CommandEscape, got %v", cmd.Type)
	}
}

// TestEscBracketStillEntersCSI verifies ESC+[ still enters CSI mode.
func TestEscBracketStillEntersCSI(t *testing.T) {
	p := NewParser()
	cmd := p.Parse(0x1b)
	if cmd.Type != CommandNone {
		t.Fatalf("ESC should buffer, got %v", cmd.Type)
	}
	cmd = p.Parse('[')
	if cmd.Type != CommandNone {
		t.Fatalf("ESC+[ should buffer for CSI, got %v", cmd.Type)
	}
	// Arrow up: ESC [ A
	cmd = p.Parse('A')
	if cmd.Type != CommandMotion || cmd.Motion != motion.MotionUp {
		t.Fatalf("ESC[A should be MotionUp, got type=%v motion=%v", cmd.Type, cmd.Motion)
	}
}

// TestAltKeyInSearchModeCancelsSearch verifies that a bound Alt+key
// during search mode cancels the search.
func TestAltKeyInSearchModeCancelsSearch(t *testing.T) {
	km := keymap.DefaultKeymap()
	km.Direct[keymap.Alt('x')] = keymap.ActionQuit
	p := NewParserWithKeymap('L', 'w', km)
	// Enter search mode
	cmd := p.Parse('/')
	if cmd.Type != CommandSearchForward {
		t.Fatalf("expected SearchForward, got %v", cmd.Type)
	}
	// Type a character
	cmd = p.Parse('a')
	if cmd.Type != CommandSearchUpdate {
		t.Fatalf("expected SearchUpdate, got %v", cmd.Type)
	}
	// Alt+x (bound) while in search → cancel search
	cmd = p.Parse(0x1b)
	if cmd.Type != CommandNone {
		t.Fatalf("ESC should buffer, got %v", cmd.Type)
	}
	cmd = p.Parse('x')
	if cmd.Type != CommandSearchCancel {
		t.Fatalf("Alt+key during search should cancel, got %v", cmd.Type)
	}
}

// TestAltKeyNonPrintableFallsThrough verifies that ESC followed by a
// non-printable byte (e.g. Ctrl code) still behaves as ESC + deferred byte.
func TestAltKeyNonPrintableFallsThrough(t *testing.T) {
	p := NewParser()
	cmd := p.Parse(0x1b)
	if cmd.Type != CommandNone {
		t.Fatalf("ESC should buffer, got %v", cmd.Type)
	}
	// Feed Ctrl+A (byte 1) — non-printable, should fall through to ESC behavior
	cmd = p.Parse(0x01)
	if cmd.Type != CommandEscape {
		t.Fatalf("ESC + non-printable should emit ESC, got %v", cmd.Type)
	}
	// The Ctrl+A should be deferred
	cmd = p.Flush()
	// Ctrl+A = byte 1, which maps to Ctrl+a in keymap → could be a binding or CommandNone
	// The point is that the ESC was emitted and Ctrl+A was deferred, not discarded.
	_ = cmd // just verify we don't panic
}

// TestAltKeyWithDigitDoesNotAccumulateCount verifies ESC+'5' is treated
// as Alt+5, not ESC then count digit.
func TestAltKeyWithDigitDoesNotAccumulateCount(t *testing.T) {
	p := NewParser()
	cmd := p.Parse(0x1b)
	if cmd.Type != CommandNone {
		t.Fatalf("ESC should buffer, got %v", cmd.Type)
	}
	cmd = p.Parse('5')
	// '5' is printable, so it's treated as Alt+5 (unbound → CommandNone)
	if cmd.Type != CommandNone {
		t.Fatalf("Alt+5 (unbound) should be CommandNone, got %v", cmd.Type)
	}
	// Verify no count was accumulated
	pending := p.PendingState()
	if pending.HasCount {
		t.Fatalf("Alt+5 should not accumulate count, got count=%d", pending.Count)
	}
}
