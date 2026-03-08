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

func TestParseKeyBinding(t *testing.T) {
	tests := []struct {
		input       string
		isPrefixSeq bool
		prefix      byte
		second      byte
		spec        KeySpec
		wantErr     bool
	}{
		// Prefix sequences
		{"g-g", true, 'g', 'g', KeySpec{}, false},
		{"z-t", true, 'z', 't', KeySpec{}, false},
		{"y-y", true, 'y', 'y', KeySpec{}, false},
		{"g-j", true, 'g', 'j', KeySpec{}, false},
		{"z-b", true, 'z', 'b', KeySpec{}, false},
		// Ctrl/Alt are NOT prefix sequences (uppercase and lowercase)
		{"C-d", false, 0, 0, Ctrl('d'), false},
		{"M-t", false, 0, 0, Alt('t'), false},
		{"c-d", false, 0, 0, Ctrl('d'), false},
		{"m-t", false, 0, 0, Alt('t'), false},
		// Single keys
		{"h", false, 0, 0, Key('h'), false},
		{"H", false, 0, 0, Key('H'), false},
		{"$", false, 0, 0, Key('$'), false},
		// Special keys
		{"Enter", false, 0, 0, Key(13), false},
		// Errors
		{"", false, 0, 0, KeySpec{}, true},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := ParseKeyBinding(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Errorf("ParseKeyBinding(%q) expected error, got %+v", tt.input, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("ParseKeyBinding(%q) unexpected error: %v", tt.input, err)
			}
			if got.IsPrefixSeq != tt.isPrefixSeq {
				t.Errorf("IsPrefixSeq = %v, want %v", got.IsPrefixSeq, tt.isPrefixSeq)
			}
			if tt.isPrefixSeq {
				if got.Prefix != tt.prefix || got.Second != tt.second {
					t.Errorf("prefix=(%c,%c), want (%c,%c)", got.Prefix, got.Second, tt.prefix, tt.second)
				}
			} else {
				if got.Spec != tt.spec {
					t.Errorf("Spec = %+v, want %+v", got.Spec, tt.spec)
				}
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
		// Prefix sequence bindings
		{
			name:  "prefix bind g-g",
			input: "g-g=last_line",
			check: func(km Keymap) error {
				action, ok := km.LookupPrefix('g', 'g')
				if !ok || action != ActionLastLine {
					return errorf("Prefix['g']['g'] = (%q, %v), want (%q, true)", action, ok, ActionLastLine)
				}
				return nil
			},
		},
		{
			name:  "prefix unbind g-j",
			input: "!g-j",
			check: func(km Keymap) error {
				// Should have ActionNone in Prefix map (Merge will delete it)
				inner, ok := km.Prefix['g']
				if !ok {
					return errorf("Prefix['g'] not present")
				}
				if inner['j'] != ActionNone {
					return errorf("Prefix['g']['j'] = %q, want %q", inner['j'], ActionNone)
				}
				return nil
			},
		},
		{
			name:  "mixed direct and prefix bindings",
			input: "H=line_end,g-g=last_line,!g-j",
			check: func(km Keymap) error {
				if a, ok := km.Lookup(Key('H')); !ok || a != ActionLineEnd {
					return errorf("H: (%q, %v)", a, ok)
				}
				if a, ok := km.LookupPrefix('g', 'g'); !ok || a != ActionLastLine {
					return errorf("gg: (%q, %v)", a, ok)
				}
				inner := km.Prefix['g']
				if inner['j'] != ActionNone {
					return errorf("gj: %q, want %q", inner['j'], ActionNone)
				}
				return nil
			},
		},
		{
			name:  "new prefix group z-a",
			input: "z-a=viewport_top",
			check: func(km Keymap) error {
				action, ok := km.LookupPrefix('z', 'a')
				if !ok || action != ActionViewportTop {
					return errorf("Prefix['z']['a'] = (%q, %v), want (%q, true)", action, ok, ActionViewportTop)
				}
				return nil
			},
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
