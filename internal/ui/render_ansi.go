package ui

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/shitcoding/tmux_yankee/internal/theme"
)

// Cell represents a single character cell with styling.
type Cell struct {
	Rune  rune
	Style Style
}

// Style represents ANSI SGR styling attributes.
type Style struct {
	FgColor   int  // Foreground color code (0 = default)
	BgColor   int  // Background color code (0 = default)
	FgR, FgG, FgB int  // 24-bit foreground (valid when FgColor == -1)
	BgR, BgG, BgB int  // 24-bit background (valid when BgColor == -1)
	Bold      bool // Bold text
	Dim       bool // Dim text
	Italic    bool // Italic text
	Underline bool // Underline text
	Reverse   bool // Reverse video (swap fg/bg)
}

// DefaultStyle returns a style with no attributes.
func DefaultStyle() Style {
	return Style{}
}

// ParseANSILine parses a line with ANSI escape codes into styled cells.
// Returns a slice of cells with their original styling.
func ParseANSILine(line string) []Cell {
	var cells []Cell
	currentStyle := DefaultStyle()

	runes := []rune(line)
	i := 0

	for i < len(runes) {
		if runes[i] == '\x1b' && i+1 < len(runes) && runes[i+1] == '[' {
			// Parse CSI sequence
			i += 2 // Skip ESC [

			// Find the end of the sequence (letter)
			seqStart := i
			for i < len(runes) && !isCSITerminator(runes[i]) {
				i++
			}

			if i < len(runes) {
				terminator := runes[i]
				sequence := string(runes[seqStart:i])

				// Handle SGR (Select Graphic Rendition) - 'm' terminator
				if terminator == 'm' {
					currentStyle = applySGR(currentStyle, sequence)
				}

				i++ // Skip terminator
			}
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

// isCSITerminator checks if a rune is a CSI sequence terminator.
func isCSITerminator(r rune) bool {
	return (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z')
}

// applySGR applies SGR codes to a style.
// SGR format: ESC[<n>;<n>;<n>m where <n> are numeric codes.
func applySGR(style Style, codes string) Style {
	if codes == "" {
		codes = "0" // Empty is treated as reset
	}

	// Split by semicolon
	parts := strings.Split(codes, ";")

	for i := 0; i < len(parts); i++ {
		if parts[i] == "" {
			continue
		}

		code, err := strconv.Atoi(parts[i])
		if err != nil {
			continue
		}

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
			if i+2 < len(parts) && parts[i+1] == "5" {
				// 256-color mode: 38;5;<n>
				if n, err := strconv.Atoi(parts[i+2]); err == nil {
					style.FgColor = 256 + n // Offset to distinguish from basic colors
					i += 2
				}
			} else if i+4 < len(parts) && parts[i+1] == "2" {
				// 24-bit truecolor mode: 38;2;<r>;<g>;<b>
				r, errR := strconv.Atoi(parts[i+2])
				g, errG := strconv.Atoi(parts[i+3])
				b, errB := strconv.Atoi(parts[i+4])
				if errR == nil && errG == nil && errB == nil {
					style.FgColor = -1 // Sentinel: use FgR/FgG/FgB
					style.FgR = r
					style.FgG = g
					style.FgB = b
					i += 4
				}
			}
		case 39: // Default foreground color
			style.FgColor = 0
		case 40, 41, 42, 43, 44, 45, 46, 47: // Background colors
			style.BgColor = code
		case 48: // Extended background color
			if i+2 < len(parts) && parts[i+1] == "5" {
				// 256-color mode: 48;5;<n>
				if n, err := strconv.Atoi(parts[i+2]); err == nil {
					style.BgColor = 256 + n // Offset to distinguish from basic colors
					i += 2
				}
			} else if i+4 < len(parts) && parts[i+1] == "2" {
				// 24-bit truecolor mode: 48;2;<r>;<g>;<b>
				r, errR := strconv.Atoi(parts[i+2])
				g, errG := strconv.Atoi(parts[i+3])
				b, errB := strconv.Atoi(parts[i+4])
				if errR == nil && errG == nil && errB == nil {
					style.BgColor = -1 // Sentinel: use BgR/BgG/BgB
					style.BgR = r
					style.BgG = g
					style.BgB = b
					i += 4
				}
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
	r, g, b, ok := parseHexColor(string(hex))
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
	r, g, b, ok := parseHexColor(string(hex))
	if !ok {
		return ""
	}
	return fmt.Sprintf("38;2;%d;%d;%d", r, g, b)
}

// parseHexColor parses a "#rrggbb" string into r, g, b int components.
func parseHexColor(hex string) (r, g, b int, ok bool) {
	hex = strings.TrimPrefix(hex, "#")
	if len(hex) != 6 {
		return 0, 0, 0, false
	}
	rv, err := strconv.ParseUint(hex[0:2], 16, 8)
	if err != nil {
		return 0, 0, 0, false
	}
	gv, err := strconv.ParseUint(hex[2:4], 16, 8)
	if err != nil {
		return 0, 0, 0, false
	}
	bv, err := strconv.ParseUint(hex[4:6], 16, 8)
	if err != nil {
		return 0, 0, 0, false
	}
	return int(rv), int(gv), int(bv), true
}

// RenderCellWithPalette renders a cell with its style applied, optionally with
// cursor/selection overlay using palette-derived colors.
// For cursor/selection, palette.Cursor or palette.Selection colors are used.
// Empty HexColor values mean "use terminal default" (no escape emitted).
func RenderCellWithPalette(cell Cell, applyCursor, applySelection bool, pal theme.Palette) string {
	var b strings.Builder

	var codes []string

	if applyCursor {
		bgCode := hexToBGAnsi(pal.Cursor.BG)
		fgCode := hexToFGAnsi(pal.Cursor.FG)
		if bgCode != "" {
			codes = append(codes, bgCode)
		}
		if fgCode != "" {
			codes = append(codes, fgCode)
		}
		if pal.Cursor.Bold {
			codes = append(codes, "1")
		}
	} else if applySelection {
		bgCode := hexToBGAnsi(pal.Selection.BG)
		fgCode := hexToFGAnsi(pal.Selection.FG)
		if bgCode != "" {
			codes = append(codes, bgCode)
		}
		if fgCode != "" {
			codes = append(codes, fgCode)
		}
		if pal.Selection.Bold {
			codes = append(codes, "1")
		}
	} else if cell.Style.Reverse {
		codes = append(codes, "7")
	}

	// Apply original style attributes (only if not in cursor/selection mode)
	if !(applyCursor || applySelection) {
		if cell.Style.Bold {
			codes = append(codes, "1")
		}
		if cell.Style.Dim {
			codes = append(codes, "2")
		}
		if cell.Style.Italic {
			codes = append(codes, "3")
		}
		if cell.Style.Underline {
			codes = append(codes, "4")
		}

		if cell.Style.FgColor == -1 {
			// 24-bit truecolor foreground
			codes = append(codes, fmt.Sprintf("38;2;%d;%d;%d", cell.Style.FgR, cell.Style.FgG, cell.Style.FgB))
		} else if cell.Style.FgColor > 0 {
			if cell.Style.FgColor >= 256 {
				n := cell.Style.FgColor - 256
				codes = append(codes, fmt.Sprintf("38;5;%d", n))
			} else {
				codes = append(codes, strconv.Itoa(cell.Style.FgColor))
			}
		}

		if cell.Style.BgColor == -1 {
			// 24-bit truecolor background
			codes = append(codes, fmt.Sprintf("48;2;%d;%d;%d", cell.Style.BgR, cell.Style.BgG, cell.Style.BgB))
		} else if cell.Style.BgColor > 0 {
			if cell.Style.BgColor >= 256 {
				n := cell.Style.BgColor - 256
				codes = append(codes, fmt.Sprintf("48;5;%d", n))
			} else {
				codes = append(codes, strconv.Itoa(cell.Style.BgColor))
			}
		}
	}

	if len(codes) > 0 {
		b.WriteString("\x1b[")
		b.WriteString(strings.Join(codes, ";"))
		b.WriteString("m")
	}

	b.WriteRune(cell.Rune)

	if len(codes) > 0 {
		b.WriteString("\x1b[0m")
	}

	return b.String()
}

// RenderCell renders a cell with its style applied, optionally with cursor/selection overlay.
// Uses the default palette (empty colors = terminal default, no highlight color).
func RenderCell(cell Cell, applyCursor, applySelection bool) string {
	return RenderCellWithPalette(cell, applyCursor, applySelection, theme.Palette{})
}

// RenderLineWithPalette renders a line with ANSI colors preserved and cursor/selection overlay.
// maxWidth truncates the line if needed (accounts for visible characters, not escape codes).
// selStart and selEnd define the character-level selection range (-1 means no selection).
func RenderLineWithPalette(rawLine string, cursorCol, selStart, selEnd int, maxWidth int, pal theme.Palette) string {
	cells := ParseANSILine(rawLine)

	if len(cells) > maxWidth {
		cells = cells[:maxWidth]
	}

	var b strings.Builder

	for i, cell := range cells {
		inSelection := selStart >= 0 && i >= selStart && i <= selEnd

		applyCursor := (i == cursorCol) && !inSelection
		applySelection := inSelection

		rendered := RenderCellWithPalette(cell, applyCursor, applySelection, pal)
		b.WriteString(rendered)
	}

	// If cursor is at or past end of line, render a visible cursor block
	if cursorCol >= len(cells) && cursorCol >= 0 {
		emptyCell := Cell{Rune: ' ', Style: Style{}}
		rendered := RenderCellWithPalette(emptyCell, true, false, pal)
		b.WriteString(rendered)
	}

	return b.String()
}

// RenderLine renders a line with ANSI colors preserved and cursor/selection overlay.
// Uses the default palette (no color highlight).
// maxWidth truncates the line if needed (accounts for visible characters, not escape codes).
// selStart and selEnd define the character-level selection range (-1 means no selection).
func RenderLine(rawLine string, cursorCol, selStart, selEnd int, maxWidth int) string {
	return RenderLineWithPalette(rawLine, cursorCol, selStart, selEnd, maxWidth, theme.Palette{})
}

// RenderCellsWithPalette renders pre-parsed cells with cursor/selection overlay.
// This is the performance-critical path: cells are pre-parsed at document load,
// so this function does no ANSI parsing.
func RenderCellsWithPalette(cells []Cell, cursorCol, selStart, selEnd int, maxWidth int, pal theme.Palette) string {
	if len(cells) > maxWidth {
		cells = cells[:maxWidth]
	}

	var b strings.Builder

	for i, cell := range cells {
		inSelection := selStart >= 0 && i >= selStart && i <= selEnd
		applyCursor := (i == cursorCol) && !inSelection
		applySelection := inSelection
		b.WriteString(RenderCellWithPalette(cell, applyCursor, applySelection, pal))
	}

	if cursorCol >= len(cells) && cursorCol >= 0 {
		emptyCell := Cell{Rune: ' ', Style: Style{}}
		b.WriteString(RenderCellWithPalette(emptyCell, true, false, pal))
	}

	return b.String()
}
