package theme

import (
	"testing"
)

func TestResolve_DefaultPreset(t *testing.T) {
	p, err := Resolve(ThemeDefault, ThemeOverrides{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if p.Cursor.FG != "#1d2021" {
		t.Errorf("Cursor.FG: got %q, want %q", p.Cursor.FG, "#1d2021")
	}
	if p.Cursor.BG != "#fe8018" {
		t.Errorf("Cursor.BG: got %q, want %q", p.Cursor.BG, "#fe8018")
	}
	if p.Selection.FG != "#fbf1c7" {
		t.Errorf("Selection.FG: got %q, want %q", p.Selection.FG, "#fbf1c7")
	}
	if p.Selection.BG != "#458588" {
		t.Errorf("Selection.BG: got %q, want %q", p.Selection.BG, "#458588")
	}
	if p.Gutter.FG != "#a89984" {
		t.Errorf("Gutter.FG: got %q, want %q", p.Gutter.FG, "#a89984")
	}
	if p.Gutter.Separator != "#665c54" {
		t.Errorf("Gutter.Separator: got %q, want %q", p.Gutter.Separator, "#665c54")
	}
	if p.LineNum.AbsoluteFG != "#d5c4a1" {
		t.Errorf("LineNum.AbsoluteFG: got %q, want %q", p.LineNum.AbsoluteFG, "#d5c4a1")
	}
	if p.LineNum.RelativeFG != "#fabd2f" {
		t.Errorf("LineNum.RelativeFG: got %q, want %q", p.LineNum.RelativeFG, "#fabd2f")
	}
	if p.LineNum.CursorFG != "#b8bb26" {
		t.Errorf("LineNum.CursorFG: got %q, want %q", p.LineNum.CursorFG, "#b8bb26")
	}
	if !p.LineNum.CursorBold {
		t.Errorf("LineNum.CursorBold: got false, want true")
	}
	if p.Status.FG != "#ebdbb2" {
		t.Errorf("Status.FG: got %q, want %q", p.Status.FG, "#ebdbb2")
	}
	if p.Status.BG != "#3c3836" {
		t.Errorf("Status.BG: got %q, want %q", p.Status.BG, "#3c3836")
	}
}

func TestResolve_OverridePrecedence(t *testing.T) {
	overrides := ThemeOverrides{
		CursorFG: "#aabbcc",
	}
	p, err := Resolve(ThemeDefault, overrides)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Only CursorFG should be overridden
	if p.Cursor.FG != "#aabbcc" {
		t.Errorf("Cursor.FG: got %q, want %q", p.Cursor.FG, "#aabbcc")
	}
	// All other fields should remain as preset values
	if p.Cursor.BG != "#fe8018" {
		t.Errorf("Cursor.BG: got %q, want %q (should not be overridden)", p.Cursor.BG, "#fe8018")
	}
	if p.Selection.FG != "#fbf1c7" {
		t.Errorf("Selection.FG: got %q, want %q (should not be overridden)", p.Selection.FG, "#fbf1c7")
	}
}

func TestResolve_UnknownThemeFallsBackToDefault(t *testing.T) {
	p, err := Resolve(ThemeName("nonexistent"), ThemeOverrides{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should match default preset
	defaultPreset := Presets[ThemeDefault]
	if p.Cursor.BG != defaultPreset.Cursor.BG {
		t.Errorf("Cursor.BG: got %q, want %q (default fallback)", p.Cursor.BG, defaultPreset.Cursor.BG)
	}
}

func TestResolve_LineNumCursorBoldOverrideOn(t *testing.T) {
	// Start with a preset where CursorBold is true, override to "on" should keep it true
	overrides := ThemeOverrides{
		LineNumCursorBold: "on",
	}
	p, err := Resolve(ThemeDefault, overrides)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !p.LineNum.CursorBold {
		t.Errorf("LineNum.CursorBold: got false, want true")
	}
}

func TestResolve_LineNumCursorBoldOverrideOff(t *testing.T) {
	overrides := ThemeOverrides{
		LineNumCursorBold: "off",
	}
	p, err := Resolve(ThemeDefault, overrides)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.LineNum.CursorBold {
		t.Errorf("LineNum.CursorBold: got true, want false")
	}
}

func TestResolve_LineNumCursorBoldOverrideEmpty(t *testing.T) {
	// Empty override should keep the preset value (true for default)
	overrides := ThemeOverrides{
		LineNumCursorBold: "",
	}
	p, err := Resolve(ThemeDefault, overrides)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Default preset has CursorBold = true
	if !p.LineNum.CursorBold {
		t.Errorf("LineNum.CursorBold: got false, want true (preset value preserved)")
	}
}

func TestResolve_AllFivePresets(t *testing.T) {
	tests := []struct {
		name   ThemeName
		wantBG HexColor
	}{
		{ThemeDefault, "#fe8018"},
		{ThemeDracula, "#ffb86c"},
		{ThemeGruvbox, "#fe8019"},
		{ThemeNord, "#88c0d0"},
		{ThemeSolarized, "#cb4b16"},
	}

	for _, tt := range tests {
		t.Run(string(tt.name), func(t *testing.T) {
			p, err := Resolve(tt.name, ThemeOverrides{})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if p.Cursor.BG != tt.wantBG {
				t.Errorf("Cursor.BG: got %q, want %q", p.Cursor.BG, tt.wantBG)
			}
		})
	}
}
