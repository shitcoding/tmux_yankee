package ui

import (
	"testing"

	"github.com/shitcoding/tmux_yankee/internal/input"
	vmode "github.com/shitcoding/tmux_yankee/internal/mode"
	"github.com/shitcoding/tmux_yankee/internal/motion"
	"github.com/shitcoding/tmux_yankee/internal/selection"
)

// makeCancelTUI returns a TUI populated enough to exercise search
// commands directly through handleCommand.
func makeCancelTUI(lines []string) *TUI {
	t := &TUI{
		doc:            NewDocument(lines),
		searchMatchIdx: -1,
		height:         24,
		width:          80,
		modeMachine:    vmode.NewMachine(),
		motionHandler:  motion.NewVimHandler(),
		parser:         input.NewParser(),
	}
	return t
}

func TestSearchCancel_RestoresPattern(t *testing.T) {
	ti := makeCancelTUI([]string{"foo bar", "baz foo", "qux"})

	// Confirm /foo
	ti.handleCommand(input.Command{Type: input.CommandSearchForward})
	ti.handleCommand(input.Command{Type: input.CommandSearchUpdate, SearchPattern: "foo"})
	ti.handleCommand(input.Command{Type: input.CommandSearchConfirm, SearchPattern: "foo"})

	if ti.searchPattern != "foo" {
		t.Fatalf("after confirm: searchPattern = %q, want foo", ti.searchPattern)
	}
	confirmedMatchCount := len(ti.searchMatches)
	if confirmedMatchCount == 0 {
		t.Fatal("expected matches for foo")
	}

	// Type provisional /bar then cancel.
	ti.handleCommand(input.Command{Type: input.CommandSearchForward})
	ti.handleCommand(input.Command{Type: input.CommandSearchUpdate, SearchPattern: "bar"})
	ti.handleCommand(input.Command{Type: input.CommandSearchCancel})

	if ti.searchPattern != "foo" {
		t.Errorf("after cancel: searchPattern = %q, want foo", ti.searchPattern)
	}
	if !ti.searchActive {
		t.Error("after cancel: searchActive = false, want true (foo was still active)")
	}
	if len(ti.searchMatches) != confirmedMatchCount {
		t.Errorf("after cancel: %d matches, want %d (foo's match list)", len(ti.searchMatches), confirmedMatchCount)
	}
}

func TestSearchCancel_RestoresDirectionBackward(t *testing.T) {
	ti := makeCancelTUI([]string{"foo", "bar", "foo", "baz"})

	// Confirm backward search ?foo
	ti.handleCommand(input.Command{Type: input.CommandSearchBackward})
	ti.handleCommand(input.Command{Type: input.CommandSearchUpdate, SearchPattern: "foo"})
	ti.handleCommand(input.Command{Type: input.CommandSearchConfirm, SearchPattern: "foo"})
	if ti.searchDirection != -1 {
		t.Fatalf("after backward confirm: direction = %d, want -1", ti.searchDirection)
	}

	// Provisional forward /b, then cancel.
	ti.handleCommand(input.Command{Type: input.CommandSearchForward})
	ti.handleCommand(input.Command{Type: input.CommandSearchUpdate, SearchPattern: "b"})
	ti.handleCommand(input.Command{Type: input.CommandSearchCancel})

	if ti.searchDirection != -1 {
		t.Errorf("after cancel: direction = %d, want -1 (the prior backward search must win)", ti.searchDirection)
	}
}

func TestSearchCancel_RestoresViewport(t *testing.T) {
	lines := make([]string, 100)
	for i := range lines {
		lines[i] = "line text"
	}
	lines[60] = "match here"
	ti := makeCancelTUI(lines)

	ti.viewportTop = 0
	ti.cursorLine = 0
	ti.cursorCol = 0

	// Confirm /match — viewport will move to make match visible.
	ti.handleCommand(input.Command{Type: input.CommandSearchForward})
	ti.handleCommand(input.Command{Type: input.CommandSearchUpdate, SearchPattern: "match"})
	ti.handleCommand(input.Command{Type: input.CommandSearchConfirm, SearchPattern: "match"})
	confirmedVTop := ti.viewportTop

	// Provisional pattern that has no matches — should NOT change viewport much,
	// but if it did, cancel must restore.
	ti.handleCommand(input.Command{Type: input.CommandSearchForward})
	ti.handleCommand(input.Command{Type: input.CommandSearchUpdate, SearchPattern: "zzznotfound"})
	// Force viewport to a different position to prove restoration.
	ti.viewportTop = 0
	ti.handleCommand(input.Command{Type: input.CommandSearchCancel})

	if ti.viewportTop != confirmedVTop {
		t.Errorf("after cancel: viewportTop = %d, want %d (post-confirm position)", ti.viewportTop, confirmedVTop)
	}
}

func TestSearchCancel_NoPriorSearch(t *testing.T) {
	ti := makeCancelTUI([]string{"foo bar baz"})

	// Provisional only, no confirmed search before.
	ti.handleCommand(input.Command{Type: input.CommandSearchForward})
	ti.handleCommand(input.Command{Type: input.CommandSearchUpdate, SearchPattern: "foo"})
	ti.handleCommand(input.Command{Type: input.CommandSearchCancel})

	if ti.searchActive {
		t.Error("after cancel with no prior search: searchActive = true, want false")
	}
	if ti.searchPattern != "" {
		t.Errorf("after cancel with no prior search: pattern = %q, want \"\"", ti.searchPattern)
	}
	if len(ti.searchMatches) != 0 {
		t.Errorf("after cancel with no prior search: %d matches, want 0", len(ti.searchMatches))
	}
}

func TestSearchCancel_RestoresHOffset(t *testing.T) {
	ti := makeCancelTUI([]string{"this is a long line containing foo somewhere"})
	ti.hOffset = 7

	ti.handleCommand(input.Command{Type: input.CommandSearchForward})
	ti.handleCommand(input.Command{Type: input.CommandSearchUpdate, SearchPattern: "xxx"})
	// Simulate hOffset changing during provisional search.
	ti.hOffset = 25
	ti.handleCommand(input.Command{Type: input.CommandSearchCancel})

	if ti.hOffset != 7 {
		t.Errorf("after cancel: hOffset = %d, want 7", ti.hOffset)
	}
}

func TestSearchCancel_PreservesConfirmedMatchesAfterCancel(t *testing.T) {
	// Verifies the snapshot/restore pipeline correctly preserves the
	// match-list backing array (deep copy avoids aliasing).
	ti := makeCancelTUI([]string{"foo bar foo", "foo baz"})

	ti.handleCommand(input.Command{Type: input.CommandSearchForward})
	ti.handleCommand(input.Command{Type: input.CommandSearchUpdate, SearchPattern: "foo"})
	ti.handleCommand(input.Command{Type: input.CommandSearchConfirm, SearchPattern: "foo"})

	confirmed := make([]searchMatch, len(ti.searchMatches))
	copy(confirmed, ti.searchMatches)

	// Provisional search mutates the live slice.
	ti.handleCommand(input.Command{Type: input.CommandSearchForward})
	ti.handleCommand(input.Command{Type: input.CommandSearchUpdate, SearchPattern: "baz"})
	ti.handleCommand(input.Command{Type: input.CommandSearchCancel})

	if len(ti.searchMatches) != len(confirmed) {
		t.Fatalf("after cancel: %d matches, want %d", len(ti.searchMatches), len(confirmed))
	}
	for i := range confirmed {
		if ti.searchMatches[i] != confirmed[i] {
			t.Errorf("match %d: got %+v, want %+v", i, ti.searchMatches[i], confirmed[i])
		}
	}
}

func TestSearchCancel_NAfterCancelJumpsConfirmedPattern(t *testing.T) {
	// End-to-end: after canceling a provisional search, n must navigate
	// the originally confirmed pattern's matches, not the provisional one.
	ti := makeCancelTUI([]string{"foo line", "bar line", "foo line", "bar line"})

	ti.handleCommand(input.Command{Type: input.CommandSearchForward})
	ti.handleCommand(input.Command{Type: input.CommandSearchUpdate, SearchPattern: "foo"})
	ti.handleCommand(input.Command{Type: input.CommandSearchConfirm, SearchPattern: "foo"})
	confirmedMatches := append([]searchMatch(nil), ti.searchMatches...)

	// Provisional pattern, then cancel.
	ti.handleCommand(input.Command{Type: input.CommandSearchForward})
	ti.handleCommand(input.Command{Type: input.CommandSearchUpdate, SearchPattern: "bar"})
	ti.handleCommand(input.Command{Type: input.CommandSearchCancel})

	// After cancel, searchMatches must equal the confirmed foo matches.
	if len(ti.searchMatches) != len(confirmedMatches) {
		t.Fatalf("post-cancel: %d matches, want %d (foo's matches)", len(ti.searchMatches), len(confirmedMatches))
	}
	for i := range confirmedMatches {
		if ti.searchMatches[i] != confirmedMatches[i] {
			t.Fatalf("post-cancel match %d: got %+v, want %+v", i, ti.searchMatches[i], confirmedMatches[i])
		}
	}
	// n moves to the next foo match, which must be a line containing "foo",
	// not a line containing "bar".
	ti.handleCommand(input.Command{Type: input.CommandSearchNext})
	if ti.doc.Line(ti.cursorLine) != "foo line" {
		t.Errorf("n jumped to line %d (%q); expected a foo line", ti.cursorLine, ti.doc.Line(ti.cursorLine))
	}
}

func TestSearchCancel_VisualModeSelectionRestored(t *testing.T) {
	// In visual mode, provisional incremental search moves the cursor via
	// jumpToMatch → modeMachine.OnCursorMoved, which extends the selection.
	// After cancel, the selection must collapse back to the original cursor.
	ti := makeCancelTUI([]string{"alpha beta gamma", "delta epsilon foo zeta"})
	ti.cursorLine = 0
	ti.cursorCol = 0

	// Enter visual-char mode at line 0, col 0.
	ti.modeMachine.Handle(vmode.EventToggleVisualChar, selection.Pos{Line: 0, Col: 0})
	startRegion := ti.modeMachine.Region()

	// Provisional /foo — without the cancel-time OnCursorMoved, the region
	// end would extend to line 1 (where foo lives).
	ti.handleCommand(input.Command{Type: input.CommandSearchForward})
	ti.handleCommand(input.Command{Type: input.CommandSearchUpdate, SearchPattern: "foo"})

	regionDuring := ti.modeMachine.Region()
	if regionDuring.End == startRegion.End {
		t.Skip("provisional search did not move cursor, so this test cannot exercise the cancel-restore path")
	}

	ti.handleCommand(input.Command{Type: input.CommandSearchCancel})

	regionAfter := ti.modeMachine.Region()
	if regionAfter.End.Line != 0 || regionAfter.End.Col != 0 {
		t.Errorf("after cancel: region.End = %+v, want {Line:0 Col:0} (collapsed to anchor)", regionAfter.End)
	}
}

func TestSearchCancel_SnapshotClearedAfterConfirmAndCancel(t *testing.T) {
	// After SearchConfirm or SearchCancel the snapshot is no longer needed.
	// Both code paths should release the saved-match backing storage and
	// reset snapshot bookkeeping fields.
	ti := makeCancelTUI([]string{"foo line", "bar"})

	// Confirm path.
	ti.handleCommand(input.Command{Type: input.CommandSearchForward})
	ti.handleCommand(input.Command{Type: input.CommandSearchUpdate, SearchPattern: "foo"})
	ti.handleCommand(input.Command{Type: input.CommandSearchConfirm, SearchPattern: "foo"})
	if ti.searchSavedMatches != nil || ti.searchSavedPattern != "" || ti.searchSavedRegex != nil {
		t.Errorf("after confirm: snapshot not cleared: matches=%v pattern=%q regex=%v",
			ti.searchSavedMatches, ti.searchSavedPattern, ti.searchSavedRegex)
	}

	// Cancel path.
	ti.handleCommand(input.Command{Type: input.CommandSearchForward})
	ti.handleCommand(input.Command{Type: input.CommandSearchUpdate, SearchPattern: "xxx"})
	ti.handleCommand(input.Command{Type: input.CommandSearchCancel})
	if ti.searchSavedMatches != nil || ti.searchSavedPattern != "" || ti.searchSavedRegex != nil {
		t.Errorf("after cancel: snapshot not cleared: matches=%v pattern=%q regex=%v",
			ti.searchSavedMatches, ti.searchSavedPattern, ti.searchSavedRegex)
	}
}

func TestSearchCancel_ClearSearchDefensivelyClearsSnapshot(t *testing.T) {
	// CommandClearSearch is the explicit "drop everything search-related"
	// path. It should defensively reset snapshot fields even if a prior
	// command already did (idempotent).
	ti := makeCancelTUI([]string{"foo line", "bar"})

	// Manually populate snapshot to simulate residue.
	ti.searchSavedPattern = "leftover"
	ti.searchSavedActive = true
	ti.searchSavedMatches = []searchMatch{{Line: 0, ColStart: 0, ColEnd: 0}}

	ti.handleCommand(input.Command{Type: input.CommandClearSearch})

	if ti.searchSavedActive || ti.searchSavedPattern != "" || ti.searchSavedMatches != nil {
		t.Errorf("after ClearSearch: snapshot residue not cleared: active=%v pattern=%q matches=%v",
			ti.searchSavedActive, ti.searchSavedPattern, ti.searchSavedMatches)
	}
}
