package input

import (
	"github.com/shitcoding/tmux_yankee/internal/keymap"
	"github.com/shitcoding/tmux_yankee/internal/motion"
)

// ActionToCommand converts a keymap.Action to the corresponding Command.
// count is the accumulated count prefix (0 = no count typed).
// capturedChar is the character captured by char-capture prefixes (f/t/F/T/m/`/').
func ActionToCommand(action keymap.Action, count int, capturedChar byte) Command {
	switch action {
	// Motion commands
	case keymap.ActionMoveUp:
		return Command{Type: CommandMotion, Motion: motion.MotionUp, Count: count}
	case keymap.ActionMoveDown:
		return Command{Type: CommandMotion, Motion: motion.MotionDown, Count: count}
	case keymap.ActionMoveLeft:
		return Command{Type: CommandMotion, Motion: motion.MotionLeft, Count: count}
	case keymap.ActionMoveRight:
		return Command{Type: CommandMotion, Motion: motion.MotionRight, Count: count}
	case keymap.ActionLineStart:
		return Command{Type: CommandMotion, Motion: motion.MotionLineStart, Count: count}
	case keymap.ActionLineEnd:
		return Command{Type: CommandMotion, Motion: motion.MotionLineEnd, Count: count}
	case keymap.ActionFirstNonBlank:
		return Command{Type: CommandMotion, Motion: motion.MotionFirstNonBlank, Count: count}
	case keymap.ActionLastNonBlank:
		return Command{Type: CommandMotion, Motion: motion.MotionLastNonBlank, Count: count}
	case keymap.ActionFirstLine:
		return Command{Type: CommandMotion, Motion: motion.MotionFirstLine, Count: count}
	case keymap.ActionLastLine:
		return Command{Type: CommandMotion, Motion: motion.MotionLastLine, Count: count}
	case keymap.ActionWordForward:
		return Command{Type: CommandMotion, Motion: motion.MotionWordForward, Count: count}
	case keymap.ActionWordBackward:
		return Command{Type: CommandMotion, Motion: motion.MotionWordBackward, Count: count}
	case keymap.ActionWordEnd:
		return Command{Type: CommandMotion, Motion: motion.MotionWordEnd, Count: count}
	case keymap.ActionWordEndBackward:
		return Command{Type: CommandMotion, Motion: motion.MotionWordEndBackward, Count: count}
	case keymap.ActionWORDForward:
		return Command{Type: CommandMotion, Motion: motion.MotionWORDForward, Count: count}
	case keymap.ActionWORDBackward:
		return Command{Type: CommandMotion, Motion: motion.MotionWORDBackward, Count: count}
	case keymap.ActionWORDEnd:
		return Command{Type: CommandMotion, Motion: motion.MotionWORDEnd, Count: count}
	case keymap.ActionWORDEndBackward:
		return Command{Type: CommandMotion, Motion: motion.MotionWORDEndBackward, Count: count}
	case keymap.ActionParagraphForward:
		return Command{Type: CommandMotion, Motion: motion.MotionParagraphForward, Count: count}
	case keymap.ActionParagraphBackward:
		return Command{Type: CommandMotion, Motion: motion.MotionParagraphBackward, Count: count}
	case keymap.ActionHalfPageUp:
		return Command{Type: CommandMotion, Motion: motion.MotionHalfPageUp, Count: count}
	case keymap.ActionHalfPageDown:
		return Command{Type: CommandMotion, Motion: motion.MotionHalfPageDown, Count: count}
	case keymap.ActionPageUp:
		return Command{Type: CommandMotion, Motion: motion.MotionPageUp, Count: count}
	case keymap.ActionPageDown:
		return Command{Type: CommandMotion, Motion: motion.MotionPageDown, Count: count}
	case keymap.ActionScreenTop:
		return Command{Type: CommandMotion, Motion: motion.MotionScreenTop, Count: count}
	case keymap.ActionScreenMiddle:
		return Command{Type: CommandMotion, Motion: motion.MotionScreenMiddle, Count: count}
	case keymap.ActionScreenBottom:
		return Command{Type: CommandMotion, Motion: motion.MotionScreenBottom, Count: count}
	case keymap.ActionMatchBracket:
		return Command{Type: CommandMotion, Motion: motion.MotionMatchBracket, Count: count}

	// Viewport positioning
	case keymap.ActionViewportTop:
		return Command{Type: CommandMotion, Motion: motion.MotionViewportTop, Count: 0}
	case keymap.ActionViewportCenter:
		return Command{Type: CommandMotion, Motion: motion.MotionViewportCenter, Count: 0}
	case keymap.ActionViewportBottom:
		return Command{Type: CommandMotion, Motion: motion.MotionViewportBottom, Count: 0}

	// Display line motions
	case keymap.ActionDisplayLineDown:
		return Command{Type: CommandDisplayLineDown, Count: count}
	case keymap.ActionDisplayLineUp:
		return Command{Type: CommandDisplayLineUp, Count: count}

	// Scroll line
	case keymap.ActionScrollLineUp:
		return Command{Type: CommandScrollLineUp}
	case keymap.ActionScrollLineDown:
		return Command{Type: CommandScrollLineDown}

	// Jump/marks
	case keymap.ActionJumpBack:
		return Command{Type: CommandJumpBack}
	case keymap.ActionJumpListBack:
		return Command{Type: CommandJumpListBack, Count: count}
	case keymap.ActionJumpListForward:
		return Command{Type: CommandJumpListForward, Count: count}
	case keymap.ActionSetMark:
		return Command{Type: CommandSetMark, MarkChar: capturedChar}
	case keymap.ActionGoToMark:
		return Command{Type: CommandGoToMark, MarkChar: capturedChar}
	case keymap.ActionGoToMarkLine:
		return Command{Type: CommandGoToMark, MarkChar: capturedChar}

	// Visual mode
	case keymap.ActionVisualChar:
		return Command{Type: CommandVisual}
	case keymap.ActionVisualLine:
		return Command{Type: CommandVisualLine}
	case keymap.ActionVisualBlock:
		return Command{Type: CommandVisualBlock}
	case keymap.ActionSwapEnd:
		return Command{Type: CommandSwapEnd}
	case keymap.ActionSwapCorner:
		return Command{Type: CommandSwapCorner}

	// Yank
	case keymap.ActionYank:
		return Command{Type: CommandYank}
	case keymap.ActionYankLine:
		return Command{Type: CommandYankLine}

	// Search
	case keymap.ActionSearchForward:
		return Command{Type: CommandSearchForward}
	case keymap.ActionSearchBackward:
		return Command{Type: CommandSearchBackward}
	case keymap.ActionSearchNext:
		return Command{Type: CommandSearchNext, Count: count}
	case keymap.ActionSearchPrev:
		return Command{Type: CommandSearchPrev, Count: count}
	case keymap.ActionSearchWordForward:
		return Command{Type: CommandSearchWordForward}
	case keymap.ActionSearchWordBackward:
		return Command{Type: CommandSearchWordBackward}
	case keymap.ActionSearchSelect:
		return Command{Type: CommandSearchSelect}
	case keymap.ActionSearchSelectBack:
		return Command{Type: CommandSearchSelectBack}

	// Clear search
	case keymap.ActionClearSearch:
		return Command{Type: CommandClearSearch}

	// Char search
	case keymap.ActionCharSearchF:
		return Command{Type: CommandCharSearch, SearchKind: SearchFindForward, SearchChar: capturedChar, Count: count}
	case keymap.ActionCharSearchT:
		return Command{Type: CommandCharSearch, SearchKind: SearchTillForward, SearchChar: capturedChar, Count: count}
	case keymap.ActionCharSearchFBack:
		return Command{Type: CommandCharSearch, SearchKind: SearchFindBackward, SearchChar: capturedChar, Count: count}
	case keymap.ActionCharSearchTBack:
		return Command{Type: CommandCharSearch, SearchKind: SearchTillBackward, SearchChar: capturedChar, Count: count}
	case keymap.ActionCharSearchRepeat:
		return Command{Type: CommandCharSearch, SearchKind: SearchRepeat, Count: count}
	case keymap.ActionCharSearchReverse:
		return Command{Type: CommandCharSearch, SearchKind: SearchRepeatReverse, Count: count}

	// Text objects
	case keymap.ActionTextObjectInnerWord, keymap.ActionTextObjectAWord,
		keymap.ActionTextObjectInnerWORD, keymap.ActionTextObjectAWORD,
		keymap.ActionTextObjectInnerParagraph, keymap.ActionTextObjectAParagraph,
		keymap.ActionTextObjectInnerQuote, keymap.ActionTextObjectAQuote,
		keymap.ActionTextObjectInnerParen, keymap.ActionTextObjectAParen,
		keymap.ActionTextObjectInnerBrace, keymap.ActionTextObjectABrace,
		keymap.ActionTextObjectInnerBracket, keymap.ActionTextObjectABracket,
		keymap.ActionTextObjectInnerAngle, keymap.ActionTextObjectAAngle:
		return Command{Type: CommandTextObject, TextObject: string(action)}

	// Mode control
	case keymap.ActionToggleLineMode:
		return Command{Type: CommandToggleLineMode}
	case keymap.ActionToggleWrapMode:
		return Command{Type: CommandToggleWrapMode}
	case keymap.ActionEscape:
		return Command{Type: CommandEscape}
	case keymap.ActionQuit:
		return Command{Type: CommandQuit}

	// Demo
	case keymap.ActionDemoNext:
		return Command{Type: CommandDemoNext}
	case keymap.ActionDemoPrev:
		return Command{Type: CommandDemoPrev}
	case keymap.ActionDemoThemeNext:
		return Command{Type: CommandDemoThemeNext}
	case keymap.ActionDemoThemePrev:
		return Command{Type: CommandDemoThemePrev}

	default:
		return Command{Type: CommandNone}
	}
}
