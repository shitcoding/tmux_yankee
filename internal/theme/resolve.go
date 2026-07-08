package theme

import "strings"

// Resolve loads the named preset and applies any non-empty overrides.
// Color values are validated upstream by config.Validate, so resolution cannot fail.
func Resolve(name ThemeName, overrides ThemeOverrides) Palette {
	preset, ok := Presets[name]
	if !ok {
		// Fall back to default if name is unknown (shouldn't happen after validation, but be safe)
		preset = Presets[ThemeDefault]
	}

	p := preset // copy to apply overrides

	applyColorOverride(overrides.CursorFG, &p.Cursor.FG)
	applyColorOverride(overrides.CursorBG, &p.Cursor.BG)
	applyColorOverride(overrides.SelectionFG, &p.Selection.FG)
	applyColorOverride(overrides.SelectionBG, &p.Selection.BG)
	applyColorOverride(overrides.GutterFG, &p.Gutter.FG)
	applyColorOverride(overrides.GutterBG, &p.Gutter.BG)
	applyColorOverride(overrides.GutterSeparatorFG, &p.Gutter.SeparatorFG)
	applyColorOverride(overrides.GutterSeparatorBG, &p.Gutter.SeparatorBG)
	applyColorOverride(overrides.LineNumAbsoluteFG, &p.LineNum.AbsoluteFG)
	applyColorOverride(overrides.LineNumRelativeFG, &p.LineNum.RelativeFG)
	applyColorOverride(overrides.LineNumCursorFG, &p.LineNum.CursorFG)
	applyColorOverride(overrides.StatusFG, &p.StatusBar.Fill.FG)
	applyColorOverride(overrides.StatusBG, &p.StatusBar.Fill.BG)

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
	applyColorOverride(overrides.FlashLabelFG, &p.FlashLabel.FG)
	applyColorOverride(overrides.FlashLabelBG, &p.FlashLabel.BG)
	applyColorOverride(overrides.FlashMatchFG, &p.FlashMatch.FG)
	applyColorOverride(overrides.FlashMatchBG, &p.FlashMatch.BG)
	applyColorOverride(overrides.FlashBackdrop, &p.FlashBackdrop.FG)

	return p
}

// applyColorOverride sets *dst to the normalized hex color if override is non-empty.
// Values are validated upstream by config.Validate; this only lowercases.
func applyColorOverride(override string, dst *HexColor) {
	if override == "" {
		return
	}
	*dst = HexColor(strings.ToLower(override))
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
