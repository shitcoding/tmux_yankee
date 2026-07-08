package input

import (
	"testing"

	"github.com/shitcoding/tmux_yankee/internal/keymap"
	"github.com/shitcoding/tmux_yankee/internal/motion"
)

func TestParser_SingleMotions(t *testing.T) {
	tests := []struct {
		name     string
		input    byte
		wantType CommandType
		wantMot  motion.Motion
		wantCnt  int
	}{
		{"j down", 'j', CommandMotion, motion.MotionDown, 0},
		{"k up", 'k', CommandMotion, motion.MotionUp, 0},
		{"h left", 'h', CommandMotion, motion.MotionLeft, 0},
		{"l right", 'l', CommandMotion, motion.MotionRight, 0},
		{"0 line start", '0', CommandMotion, motion.MotionLineStart, 0},
		{"$ line end", '$', CommandMotion, motion.MotionLineEnd, 0},
		{"G last line", 'G', CommandMotion, motion.MotionLastLine, 0},
		{"Ctrl-D half page down", 4, CommandMotion, motion.MotionHalfPageDown, 0},
		{"Ctrl-U half page up", 21, CommandMotion, motion.MotionHalfPageUp, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewParser()
			cmd := p.Parse(tt.input)

			if cmd.Type != tt.wantType {
				t.Errorf("Type = %v, want %v", cmd.Type, tt.wantType)
			}
			if cmd.Motion != tt.wantMot {
				t.Errorf("Motion = %v, want %v", cmd.Motion, tt.wantMot)
			}
			if cmd.Count != tt.wantCnt {
				t.Errorf("Count = %v, want %v", cmd.Count, tt.wantCnt)
			}
		})
	}
}

func TestParser_CountedMotions(t *testing.T) {
	tests := []struct {
		name     string
		input    []byte
		wantType CommandType
		wantMot  motion.Motion
		wantCnt  int
	}{
		{"5j", []byte{'5', 'j'}, CommandMotion, motion.MotionDown, 5},
		{"10k", []byte{'1', '0', 'k'}, CommandMotion, motion.MotionUp, 10},
		{"42h", []byte{'4', '2', 'h'}, CommandMotion, motion.MotionLeft, 42},
		{"7l", []byte{'7', 'l'}, CommandMotion, motion.MotionRight, 7},
		{"3G", []byte{'3', 'G'}, CommandMotion, motion.MotionLastLine, 3},
		{"100$", []byte{'1', '0', '0', '$'}, CommandMotion, motion.MotionLineEnd, 100},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewParser()
			var cmd Command

			// Feed all input bytes
			for _, b := range tt.input {
				cmd = p.Parse(b)
			}

			if cmd.Type != tt.wantType {
				t.Errorf("Type = %v, want %v", cmd.Type, tt.wantType)
			}
			if cmd.Motion != tt.wantMot {
				t.Errorf("Motion = %v, want %v", cmd.Motion, tt.wantMot)
			}
			if cmd.Count != tt.wantCnt {
				t.Errorf("Count = %v, want %v", cmd.Count, tt.wantCnt)
			}
		})
	}
}

func TestParser_ggPrefix(t *testing.T) {
	tests := []struct {
		name     string
		input    []byte
		wantType CommandType
		wantMot  motion.Motion
		wantCnt  int
	}{
		{"gg", []byte{'g', 'g'}, CommandMotion, motion.MotionFirstLine, 0},
		{"5gg", []byte{'5', 'g', 'g'}, CommandMotion, motion.MotionFirstLine, 5},
		{"42gg", []byte{'4', '2', 'g', 'g'}, CommandMotion, motion.MotionFirstLine, 42},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewParser()
			var cmd Command

			// Feed all input bytes
			for _, b := range tt.input {
				cmd = p.Parse(b)
			}

			if cmd.Type != tt.wantType {
				t.Errorf("Type = %v, want %v", cmd.Type, tt.wantType)
			}
			if cmd.Motion != tt.wantMot {
				t.Errorf("Motion = %v, want %v", cmd.Motion, tt.wantMot)
			}
			if cmd.Count != tt.wantCnt {
				t.Errorf("Count = %v, want %v", cmd.Count, tt.wantCnt)
			}
		})
	}
}

func TestParser_yyPrefix(t *testing.T) {
	t.Run("yy produces CommandYankLine", func(t *testing.T) {
		p := NewParser()
		if cmd := p.Parse('y'); cmd.Type != CommandNone {
			t.Errorf("first y: want CommandNone, got %v", cmd.Type)
		}
		if cmd := p.Parse('y'); cmd.Type != CommandYankLine {
			t.Errorf("second y: want CommandYankLine, got %v", cmd.Type)
		}
	})

	t.Run("y followed by non-y clears pending", func(t *testing.T) {
		p := NewParser()
		p.Parse('y')
		if cmd := p.Parse('j'); cmd.Type != CommandNone {
			t.Errorf("y+j: want CommandNone (cleared), got %v", cmd.Type)
		}
	})
}

func TestParser_ZeroHandling(t *testing.T) {
	t.Run("0 as motion when no count", func(t *testing.T) {
		p := NewParser()
		cmd := p.Parse('0')

		if cmd.Type != CommandMotion {
			t.Errorf("Type = %v, want CommandMotion", cmd.Type)
		}
		if cmd.Motion != motion.MotionLineStart {
			t.Errorf("Motion = %v, want MotionLineStart", cmd.Motion)
		}
		if cmd.Count != 0 {
			t.Errorf("Count = %v, want 0", cmd.Count)
		}
	})

	t.Run("0 as digit when count exists", func(t *testing.T) {
		p := NewParser()

		// Parse "10j"
		cmd := p.Parse('1')
		if cmd.Type != CommandNone {
			t.Errorf("Intermediate '1': Type = %v, want CommandNone", cmd.Type)
		}

		cmd = p.Parse('0')
		if cmd.Type != CommandNone {
			t.Errorf("Intermediate '0': Type = %v, want CommandNone", cmd.Type)
		}

		cmd = p.Parse('j')
		if cmd.Type != CommandMotion {
			t.Errorf("Type = %v, want CommandMotion", cmd.Type)
		}
		if cmd.Count != 10 {
			t.Errorf("Count = %v, want 10", cmd.Count)
		}
	})
}

func TestParser_InvalidSequences(t *testing.T) {
	t.Run("g alone then invalid", func(t *testing.T) {
		p := NewParser()

		cmd := p.Parse('g')
		if cmd.Type != CommandNone {
			t.Errorf("'g' alone should return CommandNone, got %v", cmd.Type)
		}

		cmd = p.Parse('x')
		if cmd.Type != CommandNone {
			t.Errorf("'gx' should return CommandNone, got %v", cmd.Type)
		}

		// Verify pending is cleared - next command should work normally
		cmd = p.Parse('j')
		if cmd.Type != CommandMotion {
			t.Errorf("After invalid sequence, new command should work. Got Type = %v", cmd.Type)
		}
		if cmd.Count != 0 {
			t.Errorf("Count should be 0, got %v", cmd.Count)
		}
	})

	t.Run("g alone then motion gj", func(t *testing.T) {
		p := NewParser()

		cmd := p.Parse('g')
		if cmd.Type != CommandNone {
			t.Errorf("'g' alone should return CommandNone, got %v", cmd.Type)
		}

		cmd = p.Parse('j')
		if cmd.Type != CommandDisplayLineDown {
			t.Errorf("'gj' should return CommandDisplayLineDown, got %v", cmd.Type)
		}
	})

	t.Run("gk display line up", func(t *testing.T) {
		p := NewParser()
		p.Parse('g')
		cmd := p.Parse('k')
		if cmd.Type != CommandDisplayLineUp {
			t.Errorf("'gk' should return CommandDisplayLineUp, got %v", cmd.Type)
		}
	})

	t.Run("5gj display line down with count", func(t *testing.T) {
		p := NewParser()
		p.Parse('5')
		p.Parse('g')
		cmd := p.Parse('j')
		if cmd.Type != CommandDisplayLineDown {
			t.Errorf("'5gj' should return CommandDisplayLineDown, got %v", cmd.Type)
		}
		if cmd.Count != 5 {
			t.Errorf("'5gj' Count = %v, want 5", cmd.Count)
		}
	})

	t.Run("count persists until motion", func(t *testing.T) {
		p := NewParser()

		cmd := p.Parse('5')
		if cmd.Type != CommandNone {
			t.Errorf("'5' alone should return CommandNone, got %v", cmd.Type)
		}

		// Count should persist and apply to next motion
		cmd = p.Parse('j')
		if cmd.Type != CommandMotion {
			t.Errorf("'5j' should be a motion, got Type = %v", cmd.Type)
		}
		if cmd.Count != 5 {
			t.Errorf("Count should be 5, got %v", cmd.Count)
		}

		// After motion, count should clear
		cmd = p.Parse('k')
		if cmd.Count != 0 {
			t.Errorf("After motion, count should clear. Got Count = %v", cmd.Count)
		}
	})
}

func TestParser_OtherCommands(t *testing.T) {
	tests := []struct {
		name     string
		input    byte
		wantType CommandType
	}{
		{"v visual char", 'v', CommandVisual},
		{"V visual line", 'V', CommandVisualLine},
		{"Enter yank", 13, CommandYank},
		{"L screen bottom", 'L', CommandMotion},
		{"q quit", 'q', CommandQuit},
		{"Ctrl-C quit", 3, CommandQuit},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewParser()
			cmd := p.Parse(tt.input)

			if cmd.Type != tt.wantType {
				t.Errorf("Type = %v, want %v", cmd.Type, tt.wantType)
			}
		})
	}

	// Alt+key detection: ESC followed by a printable byte is treated as Alt+key.
	// With no Alt binding, both bytes are discarded (CommandNone).
	// Standalone ESC (via Flush) still produces CommandEscape.
	t.Run("Escape resolves on next non-bracket byte", func(t *testing.T) {
		p := NewParser()
		// ESC is buffered.
		cmd := p.Parse(27)
		if cmd.Type != CommandNone {
			t.Errorf("ESC alone: Type = %v, want CommandNone (deferred)", cmd.Type)
		}
		// 'q' is printable → treated as Alt+q (unbound → discarded).
		cmd = p.Parse('q')
		if cmd.Type != CommandNone {
			t.Errorf("ESC+q (Alt+q unbound): Type = %v, want CommandNone", cmd.Type)
		}
		// Standalone ESC via Flush still works.
		cmd = p.Parse(27)
		if cmd.Type != CommandNone {
			t.Errorf("ESC alone: Type = %v, want CommandNone (buffered)", cmd.Type)
		}
		cmd = p.Flush()
		if cmd.Type != CommandEscape {
			t.Errorf("Flush standalone ESC: Type = %v, want CommandEscape", cmd.Type)
		}
	})
}

func TestParser_CustomToggleKey(t *testing.T) {
	// Parser with 'P' as toggle key: 'P' produces ToggleMode (P is not in keymap)
	p := NewParserWithKeys('P', 'w')

	t.Run("P produces ToggleMode", func(t *testing.T) {
		cmd := p.Parse('P')
		if cmd.Type != CommandToggleLineMode {
			t.Errorf("Type = %v, want CommandToggleLineMode", cmd.Type)
		}
	})

	t.Run("L does not produce ToggleMode", func(t *testing.T) {
		cmd := p.Parse('L')
		if cmd.Type == CommandToggleLineMode {
			t.Errorf("Type = CommandToggleLineMode, want something else (L is not the toggle key)")
		}
	})
}

func TestParser_PendingState(t *testing.T) {
	t.Run("count accumulation", func(t *testing.T) {
		p := NewParser()

		p.Parse('5')
		pending := p.PendingState()
		if !pending.HasCount || pending.Count != 5 {
			t.Errorf("After '5': HasCount=%v, Count=%v, want true, 5", pending.HasCount, pending.Count)
		}

		p.Parse('2')
		pending = p.PendingState()
		if !pending.HasCount || pending.Count != 52 {
			t.Errorf("After '52': HasCount=%v, Count=%v, want true, 52", pending.HasCount, pending.Count)
		}
	})

	t.Run("g prefix", func(t *testing.T) {
		p := NewParser()

		p.Parse('g')
		pending := p.PendingState()
		if pending.Prefix != 'g' {
			t.Errorf("After 'g': Prefix=%v, want 'g'", pending.Prefix)
		}
	})

	t.Run("clear after command", func(t *testing.T) {
		p := NewParser()

		p.Parse('5')
		p.Parse('j')
		pending := p.PendingState()
		if pending.HasCount || pending.Count != 0 || pending.Prefix != 0 {
			t.Errorf("After '5j': pending should be cleared, got %+v", pending)
		}
	})
}

func TestParser_CountWithMultipleDigits(t *testing.T) {
	tests := []struct {
		name  string
		input []byte
		want  int
	}{
		{"999j", []byte{'9', '9', '9', 'j'}, 999},
		{"1234k", []byte{'1', '2', '3', '4', 'k'}, 1234},
		{"10000G", []byte{'1', '0', '0', '0', '0', 'G'}, 10000},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewParser()
			var cmd Command

			for _, b := range tt.input {
				cmd = p.Parse(b)
			}

			if cmd.Count != tt.want {
				t.Errorf("Count = %v, want %v", cmd.Count, tt.want)
			}
		})
	}
}

func TestParser_CharSearch_FindForward(t *testing.T) {
	p := NewParser()
	cmd := p.Parse('f')
	if cmd.Type != CommandNone {
		t.Fatalf("'f' should return CommandNone, got %d", cmd.Type)
	}
	cmd = p.Parse('a')
	if cmd.Type != CommandCharSearch {
		t.Fatalf("'fa' should return CommandCharSearch, got %d", cmd.Type)
	}
	if cmd.SearchKind != SearchFindForward {
		t.Errorf("SearchKind = %d, want SearchFindForward", cmd.SearchKind)
	}
	if cmd.SearchChar != 'a' {
		t.Errorf("SearchChar = %c, want 'a'", cmd.SearchChar)
	}
}

func TestParser_CharSearch_TillForward(t *testing.T) {
	p := NewParser()
	p.Parse('t')
	cmd := p.Parse('x')
	if cmd.Type != CommandCharSearch {
		t.Fatalf("'tx' should return CommandCharSearch, got %d", cmd.Type)
	}
	if cmd.SearchKind != SearchTillForward {
		t.Errorf("SearchKind = %d, want SearchTillForward", cmd.SearchKind)
	}
	if cmd.SearchChar != 'x' {
		t.Errorf("SearchChar = %c, want 'x'", cmd.SearchChar)
	}
}

func TestParser_CharSearch_FindBackward(t *testing.T) {
	p := NewParser()
	p.Parse('F')
	cmd := p.Parse('b')
	if cmd.Type != CommandCharSearch {
		t.Fatalf("'Fb' should return CommandCharSearch, got %d", cmd.Type)
	}
	if cmd.SearchKind != SearchFindBackward {
		t.Errorf("SearchKind = %d, want SearchFindBackward", cmd.SearchKind)
	}
}

func TestParser_CharSearch_TillBackward(t *testing.T) {
	p := NewParser()
	p.Parse('T')
	cmd := p.Parse('c')
	if cmd.Type != CommandCharSearch {
		t.Fatalf("'Tc' should return CommandCharSearch, got %d", cmd.Type)
	}
	if cmd.SearchKind != SearchTillBackward {
		t.Errorf("SearchKind = %d, want SearchTillBackward", cmd.SearchKind)
	}
}

func TestParser_CharSearch_WithCount(t *testing.T) {
	p := NewParser()
	p.Parse('3')
	p.Parse('f')
	cmd := p.Parse('a')
	if cmd.Type != CommandCharSearch {
		t.Fatalf("'3fa' should return CommandCharSearch, got %d", cmd.Type)
	}
	if cmd.Count != 3 {
		t.Errorf("Count = %d, want 3", cmd.Count)
	}
	if cmd.SearchChar != 'a' {
		t.Errorf("SearchChar = %c, want 'a'", cmd.SearchChar)
	}
}

func TestParser_CharSearch_Repeat(t *testing.T) {
	p := NewParser()
	cmd := p.Parse(';')
	if cmd.Type != CommandCharSearch {
		t.Fatalf("';' should return CommandCharSearch, got %d", cmd.Type)
	}
	if cmd.SearchKind != SearchRepeat {
		t.Errorf("SearchKind = %d, want SearchRepeat", cmd.SearchKind)
	}
}

func TestParser_CharSearch_RepeatReverse(t *testing.T) {
	p := NewParser()
	cmd := p.Parse(',')
	if cmd.Type != CommandCharSearch {
		t.Fatalf("',' should return CommandCharSearch, got %d", cmd.Type)
	}
	if cmd.SearchKind != SearchRepeatReverse {
		t.Errorf("SearchKind = %d, want SearchRepeatReverse", cmd.SearchKind)
	}
}

func TestParser_CharSearch_CountWithRepeat(t *testing.T) {
	p := NewParser()
	p.Parse('2')
	cmd := p.Parse(';')
	if cmd.Type != CommandCharSearch {
		t.Fatalf("'2;' should return CommandCharSearch, got %d", cmd.Type)
	}
	if cmd.Count != 2 {
		t.Errorf("Count = %d, want 2", cmd.Count)
	}
	if cmd.SearchKind != SearchRepeat {
		t.Errorf("SearchKind = %d, want SearchRepeat", cmd.SearchKind)
	}
}

func TestParser_CharSearch_EscapeCancelsPrefix(t *testing.T) {
	p := NewParser()
	p.Parse('f')
	// ESC byte triggers mouse detection, but eventually resolves
	// After 'f' prefix, any non-target should cancel
	cmd := p.Parse(27) // ESC
	if cmd.Type == CommandCharSearch {
		t.Error("ESC after 'f' should not produce CommandCharSearch")
	}
}

func TestParser_Flush_PendingESC(t *testing.T) {
	p := NewParser()
	// ESC gets buffered for mouse disambiguation
	cmd := p.Parse(0x1b)
	if cmd.Type != CommandNone {
		t.Fatalf("ESC should buffer: got type %d", cmd.Type)
	}
	// Flush resolves it as standalone ESC
	cmd = p.Flush()
	if cmd.Type != CommandEscape {
		t.Errorf("Flush after ESC: got type %d, want CommandEscape", cmd.Type)
	}
}

func TestParser_Flush_NoOp(t *testing.T) {
	p := NewParser()
	// Nothing pending — flush is no-op
	cmd := p.Flush()
	if cmd.Type != CommandNone {
		t.Errorf("Flush with nothing pending: got type %d, want CommandNone", cmd.Type)
	}
}

func TestParser_Tab_JumpListForward(t *testing.T) {
	p := NewParser()
	cmd := p.Parse(9) // Tab/Ctrl-I = 0x09
	if cmd.Type != CommandJumpListForward {
		t.Errorf("Tab/Ctrl-I: got type %d, want CommandJumpListForward", cmd.Type)
	}
}

func TestParser_ShiftTab_DemoPrev(t *testing.T) {
	p := NewParser()
	// Shift+Tab sends ESC [ Z
	cmd := p.Parse(0x1b)
	if cmd.Type != CommandNone {
		t.Fatalf("ESC: got type %d, want CommandNone (buffered)", cmd.Type)
	}
	cmd = p.Parse('[')
	if cmd.Type != CommandNone {
		t.Fatalf("ESC [: got type %d, want CommandNone (buffered)", cmd.Type)
	}
	cmd = p.Parse('Z')
	if cmd.Type != CommandDemoPrev {
		t.Errorf("ESC [ Z: got type %d, want CommandDemoPrev", cmd.Type)
	}
}

func TestParser_Flush_ClearsPendingCount(t *testing.T) {
	p := NewParser()
	// Accumulate count, then ESC via Flush
	p.Parse('5')
	p.Parse(0x1b)
	cmd := p.Flush()
	if cmd.Type != CommandEscape {
		t.Fatalf("Flush after 5+ESC: got type %d, want CommandEscape", cmd.Type)
	}
	// Pending count should be cleared — next motion should have count=0
	cmd = p.Parse('j')
	if cmd.Type != CommandMotion {
		t.Fatalf("j after ESC: got type %d, want CommandMotion", cmd.Type)
	}
	if cmd.Count != 0 {
		t.Errorf("j after ESC: count = %d, want 0 (ESC should clear pending)", cmd.Count)
	}
}

func TestParser_Flush_ClearsPendingPrefix(t *testing.T) {
	p := NewParser()
	// Set 'f' prefix, then ESC via Flush
	p.Parse('f')
	p.Parse(0x1b)
	cmd := p.Flush()
	if cmd.Type != CommandEscape {
		t.Fatalf("Flush after f+ESC: got type %d, want CommandEscape", cmd.Type)
	}
	// Prefix should be cleared — 'j' should be motion, not char search target
	cmd = p.Parse('j')
	if cmd.Type != CommandMotion {
		t.Fatalf("j after f+ESC: got type %d, want CommandMotion", cmd.Type)
	}
}

func TestParser_ESC_ClearsPendingCount_InlineResolution(t *testing.T) {
	p := NewParser()
	// "5 ESC q" where ESC and q arrive in same read.
	// Alt+key detection: ESC+'q' is treated as Alt+q (unbound → discarded).
	// The pending count must be cleared so it doesn't leak to the next key.
	p.Parse('5')
	p.Parse(0x1b)
	cmd := p.Parse('q') // Alt+q: unbound, discarded
	if cmd.Type != CommandNone {
		t.Fatalf("5+ESC+q: got type %d, want CommandNone (Alt+q discarded)", cmd.Type)
	}
	// Count was cleared by ESC handler. Next 'q' should be CommandQuit with count=0.
	cmd = p.Parse('q')
	if cmd.Type != CommandQuit {
		t.Fatalf("q after Alt: got type %d, want CommandQuit", cmd.Type)
	}
	if cmd.Count != 0 {
		t.Fatalf("q after Alt: count=%d, want 0 (count should be cleared)", cmd.Count)
	}
}

func TestParser_CSI_ArrowKeys(t *testing.T) {
	tests := []struct {
		name    string
		final   byte
		wantMot motion.Motion
	}{
		{"arrow up", 'A', motion.MotionUp},
		{"arrow down", 'B', motion.MotionDown},
		{"arrow right", 'C', motion.MotionRight},
		{"arrow left", 'D', motion.MotionLeft},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewParser()
			p.Parse(0x1b)
			p.Parse('[')
			cmd := p.Parse(tt.final)
			if cmd.Type != CommandMotion {
				t.Fatalf("ESC[%c: got type %d, want CommandMotion", tt.final, cmd.Type)
			}
			if cmd.Motion != tt.wantMot {
				t.Errorf("ESC[%c: motion = %v, want %v", tt.final, cmd.Motion, tt.wantMot)
			}
		})
	}
}

func TestParser_CSI_HomeEnd(t *testing.T) {
	tests := []struct {
		name    string
		final   byte
		wantMot motion.Motion
	}{
		{"Home", 'H', motion.MotionLineStart},
		{"End", 'F', motion.MotionLineEnd},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewParser()
			p.Parse(0x1b)
			p.Parse('[')
			cmd := p.Parse(tt.final)
			if cmd.Type != CommandMotion {
				t.Fatalf("ESC[%c: got type %d, want CommandMotion", tt.final, cmd.Type)
			}
			if cmd.Motion != tt.wantMot {
				t.Errorf("ESC[%c: motion = %v, want %v", tt.final, cmd.Motion, tt.wantMot)
			}
		})
	}
}

func TestParser_CSI_PageUpDown(t *testing.T) {
	tests := []struct {
		name    string
		param   byte
		wantMot motion.Motion
	}{
		{"Page Up ESC[5~", '5', motion.MotionHalfPageUp},
		{"Page Down ESC[6~", '6', motion.MotionHalfPageDown},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewParser()
			p.Parse(0x1b)
			p.Parse('[')
			p.Parse(tt.param)
			cmd := p.Parse('~')
			if cmd.Type != CommandMotion {
				t.Fatalf("%s: got type %d, want CommandMotion", tt.name, cmd.Type)
			}
			if cmd.Motion != tt.wantMot {
				t.Errorf("%s: motion = %v, want %v", tt.name, cmd.Motion, tt.wantMot)
			}
		})
	}
}

func TestParser_CSI_UnknownConsumedSilently(t *testing.T) {
	// ESC[X where X is an unrecognized CSI final should not produce ESC
	p := NewParser()
	p.Parse(0x1b)
	p.Parse('[')
	cmd := p.Parse('X') // Unknown CSI final
	if cmd.Type != CommandNone {
		t.Errorf("ESC[X: got type %d, want CommandNone (silently consumed)", cmd.Type)
	}
}

func TestParser_CSI_ArrowDoesNotEmitEscape(t *testing.T) {
	// Regression: arrow keys used to emit CommandEscape + misclassified byte
	p := NewParser()
	p.Parse(0x1b)
	p.Parse('[')
	cmd := p.Parse('A') // Arrow up
	if cmd.Type == CommandEscape {
		t.Error("ESC[A should NOT produce CommandEscape (it's arrow up)")
	}
}

func TestParser_Flush_IncompleteCSI(t *testing.T) {
	p := NewParser()
	// Start a parameterized CSI: ESC [ 5 (incomplete — no final byte)
	p.Parse(0x1b)
	p.Parse('[')
	p.Parse('5')
	// Flush should discard the incomplete sequence
	cmd := p.Flush()
	if cmd.Type != CommandNone {
		t.Errorf("Flush of incomplete CSI: got type %d, want CommandNone", cmd.Type)
	}
}

func TestParser_Flush_MouseSequenceNotFlushed(t *testing.T) {
	p := NewParser()
	// Start a real mouse sequence: ESC [ <
	p.Parse(0x1b)
	p.Parse('[')
	p.Parse('<')
	// Now in mouse mode — Flush should NOT emit ESC
	cmd := p.Flush()
	if cmd.Type != CommandNone {
		t.Errorf("Flush during mouse sequence: got type %d, want CommandNone", cmd.Type)
	}
}

func TestParser_Flush_ReturnsDeferredCommand(t *testing.T) {
	// ESC followed by a non-printable byte (Ctrl code) defers the byte's
	// command. Flush must return the deferred command.
	// Note: ESC + printable is now treated as Alt+key (discarded if unbound).
	p := NewParser()
	cmd := p.Parse(0x1b)
	if cmd.Type != CommandNone {
		t.Fatalf("ESC byte: got type %d, want CommandNone", cmd.Type)
	}
	// Ctrl-D (byte 4) is non-printable, triggers ESC fallthrough + deferred.
	cmd = p.Parse(0x04)
	if cmd.Type != CommandEscape {
		t.Fatalf("ESC+Ctrl-D: got type %d, want CommandEscape", cmd.Type)
	}
	// Deferred Ctrl-D (half page down) must be returned by Flush.
	cmd = p.Flush()
	if cmd.Type != CommandMotion {
		t.Errorf("Flush deferred: got type %d, want CommandMotion", cmd.Type)
	}
	if cmd.Motion != motion.MotionHalfPageDown {
		t.Errorf("Flush deferred motion: got %v, want MotionHalfPageDown", cmd.Motion)
	}
}

func TestParser_CSI_ClearsPendingCount(t *testing.T) {
	// Regression: "5 ESC[A j" should NOT apply count=5 to j.
	// The CSI arrow should clear the pending count.
	p := NewParser()
	// Accumulate count 5
	p.Parse('5')
	// Send ESC [ A (arrow up)
	p.Parse(0x1b)
	p.Parse('[')
	cmd := p.Parse('A')
	if cmd.Type != CommandMotion || cmd.Motion != motion.MotionUp {
		t.Fatalf("CSI A: got type=%d motion=%v, want MotionUp", cmd.Type, cmd.Motion)
	}
	// Now press 'j' — count should be 0 (default, no explicit count), not 5
	cmd = p.Parse('j')
	if cmd.Type != CommandMotion {
		t.Fatalf("j after CSI: got type=%d, want CommandMotion", cmd.Type)
	}
	if cmd.Count != 0 {
		t.Errorf("j count after CSI: got %d, want 0 (pending should be cleared)", cmd.Count)
	}
}

func TestParser_TextObject_InnerParen(t *testing.T) {
	p := NewParser()
	// v → visual mode command
	cmd := p.Parse('v')
	if cmd.Type != CommandVisual {
		t.Fatalf("v: got type=%d, want CommandVisual(%d)", cmd.Type, CommandVisual)
	}
	// i → should set pending prefix (text object prefix)
	cmd = p.Parse('i')
	if cmd.Type != CommandNone {
		t.Fatalf("i: got type=%d, want CommandNone (pending text object prefix)", cmd.Type)
	}
	// ( → should produce text object
	cmd = p.Parse('(')
	if cmd.Type != CommandTextObject {
		t.Fatalf("(: got type=%d, want CommandTextObject(%d)", cmd.Type, CommandTextObject)
	}
	if cmd.TextObject != "inner_paren" {
		t.Errorf("(: got TextObject=%q, want %q", cmd.TextObject, "inner_paren")
	}
}

func TestParser_TextObject_InnerBracket(t *testing.T) {
	p := NewParser()
	p.Parse('v') // visual mode
	p.Parse('i') // text object prefix
	cmd := p.Parse('[')
	if cmd.Type != CommandTextObject {
		t.Fatalf("[: got type=%d, want CommandTextObject(%d)", cmd.Type, CommandTextObject)
	}
	if cmd.TextObject != "inner_bracket" {
		t.Errorf("[: got TextObject=%q, want %q", cmd.TextObject, "inner_bracket")
	}
}

func TestParser_TextObject_InnerAngle(t *testing.T) {
	p := NewParser()
	p.Parse('v') // visual mode
	p.Parse('i') // text object prefix
	cmd := p.Parse('<')
	if cmd.Type != CommandTextObject {
		t.Fatalf("<: got type=%d, want CommandTextObject(%d)", cmd.Type, CommandTextObject)
	}
	if cmd.TextObject != "inner_angle" {
		t.Errorf("<: got TextObject=%q, want %q", cmd.TextObject, "inner_angle")
	}
}

func TestParser_CSI_ClearsPendingPrefix(t *testing.T) {
	// Regression: "f ESC[B j" should NOT treat j as char-search target.
	// The CSI arrow should clear the pending f prefix.
	p := NewParser()
	// Start char search prefix
	p.Parse('f')
	// Send ESC [ B (arrow down)
	p.Parse(0x1b)
	p.Parse('[')
	cmd := p.Parse('B')
	if cmd.Type != CommandMotion || cmd.Motion != motion.MotionDown {
		t.Fatalf("CSI B: got type=%d motion=%v, want MotionDown", cmd.Type, cmd.Motion)
	}
	// Now press 'j' — should be plain motion, not char search for 'j'
	cmd = p.Parse('j')
	if cmd.Type != CommandMotion {
		t.Fatalf("j after CSI: got type=%d, want CommandMotion", cmd.Type)
	}
	if cmd.Motion != motion.MotionDown {
		t.Errorf("j motion after CSI: got %v, want MotionDown (not char search)", cmd.Motion)
	}
}

func TestParser_SetKeymap(t *testing.T) {
	km1 := keymap.DefaultKeymap()
	p := NewParserWithKeymap('L', 'w', km1)

	// Initially 'H' should be screen_top
	cmd := p.Parse('H')
	if cmd.Type != CommandMotion || cmd.Motion != motion.MotionScreenTop {
		t.Errorf("before SetKeymap: H = %+v, want CommandMotion/ScreenTop", cmd)
	}

	// Swap keymap: rebind H to line_end
	km2 := km1
	km2.Direct = make(map[keymap.KeySpec]keymap.Action, len(km1.Direct))
	for k, v := range km1.Direct {
		km2.Direct[k] = v
	}
	km2.Direct[keymap.Key('H')] = keymap.ActionLineEnd
	p.SetKeymap(km2)

	// Now H should be line_end
	cmd = p.Parse('H')
	if cmd.Type != CommandMotion || cmd.Motion != motion.MotionLineEnd {
		t.Errorf("after SetKeymap: H = %+v, want CommandMotion/LineEnd", cmd)
	}
}

func TestFlush_ClearsCSIPendingState(t *testing.T) {
	// M3: Flush() must clear pending count/prefix when discarding incomplete CSI.
	// Simulate: user types "3" (count), then ESC [ 5 (incomplete CSI param), then Flush.
	// After flush, the pending count must be zero so the next key isn't affected.
	p := NewParserWithKeys('L', 'w')

	// Type a count prefix
	p.Parse('3')
	ps := p.PendingState()
	if ps.Count != 3 {
		t.Fatalf("expected pending count 3, got %d", ps.Count)
	}

	// Start a CSI sequence with a parameter byte to enter inCSI mode:
	// ESC → mouseBuf=[ESC], [ → mouseBuf=[ESC,[], 5 → inCSI=true
	p.Parse(0x1b) // ESC
	p.Parse('[')   // ESC [
	p.Parse('5')   // CSI parameter byte → enters inCSI mode

	// Flush discards incomplete CSI
	cmd := p.Flush()
	if cmd.Type != CommandNone {
		t.Errorf("Flush() returned %v, want CommandNone", cmd.Type)
	}

	// Pending state must be cleared
	ps = p.PendingState()
	if ps.Count != 0 {
		t.Errorf("after Flush of incomplete CSI, pending count = %d, want 0", ps.Count)
	}
}

func TestParseSearchByte_Unicode(t *testing.T) {
	// M7: Unicode search input must work (multi-byte UTF-8 sequences).
	p := NewParserWithKeys('L', 'w')

	// Enter search mode
	cmd := p.Parse('/')
	if cmd.Type != CommandSearchForward {
		t.Fatalf("expected CommandSearchForward, got %v", cmd.Type)
	}

	// Type a 2-byte UTF-8 character: ñ (U+00F1 = 0xC3 0xB1)
	cmd = p.Parse(0xC3)
	if cmd.Type != CommandNone {
		t.Errorf("first byte of ñ: got %v, want CommandNone", cmd.Type)
	}
	cmd = p.Parse(0xB1)
	if cmd.Type != CommandSearchUpdate {
		t.Fatalf("second byte of ñ: got %v, want CommandSearchUpdate", cmd.Type)
	}
	if cmd.SearchPattern != "ñ" {
		t.Errorf("search pattern = %q, want %q", cmd.SearchPattern, "ñ")
	}

	// Type a 3-byte UTF-8 character: 日 (U+65E5 = 0xE6 0x97 0xA5)
	p.Parse(0xE6)
	p.Parse(0x97)
	cmd = p.Parse(0xA5)
	if cmd.Type != CommandSearchUpdate {
		t.Fatalf("third byte of 日: got %v, want CommandSearchUpdate", cmd.Type)
	}
	if cmd.SearchPattern != "ñ日" {
		t.Errorf("search pattern = %q, want %q", cmd.SearchPattern, "ñ日")
	}

	// Confirm search
	cmd = p.Parse(13) // Enter
	if cmd.Type != CommandSearchConfirm {
		t.Fatalf("Enter: got %v, want CommandSearchConfirm", cmd.Type)
	}
	if cmd.SearchPattern != "ñ日" {
		t.Errorf("confirmed pattern = %q, want %q", cmd.SearchPattern, "ñ日")
	}
}
