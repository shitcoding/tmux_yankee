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
	if err := applyColorOverride(overrides.GutterSeparatorFG, &p.Gutter.SeparatorFG); err != nil {
		return Palette{}, fmt.Errorf("gutter-separator-fg: %w", err)
	}
	if err := applyColorOverride(overrides.GutterSeparatorBG, &p.Gutter.SeparatorBG); err != nil {
		return Palette{}, fmt.Errorf("gutter-separator-bg: %w", err)
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
	if err := applyColorOverride(overrides.StatusFG, &p.StatusBar.Fill.FG); err != nil {
		return Palette{}, fmt.Errorf("status-fg: %w", err)
	}
	if err := applyColorOverride(overrides.StatusBG, &p.StatusBar.Fill.BG); err != nil {
		return Palette{}, fmt.Errorf("status-bg: %w", err)
	}

	// Separator char override
	if overrides.GutterSeparatorChar != "" {
		p.Gutter.SeparatorChar = overrides.GutterSeparatorChar
	}

	// LineNumCursorBold: backward compat "on"/"off"/"" → CursorStyle.Bold
	applyBoolOverride(overrides.LineNumCursorBold, &p.LineNum.CursorStyle.Bold)

	// LineNum style overrides
	applyBoolOverride(overrides.LineNumAbsoluteBold, &p.LineNum.AbsoluteStyle.Bold)
	applyBoolOverride(overrides.LineNumAbsoluteDim, &p.LineNum.AbsoluteStyle.Dim)
	applyBoolOverride(overrides.LineNumAbsoluteItalic, &p.LineNum.AbsoluteStyle.Italic)
	applyBoolOverride(overrides.LineNumRelativeBold, &p.LineNum.RelativeStyle.Bold)
	applyBoolOverride(overrides.LineNumRelativeDim, &p.LineNum.RelativeStyle.Dim)
	applyBoolOverride(overrides.LineNumRelativeItalic, &p.LineNum.RelativeStyle.Italic)
	applyBoolOverride(overrides.LineNumCursorDim, &p.LineNum.CursorStyle.Dim)
	applyBoolOverride(overrides.LineNumCursorItalic, &p.LineNum.CursorStyle.Italic)

	// Status style overrides
	applyBoolOverride(overrides.StatusBold, &p.StatusBar.Fill.Style.Bold)
	applyBoolOverride(overrides.StatusDim, &p.StatusBar.Fill.Style.Dim)

	// Cursor style overrides
	applyBoolOverride(overrides.CursorDim, &p.Cursor.Style.Dim)
	applyBoolOverride(overrides.CursorItalic, &p.Cursor.Style.Italic)

	// Selection style overrides
	applyBoolOverride(overrides.SelectionDim, &p.Selection.Style.Dim)
	applyBoolOverride(overrides.SelectionItalic, &p.Selection.Style.Italic)

	// Flash color overrides
	if err := applyColorOverride(overrides.FlashLabelFG, &p.FlashLabel.FG); err != nil {
		return Palette{}, fmt.Errorf("flash-label-fg: %w", err)
	}
	if err := applyColorOverride(overrides.FlashLabelBG, &p.FlashLabel.BG); err != nil {
		return Palette{}, fmt.Errorf("flash-label-bg: %w", err)
	}
	if err := applyColorOverride(overrides.FlashMatchFG, &p.FlashMatch.FG); err != nil {
		return Palette{}, fmt.Errorf("flash-match-fg: %w", err)
	}
	if err := applyColorOverride(overrides.FlashMatchBG, &p.FlashMatch.BG); err != nil {
		return Palette{}, fmt.Errorf("flash-match-bg: %w", err)
	}
	if err := applyColorOverride(overrides.FlashBackdrop, &p.FlashBackdrop.FG); err != nil {
		return Palette{}, fmt.Errorf("flash-backdrop: %w", err)
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

// applyBoolOverride applies an "on"/"off"/"" override to a bool pointer.
func applyBoolOverride(override string, dst *bool) {
	switch override {
	case "on":
		*dst = true
	case "off":
		*dst = false
	case "":
		// keep preset value
	}
}
