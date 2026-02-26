package keymap

// DefaultKeymap returns the full default keymap matching all current bindings
// plus new Tier 1/2/3 defaults.
//
// Note: toggleKey and wrapKey are NOT in the keymap. The parser checks them
// first for backward compatibility. The default keymap does NOT include 'L'
// for screen_bottom — it's reserved for the toggle key. Users who want H/M/L
// must change @yankee_toggle_mode_key.
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

			// Jump back
			Key('`'):  ActionJumpBack,
			Key('\''): ActionJumpBack,

			// Mode/quit
			Key('q'): ActionQuit,
			Ctrl('c'): ActionQuit,

			// Demo (Tab, Shift-Tab handled separately via CSI)
			Key(9):   ActionDemoNext, // Tab
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
			'f': ActionCharSearchF,
			't': ActionCharSearchT,
			'F': ActionCharSearchFBack,
			'T': ActionCharSearchTBack,
			'm': ActionSetMark,
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
			{'i', '\''}: ActionTextObjectInnerQuote,
			{'a', '\''}: ActionTextObjectAQuote,
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
		},
	}
}
