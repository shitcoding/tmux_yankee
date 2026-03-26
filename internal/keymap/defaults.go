package keymap

// DefaultKeymap returns the full default keymap matching all current bindings
// plus new Tier 1/2/3 defaults.
//
// Note: wrapKey is NOT in the keymap. The parser checks it separately.
// toggleKey is a legacy fallback — keymap bindings take priority.
// Alt+Shift+L (M-L) is the default binding for toggle_line_mode.
// L is bound to screen_bottom (vim default).
func DefaultKeymap() Keymap {
	return Keymap{
		Direct: map[KeySpec]Action{
			// Motions
			Key('h'):  ActionMoveLeft,
			Key('j'):  ActionMoveDown,
			Key('k'):  ActionMoveUp,
			Key('l'):  ActionMoveRight,
			Key('$'):  ActionLineEnd,
			Key('^'):  ActionFirstNonBlank,
			Key('G'):  ActionLastLine,
			Key('w'):  ActionWordForward,
			Key('b'):  ActionWordBackward,
			Key('e'):  ActionWordEnd,
			Key('W'):  ActionWORDForward,
			Key('B'):  ActionWORDBackward,
			Key('E'):  ActionWORDEnd,
			Key('{'):  ActionParagraphBackward,
			Key('}'):  ActionParagraphForward,
			Key('H'):  ActionScreenTop,
			Key('M'):  ActionScreenMiddle,
			Key('L'):  ActionScreenBottom,
			Key('%'):  ActionMatchBracket,

			// Scroll/page
			Ctrl('d'): ActionHalfPageDown,
			Ctrl('u'): ActionHalfPageUp,
			Ctrl('f'): ActionPageDown,
			Ctrl('b'): ActionPageUp,
			Ctrl('e'): ActionScrollLineDown,
			Ctrl('y'): ActionScrollLineUp,

			// Jump list
			Ctrl('o'): ActionJumpListBack,
			Ctrl('i'): ActionJumpListForward,

			// Visual mode
			Key('v'):  ActionVisualChar,
			Key('V'):  ActionVisualLine,
			Ctrl('v'): ActionVisualBlock,
			Key('o'):  ActionSwapEnd,
			Key('O'):  ActionSwapCorner,

			// Yank
			Key(13): ActionYank, // Enter

			// Search
			Key('/'):  ActionSearchForward,
			Key('?'):  ActionSearchBackward,
			Key('n'):  ActionSearchNext,
			Key('N'):  ActionSearchPrev,
			Key('*'):  ActionSearchWordForward,
			Key('#'):  ActionSearchWordBackward,

			// Char search repeat
			Key(';'): ActionCharSearchRepeat,
			Key(','): ActionCharSearchReverse,

			// Colon command-line
			Key(':'): ActionColonMode,

			// Clear search highlights
			Key('\\'): ActionClearSearch,

			// Flash
			Key('s'): ActionFlash,

			// Mode/quit
			Key('q'):  ActionQuit,
			Ctrl('c'): ActionQuit,

			// Theme cycling
			Alt('t'): ActionThemeNext,
			Alt('L'): ActionToggleLineMode,

			// Demo (Shift-Tab handled via CSI)
			Key(']'): ActionDemoThemeNext,
			Key('['): ActionDemoThemePrev,
		},

		Prefix: map[byte]map[byte]Action{
			'g': {
				'g': ActionFirstLine,
				'j': ActionDisplayLineDown,
				'k': ActionDisplayLineUp,
				// 'w' is handled specially via wrapKey in parser
				'e': ActionWordEndBackward,
				'E': ActionWORDEndBackward,
				'_': ActionLastNonBlank,
				'n': ActionSearchSelect,
				'N': ActionSearchSelectBack,
			},
			'z': {
				't': ActionViewportTop,
				'z': ActionViewportCenter,
				'b': ActionViewportBottom,
			},
			'y': {
				'y': ActionYankLine,
			},
		},

		CharCapture: map[byte]Action{
			'f':    ActionCharSearchF,
			't':    ActionCharSearchT,
			'F':    ActionCharSearchFBack,
			'T':    ActionCharSearchTBack,
			'm':    ActionSetMark,
			'`':    ActionGoToMark,
			'\'':   ActionGoToMarkLine,
		},

		TextObjects: map[[2]byte]Action{
			{'i', 'w'}:  ActionTextObjectInnerWord,
			{'a', 'w'}:  ActionTextObjectAWord,
			{'i', 'W'}:  ActionTextObjectInnerWORD,
			{'a', 'W'}:  ActionTextObjectAWORD,
			{'i', 'p'}:  ActionTextObjectInnerParagraph,
			{'a', 'p'}:  ActionTextObjectAParagraph,
			{'i', '"'}:  ActionTextObjectInnerQuote,
			{'a', '"'}:  ActionTextObjectAQuote,
			{'i', '\''}: ActionTextObjectInnerSingleQuote,
			{'a', '\''}: ActionTextObjectASingleQuote,
			{'i', '`'}:  ActionTextObjectInnerBacktick,
			{'a', '`'}:  ActionTextObjectABacktick,
			{'i', '('}:  ActionTextObjectInnerParen,
			{'a', '('}:  ActionTextObjectAParen,
			{'i', ')'}:  ActionTextObjectInnerParen,
			{'a', ')'}:  ActionTextObjectAParen,
			{'i', 'b'}:  ActionTextObjectInnerParen,
			{'a', 'b'}:  ActionTextObjectAParen,
			{'i', '{'}:  ActionTextObjectInnerBrace,
			{'a', '{'}:  ActionTextObjectABrace,
			{'i', '}'}:  ActionTextObjectInnerBrace,
			{'a', '}'}:  ActionTextObjectABrace,
			{'i', 'B'}:  ActionTextObjectInnerBrace,
			{'a', 'B'}:  ActionTextObjectABrace,
			{'i', '['}:  ActionTextObjectInnerBracket,
			{'a', '['}:  ActionTextObjectABracket,
			{'i', ']'}:  ActionTextObjectInnerBracket,
			{'a', ']'}:  ActionTextObjectABracket,
			{'i', '<'}:  ActionTextObjectInnerAngle,
			{'a', '<'}:  ActionTextObjectAAngle,
			{'i', '>'}:  ActionTextObjectInnerAngle,
			{'a', '>'}:  ActionTextObjectAAngle,
		},
	}
}
