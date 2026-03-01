package keymap

// Action represents a bindable yankee action.
type Action string

const (
	// Special
	ActionNone Action = "" // unbind / no action

	// Motions
	ActionMoveUp            Action = "move_up"
	ActionMoveDown          Action = "move_down"
	ActionMoveLeft          Action = "move_left"
	ActionMoveRight         Action = "move_right"
	ActionLineStart         Action = "line_start"         // 0
	ActionLineEnd           Action = "line_end"           // $
	ActionFirstNonBlank     Action = "first_nonblank"     // ^
	ActionLastNonBlank      Action = "last_nonblank"      // g_
	ActionFirstLine         Action = "first_line"         // gg
	ActionLastLine          Action = "last_line"          // G
	ActionWordForward       Action = "word_forward"       // w
	ActionWordBackward      Action = "word_backward"      // b
	ActionWordEnd           Action = "word_end"           // e
	ActionWordEndBackward   Action = "word_end_backward"  // ge
	ActionWORDForward       Action = "WORD_forward"       // W
	ActionWORDBackward      Action = "WORD_backward"      // B
	ActionWORDEnd           Action = "WORD_end"           // E
	ActionWORDEndBackward   Action = "WORD_end_backward"  // gE
	ActionParagraphForward  Action = "paragraph_forward"  // }
	ActionParagraphBackward Action = "paragraph_backward" // {
	ActionHalfPageUp        Action = "half_page_up"       // Ctrl-U
	ActionHalfPageDown      Action = "half_page_down"     // Ctrl-D
	ActionPageUp            Action = "page_up"            // Ctrl-B
	ActionPageDown          Action = "page_down"          // Ctrl-F
	ActionScreenTop         Action = "screen_top"         // H
	ActionScreenMiddle      Action = "screen_middle"      // M
	ActionScreenBottom      Action = "screen_bottom"      // L
	ActionScrollLineUp      Action = "scroll_line_up"     // Ctrl-Y
	ActionScrollLineDown    Action = "scroll_line_down"   // Ctrl-E
	ActionMatchBracket      Action = "match_bracket"      // %

	// Viewport positioning
	ActionViewportTop    Action = "viewport_top"    // zt
	ActionViewportCenter Action = "viewport_center" // zz
	ActionViewportBottom Action = "viewport_bottom" // zb

	// Display line motions (wrap mode)
	ActionDisplayLineDown Action = "display_line_down" // gj
	ActionDisplayLineUp   Action = "display_line_up"   // gk

	// Jump list & marks
	ActionJumpBack         Action = "jump_back"         // `` or ''
	ActionJumpListBack     Action = "jumplist_back"     // Ctrl-O
	ActionJumpListForward  Action = "jumplist_forward"  // Ctrl-I
	ActionSetMark          Action = "set_mark"          // m{char}
	ActionGoToMark         Action = "goto_mark"         // `{char}
	ActionGoToMarkLine     Action = "goto_mark_line"    // '{char}

	// Visual mode
	ActionVisualChar  Action = "visual_char"  // v
	ActionVisualLine  Action = "visual_line"  // V
	ActionVisualBlock Action = "visual_block" // Ctrl-V
	ActionSwapEnd     Action = "swap_end"     // o
	ActionSwapCorner  Action = "swap_corner"  // O

	// Yank
	ActionYank     Action = "yank"      // y (in visual), Enter
	ActionYankLine Action = "yank_line" // yy

	// Search
	ActionSearchForward      Action = "search_forward"       // /
	ActionSearchBackward     Action = "search_backward"      // ?
	ActionSearchNext         Action = "search_next"          // n
	ActionSearchPrev         Action = "search_prev"          // N
	ActionSearchWordForward  Action = "search_word_forward"  // *
	ActionSearchWordBackward Action = "search_word_backward" // #
	ActionSearchSelect       Action = "search_select"        // gn
	ActionSearchSelectBack   Action = "search_select_back"   // gN

	// Char search (f/t/F/T are char-capture, ;/, are direct)
	ActionCharSearchF       Action = "char_search_f"       // f{char}
	ActionCharSearchT       Action = "char_search_t"       // t{char}
	ActionCharSearchFBack   Action = "char_search_F"       // F{char}
	ActionCharSearchTBack   Action = "char_search_T"       // T{char}
	ActionCharSearchRepeat  Action = "char_search_repeat"  // ;
	ActionCharSearchReverse Action = "char_search_reverse" // ,

	// Text objects (visual mode only)
	ActionTextObjectInnerWord      Action = "inner_word"      // iw
	ActionTextObjectAWord          Action = "a_word"          // aw
	ActionTextObjectInnerWORD      Action = "inner_WORD"      // iW
	ActionTextObjectAWORD          Action = "a_WORD"          // aW
	ActionTextObjectInnerParagraph Action = "inner_paragraph" // ip
	ActionTextObjectAParagraph     Action = "a_paragraph"     // ap
	ActionTextObjectInnerQuote     Action = "inner_quote"     // i" / i'
	ActionTextObjectAQuote         Action = "a_quote"         // a" / a'
	ActionTextObjectInnerParen     Action = "inner_paren"     // ib / i(
	ActionTextObjectAParen         Action = "a_paren"         // ab / a(
	ActionTextObjectInnerBrace     Action = "inner_brace"     // iB / i{
	ActionTextObjectABrace         Action = "a_brace"         // aB / a{
	ActionTextObjectInnerBracket   Action = "inner_bracket"   // i[
	ActionTextObjectABracket       Action = "a_bracket"       // a[
	ActionTextObjectInnerAngle     Action = "inner_angle"     // i<
	ActionTextObjectAAngle         Action = "a_angle"         // a<

	// Search control
	ActionClearSearch Action = "clear_search" // \ — clear search highlights

	// Colon command-line
	ActionColonMode Action = "colon_mode" // : — enter colon command mode

	// Mode control
	ActionToggleLineMode Action = "toggle_line_mode"
	ActionToggleWrapMode Action = "toggle_wrap_mode" // gw
	ActionEscape         Action = "escape"
	ActionQuit           Action = "quit"

	// Demo
	ActionDemoNext      Action = "demo_next"
	ActionDemoPrev      Action = "demo_prev"
	ActionDemoThemeNext Action = "demo_theme_next"
	ActionDemoThemePrev Action = "demo_theme_prev"
)

// Modifier represents a keyboard modifier.
type Modifier int

const (
	ModNone Modifier = 0
	ModCtrl Modifier = 1
	ModAlt  Modifier = 2
)

// KeySpec represents a keyboard key, possibly with modifiers.
type KeySpec struct {
	Key byte     // ASCII byte value (0 if not applicable)
	Mod Modifier // Ctrl, Alt, or None
}

// Key creates a KeySpec for a plain key.
func Key(b byte) KeySpec {
	return KeySpec{Key: b, Mod: ModNone}
}

// Ctrl creates a KeySpec for a Ctrl+letter combination.
func Ctrl(letter byte) KeySpec {
	return KeySpec{Key: letter, Mod: ModCtrl}
}

// Alt creates a KeySpec for an Alt+key combination.
func Alt(b byte) KeySpec {
	return KeySpec{Key: b, Mod: ModAlt}
}

// ToByte converts a KeySpec to the byte the terminal sends.
// For Ctrl keys: Ctrl+A = 1, Ctrl+B = 2, ..., Ctrl+Z = 26.
// For Alt keys: returns the key byte (ESC prefix handled separately by parser).
// For plain keys: returns the key byte directly.
func (k KeySpec) ToByte() byte {
	switch k.Mod {
	case ModCtrl:
		// Ctrl+a = 1, Ctrl+b = 2, ..., Ctrl+z = 26
		if k.Key >= 'a' && k.Key <= 'z' {
			return k.Key - 'a' + 1
		}
		if k.Key >= 'A' && k.Key <= 'Z' {
			return k.Key - 'A' + 1
		}
		return k.Key
	default:
		return k.Key
	}
}

// Keymap holds all keybinding mappings organized by binding type.
type Keymap struct {
	// Direct: single key → action
	Direct map[KeySpec]Action

	// Prefix: prefix byte → (second byte → action)
	Prefix map[byte]map[byte]Action

	// CharCapture: prefix byte → action (next char is captured as parameter)
	CharCapture map[byte]Action

	// TextObjects: [prefix, object] pair → action (visual mode only)
	TextObjects map[[2]byte]Action
}

// Lookup checks the Direct map for a KeySpec and returns the action.
func (km *Keymap) Lookup(key KeySpec) (Action, bool) {
	if km.Direct == nil {
		return ActionNone, false
	}
	action, ok := km.Direct[key]
	return action, ok
}

// IsPrefix returns true if the byte is a prefix key (starts a prefix sequence).
func (km *Keymap) IsPrefix(b byte) bool {
	if km.Prefix == nil {
		return false
	}
	_, ok := km.Prefix[b]
	return ok
}

// IsCharCapture returns true if the byte is a char-capture prefix key.
func (km *Keymap) IsCharCapture(b byte) bool {
	if km.CharCapture == nil {
		return false
	}
	_, ok := km.CharCapture[b]
	return ok
}

// HasTextObjectPrefix returns true if the byte is a text object prefix ('i' or 'a').
func (km *Keymap) HasTextObjectPrefix(b byte) bool {
	if km.TextObjects == nil {
		return false
	}
	for key := range km.TextObjects {
		if key[0] == b {
			return true
		}
	}
	return false
}

// LookupPrefix looks up a two-key prefix sequence (e.g., 'g'+'g' → first_line).
func (km *Keymap) LookupPrefix(prefix, second byte) (Action, bool) {
	if km.Prefix == nil {
		return ActionNone, false
	}
	secondMap, ok := km.Prefix[prefix]
	if !ok {
		return ActionNone, false
	}
	action, ok := secondMap[second]
	return action, ok
}

// LookupCharCapture looks up a char-capture prefix (e.g., 'f' → char_search_f).
func (km *Keymap) LookupCharCapture(prefix byte) (Action, bool) {
	if km.CharCapture == nil {
		return ActionNone, false
	}
	action, ok := km.CharCapture[prefix]
	return action, ok
}

// LookupTextObject looks up a text object binding (e.g., 'i'+'w' → inner_word).
func (km *Keymap) LookupTextObject(prefix, second byte) (Action, bool) {
	if km.TextObjects == nil {
		return ActionNone, false
	}
	action, ok := km.TextObjects[[2]byte{prefix, second}]
	return action, ok
}
