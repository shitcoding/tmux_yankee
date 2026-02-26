package keymap

import (
	"fmt"
	"testing"
)

func TestParseKeyNotation(t *testing.T) {
	tests := []struct {
		input   string
		want    KeySpec
		wantErr bool
	}{
		{"h", Key('h'), false},
		{"H", Key('H'), false},
		{"$", Key('$'), false},
		{"C-d", Ctrl('d'), false},
		{"C-f", Ctrl('f'), false},
		{"M-h", Alt('h'), false},
		{"M-H", Alt('H'), false},
		{"Enter", Key(13), false},
		{"Tab", Key(9), false},
		{"Esc", Key(27), false},
		{"Space", Key(32), false},
		{"", KeySpec{}, true},
		{"C-", KeySpec{}, true},  // incomplete
		{"xx", KeySpec{}, true},  // unknown multi-char
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := ParseKeyNotation(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Errorf("ParseKeyNotation(%q) expected error, got %+v", tt.input, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("ParseKeyNotation(%q) unexpected error: %v", tt.input, err)
			}
			if got != tt.want {
				t.Errorf("ParseKeyNotation(%q) = %+v, want %+v", tt.input, got, tt.want)
			}
		})
	}
}

func TestParseBindings(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		check   func(Keymap) error
		wantErr bool
	}{
		{
			name:  "empty string",
			input: "",
			check: func(km Keymap) error { return nil },
		},
		{
			name:  "single bind",
			input: "H=line_end",
			check: func(km Keymap) error {
				action, ok := km.Lookup(Key('H'))
				if !ok || action != ActionLineEnd {
					return errorf("Direct[Key('H')] = (%q, %v), want (%q, true)", action, ok, ActionLineEnd)
				}
				return nil
			},
		},
		{
			name:  "ctrl bind",
			input: "C-d=half_page_down",
			check: func(km Keymap) error {
				action, ok := km.Lookup(Ctrl('d'))
				if !ok || action != ActionHalfPageDown {
					return errorf("Direct[Ctrl('d')] = (%q, %v), want (%q, true)", action, ok, ActionHalfPageDown)
				}
				return nil
			},
		},
		{
			name:  "unbind",
			input: "!H",
			check: func(km Keymap) error {
				action, ok := km.Lookup(Key('H'))
				if !ok || action != ActionNone {
					return errorf("Direct[Key('H')] = (%q, %v), want (%q, true)", action, ok, ActionNone)
				}
				return nil
			},
		},
		{
			name:  "multiple bindings",
			input: "H=line_end,L=line_start,C-n=scroll_line_down",
			check: func(km Keymap) error {
				if a, ok := km.Lookup(Key('H')); !ok || a != ActionLineEnd {
					return errorf("H: (%q, %v)", a, ok)
				}
				if a, ok := km.Lookup(Key('L')); !ok || a != ActionLineStart {
					return errorf("L: (%q, %v)", a, ok)
				}
				if a, ok := km.Lookup(Ctrl('n')); !ok || a != ActionScrollLineDown {
					return errorf("C-n: (%q, %v)", a, ok)
				}
				return nil
			},
		},
		{
			name:  "spaces around commas",
			input: " H=line_end , L=line_start ",
			check: func(km Keymap) error {
				if a, ok := km.Lookup(Key('H')); !ok || a != ActionLineEnd {
					return errorf("H: (%q, %v)", a, ok)
				}
				if a, ok := km.Lookup(Key('L')); !ok || a != ActionLineStart {
					return errorf("L: (%q, %v)", a, ok)
				}
				return nil
			},
		},
		{
			name:    "invalid action",
			input:   "H=nonexistent_action",
			wantErr: true,
		},
		{
			name:    "missing equals",
			input:   "H",
			wantErr: true,
		},
		{
			name:    "invalid key",
			input:   "C-=move_left",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			km, err := ParseBindings(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Errorf("ParseBindings(%q) expected error", tt.input)
				}
				return
			}
			if err != nil {
				t.Fatalf("ParseBindings(%q) unexpected error: %v", tt.input, err)
			}
			if tt.check != nil {
				if err := tt.check(km); err != nil {
					t.Error(err)
				}
			}
		})
	}
}

func errorf(format string, args ...interface{}) error {
	return fmt.Errorf(format, args...)
}
