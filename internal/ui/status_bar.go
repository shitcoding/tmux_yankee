package ui

import (
	"fmt"
	"strings"

	vmode "github.com/shitcoding/tmux_yankee/internal/mode"
	"github.com/shitcoding/tmux_yankee/internal/selection"
	"github.com/shitcoding/tmux_yankee/internal/theme"
)

// Powerline separator characters
const (
	sepRight = "\ue0b0" //
	sepLeft  = "\ue0b2" //
)

// statusSegment is a colored text chunk in the status bar.
type statusSegment struct {
	text string
	pal  theme.CellPalette
}

// renderStatusBar renders the powerline-style status bar on the last terminal row.
func (t *TUI) renderStatusBar() {
	width := t.width
	if width <= 0 {
		return
	}

	// Search input prompt: replace entire status bar.
	if t.parser.InSearchMode() {
		t.renderSearchPrompt(width)
		return
	}

	// Colon input prompt: replace entire status bar.
	if t.parser.InColonMode() {
		t.renderColonPrompt(width)
		return
	}

	// Flash mode prompt: replace entire status bar.
	if t.flash != nil && t.flash.Active {
		t.renderFlashPrompt(width)
		return
	}

	pal := t.palette.StatusBar
	mode := t.modeMachine.Mode()
	region := t.modeMachine.Region()

	// Pick mode palette and label
	var modePal theme.CellPalette
	var modeLabel string
	switch mode {
	case vmode.VisualChar:
		modePal = pal.ModeVisualChar
		modeLabel = " VISUAL "
	case vmode.VisualLine:
		modePal = pal.ModeVisualLine
		modeLabel = " V-LINE "
	case vmode.VisualBlock:
		modePal = pal.ModeVisualBlock
		modeLabel = " V-BLOCK "
	default:
		modePal = pal.ModeNormal
		modeLabel = " NORMAL "
	}

	// Build position info
	lineCount := t.doc.LineCount()
	curLine := t.cursorLine + 1 // 1-based display
	curCol := t.cursorCol + 1

	posText := fmt.Sprintf(" L%d/%d:C%d ", curLine, lineCount, curCol)

	// Scroll percentage
	var pct int
	if lineCount <= 1 {
		pct = 100
	} else {
		pct = 100 * t.cursorLine / (lineCount - 1)
	}
	pctText := fmt.Sprintf(" %d%% ", pct)

	// Selection stats (only in visual mode)
	selText := ""
	if region.Kind != selection.KindNone {
		selLines, selChars := t.selectionStats(region)
		if region.Kind == selection.KindLine {
			selText = fmt.Sprintf(" %dL ", selLines)
		} else {
			selText = fmt.Sprintf(" %dL %dC ", selLines, selChars)
		}
	}

	// Secondary info: line number mode + wrap
	numMode := strings.ToUpper(t.lineNumMode)
	if len(numMode) > 3 {
		numMode = numMode[:3]
	}
	secText := fmt.Sprintf(" %s ", numMode)

	// Demo extras
	demoText := ""
	if t.isDemo {
		pageName := "Demo"
		if t.demoPageIndex < len(t.demoPageNames) {
			pageName = t.demoPageNames[t.demoPageIndex]
		}
		themeName := string(t.demoThemeName)
		if themeName == "" {
			themeName = "default"
		}
		demoText = fmt.Sprintf(" %s │ %s ", pageName, themeName)
	}

	// Search match count
	searchText := ""
	if t.searchActive && len(t.searchMatches) > 0 {
		current := t.searchMatchIdx + 1
		if current < 1 {
			current = 0
		}
		searchText = fmt.Sprintf(" [%d/%d] ", current, len(t.searchMatches))
	} else if t.searchActive {
		searchText = " [0/0] "
	}

	// Build left and right segment lists
	leftSegs := []statusSegment{
		{text: modeLabel, pal: modePal},
	}
	if searchText != "" {
		leftSegs = append(leftSegs, statusSegment{text: searchText, pal: pal.InfoPrimary})
	}
	if selText != "" {
		leftSegs = append(leftSegs, statusSegment{text: selText, pal: pal.InfoPrimary})
	}
	leftSegs = append(leftSegs, statusSegment{text: posText, pal: pal.InfoSecondary})

	rightSegs := []statusSegment{
		{text: secText, pal: pal.InfoSecondary},
		{text: pctText, pal: pal.InfoPrimary},
	}

	// Calculate total width of segments + separators
	leftWidth := 0
	for _, s := range leftSegs {
		leftWidth += len([]rune(s.text))
	}
	leftWidth += len(leftSegs) // separators between segments (1 char each)

	rightWidth := 0
	for _, s := range rightSegs {
		rightWidth += len([]rune(s.text))
	}
	rightWidth += len(rightSegs) // separators

	demoWidth := len([]rune(demoText))

	// Width negotiation: drop segments if too narrow
	totalUsed := leftWidth + rightWidth
	if totalUsed > width {
		// Drop secondary info from right
		if len(rightSegs) > 1 {
			rightSegs = rightSegs[1:] // keep only percentage
			rightWidth = len([]rune(rightSegs[0].text)) + 1
			totalUsed = leftWidth + rightWidth
		}
	}
	if totalUsed > width {
		// Drop position info from left, keep only mode
		leftSegs = leftSegs[:1]
		leftWidth = len([]rune(leftSegs[0].text)) + 1
		totalUsed = leftWidth + rightWidth
	}
	if totalUsed > width {
		// Compact mode label
		switch mode {
		case vmode.VisualChar:
			leftSegs[0].text = " V "
		case vmode.VisualLine:
			leftSegs[0].text = "VL "
		case vmode.VisualBlock:
			leftSegs[0].text = "VB "
		default:
			leftSegs[0].text = " N "
		}
		leftWidth = len([]rune(leftSegs[0].text)) + 1
		totalUsed = leftWidth + rightWidth
	}

	// Render the status bar
	var b strings.Builder
	b.WriteString("\r\n") // move to next line after content

	// Left segments with right-pointing separators
	for i, seg := range leftSegs {
		b.WriteString(cellPaletteSGR(seg.pal))
		b.WriteString(seg.text)

		// Separator: fg = current segment bg, bg = next segment bg (or fill bg)
		var nextBG theme.HexColor
		if i+1 < len(leftSegs) {
			nextBG = leftSegs[i+1].pal.BG
		} else {
			nextBG = pal.Fill.BG
		}
		b.WriteString(transitionSGR(seg.pal.BG, nextBG))
		b.WriteString(sepRight)
	}

	// Fill the middle
	fillWidth := width - totalUsed - demoWidth
	if fillWidth < 0 {
		fillWidth = 0
		demoText = "" // drop demo text if no room
	}

	b.WriteString(cellPaletteSGR(pal.Fill))
	if demoText != "" {
		// Center demo text in fill
		leftPad := (fillWidth) / 2
		rightPad := fillWidth - leftPad
		b.WriteString(strings.Repeat(" ", leftPad))
		b.WriteString(demoText)
		b.WriteString(strings.Repeat(" ", rightPad))
	} else {
		b.WriteString(strings.Repeat(" ", fillWidth))
	}

	// Right segments with left-pointing separators
	for i, seg := range rightSegs {
		// Separator: fg = current segment bg, bg = previous segment bg (or fill bg)
		var prevBG theme.HexColor
		if i == 0 {
			prevBG = pal.Fill.BG
		} else {
			prevBG = rightSegs[i-1].pal.BG
		}
		b.WriteString(transitionSGR(seg.pal.BG, prevBG))
		b.WriteString(sepLeft)

		b.WriteString(cellPaletteSGR(seg.pal))
		b.WriteString(seg.text)
	}

	b.WriteString("\x1b[0m")

	fmt.Print(b.String())
}

// renderSearchPrompt renders the search input prompt as the status bar.
func (t *TUI) renderSearchPrompt(width int) {
	pal := t.palette.StatusBar.Fill
	dir := t.parser.SearchDir()
	buf := t.parser.SearchBuffer()

	// Format: /{pattern}▏ or ?{pattern}▏
	prompt := string(dir) + buf + "▏"

	// Truncate if too wide.
	runes := []rune(prompt)
	if len(runes) > width {
		runes = runes[len(runes)-width:]
	}

	var b strings.Builder
	b.WriteString("\r\n")
	b.WriteString(cellPaletteSGR(pal))
	b.WriteString(string(runes))
	// Fill remaining width.
	remaining := width - len(runes)
	if remaining > 0 {
		b.WriteString(strings.Repeat(" ", remaining))
	}
	b.WriteString("\x1b[0m")
	fmt.Print(b.String())
}

// renderColonPrompt renders the colon input prompt as the status bar.
func (t *TUI) renderColonPrompt(width int) {
	pal := t.palette.StatusBar.Fill
	buf := t.parser.ColonBuffer()

	// Format: :{digits}▏
	prompt := ":" + buf + "▏"

	runes := []rune(prompt)
	if len(runes) > width {
		runes = runes[len(runes)-width:]
	}

	var b strings.Builder
	b.WriteString("\r\n")
	b.WriteString(cellPaletteSGR(pal))
	b.WriteString(string(runes))
	remaining := width - len(runes)
	if remaining > 0 {
		b.WriteString(strings.Repeat(" ", remaining))
	}
	b.WriteString("\x1b[0m")
	fmt.Print(b.String())
}

// renderFlashPrompt renders the flash mode status bar with mode indicator, pattern, and match count.
func (t *TUI) renderFlashPrompt(width int) {
	ov := t.flash.Overlay()
	pattern := t.flash.Pattern
	if ov != nil && ov.Prompt != "" {
		pattern = ov.Prompt
	}

	// Count labeled matches
	labeled := 0
	for _, m := range t.flash.Matches {
		if m.Label != 0 {
			labeled++
		}
	}
	total := len(t.flash.Matches)

	// Build segments
	flashModePal := t.palette.FlashLabel
	modeSeg := statusSegment{text: " FLASH ", pal: flashModePal}

	promptText := fmt.Sprintf(" /%s ", pattern)
	countText := fmt.Sprintf(" [%d/%d] ", labeled, total)

	pal := t.palette.StatusBar
	segments := []statusSegment{
		modeSeg,
		{text: promptText, pal: pal.ModeNormal},
		{text: countText, pal: pal.InfoPrimary},
	}

	var sb strings.Builder
	sb.WriteString("\r\n")

	totalLen := 0
	for _, seg := range segments {
		totalLen += len([]rune(seg.text))
	}
	totalLen += len(segments) - 1 // separators

	for idx, seg := range segments {
		sb.WriteString(cellPaletteSGR(seg.pal))
		sb.WriteString(seg.text)
		if idx < len(segments)-1 {
			nextPal := segments[idx+1].pal
			sb.WriteString(transitionSGR(seg.pal.BG, nextPal.BG))
			sb.WriteString(sepRight)
		}
	}

	// Fill remaining width with status bar background
	remaining := width - totalLen
	if remaining > 0 {
		sb.WriteString(cellPaletteSGR(pal.Fill))
		sb.WriteString(strings.Repeat(" ", remaining))
	}

	sb.WriteString("\x1b[0m")
	fmt.Print(sb.String())
}

// selectionStats returns the number of lines and characters in the current selection.
func (t *TUI) selectionStats(region selection.Region) (lines, chars int) {
	start, end := region.Start, region.End
	if start.Line > end.Line || (start.Line == end.Line && start.Col > end.Col) {
		start, end = end, start
	}
	lines = end.Line - start.Line + 1
	if region.Kind == selection.KindLine {
		return lines, 0
	}
	if region.Kind == selection.KindBlock {
		// Count characters in block-wise selection
		minCol := region.Start.Col
		maxCol := region.End.Col
		if minCol > maxCol {
			minCol, maxCol = maxCol, minCol
		}
		chars = 0
		for i := start.Line; i <= end.Line && i < t.doc.LineCount(); i++ {
			lineLen := t.doc.LineRuneCount(i)
			if minCol >= lineLen {
				// Line shorter than block start
				continue
			}
			colEnd := maxCol
			if colEnd >= lineLen {
				colEnd = lineLen - 1
			}
			chars += colEnd - minCol + 1
		}
		if chars < 0 {
			chars = 0
		}
		return lines, chars
	}
	// Count characters in char-wise selection
	chars = 0
	for i := start.Line; i <= end.Line && i < t.doc.LineCount(); i++ {
		lineLen := t.doc.LineRuneCount(i)
		if i == start.Line && i == end.Line {
			chars += end.Col - start.Col + 1
		} else if i == start.Line {
			chars += lineLen - start.Col
		} else if i == end.Line {
			chars += end.Col + 1
		} else {
			chars += lineLen
		}
	}
	if chars < 0 {
		chars = 0
	}
	return lines, chars
}

// cellPaletteSGR builds an SGR escape for a CellPalette.
func cellPaletteSGR(p theme.CellPalette) string {
	var codes []string
	if p.FG != "" {
		r, g, b, ok := parseStatusHex(string(p.FG))
		if ok {
			codes = append(codes, fmt.Sprintf("38;2;%d;%d;%d", r, g, b))
		}
	}
	if p.BG != "" {
		r, g, b, ok := parseStatusHex(string(p.BG))
		if ok {
			codes = append(codes, fmt.Sprintf("48;2;%d;%d;%d", r, g, b))
		}
	}
	if p.Style.Bold {
		codes = append(codes, "1")
	}
	if p.Style.Dim {
		codes = append(codes, "2")
	}
	if p.Style.Italic {
		codes = append(codes, "3")
	}
	if p.Style.Underline {
		codes = append(codes, "4")
	}
	if len(codes) == 0 {
		return "\x1b[0m"
	}
	return "\x1b[" + strings.Join(codes, ";") + "m"
}

// transitionSGR builds an SGR for a powerline separator character.
// The separator fg = fromBG (the color we're transitioning from),
// the separator bg = toBG (the color we're transitioning to).
func transitionSGR(fromBG, toBG theme.HexColor) string {
	var codes []string
	if fromBG != "" {
		r, g, b, ok := parseStatusHex(string(fromBG))
		if ok {
			codes = append(codes, fmt.Sprintf("38;2;%d;%d;%d", r, g, b))
		}
	}
	if toBG != "" {
		r, g, b, ok := parseStatusHex(string(toBG))
		if ok {
			codes = append(codes, fmt.Sprintf("48;2;%d;%d;%d", r, g, b))
		}
	}
	if len(codes) == 0 {
		return "\x1b[0m"
	}
	return "\x1b[" + strings.Join(codes, ";") + "m"
}
