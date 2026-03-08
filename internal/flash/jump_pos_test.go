package flash

import "testing"

func TestParseJumpPos(t *testing.T) {
	tests := []struct {
		input    string
		fallback JumpPos
		want     JumpPos
	}{
		{"match_end", JumpPosMatchStart, JumpPosMatchEnd},
		{"match_start", JumpPosMatchEnd, JumpPosMatchStart},
		{"word_start", JumpPosMatchEnd, JumpPosWordStart},
		{"word_end", JumpPosMatchEnd, JumpPosWordEnd},
		{"off", JumpPosMatchEnd, JumpPosOff},
		{"", JumpPosMatchEnd, JumpPosMatchEnd},
		{"unknown", JumpPosWordStart, JumpPosWordStart},
		{"MATCH_END", JumpPosMatchStart, JumpPosMatchStart}, // case-sensitive
	}

	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			got := ParseJumpPos(tc.input, tc.fallback)
			if got != tc.want {
				t.Errorf("ParseJumpPos(%q, %v) = %v, want %v", tc.input, tc.fallback, got, tc.want)
			}
		})
	}
}

func TestJumpPosString(t *testing.T) {
	tests := []struct {
		pos  JumpPos
		want string
	}{
		{JumpPosMatchEnd, "match_end"},
		{JumpPosMatchStart, "match_start"},
		{JumpPosWordStart, "word_start"},
		{JumpPosWordEnd, "word_end"},
		{JumpPosOff, "off"},
	}

	for _, tc := range tests {
		t.Run(tc.want, func(t *testing.T) {
			got := tc.pos.String()
			if got != tc.want {
				t.Errorf("%v.String() = %q, want %q", tc.pos, got, tc.want)
			}
		})
	}
}

func TestJumpPosRoundTrip(t *testing.T) {
	positions := []JumpPos{
		JumpPosMatchEnd, JumpPosMatchStart, JumpPosWordStart, JumpPosWordEnd, JumpPosOff,
	}
	for _, pos := range positions {
		s := pos.String()
		got := ParseJumpPos(s, JumpPosOff)
		if got != pos {
			t.Errorf("round-trip failed: %v -> %q -> %v", pos, s, got)
		}
	}
}

func TestResolveJumpCol(t *testing.T) {
	// "the quick brown fox jumps"
	//  0123456789012345678901234
	//  t h e   q u i c k   b r o w n   f o x   j u m p s
	//  0 1 2 3 4 5 6 7 8 9 ...
	line := "the quick brown fox jumps"

	tests := []struct {
		name string
		line string
		m    Match
		pos  JumpPos
		want int
	}{
		// MatchEnd: last char of match (ColEnd-1)
		{
			name: "match_end for 'ow' in brown",
			line: line,
			m:    Match{Line: 0, ColStart: 12, ColEnd: 14}, // "ow" at cols 12-13
			pos:  JumpPosMatchEnd,
			want: 13,
		},
		// MatchStart: first char of match
		{
			name: "match_start for 'ow' in brown",
			line: line,
			m:    Match{Line: 0, ColStart: 12, ColEnd: 14},
			pos:  JumpPosMatchStart,
			want: 12,
		},
		// WordStart: start of word containing match start
		// "ow" starts at col 12, which is inside "brown" (cols 10-14)
		{
			name: "word_start for 'ow' in brown",
			line: line,
			m:    Match{Line: 0, ColStart: 12, ColEnd: 14},
			pos:  JumpPosWordStart,
			want: 10,
		},
		// WordEnd: end of word containing match end
		// "ow" ends at col 14 (exclusive), last char is col 13 ('w')
		// word "brown" ends at col 14 ('n'), so word_end should be 14
		{
			name: "word_end for 'ow' in brown",
			line: line,
			m:    Match{Line: 0, ColStart: 12, ColEnd: 14},
			pos:  JumpPosWordEnd,
			want: 14, // 'n' of "brown"
		},
		// Match at word boundary: "jumps" starts at col 20
		{
			name: "match_start for 'jumps'",
			line: line,
			m:    Match{Line: 0, ColStart: 20, ColEnd: 25},
			pos:  JumpPosMatchStart,
			want: 20,
		},
		{
			name: "match_end for 'jumps'",
			line: line,
			m:    Match{Line: 0, ColStart: 20, ColEnd: 25},
			pos:  JumpPosMatchEnd,
			want: 24,
		},
		{
			name: "word_start for 'jumps' (already at word start)",
			line: line,
			m:    Match{Line: 0, ColStart: 20, ColEnd: 25},
			pos:  JumpPosWordStart,
			want: 20,
		},
		{
			name: "word_end for 'jumps' (already at word end)",
			line: line,
			m:    Match{Line: 0, ColStart: 20, ColEnd: 25},
			pos:  JumpPosWordEnd,
			want: 24,
		},
		// Single-char match
		{
			name: "match_end single char",
			line: line,
			m:    Match{Line: 0, ColStart: 4, ColEnd: 5}, // 'q'
			pos:  JumpPosMatchEnd,
			want: 4,
		},
		{
			name: "word_start for single char 'q' in 'quick'",
			line: line,
			m:    Match{Line: 0, ColStart: 4, ColEnd: 5},
			pos:  JumpPosWordStart,
			want: 4, // 'q' is already the start of 'quick'
		},
		{
			name: "word_end for single char 'q' in 'quick'",
			line: line,
			m:    Match{Line: 0, ColStart: 4, ColEnd: 5},
			pos:  JumpPosWordEnd,
			want: 8, // end of 'quick' is col 8
		},
		// Match spanning word boundary with punctuation
		{
			name: "word_start with punctuation",
			line: "foo.bar baz",
			m:    Match{Line: 0, ColStart: 3, ColEnd: 5}, // ".b"
			pos:  JumpPosWordStart,
			want: 3, // '.' is punctuation, stands alone
		},
		{
			name: "word_end with punctuation in match",
			line: "foo.bar baz",
			m:    Match{Line: 0, ColStart: 3, ColEnd: 5}, // ".b"
			pos:  JumpPosWordEnd,
			want: 6, // 'r' end of "bar"
		},
		// Tab characters
		{
			name: "match after tab",
			line: "\thello world",
			m:    Match{Line: 0, ColStart: 1, ColEnd: 6}, // "hello"
			pos:  JumpPosWordStart,
			want: 1,
		},
		{
			name: "match_end after tab",
			line: "\thello world",
			m:    Match{Line: 0, ColStart: 1, ColEnd: 6}, // "hello"
			pos:  JumpPosMatchEnd,
			want: 5,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := ResolveJumpCol(tc.line, tc.m, tc.pos)
			if got != tc.want {
				t.Errorf("ResolveJumpCol(%q, {ColStart:%d, ColEnd:%d}, %v) = %d, want %d",
					tc.line, tc.m.ColStart, tc.m.ColEnd, tc.pos, got, tc.want)
			}
		})
	}
}

func TestResolveJumpCol_EdgeCases(t *testing.T) {
	tests := []struct {
		name string
		line string
		m    Match
		pos  JumpPos
		want int
	}{
		// Empty line
		{
			name: "empty line match_start",
			line: "",
			m:    Match{Line: 0, ColStart: 0, ColEnd: 0},
			pos:  JumpPosMatchStart,
			want: 0,
		},
		{
			name: "empty line match_end",
			line: "",
			m:    Match{Line: 0, ColStart: 0, ColEnd: 0},
			pos:  JumpPosMatchEnd,
			want: 0,
		},
		{
			name: "empty line word_start",
			line: "",
			m:    Match{Line: 0, ColStart: 0, ColEnd: 0},
			pos:  JumpPosWordStart,
			want: 0,
		},
		{
			name: "empty line word_end",
			line: "",
			m:    Match{Line: 0, ColStart: 0, ColEnd: 0},
			pos:  JumpPosWordEnd,
			want: 0,
		},
		// Out-of-range ColStart
		{
			name: "ColStart beyond line length word_start",
			line: "abc",
			m:    Match{Line: 0, ColStart: 10, ColEnd: 12},
			pos:  JumpPosWordStart,
			want: 10, // returns ColStart as-is
		},
		{
			name: "ColEnd beyond line length word_end",
			line: "abc",
			m:    Match{Line: 0, ColStart: 0, ColEnd: 10},
			pos:  JumpPosWordEnd,
			want: 2, // len(runes)-1
		},
		// Default/unknown JumpPos falls back to ColStart
		{
			name: "unknown JumpPos",
			line: "hello",
			m:    Match{Line: 0, ColStart: 2, ColEnd: 4},
			pos:  JumpPos(99),
			want: 2,
		},
		// Match at very start of line
		{
			name: "word_start at col 0",
			line: "hello",
			m:    Match{Line: 0, ColStart: 0, ColEnd: 3},
			pos:  JumpPosWordStart,
			want: 0,
		},
		// Match at very end of line
		{
			name: "word_end at end of line",
			line: "hello",
			m:    Match{Line: 0, ColStart: 3, ColEnd: 5},
			pos:  JumpPosWordEnd,
			want: 4, // last rune index
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := ResolveJumpCol(tc.line, tc.m, tc.pos)
			if got != tc.want {
				t.Errorf("ResolveJumpCol(%q, {ColStart:%d, ColEnd:%d}, %v) = %d, want %d",
					tc.line, tc.m.ColStart, tc.m.ColEnd, tc.pos, got, tc.want)
			}
		})
	}
}

func TestGetFlashCharType(t *testing.T) {
	tests := []struct {
		r    rune
		want flashCharType
	}{
		{'a', flashCharWord},
		{'Z', flashCharWord},
		{'0', flashCharWord},
		{'9', flashCharWord},
		{'_', flashCharWord},
		{' ', flashCharWhitespace},
		{'\t', flashCharWhitespace},
		{'\n', flashCharWhitespace},
		{'\r', flashCharWhitespace},
		{'.', flashCharPunctuation},
		{'-', flashCharPunctuation},
		{'(', flashCharPunctuation},
		{'!', flashCharPunctuation},
	}

	for _, tc := range tests {
		t.Run(string(tc.r), func(t *testing.T) {
			got := getFlashCharType(tc.r)
			if got != tc.want {
				t.Errorf("getFlashCharType(%q) = %d, want %d", tc.r, got, tc.want)
			}
		})
	}
}
