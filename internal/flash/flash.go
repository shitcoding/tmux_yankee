package flash

// ActionType represents the result of a state machine transition.
type ActionType int

const (
	// ActionContinue means the caller should keep running the flash loop.
	ActionContinue ActionType = iota
	// ActionJump means a label was selected and the caller should jump.
	ActionJump
	// ActionAutoJump means exactly one labeled match exists and the caller should jump.
	ActionAutoJump
	// ActionCancel means flash mode was cancelled (Escape or backspace to empty).
	ActionCancel
)

// Action is the output of a state machine transition, including optional
// jump coordinates.
type Action struct {
	Type ActionType
	Line int
	Col  int
}

// Options configures flash state machine behavior.
type Options struct {
	// MinChars is the minimum pattern length before labels are assigned.
	// Defaults to 1 if less than 1.
	MinChars int

	// JumpPos controls where the cursor lands for lowercase label presses.
	// Defaults to JumpPosMatchEnd (last char of match).
	JumpPos JumpPos

	// AltJumpPos controls where the cursor lands for uppercase label presses.
	// Defaults to JumpPosMatchStart (first char of match).
	// Set to JumpPosOff to disable uppercase label jumps.
	AltJumpPos JumpPos
}

// State manages the lifecycle of a single flash-search interaction.
type State struct {
	Active        bool
	Pattern       string
	Matches       []Match
	SavedCursor   [2]int
	SavedViewport int
	labeler       *Labeler
	minChars      int
	overlay       *Overlay
	jumpPos       JumpPos
	altJumpPos    JumpPos
}

// New creates a new flash State with the given options.
func New(opts Options) *State {
	minChars := max(opts.MinChars, 1)
	// AltJumpPos defaults to JumpPosMatchStart (iota value 1) when Options
	// is zero-initialized. Since JumpPosMatchEnd is iota 0 and that's also
	// the zero value, we use a sentinel: if both are zero (default), set
	// altJumpPos to JumpPosMatchStart for ergonomic defaults.
	altJumpPos := opts.AltJumpPos
	if opts.JumpPos == JumpPosMatchEnd && opts.AltJumpPos == JumpPosMatchEnd {
		altJumpPos = JumpPosMatchStart
	}
	return &State{
		labeler:    NewLabeler(),
		minChars:   minChars,
		jumpPos:    opts.JumpPos,
		altJumpPos: altJumpPos,
	}
}

// Enter activates flash mode, saving the current cursor and viewport position.
func (s *State) Enter(cursorLine, cursorCol, viewportTop int) {
	s.Active = true
	s.Pattern = ""
	s.Matches = nil
	s.SavedCursor = [2]int{cursorLine, cursorCol}
	s.SavedViewport = viewportTop
	s.overlay = BuildOverlay(nil, "", true) // backdrop on immediately
}

// UpdatePattern recomputes matches and labels for the current pattern.
// Returns ActionAutoJump if exactly one labeled match exists, otherwise ActionContinue.
func (s *State) UpdatePattern(pattern string, lines []string, viewportTop, viewportHeight int) Action {
	s.Pattern = pattern
	s.Matches = FindMatches(lines, pattern, viewportTop, viewportHeight)

	if len(pattern) >= s.minChars && len(s.Matches) > 0 {
		// Compute forbidden labels: any character that, appended to the pattern,
		// would still produce matches. This prevents label/pattern ambiguity.
		forbidden := make(map[byte]bool)
		for c := byte('a'); c <= 'z'; c++ {
			ext := FindMatches(lines, pattern+string(rune(c)), viewportTop, viewportHeight)
			if len(ext) > 0 {
				forbidden[c] = true
			}
		}
		s.labeler.AssignWithForbidden(s.Matches, s.SavedCursor[0], s.SavedCursor[1], lines, forbidden)
	}

	s.overlay = BuildOverlay(s.Matches, pattern, true)

	// Count labeled matches for auto-jump
	labeled := 0
	var jumpMatch Match
	for _, m := range s.Matches {
		if m.Label != 0 {
			labeled++
			jumpMatch = m
		}
	}

	if labeled == 1 {
		lineText := s.getLineText(lines, jumpMatch.Line)
		col := ResolveJumpCol(lineText, jumpMatch, s.jumpPos)
		s.exit()
		return Action{
			Type: ActionAutoJump,
			Line: jumpMatch.Line,
			Col:  col,
		}
	}

	return Action{Type: ActionContinue}
}

// HandleKey processes a single keypress during flash mode.
//
// Key semantics:
//   - Escape (27): Cancel flash mode
//   - Backspace (127 or 8) with pattern length <= 1: Cancel flash mode
//   - Backspace with longer pattern: Continue (caller should shorten pattern and call UpdatePattern)
//   - Lowercase char matching a label: Jump using jumpPos
//   - Uppercase char (A-Z) when altJumpPos != Off: Look up toLower(key) as label, jump using altJumpPos
//   - Other printable: Continue (caller should append to pattern and call UpdatePattern)
//
// lines is the full document text, used to resolve word boundary positions.
// It may be nil if word_start/word_end jump positions are not configured.
func (s *State) HandleKey(key byte, lines []string) Action {
	// Escape
	if key == 27 {
		s.exit()
		return Action{Type: ActionCancel}
	}

	// Backspace
	if key == 127 || key == 8 {
		if len(s.Pattern) <= 1 {
			s.exit()
			return Action{Type: ActionCancel}
		}
		return Action{Type: ActionContinue}
	}

	// Uppercase key: alt-jump if altJumpPos is not Off
	if key >= 'A' && key <= 'Z' && s.altJumpPos != JumpPosOff {
		lower := key + 32 // 'A' -> 'a', etc.
		for _, m := range s.Matches {
			if m.Label == lower {
				lineText := s.getLineText(lines, m.Line)
				col := ResolveJumpCol(lineText, m, s.altJumpPos)
				s.exit()
				return Action{
					Type: ActionJump,
					Line: m.Line,
					Col:  col,
				}
			}
		}
	}

	// Check if key matches a label (lowercase label press)
	for _, m := range s.Matches {
		if m.Label == key {
			lineText := s.getLineText(lines, m.Line)
			col := ResolveJumpCol(lineText, m, s.jumpPos)
			s.exit()
			return Action{
				Type: ActionJump,
				Line: m.Line,
				Col:  col,
			}
		}
	}

	// Other printable character
	return Action{Type: ActionContinue}
}

// getLineText safely retrieves a line from the document for jump col resolution.
func (s *State) getLineText(lines []string, lineIdx int) string {
	if lines == nil || lineIdx < 0 || lineIdx >= len(lines) {
		return ""
	}
	return lines[lineIdx]
}

// Overlay returns the current overlay for rendering, or nil when flash is not active.
func (s *State) Overlay() *Overlay {
	if !s.Active {
		return nil
	}
	return s.overlay
}

// exit deactivates flash mode and clears state.
func (s *State) exit() {
	s.Active = false
	s.Pattern = ""
	s.Matches = nil
	s.overlay = nil
}
