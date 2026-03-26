package theme

import (
	"testing"
)

func TestResolve_DefaultPreset(t *testing.T) {
	p, err := Resolve(ThemeDefault, ThemeOverrides{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if p.Cursor.FG != "#ebdbb2" {
		t.Errorf("Cursor.FG: got %q, want %q", p.Cursor.FG, "#ebdbb2")
	}
	if p.Cursor.BG != "#FF8700" {
		t.Errorf("Cursor.BG: got %q, want %q", p.Cursor.BG, "#FF8700")
	}
	if p.Selection.FG != "" {
		t.Errorf("Selection.FG: got %q, want %q", p.Selection.FG, "")
	}
	if p.Selection.BG != "#FF8700" {
		t.Errorf("Selection.BG: got %q, want %q", p.Selection.BG, "#FF8700")
	}
	if p.Gutter.FG != "#665c54" {
		t.Errorf("Gutter.FG: got %q, want %q", p.Gutter.FG, "#665c54")
	}
	if p.Gutter.SeparatorFG != "" {
		t.Errorf("Gutter.SeparatorFG: got %q, want %q", p.Gutter.SeparatorFG, "")
	}
	if p.Gutter.SeparatorChar != "│" {
		t.Errorf("Gutter.SeparatorChar: got %q, want %q", p.Gutter.SeparatorChar, "│")
	}
	if p.LineNum.AbsoluteFG != "#7c6f64" {
		t.Errorf("LineNum.AbsoluteFG: got %q, want %q", p.LineNum.AbsoluteFG, "#7c6f64")
	}
	if p.LineNum.RelativeFG != "#7c6f64" {
		t.Errorf("LineNum.RelativeFG: got %q, want %q", p.LineNum.RelativeFG, "#7c6f64")
	}
	if p.LineNum.CursorFG != "#FF8700" {
		t.Errorf("LineNum.CursorFG: got %q, want %q", p.LineNum.CursorFG, "#FF8700")
	}
	if !p.LineNum.CursorStyle.Bold {
		t.Errorf("LineNum.CursorStyle.Bold: got false, want true")
	}
	if p.StatusBar.Fill.FG != "#fe8019" {
		t.Errorf("StatusBar.Fill.FG: got %q, want %q", p.StatusBar.Fill.FG, "#fe8019")
	}
	if p.StatusBar.Fill.BG != "#3c3836" {
		t.Errorf("StatusBar.Fill.BG: got %q, want %q", p.StatusBar.Fill.BG, "#3c3836")
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
	if p.Cursor.BG != "#FF8700" {
		t.Errorf("Cursor.BG: got %q, want %q (should not be overridden)", p.Cursor.BG, "#FF8700")
	}
	if p.Selection.FG != "" {
		t.Errorf("Selection.FG: got %q, want %q (should not be overridden)", p.Selection.FG, "")
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
	overrides := ThemeOverrides{
		LineNumCursorBold: "on",
	}
	p, err := Resolve(ThemeDefault, overrides)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !p.LineNum.CursorStyle.Bold {
		t.Errorf("LineNum.CursorStyle.Bold: got false, want true")
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
	if p.LineNum.CursorStyle.Bold {
		t.Errorf("LineNum.CursorStyle.Bold: got true, want false")
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
	// Default preset has CursorStyle.Bold = true
	if !p.LineNum.CursorStyle.Bold {
		t.Errorf("LineNum.CursorStyle.Bold: got false, want true (preset value preserved)")
	}
}

func TestResolve_AllFivePresets(t *testing.T) {
	tests := []struct {
		name   ThemeName
		wantBG HexColor
	}{
		{ThemeDefault, "#FF8700"},
		{ThemeDracula, "#ffb86c"},
		{ThemeGruvbox, "#3c3836"},
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

func TestResolve_NewStyleOverrides(t *testing.T) {
	overrides := ThemeOverrides{
		LineNumAbsoluteDim:    "on",
		LineNumRelativeItalic: "on",
		LineNumCursorDim:      "on",
		StatusBold:            "on",
		CursorDim:             "on",
		SelectionItalic:       "on",
		GutterSeparatorBG:     "#112233",
		GutterSeparatorChar:   "▎",
	}
	p, err := Resolve(ThemeDefault, overrides)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !p.LineNum.AbsoluteStyle.Dim {
		t.Error("LineNum.AbsoluteStyle.Dim: got false, want true")
	}
	if !p.LineNum.RelativeStyle.Italic {
		t.Error("LineNum.RelativeStyle.Italic: got false, want true")
	}
	if !p.LineNum.CursorStyle.Dim {
		t.Error("LineNum.CursorStyle.Dim: got false, want true")
	}
	if !p.StatusBar.Fill.Style.Bold {
		t.Error("StatusBar.Fill.Style.Bold: got false, want true")
	}
	if !p.Cursor.Style.Dim {
		t.Error("Cursor.Style.Dim: got false, want true")
	}
	if !p.Selection.Style.Italic {
		t.Error("Selection.Style.Italic: got false, want true")
	}
	if p.Gutter.SeparatorBG != "#112233" {
		t.Errorf("Gutter.SeparatorBG: got %q, want %q", p.Gutter.SeparatorBG, "#112233")
	}
	if p.Gutter.SeparatorChar != "▎" {
		t.Errorf("Gutter.SeparatorChar: got %q, want %q", p.Gutter.SeparatorChar, "▎")
	}
}
