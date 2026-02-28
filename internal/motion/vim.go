package motion

// VimHandler implements vim-like motion semantics.
type VimHandler struct {
	goalCol    int            // Remember desired column for vertical movements
	hasGoal    bool           // Track if goal column is set
	lastSearch lastCharSearch // State for ; and , repeat
}

// NewVimHandler creates a new vim motion handler.
func NewVimHandler() *VimHandler {
	return &VimHandler{}
}

// Apply executes the given motion with the specified count.
// count=0 means "no count specified" (e.g., just 'j' or 'G').
// count>=1 means explicit count (e.g., '5j' has count=5, '1G' has count=1).
func (h *VimHandler) Apply(doc Document, cursor Cursor, viewport Viewport, motion Motion, count int) Result {
	if count < 0 {
		count = 0
	}

	// For motions that don't use count, treat 0 as 1
	effectiveCount := count
	if effectiveCount == 0 {
		effectiveCount = 1
	}

	result := Result{
		Cursor:   cursor,
		Viewport: viewport,
	}

	switch motion {
	case MotionUp:
		result.Cursor = h.moveVertical(doc, cursor, -effectiveCount)
	case MotionDown:
		result.Cursor = h.moveVertical(doc, cursor, effectiveCount)
	case MotionLeft:
		result.Cursor = h.moveLeft(doc, cursor, effectiveCount)
	case MotionRight:
		result.Cursor = h.moveRight(doc, cursor, effectiveCount)
	case MotionLineStart:
		result.Cursor = h.moveLineStart(cursor)
	case MotionLineEnd:
		result.Cursor = h.moveLineEnd(doc, cursor)
	case MotionFirstLine:
		result.Cursor = h.moveFirstLine(doc, cursor, count)
	case MotionLastLine:
		result.Cursor = h.moveLastLine(doc, cursor, count)
	case MotionHalfPageUp:
		result = h.moveHalfPageUp(doc, cursor, viewport)
	case MotionHalfPageDown:
		result = h.moveHalfPageDown(doc, cursor, viewport)
	case MotionWordForward:
		result.Cursor = h.moveWordForward(doc, cursor, effectiveCount)
	case MotionWordBackward:
		result.Cursor = h.moveWordBackward(doc, cursor, effectiveCount)
	case MotionWordEnd:
		result.Cursor = h.moveWordEnd(doc, cursor, effectiveCount)
	case MotionFirstNonBlank:
		result.Cursor = h.moveFirstNonBlank(doc, cursor)
	case MotionWORDForward:
		result.Cursor = h.moveWORDForward(doc, cursor, effectiveCount)
	case MotionWORDEnd:
		result.Cursor = h.moveWORDEnd(doc, cursor, effectiveCount)
	case MotionWORDBackward:
		result.Cursor = h.moveWORDBackward(doc, cursor, effectiveCount)
	case MotionParagraphForward:
		result.Cursor = h.moveParagraphForward(doc, cursor, effectiveCount)
	case MotionParagraphBackward:
		result.Cursor = h.moveParagraphBackward(doc, cursor, effectiveCount)
	case MotionViewportTop:
		result.Viewport = h.positionViewportTop(doc, cursor, viewport)
	case MotionViewportCenter:
		result.Viewport = h.positionViewportCenter(doc, cursor, viewport)
	case MotionViewportBottom:
		result.Viewport = h.positionViewportBottom(doc, cursor, viewport)
	case MotionScreenTop:
		result.Cursor = h.moveScreenTop(doc, cursor, viewport, count)
	case MotionScreenMiddle:
		result.Cursor = h.moveScreenMiddle(doc, cursor, viewport)
	case MotionScreenBottom:
		result.Cursor = h.moveScreenBottom(doc, cursor, viewport, count)
	case MotionPageUp:
		result = h.movePageUp(doc, cursor, viewport)
	case MotionPageDown:
		result = h.movePageDown(doc, cursor, viewport)
	case MotionWordEndBackward:
		result.Cursor = h.moveWordEndBackward(doc, cursor, effectiveCount)
	case MotionWORDEndBackward:
		result.Cursor = h.moveWORDEndBackward(doc, cursor, effectiveCount)
	case MotionLastNonBlank:
		result.Cursor = h.moveLastNonBlank(doc, cursor)
	case MotionMatchBracket:
		result.Cursor = h.moveMatchBracket(doc, cursor)
	}

	// Ensure cursor is within viewport
	result.Viewport = h.adjustViewport(result.Cursor, result.Viewport, doc.LineCount())

	return result
}

// moveVertical moves the cursor up or down by delta lines, preserving goal column.
func (h *VimHandler) moveVertical(doc Document, cursor Cursor, delta int) Cursor {
	// Ensure goal column is set
	if !h.hasGoal {
		h.goalCol = cursor.Col
		h.hasGoal = true
	}

	newLine := cursor.Line + delta
	lineCount := doc.LineCount()

	// Clamp to valid line range
	if newLine < 0 {
		newLine = 0
	} else if newLine >= lineCount {
		newLine = lineCount - 1
	}

	// Apply goal column, clamping to line length
	// Cursor should be ON last character, not past it
	newCol := h.goalCol
	lineLen := doc.LineRuneCount(newLine)
	maxCol := lineLen - 1
	if maxCol < 0 {
		maxCol = 0
	}
	if newCol > maxCol {
		newCol = maxCol
	}
	if newCol < 0 {
		newCol = 0
	}

	return Cursor{Line: newLine, Col: newCol}
}

// moveLeft moves the cursor left by count columns.
func (h *VimHandler) moveLeft(doc Document, cursor Cursor, count int) Cursor {
	newCol := cursor.Col - count
	if newCol < 0 {
		newCol = 0
	}

	// Set goal column to new position
	h.goalCol = newCol
	h.hasGoal = true

	return Cursor{Line: cursor.Line, Col: newCol}
}

// moveRight moves the cursor right by count columns.
func (h *VimHandler) moveRight(doc Document, cursor Cursor, count int) Cursor {
	lineLen := doc.LineRuneCount(cursor.Line)
	// Cursor can go to last character, not past it (lineLen-1 max)
	maxCol := lineLen - 1
	if maxCol < 0 {
		maxCol = 0
	}

	newCol := cursor.Col + count
	if newCol > maxCol {
		newCol = maxCol
	}

	// Set goal column to new position
	h.goalCol = newCol
	h.hasGoal = true

	return Cursor{Line: cursor.Line, Col: newCol}
}

// moveLineStart moves cursor to column 0.
func (h *VimHandler) moveLineStart(cursor Cursor) Cursor {
	h.goalCol = 0
	h.hasGoal = true
	return Cursor{Line: cursor.Line, Col: 0}
}

// moveLineEnd moves cursor to end of current line (last character, not past it).
func (h *VimHandler) moveLineEnd(doc Document, cursor Cursor) Cursor {
	lineLen := doc.LineRuneCount(cursor.Line)
	// Vim places cursor ON the last character, not past it
	// For empty lines (lineLen=0), cursor stays at col 0
	targetCol := lineLen - 1
	if targetCol < 0 {
		targetCol = 0
	}
	h.goalCol = targetCol
	h.hasGoal = true
	return Cursor{Line: cursor.Line, Col: targetCol}
}

// moveFirstLine moves to first line (gg with no count) or to line N (Ngg).
// count=0 means no explicit count (gg → line 0).
// count>=1 means explicit line number (1gg → line 1, 5gg → line 5).
func (h *VimHandler) moveFirstLine(doc Document, cursor Cursor, count int) Cursor {
	var targetLine int
	if count == 0 {
		targetLine = 0 // gg with no count → first line
	} else {
		targetLine = count - 1 // Ngg goes to line N (0-indexed = N-1)
	}

	lineCount := doc.LineCount()
	if targetLine < 0 {
		targetLine = 0
	} else if targetLine >= lineCount {
		targetLine = lineCount - 1
	}

	// gg/G are vertical motions, so preserve goal column
	// If no goal column set, use current cursor column
	if !h.hasGoal {
		h.goalCol = cursor.Col
		h.hasGoal = true
	}

	col := h.goalCol
	lineLen := doc.LineRuneCount(targetLine)
	if col > lineLen {
		col = lineLen
	}
	if col < 0 {
		col = 0
	}

	return Cursor{Line: targetLine, Col: col}
}

// moveLastLine moves to last line (G with no count) or to line N (NG).
// count=0 means no explicit count (G → last line).
// count>=1 means explicit line number (1G → line 1, 5G → line 5).
func (h *VimHandler) moveLastLine(doc Document, cursor Cursor, count int) Cursor {
	lineCount := doc.LineCount()
	if lineCount == 0 {
		return Cursor{Line: 0, Col: 0}
	}

	var targetLine int
	if count == 0 {
		targetLine = lineCount - 1 // G with no count → last line
	} else {
		targetLine = count - 1 // NG goes to line N (0-indexed = N-1)
	}

	if targetLine < 0 {
		targetLine = 0
	} else if targetLine >= lineCount {
		targetLine = lineCount - 1
	}

	// G is a vertical motion, so preserve goal column
	// If no goal column set, use current cursor column
	if !h.hasGoal {
		h.goalCol = cursor.Col
		h.hasGoal = true
	}

	col := h.goalCol
	lineLen := doc.LineRuneCount(targetLine)
	if col > lineLen {
		col = lineLen
	}
	if col < 0 {
		col = 0
	}

	return Cursor{Line: targetLine, Col: col}
}

// moveHalfPageUp scrolls viewport and cursor up by half a page.
func (h *VimHandler) moveHalfPageUp(doc Document, cursor Cursor, viewport Viewport) Result {
	halfPage := viewport.Height / 2
	if halfPage < 1 {
		halfPage = 1
	}

	newTop := viewport.Top - halfPage
	if newTop < 0 {
		newTop = 0
	}

	// Cursor moves with viewport (maintains relative position)
	newCursorLine := cursor.Line - halfPage
	if newCursorLine < 0 {
		newCursorLine = 0
	}

	// Apply goal column
	if !h.hasGoal {
		h.goalCol = cursor.Col
		h.hasGoal = true
	}

	newCol := h.goalCol
	lineLen := doc.LineRuneCount(newCursorLine)
	if newCol > lineLen {
		newCol = lineLen
	}
	if newCol < 0 {
		newCol = 0
	}

	return Result{
		Cursor:   Cursor{Line: newCursorLine, Col: newCol},
		Viewport: Viewport{Top: newTop, Height: viewport.Height},
	}
}

// moveHalfPageDown scrolls viewport and cursor down by half a page.
func (h *VimHandler) moveHalfPageDown(doc Document, cursor Cursor, viewport Viewport) Result {
	halfPage := viewport.Height / 2
	if halfPage < 1 {
		halfPage = 1
	}

	lineCount := doc.LineCount()

	newTop := viewport.Top + halfPage
	// Don't scroll viewport past end of document
	maxTop := lineCount - viewport.Height
	if maxTop < 0 {
		maxTop = 0
	}
	if newTop > maxTop {
		newTop = maxTop
	}

	// Cursor moves with viewport (maintains relative position)
	newCursorLine := cursor.Line + halfPage
	if newCursorLine >= lineCount {
		newCursorLine = lineCount - 1
	}
	if newCursorLine < 0 {
		newCursorLine = 0
	}

	// Apply goal column
	if !h.hasGoal {
		h.goalCol = cursor.Col
		h.hasGoal = true
	}

	newCol := h.goalCol
	lineLen := doc.LineRuneCount(newCursorLine)
	if newCol > lineLen {
		newCol = lineLen
	}
	if newCol < 0 {
		newCol = 0
	}

	return Result{
		Cursor:   Cursor{Line: newCursorLine, Col: newCol},
		Viewport: Viewport{Top: newTop, Height: viewport.Height},
	}
}

// adjustViewport ensures the cursor is visible within the viewport.
// Uses minimal scrolling strategy.
func (h *VimHandler) adjustViewport(cursor Cursor, viewport Viewport, lineCount int) Viewport {
	newViewport := viewport

	// Cursor above viewport - scroll up
	if cursor.Line < viewport.Top {
		newViewport.Top = cursor.Line
	}

	// Cursor below viewport - scroll down
	bottom := viewport.Top + viewport.Height - 1
	if cursor.Line > bottom {
		newViewport.Top = cursor.Line - viewport.Height + 1
	}

	// Ensure viewport doesn't go negative
	if newViewport.Top < 0 {
		newViewport.Top = 0
	}

	// Ensure viewport doesn't scroll past end of document
	maxTop := lineCount - viewport.Height
	if maxTop < 0 {
		maxTop = 0
	}
	if newViewport.Top > maxTop {
		newViewport.Top = maxTop
	}

	return newViewport
}

// charType represents the category of a character for word motion.
type charType int

const (
	charTypeWhitespace charType = iota
	charTypeWord                // alphanumeric + underscore
	charTypePunctuation
)

// getCharType returns the category of a rune for word boundary detection.
func getCharType(r rune) charType {
	if r == ' ' || r == '\t' || r == '\n' || r == '\r' {
		return charTypeWhitespace
	}
	if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' {
		return charTypeWord
	}
	return charTypePunctuation
}

// moveWordForward moves to the start of the next word (w motion).
// Repeats count times.
func (h *VimHandler) moveWordForward(doc Document, cursor Cursor, count int) Cursor {
	line := cursor.Line
	col := cursor.Col

	for i := 0; i < count; i++ {
		runes := []rune(doc.Line(line))
		lineLen := len(runes)

		// If at or past end of line, wrap to next line
		if col >= lineLen {
			if line+1 >= doc.LineCount() {
				// At last line, can't move forward
				break
			}
			line++
			col = 0
			runes = []rune(doc.Line(line))
			lineLen = len(runes)

			// Skip leading whitespace on new line
			for col < lineLen && getCharType(runes[col]) == charTypeWhitespace {
				col++
			}

			// If landed on a word, we're done with this iteration
			if col < lineLen {
				continue
			}
			// Otherwise, empty line, continue to next word
		}

		// Skip current word (same char type)
		currentType := getCharType(runes[col])
		for col < lineLen && getCharType(runes[col]) == currentType {
			col++
		}

		// Skip whitespace after current word
		for col < lineLen && getCharType(runes[col]) == charTypeWhitespace {
			col++
		}

		// If at end of line after skipping, wrap to next line
		if col >= lineLen {
			if line+1 >= doc.LineCount() {
				// At last line
				break
			}
			line++
			col = 0
			runes = []rune(doc.Line(line))
			lineLen = len(runes)

			// Skip leading whitespace on new line
			for col < lineLen && getCharType(runes[col]) == charTypeWhitespace {
				col++
			}
		}
	}

	// Clamp to document bounds
	if line >= doc.LineCount() {
		line = doc.LineCount() - 1
	}
	if line < 0 {
		line = 0
	}

	lineLen := doc.LineRuneCount(line)
	if col > lineLen {
		col = lineLen
	}
	if col < 0 {
		col = 0
	}

	// Update goal column
	h.goalCol = col
	h.hasGoal = true

	return Cursor{Line: line, Col: col}
}

// moveWordBackward moves to the start of the previous word (b motion).
// Repeats count times.
func (h *VimHandler) moveWordBackward(doc Document, cursor Cursor, count int) Cursor {
	line := cursor.Line
	col := cursor.Col

	for i := 0; i < count; i++ {
		runes := []rune(doc.Line(line))

		// If at start of line, move to previous line
		if col <= 0 {
			if line <= 0 {
				// At first line, can't move back
				break
			}
			line--
			runes = []rune(doc.Line(line))
			col = len(runes)

			// Skip trailing whitespace
			for col > 0 && getCharType(runes[col-1]) == charTypeWhitespace {
				col--
			}

			// If empty line (all whitespace), continue to previous word
			if col == 0 {
				continue
			}

			// Move to start of the word we landed in
			currentType := getCharType(runes[col-1])
			for col > 0 && getCharType(runes[col-1]) == currentType {
				col--
			}
			continue
		}

		// Move back one position
		col--

		// Skip whitespace backwards
		for col > 0 && getCharType(runes[col]) == charTypeWhitespace {
			col--
		}

		// If we're in whitespace at position 0, wrap to previous line
		if col == 0 && len(runes) > 0 && getCharType(runes[0]) == charTypeWhitespace {
			if line <= 0 {
				break
			}
			line--
			runes = []rune(doc.Line(line))
			col = len(runes)

			// Skip trailing whitespace
			for col > 0 && getCharType(runes[col-1]) == charTypeWhitespace {
				col--
			}

			// Move to start of word
			if col > 0 {
				currentType := getCharType(runes[col-1])
				for col > 0 && getCharType(runes[col-1]) == currentType {
					col--
				}
			}
			continue
		}

		// Now we're at a non-whitespace character, move to start of its word
		if col < len(runes) {
			currentType := getCharType(runes[col])
			for col > 0 && getCharType(runes[col-1]) == currentType {
				col--
			}
		}
	}

	// Clamp to document bounds
	if line < 0 {
		line = 0
	}
	if line >= doc.LineCount() {
		line = doc.LineCount() - 1
	}

	lineLen := doc.LineRuneCount(line)
	if col > lineLen {
		col = lineLen
	}
	if col < 0 {
		col = 0
	}

	// Update goal column
	h.goalCol = col
	h.hasGoal = true

	return Cursor{Line: line, Col: col}
}

// moveWordEnd moves to the end of the current/next word (e motion).
// Repeats count times.
func (h *VimHandler) moveWordEnd(doc Document, cursor Cursor, count int) Cursor {
	line := cursor.Line
	col := cursor.Col

	for i := 0; i < count; i++ {
		// Get current line as runes
		runes := []rune(doc.Line(line))
		lineLen := len(runes)

		// Move forward one position
		col++

		// If past end of line, move to next line
		if col >= lineLen {
			if line+1 < doc.LineCount() {
				line++
				col = 0
				runes = []rune(doc.Line(line))
				lineLen = len(runes)
				// Skip leading whitespace
				for col < lineLen && getCharType(runes[col]) == charTypeWhitespace {
					col++
				}
			} else {
				// At last line, clamp to end
				col = lineLen
				break
			}
		}

		// Skip whitespace
		for col < lineLen && getCharType(runes[col]) == charTypeWhitespace {
			col++
		}

		// If reached end of line after skipping whitespace, move to next line
		if col >= lineLen {
			if line+1 < doc.LineCount() {
				line++
				col = 0
				runes = []rune(doc.Line(line))
				lineLen = len(runes)
				// Skip leading whitespace
				for col < lineLen && getCharType(runes[col]) == charTypeWhitespace {
					col++
				}
			} else {
				col = lineLen
				break
			}
		}

		// Get current character type
		if col < lineLen {
			currentType := getCharType(runes[col])

			// Move to end of word (last character of same type)
			for col+1 < lineLen && getCharType(runes[col+1]) == currentType {
				col++
			}
		}
	}

	// Clamp to document bounds
	if line >= doc.LineCount() {
		line = doc.LineCount() - 1
	}
	if line < 0 {
		line = 0
	}

	lineLen := doc.LineRuneCount(line)
	if col > lineLen {
		col = lineLen
	}
	if col < 0 {
		col = 0
	}

	// Update goal column
	h.goalCol = col
	h.hasGoal = true

	return Cursor{Line: line, Col: col}
}

// moveFirstNonBlank moves to the first non-blank character on the current line (^ motion).
func (h *VimHandler) moveFirstNonBlank(doc Document, cursor Cursor) Cursor {
	line := cursor.Line
	runes := []rune(doc.Line(line))

	// Find first non-whitespace character
	col := 0
	for col < len(runes) && (runes[col] == ' ' || runes[col] == '\t') {
		col++
	}

	// If line is all whitespace, stay at position 0
	if col >= len(runes) {
		col = 0
	}

	// Update goal column
	h.goalCol = col
	h.hasGoal = true

	return Cursor{Line: line, Col: col}
}

// isWORDChar returns true if the rune is non-whitespace (for WORD motions).
// WORD in vim is any sequence of non-whitespace characters.
func isWORDChar(r rune) bool {
	return r != ' ' && r != '\t' && r != '\n' && r != '\r'
}

// moveWORDForward moves to the start of the next WORD (W motion).
// WORD is whitespace-separated (unlike word which considers punctuation).
func (h *VimHandler) moveWORDForward(doc Document, cursor Cursor, count int) Cursor {
	line := cursor.Line
	col := cursor.Col

	for i := 0; i < count; i++ {
		runes := []rune(doc.Line(line))
		lineLen := len(runes)

		// If at or past end of line, wrap to next line
		if col >= lineLen {
			if line+1 >= doc.LineCount() {
				break
			}
			line++
			col = 0
			runes = []rune(doc.Line(line))
			lineLen = len(runes)

			// Skip leading whitespace on new line
			for col < lineLen && !isWORDChar(runes[col]) {
				col++
			}
			if col < lineLen {
				continue
			}
		}

		// Skip current WORD (non-whitespace)
		for col < lineLen && isWORDChar(runes[col]) {
			col++
		}

		// Skip whitespace after current WORD
		for col < lineLen && !isWORDChar(runes[col]) {
			col++
		}

		// If at end of line after skipping, wrap to next line
		if col >= lineLen {
			if line+1 >= doc.LineCount() {
				break
			}
			line++
			col = 0
			runes = []rune(doc.Line(line))
			lineLen = len(runes)

			// Skip leading whitespace on new line
			for col < lineLen && !isWORDChar(runes[col]) {
				col++
			}
		}
	}

	// Clamp to document bounds
	if line >= doc.LineCount() {
		line = doc.LineCount() - 1
	}
	if line < 0 {
		line = 0
	}
	lineLen := doc.LineRuneCount(line)
	if col > lineLen {
		col = lineLen
	}
	if col < 0 {
		col = 0
	}

	h.goalCol = col
	h.hasGoal = true
	return Cursor{Line: line, Col: col}
}

// moveWORDEnd moves to the end of the current/next WORD (E motion).
// WORD is whitespace-separated (unlike word which considers punctuation).
// Repeats count times.
func (h *VimHandler) moveWORDEnd(doc Document, cursor Cursor, count int) Cursor {
	line := cursor.Line
	col := cursor.Col

	for i := 0; i < count; i++ {
		runes := []rune(doc.Line(line))
		lineLen := len(runes)

		// Move forward one position
		col++

		// If past end of line, move to next line
		if col >= lineLen {
			if line+1 < doc.LineCount() {
				line++
				col = 0
				runes = []rune(doc.Line(line))
				lineLen = len(runes)
				// Skip leading whitespace
				for col < lineLen && !isWORDChar(runes[col]) {
					col++
				}
			} else {
				// At last line, clamp to end
				col = lineLen
				break
			}
		}

		// Skip whitespace
		for col < lineLen && !isWORDChar(runes[col]) {
			col++
		}

		// If reached end of line after skipping whitespace, move to next line
		if col >= lineLen {
			if line+1 < doc.LineCount() {
				line++
				col = 0
				runes = []rune(doc.Line(line))
				lineLen = len(runes)
				// Skip leading whitespace
				for col < lineLen && !isWORDChar(runes[col]) {
					col++
				}
			} else {
				col = lineLen
				break
			}
		}

		// Move to end of WORD (last non-whitespace character before whitespace)
		if col < lineLen {
			for col+1 < lineLen && isWORDChar(runes[col+1]) {
				col++
			}
		}
	}

	// Clamp to document bounds
	if line >= doc.LineCount() {
		line = doc.LineCount() - 1
	}
	if line < 0 {
		line = 0
	}

	lineLen := doc.LineRuneCount(line)
	if col > lineLen {
		col = lineLen
	}
	if col < 0 {
		col = 0
	}

	// Update goal column
	h.goalCol = col
	h.hasGoal = true

	return Cursor{Line: line, Col: col}
}

// moveWORDBackward moves to the start of the previous WORD (B motion).
// WORD is whitespace-separated (unlike word which considers punctuation).
// Repeats count times.
func (h *VimHandler) moveWORDBackward(doc Document, cursor Cursor, count int) Cursor {
	line := cursor.Line
	col := cursor.Col

	for i := 0; i < count; i++ {
		runes := []rune(doc.Line(line))

		// If at start of line, move to previous line
		if col <= 0 {
			if line <= 0 {
				// At first line, can't move back
				break
			}
			line--
			runes = []rune(doc.Line(line))
			col = len(runes)

			// Skip trailing whitespace
			for col > 0 && !isWORDChar(runes[col-1]) {
				col--
			}

			// If empty line (all whitespace), continue to previous WORD
			if col == 0 {
				continue
			}

			// Move to start of the WORD we landed in
			for col > 0 && isWORDChar(runes[col-1]) {
				col--
			}
			continue
		}

		// Move back one position
		col--

		// Skip whitespace backwards
		for col > 0 && !isWORDChar(runes[col]) {
			col--
		}

		// If we're in whitespace at position 0, wrap to previous line
		if col == 0 && len(runes) > 0 && !isWORDChar(runes[0]) {
			if line <= 0 {
				break
			}
			line--
			runes = []rune(doc.Line(line))
			col = len(runes)

			// Skip trailing whitespace
			for col > 0 && !isWORDChar(runes[col-1]) {
				col--
			}

			// Move to start of WORD
			if col > 0 {
				for col > 0 && isWORDChar(runes[col-1]) {
					col--
				}
			}
			continue
		}

		// Now we're at a non-whitespace character, move to start of its WORD
		if col < len(runes) && isWORDChar(runes[col]) {
			for col > 0 && isWORDChar(runes[col-1]) {
				col--
			}
		}
	}

	// Clamp to document bounds
	if line < 0 {
		line = 0
	}
	if line >= doc.LineCount() {
		line = doc.LineCount() - 1
	}

	lineLen := doc.LineRuneCount(line)
	if col > lineLen {
		col = lineLen
	}
	if col < 0 {
		col = 0
	}

	// Update goal column
	h.goalCol = col
	h.hasGoal = true

	return Cursor{Line: line, Col: col}
}

// isEmptyLine returns true if the line is empty or contains only whitespace.
// Vim considers a line a paragraph boundary if it is completely empty (zero length).
func isEmptyLine(doc Document, line int) bool {
	return doc.LineRuneCount(line) == 0
}

// moveParagraphForward moves the cursor to the next empty line (} motion).
// If already on an empty line, skips consecutive empty lines first.
func (h *VimHandler) moveParagraphForward(doc Document, cursor Cursor, count int) Cursor {
	lineCount := doc.LineCount()
	line := cursor.Line

	for i := 0; i < count; i++ {
		// Skip current empty lines
		for line < lineCount-1 && isEmptyLine(doc, line) {
			line++
		}
		// Find next empty line
		for line < lineCount-1 && !isEmptyLine(doc, line) {
			line++
		}
	}

	h.goalCol = 0
	h.hasGoal = true
	return Cursor{Line: line, Col: 0}
}

// moveParagraphBackward moves the cursor to the previous empty line ({ motion).
// If already on an empty line, skips consecutive empty lines first.
func (h *VimHandler) moveParagraphBackward(doc Document, cursor Cursor, count int) Cursor {
	line := cursor.Line

	for i := 0; i < count; i++ {
		// Skip current empty lines
		for line > 0 && isEmptyLine(doc, line) {
			line--
		}
		// Find previous empty line
		for line > 0 && !isEmptyLine(doc, line) {
			line--
		}
	}

	h.goalCol = 0
	h.hasGoal = true
	return Cursor{Line: line, Col: 0}
}

// positionViewportTop positions the cursor line at the top of the viewport (zt motion).
func (h *VimHandler) positionViewportTop(doc Document, cursor Cursor, viewport Viewport) Viewport {
	newTop := cursor.Line
	
	// Ensure viewport doesn't go past document end
	lineCount := doc.LineCount()
	maxTop := lineCount - viewport.Height
	if maxTop < 0 {
		maxTop = 0
	}
	if newTop > maxTop {
		newTop = maxTop
	}
	
	// Ensure viewport doesn't go negative
	if newTop < 0 {
		newTop = 0
	}
	
	return Viewport{Top: newTop, Height: viewport.Height}
}

// positionViewportCenter positions the cursor line at the center of the viewport (zz motion).
func (h *VimHandler) positionViewportCenter(doc Document, cursor Cursor, viewport Viewport) Viewport {
	halfHeight := viewport.Height / 2
	newTop := cursor.Line - halfHeight
	
	// Ensure viewport doesn't go negative
	if newTop < 0 {
		newTop = 0
	}
	
	// Ensure viewport doesn't go past document end
	lineCount := doc.LineCount()
	maxTop := lineCount - viewport.Height
	if maxTop < 0 {
		maxTop = 0
	}
	if newTop > maxTop {
		newTop = maxTop
	}
	
	return Viewport{Top: newTop, Height: viewport.Height}
}

// positionViewportBottom positions the cursor line at the bottom of the viewport (zb motion).
func (h *VimHandler) positionViewportBottom(doc Document, cursor Cursor, viewport Viewport) Viewport {
	newTop := cursor.Line - viewport.Height + 1

	// Ensure viewport doesn't go negative
	if newTop < 0 {
		newTop = 0
	}

	// Ensure viewport doesn't go past document end
	lineCount := doc.LineCount()
	maxTop := lineCount - viewport.Height
	if maxTop < 0 {
		maxTop = 0
	}
	if newTop > maxTop {
		newTop = maxTop
	}

	return Viewport{Top: newTop, Height: viewport.Height}
}

// moveScreenTop moves cursor to the top of the visible screen (H).
// count=0 → first visible line, count=N → Nth line from top (1-indexed).
func (h *VimHandler) moveScreenTop(doc Document, cursor Cursor, viewport Viewport, count int) Cursor {
	h.hasGoal = false

	targetLine := viewport.Top
	if count > 0 {
		targetLine = viewport.Top + count - 1
	}

	maxLine := viewport.Top + viewport.Height - 1
	lineCount := doc.LineCount()
	if maxLine >= lineCount {
		maxLine = lineCount - 1
	}
	if targetLine > maxLine {
		targetLine = maxLine
	}
	if targetLine < 0 {
		targetLine = 0
	}

	col := firstNonBlankCol(doc, targetLine)
	return Cursor{Line: targetLine, Col: col}
}

// moveScreenMiddle moves cursor to the middle of the visible screen (M).
func (h *VimHandler) moveScreenMiddle(doc Document, cursor Cursor, viewport Viewport) Cursor {
	h.hasGoal = false

	midLine := viewport.Top + viewport.Height/2
	lineCount := doc.LineCount()
	if midLine >= lineCount {
		midLine = lineCount - 1
	}
	if midLine < 0 {
		midLine = 0
	}

	col := firstNonBlankCol(doc, midLine)
	return Cursor{Line: midLine, Col: col}
}

// moveScreenBottom moves cursor to the bottom of the visible screen (L).
// count=0 → last visible line, count=N → Nth line from bottom (1-indexed).
func (h *VimHandler) moveScreenBottom(doc Document, cursor Cursor, viewport Viewport, count int) Cursor {
	h.hasGoal = false

	maxLine := viewport.Top + viewport.Height - 1
	lineCount := doc.LineCount()
	if maxLine >= lineCount {
		maxLine = lineCount - 1
	}

	targetLine := maxLine
	if count > 0 {
		targetLine = maxLine - count + 1
	}
	if targetLine < viewport.Top {
		targetLine = viewport.Top
	}
	if targetLine < 0 {
		targetLine = 0
	}

	col := firstNonBlankCol(doc, targetLine)
	return Cursor{Line: targetLine, Col: col}
}

// movePageUp scrolls viewport and cursor up by one full page (Ctrl-B).
func (h *VimHandler) movePageUp(doc Document, cursor Cursor, viewport Viewport) Result {
	page := viewport.Height
	if page < 1 {
		page = 1
	}

	newTop := viewport.Top - page
	if newTop < 0 {
		newTop = 0
	}

	newCursorLine := cursor.Line - page
	if newCursorLine < 0 {
		newCursorLine = 0
	}

	if !h.hasGoal {
		h.goalCol = cursor.Col
		h.hasGoal = true
	}

	newCol := h.goalCol
	lineLen := doc.LineRuneCount(newCursorLine)
	if newCol > lineLen {
		newCol = lineLen
	}
	if newCol < 0 {
		newCol = 0
	}

	return Result{
		Cursor:   Cursor{Line: newCursorLine, Col: newCol},
		Viewport: Viewport{Top: newTop, Height: viewport.Height},
	}
}

// movePageDown scrolls viewport and cursor down by one full page (Ctrl-F).
func (h *VimHandler) movePageDown(doc Document, cursor Cursor, viewport Viewport) Result {
	page := viewport.Height
	if page < 1 {
		page = 1
	}

	lineCount := doc.LineCount()

	newTop := viewport.Top + page
	maxTop := lineCount - viewport.Height
	if maxTop < 0 {
		maxTop = 0
	}
	if newTop > maxTop {
		newTop = maxTop
	}

	newCursorLine := cursor.Line + page
	if newCursorLine >= lineCount {
		newCursorLine = lineCount - 1
	}
	if newCursorLine < 0 {
		newCursorLine = 0
	}

	if !h.hasGoal {
		h.goalCol = cursor.Col
		h.hasGoal = true
	}

	newCol := h.goalCol
	lineLen := doc.LineRuneCount(newCursorLine)
	if newCol > lineLen {
		newCol = lineLen
	}
	if newCol < 0 {
		newCol = 0
	}

	return Result{
		Cursor:   Cursor{Line: newCursorLine, Col: newCol},
		Viewport: Viewport{Top: newTop, Height: viewport.Height},
	}
}

// firstNonBlankCol returns the column of the first non-whitespace rune on a line.
func firstNonBlankCol(doc Document, line int) int {
	content := doc.Line(line)
	for i, r := range content {
		if r != ' ' && r != '\t' {
			return i
		}
	}
	return 0
}

// moveWordEndBackward moves to the end of the previous word (ge).
// Algorithm:
//  1. Move back one position
//  2. Skip whitespace backward → if we land on non-ws, that's the word end, done
//  3. If no whitespace was skipped (still in same word), skip same-type chars
//     backward to exit the current word, then skip whitespace → done
func (h *VimHandler) moveWordEndBackward(doc Document, cursor Cursor, count int) Cursor {
	line := cursor.Line
	col := cursor.Col

	prevCol := func() bool {
		col--
		for col < 0 {
			if line <= 0 {
				col = 0
				return false
			}
			line--
			col = len([]rune(doc.Line(line))) - 1
			if col < 0 {
				col = -1 // empty line, keep going
			}
		}
		return true
	}

	charAt := func() charType {
		runes := []rune(doc.Line(line))
		if col >= 0 && col < len(runes) {
			return getCharType(runes[col])
		}
		return charTypeWhitespace
	}

	for i := 0; i < count; i++ {
		// Save original char type to detect word boundary crossing
		origType := charAt()

		if !prevCol() {
			break
		}

		curType := charAt()

		// Case 1: landed on whitespace → skip whitespace, done at word end
		if curType == charTypeWhitespace {
			for curType == charTypeWhitespace {
				if !prevCol() {
					goto done
				}
				curType = charAt()
			}
			continue
		}

		// Case 2: crossed word boundary (different non-ws type) → done
		if origType != charTypeWhitespace && curType != origType {
			continue
		}

		// Case 3: still in same word → skip rest of word, then whitespace
		for {
			if !prevCol() {
				goto done
			}
			ct := charAt()
			if ct != curType {
				// Exited the word. If whitespace, skip it too.
				if ct == charTypeWhitespace {
					for ct == charTypeWhitespace {
						if !prevCol() {
							goto done
						}
						ct = charAt()
					}
				}
				// Now at end of previous word
				break
			}
		}
	}

done:
	if line < 0 {
		line = 0
	}
	if line >= doc.LineCount() {
		line = doc.LineCount() - 1
	}
	lineLen := doc.LineRuneCount(line)
	if col >= lineLen {
		col = lineLen - 1
	}
	if col < 0 {
		col = 0
	}

	h.goalCol = col
	h.hasGoal = true
	return Cursor{Line: line, Col: col}
}

// moveWORDEndBackward moves to the end of the previous WORD (gE).
// Same as ge but uses WORD boundaries (whitespace-separated only).
func (h *VimHandler) moveWORDEndBackward(doc Document, cursor Cursor, count int) Cursor {
	line := cursor.Line
	col := cursor.Col

	prevCol := func() bool {
		col--
		for col < 0 {
			if line <= 0 {
				col = 0
				return false
			}
			line--
			col = len([]rune(doc.Line(line))) - 1
			if col < 0 {
				col = -1
			}
		}
		return true
	}

	isWS := func() bool {
		runes := []rune(doc.Line(line))
		if col >= 0 && col < len(runes) {
			return !isWORDChar(runes[col])
		}
		return true
	}

	for i := 0; i < count; i++ {
		origWS := isWS()

		if !prevCol() {
			break
		}

		curWS := isWS()

		// Case 1: landed on whitespace → skip it, done at WORD end
		if curWS {
			for curWS {
				if !prevCol() {
					goto done
				}
				curWS = isWS()
			}
			continue
		}

		// Case 2: crossed from ws to non-ws → already at word end
		if origWS {
			continue
		}

		// Case 3: still in same WORD → skip to start of WORD, then whitespace
		for {
			if !prevCol() {
				goto done
			}
			if isWS() {
				// Skip whitespace
				for isWS() {
					if !prevCol() {
						goto done
					}
				}
				break
			}
		}
	}

done:
	if line < 0 {
		line = 0
	}
	if line >= doc.LineCount() {
		line = doc.LineCount() - 1
	}
	lineLen := doc.LineRuneCount(line)
	if col >= lineLen {
		col = lineLen - 1
	}
	if col < 0 {
		col = 0
	}

	h.goalCol = col
	h.hasGoal = true
	return Cursor{Line: line, Col: col}
}

// moveLastNonBlank moves to the last non-whitespace character on the line (g_).
func (h *VimHandler) moveLastNonBlank(doc Document, cursor Cursor) Cursor {
	h.hasGoal = false
	runes := []rune(doc.Line(cursor.Line))
	col := len(runes) - 1
	for col >= 0 && (runes[col] == ' ' || runes[col] == '\t') {
		col--
	}
	if col < 0 {
		col = 0
	}
	return Cursor{Line: cursor.Line, Col: col}
}

// moveMatchBracket jumps to the matching bracket (%): (), {}, [].
func (h *VimHandler) moveMatchBracket(doc Document, cursor Cursor) Cursor {
	h.hasGoal = false
	runes := []rune(doc.Line(cursor.Line))
	if len(runes) == 0 {
		return cursor
	}

	// Find bracket at or after cursor on the current line
	bracketCol := -1
	for c := cursor.Col; c < len(runes); c++ {
		if isBracket(runes[c]) {
			bracketCol = c
			break
		}
	}
	if bracketCol < 0 {
		return cursor
	}

	ch := runes[bracketCol]
	match, forward := bracketPair(ch)
	if match == 0 {
		return cursor
	}

	// Scan for matching bracket
	depth := 1
	line := cursor.Line
	col := bracketCol

	for depth > 0 {
		if forward {
			col++
		} else {
			col--
		}

		// Handle line wrapping
		for col < 0 || col >= len([]rune(doc.Line(line))) {
			if forward {
				line++
				if line >= doc.LineCount() {
					return cursor // no match found
				}
				col = 0
			} else {
				line--
				if line < 0 {
					return cursor // no match found
				}
				rr := []rune(doc.Line(line))
				col = len(rr) - 1
				if col < 0 {
					continue // empty line
				}
			}
		}

		r := []rune(doc.Line(line))[col]
		if r == ch {
			depth++
		} else if r == match {
			depth--
		}
	}

	return Cursor{Line: line, Col: col}
}

func isBracket(r rune) bool {
	switch r {
	case '(', ')', '{', '}', '[', ']':
		return true
	}
	return false
}

func bracketPair(r rune) (rune, bool) {
	switch r {
	case '(':
		return ')', true
	case ')':
		return '(', false
	case '{':
		return '}', true
	case '}':
		return '{', false
	case '[':
		return ']', true
	case ']':
		return '[', false
	}
	return 0, false
}
