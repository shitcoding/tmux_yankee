package keymap

import (
	"fmt"
	"strings"
)

// ParseKeyNotation parses a human-readable key notation string into a KeySpec.
// Supported formats:
//   - Single char: "h", "H", "$", "^", etc.
//   - Ctrl+letter: "C-d", "C-f"
//   - Alt+key: "M-h", "M-H"
//   - Special keys: "Enter", "Tab", "Esc", "Space"
func ParseKeyNotation(s string) (KeySpec, error) {
	if s == "" {
		return KeySpec{}, fmt.Errorf("empty key notation")
	}

	// Ctrl modifier: C-x
	if strings.HasPrefix(s, "C-") && len(s) == 3 {
		letter := s[2]
		return Ctrl(letter), nil
	}

	// Alt modifier: M-x
	if strings.HasPrefix(s, "M-") && len(s) == 3 {
		key := s[2]
		return Alt(key), nil
	}

	// Special key names
	switch s {
	case "Enter":
		return Key(13), nil
	case "Tab":
		return Key(9), nil
	case "Esc":
		return Key(27), nil
	case "Space":
		return Key(32), nil
	}

	// Single character
	if len(s) == 1 {
		return Key(s[0]), nil
	}

	return KeySpec{}, fmt.Errorf("unrecognized key notation: %q", s)
}

// ParseBindings parses a comma-separated bindings string into a Keymap of overrides.
// Format: "key=action,key=action,!key" where !key means unbind.
// Examples:
//   - "H=line_end" → Direct[Key('H')] = ActionLineEnd
//   - "C-d=half_page_down" → Direct[Ctrl('d')] = ActionHalfPageDown
//   - "!H" → Direct[Key('H')] = ActionNone (unbind)
func ParseBindings(s string) (Keymap, error) {
	result := Keymap{
		Direct:      make(map[KeySpec]Action),
		Prefix:      make(map[byte]map[byte]Action),
		CharCapture: make(map[byte]Action),
		TextObjects: make(map[[2]byte]Action),
	}

	if s == "" {
		return result, nil
	}

	parts := strings.Split(s, ",")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		// Unbind: "!key"
		if strings.HasPrefix(part, "!") {
			keyStr := part[1:]
			key, err := ParseKeyNotation(keyStr)
			if err != nil {
				return Keymap{}, fmt.Errorf("unbind %q: %w", keyStr, err)
			}
			result.Direct[key] = ActionNone
			continue
		}

		// Bind: "key=action"
		eq := strings.IndexByte(part, '=')
		if eq < 0 {
			return Keymap{}, fmt.Errorf("invalid binding %q: expected key=action", part)
		}
		keyStr := part[:eq]
		actionStr := part[eq+1:]

		key, err := ParseKeyNotation(keyStr)
		if err != nil {
			return Keymap{}, fmt.Errorf("binding %q: %w", part, err)
		}

		action := Action(actionStr)
		if !isValidAction(action) {
			return Keymap{}, fmt.Errorf("binding %q: unknown action %q", part, actionStr)
		}

		result.Direct[key] = action
	}

	return result, nil
}

// isValidAction checks whether an action string corresponds to a known action.
func isValidAction(a Action) bool {
	switch a {
	case ActionMoveUp, ActionMoveDown, ActionMoveLeft, ActionMoveRight,
		ActionLineStart, ActionLineEnd, ActionFirstNonBlank, ActionLastNonBlank,
		ActionFirstLine, ActionLastLine,
		ActionWordForward, ActionWordBackward, ActionWordEnd, ActionWordEndBackward,
		ActionWORDForward, ActionWORDBackward, ActionWORDEnd, ActionWORDEndBackward,
		ActionParagraphForward, ActionParagraphBackward,
		ActionHalfPageUp, ActionHalfPageDown, ActionPageUp, ActionPageDown,
		ActionScreenTop, ActionScreenMiddle, ActionScreenBottom,
		ActionScrollLineUp, ActionScrollLineDown,
		ActionMatchBracket,
		ActionViewportTop, ActionViewportCenter, ActionViewportBottom,
		ActionDisplayLineDown, ActionDisplayLineUp,
		ActionJumpBack, ActionJumpListBack, ActionJumpListForward,
		ActionSetMark, ActionGoToMark, ActionGoToMarkLine,
		ActionVisualChar, ActionVisualLine, ActionVisualBlock,
		ActionSwapEnd, ActionSwapCorner,
		ActionYank, ActionYankLine,
		ActionSearchForward, ActionSearchBackward, ActionSearchNext, ActionSearchPrev,
		ActionSearchWordForward, ActionSearchWordBackward,
		ActionSearchSelect, ActionSearchSelectBack,
		ActionCharSearchF, ActionCharSearchT, ActionCharSearchFBack, ActionCharSearchTBack,
		ActionCharSearchRepeat, ActionCharSearchReverse,
		ActionTextObjectInnerWord, ActionTextObjectAWord,
		ActionTextObjectInnerWORD, ActionTextObjectAWORD,
		ActionTextObjectInnerParagraph, ActionTextObjectAParagraph,
		ActionTextObjectInnerQuote, ActionTextObjectAQuote,
		ActionTextObjectInnerParen, ActionTextObjectAParen,
		ActionTextObjectInnerBrace, ActionTextObjectABrace,
		ActionTextObjectInnerBracket, ActionTextObjectABracket,
		ActionTextObjectInnerAngle, ActionTextObjectAAngle,
		ActionClearSearch,
		ActionColonMode,
		ActionToggleLineMode, ActionToggleWrapMode,
		ActionEscape, ActionQuit,
		ActionThemeNext, ActionThemePrev,
		ActionDemoNext, ActionDemoPrev, ActionDemoThemeNext, ActionDemoThemePrev:
		return true
	}
	return false
}
