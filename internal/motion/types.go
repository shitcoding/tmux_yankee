package motion

// Motion represents a cursor movement action.
type Motion int

const (
	MotionUp Motion = iota
	MotionDown
	MotionLeft
	MotionRight
	MotionLineStart    // 0 - beginning of line
	MotionLineEnd      // $ - end of line
	MotionFirstLine    // gg - first line of document
	MotionLastLine     // G - last line of document
	MotionHalfPageUp   // Ctrl-U
	MotionHalfPageDown // Ctrl-D
	MotionWordForward     // w - next word start
	MotionWordBackward    // b - previous word start
	MotionWordEnd         // e - current/next word end
	MotionFirstNonBlank   // ^ - first non-blank character
	MotionWORDEnd         // E - current/next WORD end (whitespace-separated)
	MotionWORDBackward    // B - previous WORD start (whitespace-separated)
	MotionViewportTop     // zt - position cursor line at top of viewport
	MotionViewportCenter  // zz - position cursor line at center of viewport
	MotionViewportBottom  // zb - position cursor line at bottom of viewport
)

// Cursor represents the cursor position in the document.
type Cursor struct {
	Line int // 0-indexed line number
	Col  int // 0-indexed column (rune offset, not byte)
}

// Viewport represents the visible portion of the document.
type Viewport struct {
	Top    int // 0-indexed first visible line
	Height int // number of visible lines
}

// Result contains the new cursor and viewport after applying a motion.
type Result struct {
	Cursor   Cursor
	Viewport Viewport
}

// Document provides read-only access to document content for motion calculations.
type Document interface {
	// LineCount returns the total number of lines in the document.
	LineCount() int

	// Line returns the content of the line at the given index.
	// Returns empty string if index is out of bounds.
	Line(index int) string

	// LineRuneCount returns the number of runes (Unicode characters) in the line.
	// Returns 0 if index is out of bounds.
	LineRuneCount(index int) int
}

// Handler applies motions to cursor and viewport positions.
type Handler interface {
	// Apply executes the given motion with the specified count.
	//
	// Count semantics:
	//   count=0: No explicit count (e.g., plain "j" or "G")
	//   count>=1: Explicit count (e.g., "5j" has count=5, "1G" has count=1)
	//
	// For most motions (j/k/h/l), count=0 behaves as count=1.
	// For gg/G, count=0 has special meaning:
	//   - MotionFirstLine with count=0: go to first line (gg)
	//   - MotionFirstLine with count=N: go to line N (Ngg)
	//   - MotionLastLine with count=0: go to last line (G)
	//   - MotionLastLine with count=N: go to line N (NG)
	Apply(doc Document, cursor Cursor, viewport Viewport, motion Motion, count int) Result
}
