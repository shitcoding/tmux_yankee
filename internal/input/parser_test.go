package input

import (
	"testing"

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

	t.Run("g alone then motion", func(t *testing.T) {
		p := NewParser()

		cmd := p.Parse('g')
		if cmd.Type != CommandNone {
			t.Errorf("'g' alone should return CommandNone, got %v", cmd.Type)
		}

		cmd = p.Parse('j')
		if cmd.Type != CommandNone {
			t.Errorf("'gj' should return CommandNone (invalid), got %v", cmd.Type)
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
		{"y yank", 'y', CommandYank},
		{"Enter yank", 13, CommandYank},
		{"Escape", 27, CommandEscape},
		{"L toggle mode", 'L', CommandToggleLineMode},
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
