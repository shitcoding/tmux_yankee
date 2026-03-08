package flash

import "fmt"

// JumpPos defines where the cursor lands when a flash label is pressed.
type JumpPos int

const (
	JumpPosMatchEnd   JumpPos = iota // last char of match (default for lowercase)
	JumpPosMatchStart                // first char of match (default for uppercase/alt)
	JumpPosWordStart                 // start of word containing match start
	JumpPosWordEnd                   // end of word containing match end
	JumpPosOff                       // alt jump disabled
)

// ParseJumpPos converts a string to JumpPos, returning fallback for unknown values.
func ParseJumpPos(s string, fallback JumpPos) JumpPos {
	switch s {
	case "match_end":
		return JumpPosMatchEnd
	case "match_start":
		return JumpPosMatchStart
	case "word_start":
		return JumpPosWordStart
	case "word_end":
		return JumpPosWordEnd
	case "off":
		return JumpPosOff
	default:
		return fallback
	}
}

// String returns the config string representation of a JumpPos.
func (j JumpPos) String() string {
	switch j {
	case JumpPosMatchEnd:
		return "match_end"
	case JumpPosMatchStart:
		return "match_start"
	case JumpPosWordStart:
		return "word_start"
	case JumpPosWordEnd:
		return "word_end"
	case JumpPosOff:
		return "off"
	default:
		return fmt.Sprintf("JumpPos(%d)", int(j))
	}
}

// flashCharType classifies runes for word boundary detection.
// Intentionally duplicates internal/motion/vim.go:getCharType to avoid circular dependency.
type flashCharType int

const (
	flashCharWhitespace  flashCharType = iota
	flashCharWord
	flashCharPunctuation
)

func getFlashCharType(r rune) flashCharType {
	if r == ' ' || r == '\t' || r == '\n' || r == '\r' {
		return flashCharWhitespace
	}
	if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' {
		return flashCharWord
	}
	return flashCharPunctuation
}

// ResolveJumpCol computes the cursor column for a given match and jump position.
// lineText is the full text of the line containing the match.
// m.ColStart and m.ColEnd are rune indices (ColEnd exclusive).
func ResolveJumpCol(lineText string, m Match, pos JumpPos) int {
	runes := []rune(lineText)

	switch pos {
	case JumpPosMatchStart:
		return m.ColStart

	case JumpPosMatchEnd:
		if m.ColEnd > m.ColStart {
			return m.ColEnd - 1
		}
		return m.ColStart

	case JumpPosWordStart:
		if len(runes) == 0 || m.ColStart >= len(runes) {
			return m.ColStart
		}
		ct := getFlashCharType(runes[m.ColStart])
		col := m.ColStart
		for col > 0 && getFlashCharType(runes[col-1]) == ct {
			col--
		}
		return col

	case JumpPosWordEnd:
		endCol := m.ColEnd - 1
		if endCol < 0 {
			endCol = 0
		}
		if len(runes) == 0 || endCol >= len(runes) {
			if len(runes) > 0 {
				return len(runes) - 1
			}
			return 0
		}
		ct := getFlashCharType(runes[endCol])
		col := endCol
		for col+1 < len(runes) && getFlashCharType(runes[col+1]) == ct {
			col++
		}
		return col

	default:
		return m.ColStart
	}
}
