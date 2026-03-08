package flash

import (
	"testing"
)

func TestBuildOverlay(t *testing.T) {
	matches := []Match{
		{Line: 0, ColStart: 0, ColEnd: 3, Label: 'a'},
		{Line: 1, ColStart: 5, ColEnd: 8, Label: 'b'},
		{Line: 2, ColStart: 0, ColEnd: 3, Label: 0}, // no label
	}

	o := BuildOverlay(matches, "flash>", true)

	if o.Prompt != "flash>" {
		t.Errorf("prompt = %q, want %q", o.Prompt, "flash>")
	}
	if !o.Backdrop {
		t.Error("expected backdrop to be true")
	}
	if len(o.Matches) != 3 {
		t.Errorf("expected 3 match ranges, got %d", len(o.Matches))
	}
	if len(o.Labels) != 2 {
		t.Errorf("expected 2 labels (skipping label=0), got %d", len(o.Labels))
	}
}

func TestOverlay_HasLabel(t *testing.T) {
	matches := []Match{
		{Line: 0, ColStart: 5, ColEnd: 8, Label: 'x'},
		{Line: 2, ColStart: 0, ColEnd: 3, Label: 'y'},
	}

	o := BuildOverlay(matches, "", false)

	tests := []struct {
		name string
		line int
		col  int
		want byte
	}{
		{"label after match end", 0, 8, 'x'},
		{"another label after end", 2, 3, 'y'},
		{"no label at match start", 0, 5, 0},
		{"wrong line", 1, 8, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := o.HasLabel(tt.line, tt.col)
			if got != tt.want {
				t.Errorf("HasLabel(%d,%d) = '%c' (%d), want '%c' (%d)",
					tt.line, tt.col, got, got, tt.want, tt.want)
			}
		})
	}
}

func TestOverlay_InMatch(t *testing.T) {
	matches := []Match{
		{Line: 0, ColStart: 2, ColEnd: 5, Label: 'a'},
		{Line: 1, ColStart: 0, ColEnd: 3, Label: 'b'},
	}

	o := BuildOverlay(matches, "", false)

	tests := []struct {
		name string
		line int
		col  int
		want bool
	}{
		{"start of match", 0, 2, true},
		{"middle of match", 0, 3, true},
		{"end of match (exclusive)", 0, 5, false},
		{"before match", 0, 1, false},
		{"second match", 1, 1, true},
		{"outside all matches", 2, 0, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := o.InMatch(tt.line, tt.col)
			if got != tt.want {
				t.Errorf("InMatch(%d,%d) = %v, want %v",
					tt.line, tt.col, got, tt.want)
			}
		})
	}
}

func TestOverlay_NilSafety(t *testing.T) {
	var o *Overlay

	if got := o.HasLabel(0, 0); got != 0 {
		t.Errorf("nil.HasLabel = %d, want 0", got)
	}

	if got := o.InMatch(0, 0); got {
		t.Error("nil.InMatch = true, want false")
	}
}

func TestBuildOverlay_EmptyMatches(t *testing.T) {
	o := BuildOverlay(nil, "test", false)

	if len(o.Labels) != 0 {
		t.Errorf("expected 0 labels, got %d", len(o.Labels))
	}
	if len(o.Matches) != 0 {
		t.Errorf("expected 0 matches, got %d", len(o.Matches))
	}
}

func TestBuildOverlay_MatchRanges(t *testing.T) {
	matches := []Match{
		{Line: 3, ColStart: 10, ColEnd: 15, Label: 'z'},
	}

	o := BuildOverlay(matches, "", false)

	if len(o.Matches) != 1 {
		t.Fatalf("expected 1 match range, got %d", len(o.Matches))
	}

	r := o.Matches[0]
	if r.Line != 3 || r.ColStart != 10 || r.ColEnd != 15 {
		t.Errorf("range = {%d, %d, %d}, want {3, 10, 15}", r.Line, r.ColStart, r.ColEnd)
	}
}
