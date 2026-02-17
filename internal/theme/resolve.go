package theme

import (
	"fmt"
	"strings"
)

// Resolve loads the named preset and applies any non-empty overrides.
// Returns an error if the theme name is unknown or a color value is invalid.
func Resolve(name ThemeName, overrides ThemeOverrides) (Palette, error) {
	preset, ok := Presets[name]
	if !ok {
		// Fall back to default if name is unknown (shouldn't happen after validation, but be safe)
		preset = Presets[ThemeDefault]
	}

	p := preset // copy to apply overrides

	if err := applyColorOverride(overrides.CursorFG, &p.Cursor.FG); err != nil {
		return Palette{}, fmt.Errorf("cursor-fg: %w", err)
	}
	if err := applyColorOverride(overrides.CursorBG, &p.Cursor.BG); err != nil {
		return Palette{}, fmt.Errorf("cursor-bg: %w", err)
	}
	if err := applyColorOverride(overrides.SelectionFG, &p.Selection.FG); err != nil {
		return Palette{}, fmt.Errorf("selection-fg: %w", err)
	}
	if err := applyColorOverride(overrides.SelectionBG, &p.Selection.BG); err != nil {
		return Palette{}, fmt.Errorf("selection-bg: %w", err)
	}
	if err := applyColorOverride(overrides.GutterFG, &p.Gutter.FG); err != nil {
		return Palette{}, fmt.Errorf("gutter-fg: %w", err)
	}
	if err := applyColorOverride(overrides.GutterBG, &p.Gutter.BG); err != nil {
		return Palette{}, fmt.Errorf("gutter-bg: %w", err)
	}
	if err := applyColorOverride(overrides.GutterSeparatorFG, &p.Gutter.Separator); err != nil {
		return Palette{}, fmt.Errorf("gutter-separator-fg: %w", err)
	}
	if err := applyColorOverride(overrides.LineNumAbsoluteFG, &p.LineNum.AbsoluteFG); err != nil {
		return Palette{}, fmt.Errorf("linenum-absolute-fg: %w", err)
	}
	if err := applyColorOverride(overrides.LineNumRelativeFG, &p.LineNum.RelativeFG); err != nil {
		return Palette{}, fmt.Errorf("linenum-relative-fg: %w", err)
	}
	if err := applyColorOverride(overrides.LineNumCursorFG, &p.LineNum.CursorFG); err != nil {
		return Palette{}, fmt.Errorf("linenum-cursor-fg: %w", err)
	}
	if err := applyColorOverride(overrides.StatusFG, &p.Status.FG); err != nil {
		return Palette{}, fmt.Errorf("status-fg: %w", err)
	}
	if err := applyColorOverride(overrides.StatusBG, &p.Status.BG); err != nil {
		return Palette{}, fmt.Errorf("status-bg: %w", err)
	}

	// LineNumCursorBold: "on"/"off"/""
	switch overrides.LineNumCursorBold {
	case "on":
		p.LineNum.CursorBold = true
	case "off":
		p.LineNum.CursorBold = false
	case "":
		// keep preset value
	}

	return p, nil
}

// applyColorOverride sets *dst to the normalized hex color if override is non-empty.
func applyColorOverride(override string, dst *HexColor) error {
	if override == "" {
		return nil
	}
	// Colors were already validated by config.Validate, but normalize to lowercase
	*dst = HexColor(strings.ToLower(override))
	return nil
}
