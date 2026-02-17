package input_test

import (
	"testing"

	"github.com/shitcoding/tmux_yankee/internal/input"
)

func TestParseMouseScroll_WheelUp(t *testing.T) {
	p := input.NewParser()
	// SGR wheel-up: \x1b[<64;1;1M
	seq := []byte{0x1b, '[', '<', '6', '4', ';', '1', ';', '1', 'M'}
	var cmds []input.Command
	for _, b := range seq {
		if cmd := p.Parse(b); cmd.Type != input.CommandNone {
			cmds = append(cmds, cmd)
		}
	}
	if len(cmds) != 1 {
		t.Fatalf("expected 1 command, got %d", len(cmds))
	}
	if cmds[0].Type != input.CommandMouseScroll {
		t.Fatalf("expected CommandMouseScroll, got %v", cmds[0].Type)
	}
	if cmds[0].ScrollDirection != input.ScrollUp {
		t.Fatalf("expected ScrollUp, got %v", cmds[0].ScrollDirection)
	}
}

func TestParseMouseScroll_WheelDown(t *testing.T) {
	p := input.NewParser()
	// SGR wheel-down: \x1b[<65;1;1M
	seq := []byte{0x1b, '[', '<', '6', '5', ';', '1', ';', '1', 'M'}
	var cmds []input.Command
	for _, b := range seq {
		if cmd := p.Parse(b); cmd.Type != input.CommandNone {
			cmds = append(cmds, cmd)
		}
	}
	if len(cmds) != 1 {
		t.Fatalf("expected 1 command, got %d", len(cmds))
	}
	if cmds[0].ScrollDirection != input.ScrollDown {
		t.Fatalf("expected ScrollDown, got %v", cmds[0].ScrollDirection)
	}
}

func TestParseMouseScroll_NonWheelIgnored(t *testing.T) {
	p := input.NewParser()
	// SGR left-click: \x1b[<0;1;1M — button 0, not wheel
	seq := []byte{0x1b, '[', '<', '0', ';', '1', ';', '1', 'M'}
	var cmds []input.Command
	for _, b := range seq {
		if cmd := p.Parse(b); cmd.Type != input.CommandNone {
			cmds = append(cmds, cmd)
		}
	}
	// Non-wheel mouse events produce no command
	for _, cmd := range cmds {
		if cmd.Type == input.CommandMouseScroll {
			t.Fatal("non-wheel mouse event should not produce CommandMouseScroll")
		}
	}
}

func TestParseMouseScroll_MalformedSequenceRecovery(t *testing.T) {
	p := input.NewParser()
	// Start a mouse sequence but never terminate it (> 32 bytes)
	// This should not leave the parser stuck.
	start := []byte{0x1b, '[', '<'}
	for _, b := range start {
		p.Parse(b)
	}
	// Feed 30 more bytes (total 33 = prefix 3 + 30) without M/m terminator
	for i := 0; i < 30; i++ {
		p.Parse('5')
	}
	// Parser should have recovered: normal key 'j' must now produce CommandMotion
	cmd := p.Parse('j')
	if cmd.Type != input.CommandMotion {
		t.Fatalf("parser stuck after malformed sequence: expected CommandMotion for 'j', got %v", cmd.Type)
	}
}

func TestParseMouseScroll_InterleavedKeyAfterMouse(t *testing.T) {
	p := input.NewParser()
	// Wheel-up sequence followed by 'j'
	seq := []byte{0x1b, '[', '<', '6', '4', ';', '1', ';', '1', 'M', 'j'}
	var cmds []input.Command
	for _, b := range seq {
		if cmd := p.Parse(b); cmd.Type != input.CommandNone {
			cmds = append(cmds, cmd)
		}
	}
	if len(cmds) != 2 {
		t.Fatalf("expected 2 commands (scroll + motion), got %d", len(cmds))
	}
	if cmds[0].Type != input.CommandMouseScroll {
		t.Fatalf("first command should be CommandMouseScroll, got %v", cmds[0].Type)
	}
	if cmds[1].Type != input.CommandMotion {
		t.Fatalf("second command should be CommandMotion (j), got %v", cmds[1].Type)
	}
}
