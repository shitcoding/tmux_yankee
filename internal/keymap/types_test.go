package keymap

import "testing"

func TestKeySpecToByte(t *testing.T) {
	tests := []struct {
		name string
		key  KeySpec
		want byte
	}{
		{"plain h", Key('h'), 'h'},
		{"plain H", Key('H'), 'H'},
		{"Ctrl-d", Ctrl('d'), 4},
		{"Ctrl-u", Ctrl('u'), 21},
		{"Ctrl-a", Ctrl('a'), 1},
		{"Ctrl-z", Ctrl('z'), 26},
		{"Ctrl-D uppercase", Ctrl('D'), 4},
		{"Alt-h", Alt('h'), 'h'},
		{"plain Enter", Key(13), 13},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.key.ToByte()
			if got != tt.want {
				t.Errorf("ToByte() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestKeySpecConstructors(t *testing.T) {
	k := Key('j')
	if k.Key != 'j' || k.Mod != ModNone {
		t.Errorf("Key('j') = %+v, want {Key:'j', Mod:ModNone}", k)
	}

	c := Ctrl('f')
	if c.Key != 'f' || c.Mod != ModCtrl {
		t.Errorf("Ctrl('f') = %+v, want {Key:'f', Mod:ModCtrl}", c)
	}

	a := Alt('h')
	if a.Key != 'h' || a.Mod != ModAlt {
		t.Errorf("Alt('h') = %+v, want {Key:'h', Mod:ModAlt}", a)
	}
}

func TestKeymapLookupDirect(t *testing.T) {
	km := Keymap{
		Direct: map[KeySpec]Action{
			Key('h'):  ActionMoveLeft,
			Key('j'):  ActionMoveDown,
			Ctrl('d'): ActionHalfPageDown,
			Alt('h'):  ActionMoveLeft,
		},
	}

	action, ok := km.Lookup(Key('h'))
	if !ok || action != ActionMoveLeft {
		t.Errorf("Lookup(Key('h')) = (%q, %v), want (%q, true)", action, ok, ActionMoveLeft)
	}

	action, ok = km.Lookup(Ctrl('d'))
	if !ok || action != ActionHalfPageDown {
		t.Errorf("Lookup(Ctrl('d')) = (%q, %v), want (%q, true)", action, ok, ActionHalfPageDown)
	}

	action, ok = km.Lookup(Alt('h'))
	if !ok || action != ActionMoveLeft {
		t.Errorf("Lookup(Alt('h')) = (%q, %v), want (%q, true)", action, ok, ActionMoveLeft)
	}

	_, ok = km.Lookup(Key('x'))
	if ok {
		t.Error("Lookup(Key('x')) should return false for missing key")
	}
}

func TestKeymapLookupDirect_NilMap(t *testing.T) {
	km := Keymap{}
	_, ok := km.Lookup(Key('h'))
	if ok {
		t.Error("Lookup on nil Direct map should return false")
	}
}

func TestKeymapIsPrefix(t *testing.T) {
	km := Keymap{
		Prefix: map[byte]map[byte]Action{
			'g': {'g': ActionFirstLine},
			'z': {'t': ActionViewportTop},
		},
	}

	if !km.IsPrefix('g') {
		t.Error("IsPrefix('g') should be true")
	}
	if !km.IsPrefix('z') {
		t.Error("IsPrefix('z') should be true")
	}
	if km.IsPrefix('x') {
		t.Error("IsPrefix('x') should be false")
	}
}

func TestKeymapIsCharCapture(t *testing.T) {
	km := Keymap{
		CharCapture: map[byte]Action{
			'f': ActionCharSearchF,
			't': ActionCharSearchT,
		},
	}

	if !km.IsCharCapture('f') {
		t.Error("IsCharCapture('f') should be true")
	}
	if km.IsCharCapture('x') {
		t.Error("IsCharCapture('x') should be false")
	}
}

func TestKeymapHasTextObjectPrefix(t *testing.T) {
	km := Keymap{
		TextObjects: map[[2]byte]Action{
			{'i', 'w'}: ActionTextObjectInnerWord,
			{'a', 'w'}: ActionTextObjectAWord,
		},
	}

	if !km.HasTextObjectPrefix('i') {
		t.Error("HasTextObjectPrefix('i') should be true")
	}
	if !km.HasTextObjectPrefix('a') {
		t.Error("HasTextObjectPrefix('a') should be true")
	}
	if km.HasTextObjectPrefix('x') {
		t.Error("HasTextObjectPrefix('x') should be false")
	}
}

func TestKeymapLookupPrefix(t *testing.T) {
	km := Keymap{
		Prefix: map[byte]map[byte]Action{
			'g': {
				'g': ActionFirstLine,
				'j': ActionDisplayLineDown,
			},
		},
	}

	action, ok := km.LookupPrefix('g', 'g')
	if !ok || action != ActionFirstLine {
		t.Errorf("LookupPrefix('g','g') = (%q, %v), want (%q, true)", action, ok, ActionFirstLine)
	}

	action, ok = km.LookupPrefix('g', 'j')
	if !ok || action != ActionDisplayLineDown {
		t.Errorf("LookupPrefix('g','j') = (%q, %v), want (%q, true)", action, ok, ActionDisplayLineDown)
	}

	_, ok = km.LookupPrefix('g', 'x')
	if ok {
		t.Error("LookupPrefix('g','x') should return false for missing binding")
	}

	_, ok = km.LookupPrefix('x', 'g')
	if ok {
		t.Error("LookupPrefix('x','g') should return false for missing prefix")
	}
}

func TestKeymapLookupCharCapture(t *testing.T) {
	km := Keymap{
		CharCapture: map[byte]Action{
			'f': ActionCharSearchF,
		},
	}

	action, ok := km.LookupCharCapture('f')
	if !ok || action != ActionCharSearchF {
		t.Errorf("LookupCharCapture('f') = (%q, %v), want (%q, true)", action, ok, ActionCharSearchF)
	}

	_, ok = km.LookupCharCapture('x')
	if ok {
		t.Error("LookupCharCapture('x') should return false for missing prefix")
	}
}

func TestKeymapLookupTextObject(t *testing.T) {
	km := Keymap{
		TextObjects: map[[2]byte]Action{
			{'i', 'w'}: ActionTextObjectInnerWord,
			{'a', 'w'}: ActionTextObjectAWord,
		},
	}

	action, ok := km.LookupTextObject('i', 'w')
	if !ok || action != ActionTextObjectInnerWord {
		t.Errorf("LookupTextObject('i','w') = (%q, %v), want (%q, true)", action, ok, ActionTextObjectInnerWord)
	}

	action, ok = km.LookupTextObject('a', 'w')
	if !ok || action != ActionTextObjectAWord {
		t.Errorf("LookupTextObject('a','w') = (%q, %v), want (%q, true)", action, ok, ActionTextObjectAWord)
	}

	_, ok = km.LookupTextObject('i', 'x')
	if ok {
		t.Error("LookupTextObject('i','x') should return false for missing text object")
	}
}
