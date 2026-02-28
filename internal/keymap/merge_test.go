package keymap

import "testing"

func TestDefaultKeymapHasExpectedEntries(t *testing.T) {
	km := DefaultKeymap()

	// Check a few Direct bindings
	tests := []struct {
		key    KeySpec
		action Action
	}{
		{Key('h'), ActionMoveLeft},
		{Key('j'), ActionMoveDown},
		{Key('k'), ActionMoveUp},
		{Key('l'), ActionMoveRight},
		{Key('G'), ActionLastLine},
		{Key('w'), ActionWordForward},
		{Ctrl('d'), ActionHalfPageDown},
		{Ctrl('u'), ActionHalfPageUp},
		{Ctrl('f'), ActionPageDown},
		{Ctrl('b'), ActionPageUp},
		{Key('v'), ActionVisualChar},
		{Key('/'), ActionSearchForward},
		{Key('n'), ActionSearchNext},
		{Key('H'), ActionScreenTop},
		{Key('M'), ActionScreenMiddle},
		{Key('\\'), ActionClearSearch},
	}
	for _, tt := range tests {
		action, ok := km.Lookup(tt.key)
		if !ok {
			t.Errorf("DefaultKeymap missing Direct binding for key %+v", tt.key)
			continue
		}
		if action != tt.action {
			t.Errorf("DefaultKeymap Direct[%+v] = %q, want %q", tt.key, action, tt.action)
		}
	}

	// L should be in the default keymap as screen_bottom
	if action, ok := km.Lookup(Key('L')); !ok {
		t.Error("DefaultKeymap should have Direct binding for 'L' (screen_bottom)")
	} else if action != ActionScreenBottom {
		t.Errorf("DefaultKeymap Key('L') = %v, want ActionScreenBottom", action)
	}

	// Check prefix bindings
	action, ok := km.LookupPrefix('g', 'g')
	if !ok || action != ActionFirstLine {
		t.Errorf("DefaultKeymap Prefix['g']['g'] = (%q, %v), want (%q, true)", action, ok, ActionFirstLine)
	}

	action, ok = km.LookupPrefix('g', 'e')
	if !ok || action != ActionWordEndBackward {
		t.Errorf("DefaultKeymap Prefix['g']['e'] = (%q, %v), want (%q, true)", action, ok, ActionWordEndBackward)
	}

	action, ok = km.LookupPrefix('z', 't')
	if !ok || action != ActionViewportTop {
		t.Errorf("DefaultKeymap Prefix['z']['t'] = (%q, %v), want (%q, true)", action, ok, ActionViewportTop)
	}

	// Check char capture
	action, ok = km.LookupCharCapture('f')
	if !ok || action != ActionCharSearchF {
		t.Errorf("DefaultKeymap CharCapture['f'] = (%q, %v), want (%q, true)", action, ok, ActionCharSearchF)
	}

	action, ok = km.LookupCharCapture('m')
	if !ok || action != ActionSetMark {
		t.Errorf("DefaultKeymap CharCapture['m'] = (%q, %v), want (%q, true)", action, ok, ActionSetMark)
	}

	// Check text objects
	action, ok = km.LookupTextObject('i', 'w')
	if !ok || action != ActionTextObjectInnerWord {
		t.Errorf("DefaultKeymap TextObjects['i','w'] = (%q, %v), want (%q, true)", action, ok, ActionTextObjectInnerWord)
	}
}

func TestMergeOverridesDirect(t *testing.T) {
	base := Keymap{
		Direct: map[KeySpec]Action{
			Key('h'): ActionMoveLeft,
			Key('H'): ActionScreenTop,
		},
	}
	overrides := Keymap{
		Direct: map[KeySpec]Action{
			Key('H'): ActionLineEnd, // override H from screen_top to line_end
		},
	}

	merged := base.Merge(overrides)

	// h should be unchanged
	action, ok := merged.Lookup(Key('h'))
	if !ok || action != ActionMoveLeft {
		t.Errorf("Merge: Direct['h'] = (%q, %v), want (%q, true)", action, ok, ActionMoveLeft)
	}

	// H should be overridden
	action, ok = merged.Lookup(Key('H'))
	if !ok || action != ActionLineEnd {
		t.Errorf("Merge: Direct['H'] = (%q, %v), want (%q, true)", action, ok, ActionLineEnd)
	}
}

func TestMergeUnbindsDirect(t *testing.T) {
	base := Keymap{
		Direct: map[KeySpec]Action{
			Key('h'): ActionMoveLeft,
			Key('H'): ActionScreenTop,
		},
	}
	overrides := Keymap{
		Direct: map[KeySpec]Action{
			Key('H'): ActionNone, // unbind H
		},
	}

	merged := base.Merge(overrides)

	// h should still exist
	action, ok := merged.Lookup(Key('h'))
	if !ok || action != ActionMoveLeft {
		t.Errorf("Merge: Direct['h'] = (%q, %v), want (%q, true)", action, ok, ActionMoveLeft)
	}

	// H should be removed
	_, ok = merged.Lookup(Key('H'))
	if ok {
		t.Error("Merge: Direct['H'] should be removed after unbind")
	}
}

func TestMergeOverridesPrefix(t *testing.T) {
	base := Keymap{
		Prefix: map[byte]map[byte]Action{
			'g': {
				'g': ActionFirstLine,
				'j': ActionDisplayLineDown,
			},
		},
	}
	overrides := Keymap{
		Prefix: map[byte]map[byte]Action{
			'g': {
				'j': ActionMoveDown, // override gj
			},
		},
	}

	merged := base.Merge(overrides)

	// gg unchanged
	action, ok := merged.LookupPrefix('g', 'g')
	if !ok || action != ActionFirstLine {
		t.Errorf("Merge: Prefix['g']['g'] = (%q, %v), want (%q, true)", action, ok, ActionFirstLine)
	}

	// gj overridden
	action, ok = merged.LookupPrefix('g', 'j')
	if !ok || action != ActionMoveDown {
		t.Errorf("Merge: Prefix['g']['j'] = (%q, %v), want (%q, true)", action, ok, ActionMoveDown)
	}
}

func TestMergeUnbindsPrefix(t *testing.T) {
	base := Keymap{
		Prefix: map[byte]map[byte]Action{
			'g': {
				'g': ActionFirstLine,
				'j': ActionDisplayLineDown,
			},
		},
	}
	overrides := Keymap{
		Prefix: map[byte]map[byte]Action{
			'g': {
				'j': ActionNone, // unbind gj
			},
		},
	}

	merged := base.Merge(overrides)

	// gg unchanged
	action, ok := merged.LookupPrefix('g', 'g')
	if !ok || action != ActionFirstLine {
		t.Errorf("Merge: Prefix['g']['g'] = (%q, %v), want (%q, true)", action, ok, ActionFirstLine)
	}

	// gj should be removed
	_, ok = merged.LookupPrefix('g', 'j')
	if ok {
		t.Error("Merge: Prefix['g']['j'] should be removed after unbind")
	}
}

func TestMergeDoesNotMutateBase(t *testing.T) {
	base := Keymap{
		Direct: map[KeySpec]Action{
			Key('h'): ActionMoveLeft,
		},
	}
	overrides := Keymap{
		Direct: map[KeySpec]Action{
			Key('h'): ActionMoveRight,
		},
	}

	_ = base.Merge(overrides)

	// Base should be unchanged
	if base.Direct[Key('h')] != ActionMoveLeft {
		t.Error("Merge mutated the base keymap")
	}
}

func TestMergeAddsNewBindings(t *testing.T) {
	base := Keymap{
		Direct: map[KeySpec]Action{
			Key('h'): ActionMoveLeft,
		},
	}
	overrides := Keymap{
		Direct: map[KeySpec]Action{
			Alt('h'): ActionMoveLeft, // new binding
		},
	}

	merged := base.Merge(overrides)

	action, ok := merged.Lookup(Alt('h'))
	if !ok || action != ActionMoveLeft {
		t.Errorf("Merge: Direct[Alt('h')] = (%q, %v), want (%q, true)", action, ok, ActionMoveLeft)
	}
}
