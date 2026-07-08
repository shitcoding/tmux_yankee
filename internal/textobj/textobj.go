package textobj

import (
	"github.com/shitcoding/tmux_yankee/internal/keymap"
	"github.com/shitcoding/tmux_yankee/internal/motion"
)

// Range represents a text object selection range.
type Range struct {
	StartLine int
	StartCol  int
	EndLine   int
	EndCol    int
	OK        bool // false if no valid range found
}

// Resolve computes the selection range for the given text object action.
// Only valid in visual mode — in normal mode (read-only viewer), text objects
// have no operator to apply to.
func Resolve(doc motion.Document, cursor motion.Cursor, action keymap.Action) Range {
	switch action {
	// Word text objects
	case keymap.ActionTextObjectInnerWord:
		return innerWord(doc, cursor, false)
	case keymap.ActionTextObjectAWord:
		return aWord(doc, cursor, false)
	case keymap.ActionTextObjectInnerWORD:
		return innerWord(doc, cursor, true)
	case keymap.ActionTextObjectAWORD:
		return aWord(doc, cursor, true)

	// Paragraph text objects
	case keymap.ActionTextObjectInnerParagraph:
		return innerParagraph(doc, cursor)
	case keymap.ActionTextObjectAParagraph:
		return aParagraph(doc, cursor)

	// Quote text objects
	case keymap.ActionTextObjectInnerQuote:
		return innerQuote(doc, cursor, '"')
	case keymap.ActionTextObjectAQuote:
		return aQuote(doc, cursor, '"')
	case keymap.ActionTextObjectInnerSingleQuote:
		return innerQuote(doc, cursor, '\'')
	case keymap.ActionTextObjectASingleQuote:
		return aQuote(doc, cursor, '\'')
	case keymap.ActionTextObjectInnerBacktick:
		return innerQuote(doc, cursor, '`')
	case keymap.ActionTextObjectABacktick:
		return aQuote(doc, cursor, '`')

	// Paren text objects: i( / a(
	case keymap.ActionTextObjectInnerParen:
		return innerBracket(doc, cursor, '(', ')')
	case keymap.ActionTextObjectAParen:
		return aBracket(doc, cursor, '(', ')')

	// Brace text objects: i{ / a{
	case keymap.ActionTextObjectInnerBrace:
		return innerBracket(doc, cursor, '{', '}')
	case keymap.ActionTextObjectABrace:
		return aBracket(doc, cursor, '{', '}')

	// Bracket text objects: i[ / a[
	case keymap.ActionTextObjectInnerBracket:
		return innerBracket(doc, cursor, '[', ']')
	case keymap.ActionTextObjectABracket:
		return aBracket(doc, cursor, '[', ']')

	// Angle bracket text objects: i< / a<
	case keymap.ActionTextObjectInnerAngle:
		return innerBracket(doc, cursor, '<', '>')
	case keymap.ActionTextObjectAAngle:
		return aBracket(doc, cursor, '<', '>')
	}

	return Range{}
}

// --- Word text objects ---

func isWordChar(r rune) bool {
	return (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_'
}

func isWORDChar(r rune) bool {
	return r != ' ' && r != '\t' && r != '\n' && r != '\r'
}

// innerWord selects the word under cursor. If bigWord, uses WORD boundaries.
func innerWord(doc motion.Document, cursor motion.Cursor, bigWord bool) Range {
	runes := []rune(doc.Line(cursor.Line))
	if len(runes) == 0 {
		return Range{StartLine: cursor.Line, StartCol: 0, EndLine: cursor.Line, EndCol: 0, OK: true}
	}
	col := cursor.Col
	if col >= len(runes) {
		col = len(runes) - 1
	}

	isChar := isWordChar
	if bigWord {
		isChar = isWORDChar
	}

	// Determine if cursor is on a word char or whitespace
	onWord := isChar(runes[col])

	start := col
	end := col

	if onWord {
		for start > 0 && isChar(runes[start-1]) {
			start--
		}
		for end < len(runes)-1 && isChar(runes[end+1]) {
			end++
		}
	} else {
		// On whitespace — select whitespace run
		isWS := func(r rune) bool { return r == ' ' || r == '\t' }
		for start > 0 && isWS(runes[start-1]) {
			start--
		}
		for end < len(runes)-1 && isWS(runes[end+1]) {
			end++
		}
	}

	return Range{StartLine: cursor.Line, StartCol: start, EndLine: cursor.Line, EndCol: end, OK: true}
}

// aWord selects the word + surrounding whitespace.
func aWord(doc motion.Document, cursor motion.Cursor, bigWord bool) Range {
	r := innerWord(doc, cursor, bigWord)
	if !r.OK {
		return r
	}

	runes := []rune(doc.Line(cursor.Line))
	// Try to include trailing whitespace first
	end := r.EndCol
	for end+1 < len(runes) && (runes[end+1] == ' ' || runes[end+1] == '\t') {
		end++
	}
	if end > r.EndCol {
		r.EndCol = end
		return r
	}

	// No trailing whitespace — include leading whitespace
	start := r.StartCol
	for start > 0 && (runes[start-1] == ' ' || runes[start-1] == '\t') {
		start--
	}
	r.StartCol = start
	return r
}

// --- Paragraph text objects ---

func isEmptyLine(doc motion.Document, line int) bool {
	content := doc.Line(line)
	for _, r := range content {
		if r != ' ' && r != '\t' {
			return false
		}
	}
	return true
}

// innerParagraph selects the block of non-empty lines around cursor.
func innerParagraph(doc motion.Document, cursor motion.Cursor) Range {
	lineCount := doc.LineCount()
	if lineCount == 0 {
		return Range{}
	}

	line := cursor.Line
	if line >= lineCount {
		line = lineCount - 1
	}

	empty := isEmptyLine(doc, line)

	start := line
	end := line

	if empty {
		// Select block of empty lines
		for start > 0 && isEmptyLine(doc, start-1) {
			start--
		}
		for end < lineCount-1 && isEmptyLine(doc, end+1) {
			end++
		}
	} else {
		// Select block of non-empty lines
		for start > 0 && !isEmptyLine(doc, start-1) {
			start--
		}
		for end < lineCount-1 && !isEmptyLine(doc, end+1) {
			end++
		}
	}

	endCol := doc.LineRuneCount(end)
	if endCol > 0 {
		endCol--
	}

	return Range{StartLine: start, StartCol: 0, EndLine: end, EndCol: endCol, OK: true}
}

// aParagraph selects the paragraph + trailing empty lines.
func aParagraph(doc motion.Document, cursor motion.Cursor) Range {
	r := innerParagraph(doc, cursor)
	if !r.OK {
		return r
	}

	lineCount := doc.LineCount()

	// Include trailing empty lines
	end := r.EndLine
	for end+1 < lineCount && isEmptyLine(doc, end+1) {
		end++
	}
	if end > r.EndLine {
		endCol := doc.LineRuneCount(end)
		if endCol > 0 {
			endCol--
		}
		r.EndLine = end
		r.EndCol = endCol
		return r
	}

	// No trailing empty lines — include leading empty lines
	start := r.StartLine
	for start > 0 && isEmptyLine(doc, start-1) {
		start--
	}
	r.StartLine = start
	r.StartCol = 0
	return r
}

// --- Quote text objects ---

// innerQuote finds matching quotes of the given type on the current line.
func innerQuote(doc motion.Document, cursor motion.Cursor, quote rune) Range {
	runes := []rune(doc.Line(cursor.Line))
	if len(runes) == 0 {
		return Range{}
	}

	col := cursor.Col
	if col >= len(runes) {
		col = len(runes) - 1
	}

	return findQuotePair(runes, col, quote, cursor.Line, false)
}

func aQuote(doc motion.Document, cursor motion.Cursor, quote rune) Range {
	runes := []rune(doc.Line(cursor.Line))
	if len(runes) == 0 {
		return Range{}
	}

	col := cursor.Col
	if col >= len(runes) {
		col = len(runes) - 1
	}

	return findQuotePair(runes, col, quote, cursor.Line, true)
}

func findQuotePair(runes []rune, col int, quote rune, line int, includeQuotes bool) Range {
	// Find all quote positions on the line
	var positions []int
	for i, r := range runes {
		if r == quote {
			positions = append(positions, i)
		}
	}

	if len(positions) < 2 {
		return Range{}
	}

	// Find the pair that contains the cursor, or the nearest pair ahead of it.
	for i := 0; i+1 < len(positions); i += 2 {
		open := positions[i]
		close := positions[i+1]
		if col > close {
			continue // cursor is past this pair, try next
		}
		// Cursor is inside or before this pair
		if includeQuotes {
			return Range{StartLine: line, StartCol: open, EndLine: line, EndCol: close, OK: true}
		}
		if open+1 <= close-1 {
			return Range{StartLine: line, StartCol: open + 1, EndLine: line, EndCol: close - 1, OK: true}
		}
		// Empty quotes
		return Range{StartLine: line, StartCol: open + 1, EndLine: line, EndCol: open + 1, OK: true}
	}

	return Range{}
}

// --- Bracket text objects ---

// innerBracket finds the matching bracket pair containing cursor.
// Search order: 1) backward for enclosing unmatched bracket, 2) forward for next
// opening bracket, 3) backward for nearest opening bracket (cursor past the pair).

func innerBracket(doc motion.Document, cursor motion.Cursor, open, close rune) Range {
	// Try each open-bracket strategy, validating with findCloseBracket each time.
	// A strategy that finds an open bracket but no matching close bracket is useless.
	strategies := []func() (int, int){
		func() (int, int) { return findOpenBracket(doc, cursor, open, close) },
		func() (int, int) { return findOpenBracketForward(doc, cursor, open) },
		func() (int, int) { return findOpenBracketBackward(doc, cursor, open) },
	}

	var startLine, startCol, endLine, endCol int
	found := false
	for _, fn := range strategies {
		sl, sc := fn()
		if sl < 0 {
			continue
		}
		el, ec := findCloseBracket(doc, motion.Cursor{Line: sl, Col: sc}, open, close)
		if el < 0 {
			continue
		}
		startLine, startCol, endLine, endCol = sl, sc, el, ec
		found = true
		break
	}

	if !found {
		return Range{}
	}

	// Inner: exclude the brackets themselves
	sLine, sCol := startLine, startCol+1
	eLine, eCol := endLine, endCol-1

	// If open bracket is at end of line, start on next line col 0
	lineLen := len([]rune(doc.Line(sLine)))
	if sCol >= lineLen {
		sLine++
		sCol = 0
		if sLine > eLine {
			return Range{StartLine: startLine, StartCol: startCol + 1, EndLine: startLine, EndCol: startCol + 1, OK: true}
		}
	}

	// If close bracket is at start of line, end on previous line's end
	if eCol < 0 {
		eLine--
		if eLine < sLine {
			return Range{StartLine: sLine, StartCol: sCol, EndLine: sLine, EndCol: sCol, OK: true}
		}
		eCol = max(len([]rune(doc.Line(eLine)))-1, 0)
	}

	return Range{StartLine: sLine, StartCol: sCol, EndLine: eLine, EndCol: eCol, OK: true}
}

// aBracket selects including the brackets.
func aBracket(doc motion.Document, cursor motion.Cursor, open, close rune) Range {
	strategies := []func() (int, int){
		func() (int, int) { return findOpenBracket(doc, cursor, open, close) },
		func() (int, int) { return findOpenBracketForward(doc, cursor, open) },
		func() (int, int) { return findOpenBracketBackward(doc, cursor, open) },
	}

	for _, strategy := range strategies {
		sl, sc := strategy()
		if sl < 0 {
			continue
		}
		el, ec := findCloseBracket(doc, motion.Cursor{Line: sl, Col: sc}, open, close)
		if el < 0 {
			continue
		}
		return Range{StartLine: sl, StartCol: sc, EndLine: el, EndCol: ec, OK: true}
	}

	return Range{}
}

// findOpenBracketBackward scans backward for any opening bracket (not depth-aware).
// Used as final fallback when cursor is past a bracket pair entirely.
func findOpenBracketBackward(doc motion.Document, cursor motion.Cursor, open rune) (int, int) {
	line := cursor.Line
	startCol := cursor.Col - 1
	for line >= 0 {
		runes := []rune(doc.Line(line))
		if startCol >= len(runes) {
			startCol = len(runes) - 1
		}
		for c := startCol; c >= 0; c-- {
			if runes[c] == open {
				return line, c
			}
		}
		line--
		if line >= 0 {
			startCol = len([]rune(doc.Line(line))) - 1
		}
	}
	return -1, -1
}

// findOpenBracketForward scans forward on the current line for an opening bracket.
// Used as fallback when cursor is before a bracket pair (vim behavior).
func findOpenBracketForward(doc motion.Document, cursor motion.Cursor, open rune) (int, int) {
	// Search forward from cursor position across all remaining lines
	line := cursor.Line
	startCol := cursor.Col + 1
	for line < doc.LineCount() {
		runes := []rune(doc.Line(line))
		for c := startCol; c < len(runes); c++ {
			if runes[c] == open {
				return line, c
			}
		}
		line++
		startCol = 0
	}
	return -1, -1
}

// findOpenBracket scans backward for the unmatched opening bracket.
func findOpenBracket(doc motion.Document, cursor motion.Cursor, open, close rune) (int, int) {
	line := cursor.Line
	col := cursor.Col
	runes := []rune(doc.Line(line))

	// If cursor is ON the open bracket, return it directly.
	if col >= 0 && col < len(runes) && runes[col] == open {
		return line, col
	}

	// If cursor is ON the close bracket, start scanning one position before it
	// so we don't count this close bracket as a nested one.
	if col >= 0 && col < len(runes) && runes[col] == close {
		col--
	}

	depth := 0

	for {
		runes = []rune(doc.Line(line))
		startCol := col
		if startCol >= len(runes) {
			startCol = len(runes) - 1
		}
		for c := startCol; c >= 0; c-- {
			r := runes[c]
			if r == close {
				depth++
			} else if r == open {
				if depth == 0 {
					return line, c
				}
				depth--
			}
		}
		line--
		if line < 0 {
			return -1, -1
		}
		col = len([]rune(doc.Line(line))) - 1
	}
}

// findCloseBracket scans forward for the matching closing bracket.
func findCloseBracket(doc motion.Document, cursor motion.Cursor, open, close rune) (int, int) {
	line := cursor.Line
	col := cursor.Col + 1 // start after the opening bracket
	depth := 0

	for {
		runes := []rune(doc.Line(line))
		for c := col; c < len(runes); c++ {
			r := runes[c]
			if r == open {
				depth++
			} else if r == close {
				if depth == 0 {
					return line, c
				}
				depth--
			}
		}
		line++
		if line >= doc.LineCount() {
			return -1, -1
		}
		col = 0
	}
}
