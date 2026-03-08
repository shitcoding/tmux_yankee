package flash

// OverlayPos identifies a single cell position for label rendering.
type OverlayPos struct {
	Line int
	Col  int
}

// OverlayRange identifies a match span for highlight rendering.
type OverlayRange struct {
	Line     int
	ColStart int
	ColEnd   int
}

// Overlay holds the rendering state for flash-mode matches and labels.
type Overlay struct {
	Labels   map[OverlayPos]byte
	Matches  []OverlayRange
	Backdrop bool
	Prompt   string
}

// BuildOverlay creates an Overlay from a slice of labeled matches.
func BuildOverlay(matches []Match, prompt string, backdrop bool) *Overlay {
	o := &Overlay{
		Labels:   make(map[OverlayPos]byte),
		Matches:  make([]OverlayRange, 0, len(matches)),
		Backdrop: backdrop,
		Prompt:   prompt,
	}

	for _, m := range matches {
		o.Matches = append(o.Matches, OverlayRange{
			Line:     m.Line,
			ColStart: m.ColStart,
			ColEnd:   m.ColEnd,
		})
		if m.Label != 0 {
			o.Labels[OverlayPos{Line: m.Line, Col: m.ColEnd}] = m.Label
		}
	}

	return o
}

// HasLabel returns the label character at the given position, or 0 if none.
// Nil-safe: returns 0 if the receiver is nil.
func (o *Overlay) HasLabel(line, col int) byte {
	if o == nil {
		return 0
	}
	return o.Labels[OverlayPos{Line: line, Col: col}]
}

// InMatch returns true if the given cell position falls within any match range.
// Nil-safe: returns false if the receiver is nil.
func (o *Overlay) InMatch(line, col int) bool {
	if o == nil {
		return false
	}
	for _, r := range o.Matches {
		if r.Line == line && col >= r.ColStart && col < r.ColEnd {
			return true
		}
	}
	return false
}
