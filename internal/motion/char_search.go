package motion

// lastCharSearch stores state for ; and , repeat.
type lastCharSearch struct {
	dir   CharSearchDirection
	char  byte
	valid bool
}

// ApplyCharSearch moves cursor to the count-th occurrence of char on the current line.
func (h *VimHandler) ApplyCharSearch(doc Document, cursor Cursor, dir CharSearchDirection, char byte, count int) Cursor {
	if count <= 0 {
		count = 1
	}
	h.lastSearch = lastCharSearch{dir: dir, char: char, valid: true}
	return h.doCharSearch(doc, cursor, dir, char, count, false)
}

// RepeatCharSearch repeats the last character search in the same direction.
func (h *VimHandler) RepeatCharSearch(doc Document, cursor Cursor, count int) Cursor {
	if !h.lastSearch.valid {
		return cursor
	}
	if count <= 0 {
		count = 1
	}
	return h.doCharSearch(doc, cursor, h.lastSearch.dir, h.lastSearch.char, count, true)
}

// RepeatCharSearchReverse repeats the last character search in the opposite direction.
func (h *VimHandler) RepeatCharSearchReverse(doc Document, cursor Cursor, count int) Cursor {
	if !h.lastSearch.valid {
		return cursor
	}
	if count <= 0 {
		count = 1
	}
	reversed := reverseDirection(h.lastSearch.dir)
	return h.doCharSearch(doc, cursor, reversed, h.lastSearch.char, count, true)
}

func reverseDirection(dir CharSearchDirection) CharSearchDirection {
	switch dir {
	case CharSearchFindForward:
		return CharSearchFindBackward
	case CharSearchTillForward:
		return CharSearchTillBackward
	case CharSearchFindBackward:
		return CharSearchFindForward
	case CharSearchTillBackward:
		return CharSearchTillForward
	default:
		return dir
	}
}

// doCharSearch performs the actual line-local character search.
// When repeat is true and the direction is till, the start position is adjusted
// by one extra to skip past the character the cursor is adjacent to (vim behavior).
func (h *VimHandler) doCharSearch(doc Document, cursor Cursor, dir CharSearchDirection, char byte, count int, repeat bool) Cursor {
	runes := []rune(doc.Line(cursor.Line))
	lineLen := len(runes)
	if lineLen == 0 {
		return cursor
	}

	col := cursor.Col

	switch dir {
	case CharSearchFindForward, CharSearchTillForward:
		start := col + 1
		// On repeat of till-forward, skip one extra position to avoid
		// re-finding the same adjacent character.
		if repeat && dir == CharSearchTillForward && start < lineLen {
			start++
		}
		found := 0
		for i := start; i < lineLen; i++ {
			if runes[i] < 128 && byte(runes[i]) == char {
				found++
				if found == count {
					if dir == CharSearchTillForward {
						i--
					}
					h.goalCol = i
					h.hasGoal = true
					return Cursor{Line: cursor.Line, Col: i}
				}
			}
		}
		return cursor

	case CharSearchFindBackward, CharSearchTillBackward:
		start := col - 1
		// On repeat of till-backward, skip one extra position to avoid
		// re-finding the same adjacent character.
		if repeat && dir == CharSearchTillBackward && start >= 0 {
			start--
		}
		found := 0
		for i := start; i >= 0; i-- {
			if runes[i] < 128 && byte(runes[i]) == char {
				found++
				if found == count {
					if dir == CharSearchTillBackward {
						i++
					}
					h.goalCol = i
					h.hasGoal = true
					return Cursor{Line: cursor.Line, Col: i}
				}
			}
		}
		return cursor
	}

	return cursor
}
