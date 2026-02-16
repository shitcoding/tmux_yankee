package ui

import (
	"fmt"
	"strconv"
	"strings"
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
			// Handle 256-color mode: 38;5;<n>
			if i+2 < len(parts) && parts[i+1] == "5" {
				if n, err := strconv.Atoi(parts[i+2]); err == nil {
					style.FgColor = 256 + n // Offset to distinguish from basic colors
					i += 2
				}
			}
		case 39: // Default foreground color
			style.FgColor = 0
		case 40, 41, 42, 43, 44, 45, 46, 47: // Background colors
			style.BgColor = code
		case 48: // Extended background color
			// Handle 256-color mode: 48;5;<n>
			if i+2 < len(parts) && parts[i+1] == "5" {
				if n, err := strconv.Atoi(parts[i+2]); err == nil {
					style.BgColor = 256 + n // Offset to distinguish from basic colors
					i += 2
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

// RenderCell renders a cell with its style applied, optionally with cursor/selection overlay.
func RenderCell(cell Cell, applyCursor, applySelection bool) string {
	var b strings.Builder

	// Build style sequence
	var codes []string

	// Apply selection/cursor highlight with fixed colors
	if applyCursor || applySelection {
		// Orange highlight (#FE8018 = rgb(254, 128, 24)) with black text
		codes = append(codes, "30")              // Black foreground
		codes = append(codes, "48;2;254;128;24") // Orange background (RGB)
	} else if cell.Style.Reverse {
		// Use original reverse video
		codes = append(codes, "7")
	}

	// Apply original style attributes (only if not in selection mode)
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

		// Apply original colors
		if cell.Style.FgColor > 0 {
			if cell.Style.FgColor >= 256 {
				// 256-color mode
				n := cell.Style.FgColor - 256
				codes = append(codes, fmt.Sprintf("38;5;%d", n))
			} else {
				codes = append(codes, strconv.Itoa(cell.Style.FgColor))
			}
		}

		if cell.Style.BgColor > 0 {
			if cell.Style.BgColor >= 256 {
				// 256-color mode
				n := cell.Style.BgColor - 256
				codes = append(codes, fmt.Sprintf("48;5;%d", n))
			} else {
				codes = append(codes, strconv.Itoa(cell.Style.BgColor))
			}
		}
	}

	// Emit SGR sequence if we have codes
	if len(codes) > 0 {
		b.WriteString("\x1b[")
		b.WriteString(strings.Join(codes, ";"))
		b.WriteString("m")
	}

	// Emit the character
	b.WriteRune(cell.Rune)

	// Reset after character (to avoid bleeding into gutter or next line)
	if len(codes) > 0 {
		b.WriteString("\x1b[0m")
	}

	return b.String()
}

// RenderLine renders a line with ANSI colors preserved and cursor/selection overlay.
// maxWidth truncates the line if needed (accounts for visible characters, not escape codes).
// selStart and selEnd define the character-level selection range (-1 means no selection).
func RenderLine(rawLine string, cursorCol, selStart, selEnd int, maxWidth int) string {
	// Parse the raw ANSI line into cells
	cells := ParseANSILine(rawLine)

	// Truncate to maxWidth if needed
	if len(cells) > maxWidth {
		cells = cells[:maxWidth]
	}

	var b strings.Builder

	for i, cell := range cells {
		// Check if this column is in selection range
		inSelection := selStart >= 0 && i >= selStart && i <= selEnd

		applyCursor := (i == cursorCol) && !inSelection
		applySelection := inSelection

		rendered := RenderCell(cell, applyCursor, applySelection)
		b.WriteString(rendered)
	}

	// If cursor is at or past end of line, render a visible cursor block
	if cursorCol >= len(cells) && cursorCol >= 0 {
		emptyCell := Cell{Rune: ' ', Style: Style{}}
		rendered := RenderCell(emptyCell, true, false)
		b.WriteString(rendered)
	}

	return b.String()
}
