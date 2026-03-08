package keymap

import "testing"

func TestNewModeKeymapNoOverrides(t *testing.T) {
	base := DefaultKeymap()
	empty := Keymap{}
	mk := NewModeKeymap(base, empty, empty, empty)

	// Both modes should equal the base keymap
	normalKm := mk.ForMode(false)
	visualKm := mk.ForMode(true)

	// Spot-check a few bindings exist in both
	for _, km := range []Keymap{normalKm, visualKm} {
		if a, ok := km.Lookup(Key('h')); !ok || a != ActionMoveLeft {
			t.Errorf("expected h=move_left, got (%q, %v)", a, ok)
		}
		if a, ok := km.LookupPrefix('g', 'g'); !ok || a != ActionFirstLine {
			t.Errorf("expected gg=first_line, got (%q, %v)", a, ok)
		}
	}
}

func TestModeKeymapSharedOverrides(t *testing.T) {
	base := DefaultKeymap()
	shared := Keymap{
		Direct: map[KeySpec]Action{
			Key('H'): ActionLineEnd, // override H in both modes
		},
	}
	empty := Keymap{}
	mk := NewModeKeymap(base, shared, empty, empty)

	// Both modes should have the shared override
	for _, isVisual := range []bool{false, true} {
		km := mk.ForMode(isVisual)
		if a, ok := km.Lookup(Key('H')); !ok || a != ActionLineEnd {
			t.Errorf("mode(visual=%v): H = (%q, %v), want line_end", isVisual, a, ok)
		}
	}
}

func TestModeKeymapNormalOverride(t *testing.T) {
	base := DefaultKeymap()
	empty := Keymap{}
	normalOv := Keymap{
		Direct: map[KeySpec]Action{
			Key('H'): ActionFirstNonBlank,
		},
	}
	mk := NewModeKeymap(base, empty, normalOv, empty)

	// Normal mode should have the override
	nm := mk.ForMode(false)
	if a, ok := nm.Lookup(Key('H')); !ok || a != ActionFirstNonBlank {
		t.Errorf("normal: H = (%q, %v), want first_non_blank", a, ok)
	}

	// Visual mode should retain the default (screen_top)
	vm := mk.ForMode(true)
	if a, ok := vm.Lookup(Key('H')); !ok || a != ActionScreenTop {
		t.Errorf("visual: H = (%q, %v), want screen_top", a, ok)
	}
}

func TestModeKeymapVisualOverride(t *testing.T) {
	base := DefaultKeymap()
	empty := Keymap{}
	visualOv := Keymap{
		Direct: map[KeySpec]Action{
			Key('H'): ActionScreenTop,
		},
	}
	// Shared overrides H to line_end, but visual overrides it to screen_top
	shared := Keymap{
		Direct: map[KeySpec]Action{
			Key('H'): ActionLineEnd,
		},
	}
	mk := NewModeKeymap(base, shared, empty, visualOv)

	// Normal mode should have shared override (line_end)
	nm := mk.ForMode(false)
	if a, ok := nm.Lookup(Key('H')); !ok || a != ActionLineEnd {
		t.Errorf("normal: H = (%q, %v), want line_end", a, ok)
	}

	// Visual mode should have visual override (screen_top)
	vm := mk.ForMode(true)
	if a, ok := vm.Lookup(Key('H')); !ok || a != ActionScreenTop {
		t.Errorf("visual: H = (%q, %v), want screen_top", a, ok)
	}
}

func TestModeKeymapPrefixOverrides(t *testing.T) {
	base := DefaultKeymap()
	empty := Keymap{}
	normalOv := Keymap{
		Prefix: map[byte]map[byte]Action{
			'g': {'g': ActionLastLine}, // rebind gg in normal only
		},
	}
	mk := NewModeKeymap(base, empty, normalOv, empty)

	// Normal: gg → last_line
	nm := mk.ForMode(false)
	if a, ok := nm.LookupPrefix('g', 'g'); !ok || a != ActionLastLine {
		t.Errorf("normal: gg = (%q, %v), want last_line", a, ok)
	}

	// Visual: gg → first_line (default)
	vm := mk.ForMode(true)
	if a, ok := vm.LookupPrefix('g', 'g'); !ok || a != ActionFirstLine {
		t.Errorf("visual: gg = (%q, %v), want first_line", a, ok)
	}
}

func TestModeKeymapForModeHelpers(t *testing.T) {
	base := DefaultKeymap()
	empty := Keymap{}
	mk := NewModeKeymap(base, empty, empty, empty)

	// Normal() and Visual() should match ForMode
	nm := mk.Normal()
	vm := mk.Visual()

	if a, _ := nm.Lookup(Key('h')); a != ActionMoveLeft {
		t.Error("Normal() keymap mismatch")
	}
	if a, _ := vm.Lookup(Key('h')); a != ActionMoveLeft {
		t.Error("Visual() keymap mismatch")
	}
}
