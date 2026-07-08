package ui

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/shitcoding/tmux_yankee/internal/flash"
	"github.com/shitcoding/tmux_yankee/internal/theme"
)

// Cell represents a single character cell with styling.
type Cell struct {
	Rune  rune
	Style Style
}

// Style represents ANSI SGR styling attributes.
type Style struct {
	FgColor       int  // Foreground color code (0 = default)
	BgColor       int  // Background color code (0 = default)
	FgR, FgG, FgB int  // 24-bit foreground (valid when FgColor == -1)
	BgR, BgG, BgB int  // 24-bit background (valid when BgColor == -1)
	Bold          bool // Bold text
	Dim           bool // Dim text
	Italic        bool // Italic text
	Underline     bool // Underline text
	Reverse       bool // Reverse video (swap fg/bg)
}

// DefaultStyle returns a style with no attributes.
func DefaultStyle() Style {
	return Style{}
}

// ParseANSILine parses a line with ANSI escape codes into styled cells.
// Returns a slice of cells with their original styling.
func ParseANSILine(line string) []Cell {
	// Fast path: no ANSI escapes -> avoid full parser overhead.
	if strings.IndexByte(line, 0x1b) < 0 {
		runes := []rune(line)
		cells := make([]Cell, 0, len(runes))
		defStyle := DefaultStyle()
		for _, r := range runes {
			if r == '\t' {
				tabWidth := 4
				spacesNeeded := tabWidth - (len(cells) % tabWidth)
				for j := 0; j < spacesNeeded; j++ {
					cells = append(cells, Cell{Rune: ' ', Style: defStyle})
				}
			} else {
				cells = append(cells, Cell{Rune: r, Style: defStyle})
			}
		}
		return cells
	}

	var cells []Cell
	currentStyle := DefaultStyle()

	runes := []rune(line)
	i := 0

	for i < len(runes) {
		if runes[i] == '\x1b' {
			// CSI sequences need inline handling so SGR can update style;
			// all other escape forms (OSC/DCS/APC/PM/SOS/SS2/SS3/intermediate
			// /lone-Fp-controls) are skipped by the shared scanner.
			if i+1 < len(runes) && runes[i+1] == '[' {
				seqStart := i + 2
				next := scanCSI(runes, seqStart)
				if next > seqStart && next <= len(runes) {
					terminator := runes[next-1]
					if terminator == 'm' {
						sequence := string(runes[seqStart : next-1])
						currentStyle = applySGR(currentStyle, sequence)
					}
				}
				i = next
				continue
			}
			i = scanEscape(runes, i)
			continue
		}

		// Tab expansion: replace with spaces to next 4-column tab stop
		if runes[i] == '\t' {
			tabWidth := 4
			spacesNeeded := tabWidth - (len(cells) % tabWidth)
			for j := 0; j < spacesNeeded; j++ {
				cells = append(cells, Cell{Rune: ' ', Style: currentStyle})
			}
			i++
			continue
		}

		// Regular character - add cell with current style
		cells = append(cells, Cell{
			Rune:  runes[i],
			Style: currentStyle,
		})
		i++
	}

	return cells
}

// applySGR applies SGR codes to a style.
// SGR format: ESC[<n>;<n>;<n>m where <n> are numeric codes.
// Uses a single-pass byte parser with a stack-allocated parameter array
// to avoid heap allocations from strings.Split and strconv.Atoi.
func applySGR(style Style, codes string) Style {
	if codes == "" {
		codes = "0" // Empty is treated as reset
	}

	// Parse parameters into a stack-allocated array to avoid heap alloc.
	var params [16]int
	paramCount := 0
	val := 0
	hasVal := false

	for i := 0; i <= len(codes); i++ {
		if i == len(codes) || codes[i] == ';' {
			if hasVal && paramCount < len(params) {
				params[paramCount] = val
				paramCount++
			} else if !hasVal && paramCount < len(params) {
				// Empty field (e.g. ";;" or leading ";") -> treat as 0
				params[paramCount] = 0
				paramCount++
			}
			val = 0
			hasVal = false
		} else if codes[i] >= '0' && codes[i] <= '9' {
			val = val*10 + int(codes[i]-'0')
			hasVal = true
		}
		// Non-digit, non-semicolon characters are ignored
	}

	for i := 0; i < paramCount; i++ {
		code := params[i]
		switch code {
		case 0: // Reset all attributes
			style = DefaultStyle()
		case 1: // Bold
			style.Bold = true
		case 2: // Dim
			style.Dim = true
		case 3: // Italic
			style.Italic = true
		case 4: // Underline
			style.Underline = true
		case 7: // Reverse video
			style.Reverse = true
		case 22: // Normal intensity (not bold or dim)
			style.Bold = false
			style.Dim = false
		case 23: // Not italic
			style.Italic = false
		case 24: // Not underlined
			style.Underline = false
		case 27: // Not reversed
			style.Reverse = false
		case 30, 31, 32, 33, 34, 35, 36, 37: // Foreground colors
			style.FgColor = code
		case 38: // Extended foreground color
			if i+2 < paramCount && params[i+1] == 5 {
				// 256-color mode: 38;5;<n>
				style.FgColor = 256 + params[i+2]
				i += 2
			} else if i+4 < paramCount && params[i+1] == 2 {
				// 24-bit truecolor mode: 38;2;<r>;<g>;<b>
				style.FgColor = -1 // Sentinel: use FgR/FgG/FgB
				style.FgR = params[i+2]
				style.FgG = params[i+3]
				style.FgB = params[i+4]
				i += 4
			}
		case 39: // Default foreground color
			style.FgColor = 0
		case 40, 41, 42, 43, 44, 45, 46, 47: // Background colors
			style.BgColor = code
		case 48: // Extended background color
			if i+2 < paramCount && params[i+1] == 5 {
				// 256-color mode: 48;5;<n>
				style.BgColor = 256 + params[i+2]
				i += 2
			} else if i+4 < paramCount && params[i+1] == 2 {
				// 24-bit truecolor mode: 48;2;<r>;<g>;<b>
				style.BgColor = -1 // Sentinel: use BgR/BgG/BgB
				style.BgR = params[i+2]
				style.BgG = params[i+3]
				style.BgB = params[i+4]
				i += 4
			}
		case 49: // Default background color
			style.BgColor = 0
		case 90, 91, 92, 93, 94, 95, 96, 97: // Bright foreground colors
			style.FgColor = code
		case 100, 101, 102, 103, 104, 105, 106, 107: // Bright background colors
			style.BgColor = code
		}
	}

	return style
}

// hexToBGAnsi converts a "#rrggbb" hex color to an ANSI 24-bit background sequence fragment.
// Returns empty string for empty input (transparent/terminal default).
func hexToBGAnsi(hex theme.HexColor) string {
	if hex == "" {
		return ""
	}
	r, g, b, ok := theme.ParseHex(string(hex))
	if !ok {
		return ""
	}
	return fmt.Sprintf("48;2;%d;%d;%d", r, g, b)
}

// hexToFGAnsi converts a "#rrggbb" hex color to an ANSI 24-bit foreground sequence fragment.
// Returns empty string for empty input (transparent/terminal default).
func hexToFGAnsi(hex theme.HexColor) string {
	if hex == "" {
		return ""
	}
	r, g, b, ok := theme.ParseHex(string(hex))
	if !ok {
		return ""
	}
	return fmt.Sprintf("38;2;%d;%d;%d", r, g, b)
}

// runeDisplayWidth returns the terminal display width of a rune.
// Most characters occupy 1 column; CJK, emoji, and full-width characters occupy 2.
func runeDisplayWidth(r rune) int {
	if r < 0x20 {
		return 0 // control characters
	}
	switch {
	case r >= 0x1100 && r <= 0x115F: // Hangul Jamo
		return 2
	case r >= 0x2E80 && r <= 0x303E: // CJK Radicals, Kangxi, Symbols/Punctuation
		return 2
	case r >= 0x3041 && r <= 0x33BF: // Hiragana, Katakana, Bopomofo, CJK Letters
		return 2
	case r >= 0x3400 && r <= 0x4DBF: // CJK Extension A
		return 2
	case r >= 0x4E00 && r <= 0xA4CF: // CJK Unified Ideographs + Yi
		return 2
	case r >= 0xAC00 && r <= 0xD7AF: // Hangul Syllables
		return 2
	case r >= 0xF900 && r <= 0xFAFF: // CJK Compatibility Ideographs
		return 2
	case r >= 0xFE10 && r <= 0xFE6F: // CJK Compatibility Forms
		return 2
	case r >= 0xFF01 && r <= 0xFF60: // Full-width Latin/Punctuation
		return 2
	case r >= 0xFFE0 && r <= 0xFFE6: // Full-width Symbols
		return 2
	case r >= 0x1F300 && r <= 0x1FAFF: // Emoji (Misc Symbols, Emoticons, etc.)
		return 2
	case r >= 0x20000 && r <= 0x2FA1F: // CJK Extension B-F, Supplements
		return 2
	}
	return 1
}

// cellMode tracks what overlay (cursor/selection/normal) is active for a cell.
type cellMode byte

const (
	cellModeNormal        cellMode = 0
	cellModeCursor        cellMode = 1
	cellModeSelection     cellMode = 2
	cellModeSearchMatch   cellMode = 3
	cellModeSearchCurrent cellMode = 4
	cellModeFlashLabel    cellMode = 5
	cellModeFlashMatch    cellMode = 6
	cellModeFlashBackdrop cellMode = 7
)

// cellStyleKey is a compact representation of a cell's visual style for run tracking.
// Adjacent cells with the same key need no SGR transition.
type cellStyleKey struct {
	mode  cellMode
	style Style // only compared when mode == cellModeNormal
}

// writeSGRNormal writes the SGR sequence for a normal (non-cursor, non-selection) cell
// directly to the builder, avoiding per-cell string allocations.
func writeSGRNormal(b *strings.Builder, s Style) {
	b.WriteString("\x1b[0;")
	first := true
	writeSep := func() {
		if !first {
			b.WriteByte(';')
		}
		first = false
	}
	if s.Reverse {
		writeSep()
		b.WriteByte('7')
	}
	if s.Bold {
		writeSep()
		b.WriteByte('1')
	}
	if s.Dim {
		writeSep()
		b.WriteByte('2')
	}
	if s.Italic {
		writeSep()
		b.WriteByte('3')
	}
	if s.Underline {
		writeSep()
		b.WriteByte('4')
	}
	if s.FgColor == -1 {
		writeSep()
		b.WriteString("38;2;")
		b.WriteString(strconv.Itoa(s.FgR))
		b.WriteByte(';')
		b.WriteString(strconv.Itoa(s.FgG))
		b.WriteByte(';')
		b.WriteString(strconv.Itoa(s.FgB))
	} else if s.FgColor > 0 {
		writeSep()
		if s.FgColor >= 256 {
			b.WriteString("38;5;")
			b.WriteString(strconv.Itoa(s.FgColor - 256))
		} else {
			b.WriteString(strconv.Itoa(s.FgColor))
		}
	}
	if s.BgColor == -1 {
		writeSep()
		b.WriteString("48;2;")
		b.WriteString(strconv.Itoa(s.BgR))
		b.WriteByte(';')
		b.WriteString(strconv.Itoa(s.BgG))
		b.WriteByte(';')
		b.WriteString(strconv.Itoa(s.BgB))
	} else if s.BgColor > 0 {
		writeSep()
		if s.BgColor >= 256 {
			b.WriteString("48;5;")
			b.WriteString(strconv.Itoa(s.BgColor - 256))
		} else {
			b.WriteString(strconv.Itoa(s.BgColor))
		}
	}
	b.WriteByte('m')
}

// RenderCellsWithPalette renders pre-parsed cells with cursor/selection overlay.
// This is the performance-critical path: cells are pre-parsed at document load,
// so this function does no ANSI parsing.
// startCol is the horizontal viewport offset — cells before startCol are not rendered.
// maxWidth is the maximum number of terminal display columns to render.
// cursorCol, selStart, selEnd are absolute (0-based from line start); the renderer
// maps them to viewport-relative positions internally.
// Uses style-run optimization: SGR sequences are only emitted when the effective
// style changes from the previous cell, reducing output size and allocations.
func RenderCellsWithPalette(cells []Cell, cursorCol, selStart, selEnd int,
	searchRanges [][2]int, currentSearch [2]int,
	startCol, maxWidth int, pal theme.Palette,
	flashOverlay *flash.Overlay, flashLine int) string {
	// Clamp startCol to valid range
	if startCol < 0 {
		startCol = 0
	}
	if startCol > len(cells) {
		startCol = len(cells)
	}

	var b strings.Builder
	displayCols := 0

	// Precompute overlay SGR strings once (constant for whole line).
	cursorSGR := buildOverlaySGR(pal.Cursor)
	selectionSGR := buildOverlaySGR(pal.Selection)
	searchMatchSGR := buildOverlaySGR(pal.SearchMatch)
	searchCurrentSGR := buildOverlaySGR(pal.SearchCurrent)

	var flashLabelSGR, flashMatchSGR, flashBackdropSGR string
	if flashOverlay != nil {
		flashLabelSGR = buildOverlaySGR(pal.FlashLabel)
		flashMatchSGR = buildOverlaySGR(pal.FlashMatch)
		flashBackdropSGR = buildOverlaySGR(pal.FlashBackdrop)
	}

	// Track current style to avoid redundant SGR emission.
	var prevKey cellStyleKey
	prevKey.mode = 255 // sentinel: force first cell to emit SGR
	styled := false    // whether we have emitted any SGR (need reset at end)

	// Interval pointer for search ranges (sorted, walk left-to-right).
	srIdx := 0

	for vi := 0; startCol+vi < len(cells); vi++ {
		cell := cells[startCol+vi]
		w := runeDisplayWidth(cell.Rune)
		if displayCols+w > maxWidth {
			break
		}
		absIdx := startCol + vi
		inSelection := selStart >= 0 && absIdx >= selStart && absIdx <= selEnd

		// Advance search range pointer past ranges that end before absIdx.
		for srIdx < len(searchRanges) && searchRanges[srIdx][1] < absIdx {
			srIdx++
		}
		inSearchMatch := srIdx < len(searchRanges) &&
			absIdx >= searchRanges[srIdx][0] && absIdx <= searchRanges[srIdx][1]
		inCurrentSearch := currentSearch[0] >= 0 &&
			absIdx >= currentSearch[0] && absIdx <= currentSearch[1]

		// Flash overlay checks
		var flashLabel byte
		inFlashMatch := false
		if flashOverlay != nil {
			flashLabel = flashOverlay.HasLabel(flashLine, absIdx)
			inFlashMatch = flashOverlay.InMatch(flashLine, absIdx)
		}

		// Priority: cursor > flashLabel > selection > flashMatch > searchCurrent > searchMatch > flashBackdrop > normal
		var key cellStyleKey
		if absIdx == cursorCol {
			key.mode = cellModeCursor
		} else if flashLabel != 0 {
			key.mode = cellModeFlashLabel
		} else if inSelection {
			key.mode = cellModeSelection
		} else if inFlashMatch {
			key.mode = cellModeFlashMatch
		} else if inCurrentSearch {
			key.mode = cellModeSearchCurrent
		} else if inSearchMatch {
			key.mode = cellModeSearchMatch
		} else if flashOverlay != nil && flashOverlay.Backdrop {
			key.mode = cellModeFlashBackdrop
		} else {
			key.mode = cellModeNormal
			key.style = cell.Style
		}

		// Emit SGR only when style changes from previous cell.
		if key != prevKey {
			switch key.mode {
			case cellModeCursor:
				b.WriteString(cursorSGR)
			case cellModeSelection:
				b.WriteString(selectionSGR)
			case cellModeSearchCurrent:
				b.WriteString(searchCurrentSGR)
			case cellModeSearchMatch:
				b.WriteString(searchMatchSGR)
			case cellModeFlashLabel:
				b.WriteString(flashLabelSGR)
			case cellModeFlashMatch:
				b.WriteString(flashMatchSGR)
			case cellModeFlashBackdrop:
				b.WriteString(flashBackdropSGR)
			default:
				if key.style == (Style{}) {
					// Default style — just reset.
					b.WriteString("\x1b[0m")
				} else {
					writeSGRNormal(&b, key.style)
				}
			}
			styled = true
			prevKey = key
		}

		if flashLabel != 0 {
			b.WriteByte(flashLabel)
		} else {
			b.WriteRune(cell.Rune)
		}
		displayCols += w
	}

	// Emit flash labels that fall just past end of line content (match at line end).
	if flashOverlay != nil && displayCols < maxWidth {
		pastEnd := len(cells)
		if label := flashOverlay.HasLabel(flashLine, pastEnd); label != 0 {
			b.WriteString(flashLabelSGR)
			b.WriteByte(label)
			displayCols++
			styled = true
			prevKey = cellStyleKey{mode: cellModeFlashLabel}
		}
	}

	// If cursor is at or past end of visible content, render cursor block
	if cursorCol >= startCol && cursorCol >= len(cells) && displayCols < maxWidth {
		b.WriteString(cursorSGR)
		b.WriteByte(' ')
		styled = true
	}

	if styled {
		b.WriteString("\x1b[0m")
	}

	return b.String()
}

// buildOverlaySGR precomputes the full SGR escape for a theme overlay (cursor or selection).
func buildOverlaySGR(overlay theme.CellPalette) string {
	var b strings.Builder
	b.WriteString("\x1b[0")
	if bg := hexToBGAnsi(overlay.BG); bg != "" {
		b.WriteByte(';')
		b.WriteString(bg)
	}
	if fg := hexToFGAnsi(overlay.FG); fg != "" {
		b.WriteByte(';')
		b.WriteString(fg)
	}
	if overlay.Style.Bold {
		b.WriteString(";1")
	}
	if overlay.Style.Dim {
		b.WriteString(";2")
	}
	if overlay.Style.Italic {
		b.WriteString(";3")
	}
	if overlay.Style.Underline {
		b.WriteString(";4")
	}
	b.WriteByte('m')
	return b.String()
}
