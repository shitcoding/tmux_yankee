package ui

import (
	"strings"
	"testing"

	"github.com/shitcoding/tmux_yankee/internal/config"
)

// TestStripANSI_EscapeHygiene drives stripANSI directly to prove that all
// escape forms are stripped from the plain-text path that search/motion/
// yank consume.
func TestStripANSI_EscapeHygiene(t *testing.T) {
	st := "\x1b\\"
	bel := "\x07"

	cases := []struct {
		name string
		in   string
		want string
	}{
		{"plain passthrough", "hello world", "hello world"},
		{"SGR colors stripped", "\x1b[31mred\x1b[0m", "red"},
		{"CSI ~ terminator", "\x1b[1~X", "X"},
		{"bracketed paste markers", "\x1b[200~text\x1b[201~Y", "textY"},
		{"OSC title + BEL", "\x1b]0;evil-title" + bel + "hello", "hello"},
		{"OSC8 hyperlink", "\x1b]8;;https://x" + st + "link" + "\x1b]8;;" + st + "after", "linkafter"},
		{"DCS payload", "\x1bPq#data" + st + "X", "X"},
		{"APC payload", "\x1b_APC stuff" + st + "Y", "Y"},
		{"PM payload", "\x1b^PM stuff" + st + "Z", "Z"},
		{"SOS payload", "\x1bXSOS stuff" + st + "W", "W"},
		{"SS3 + ESC bypass attempt", "\x1bO\x1b[31mY", "Y"},
		{"charset designation", "\x1b(BX", "X"},
		{"DEC alignment", "\x1b#8X", "X"},
		{"unterminated OSC drops EOL", "\x1b]0;" + strings.Repeat("Q", 2000) + "X", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := stripANSI(tc.in)
			if got != tc.want {
				t.Errorf("stripANSI(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

// TestDocument_PlainSanitizedForYankSearch proves the end-to-end invariant:
// captured raw bytes go through Document.Plain, which is what search and
// yank read. After the scanner change, every supported escape form is
// scrubbed so the plain text is safe to send to a downstream shell or
// terminal.
func TestDocument_PlainSanitizedForYankSearch(t *testing.T) {
	bel := "\x07"
	// hello + OSC (stripped) + world + SGR red ! reset → "helloworld!"
	raw := "hello\x1b]0;evil-title" + bel + "world\x1b[31m!\x1b[0m"
	doc := NewDocument([]string{raw})
	if got := doc.Line(0); got != "helloworld!" {
		t.Errorf("doc.Line(0) = %q, want %q", got, "helloworld!")
	}
}

// TestParseANSILine_EscapeHygiene proves the render path also handles
// non-CSI escapes correctly: cells emitted by the parser contain only
// printable runes, never raw ESC introducers or escape-payload bytes.
func TestParseANSILine_EscapeHygiene(t *testing.T) {
	bel := "\x07"
	st := "\x1b\\"

	cases := []struct {
		name string
		in   string
		want string // expected concatenated runes from emitted cells
	}{
		{"plain", "abc", "abc"},
		{"SGR around text", "\x1b[31mA\x1b[0mB", "AB"},
		{"OSC title swallowed", "\x1b]0;title" + bel + "after", "after"},
		{"OSC8 hyperlink swallowed, text kept", "\x1b]8;;https://x" + st + "link" + "\x1b]8;;" + st + "after", "linkafter"},
		{"DCS payload swallowed", "\x1bP1$qstuff" + st + "Z", "Z"},
		{"SS3 + ESC bypass guard", "\x1bO\x1b[31mY", "Y"},
		{"bracketed paste markers", "\x1b[200~text\x1b[201~Y", "textY"},
		{"charset designation", "\x1b(BX", "X"},
		{"DEC alignment test", "\x1b#8A", "A"},
		{"intermediate UTF-8 mode", "\x1b%GA", "A"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cells := ParseANSILine(tc.in)
			var b strings.Builder
			for _, c := range cells {
				b.WriteRune(c.Rune)
			}
			got := b.String()
			if got != tc.want {
				t.Errorf("ParseANSILine(%q) cells = %q, want %q", tc.in, got, tc.want)
			}
			// Belt and braces: no raw ESC should be present in any cell.
			for _, c := range cells {
				if c.Rune == '\x1b' {
					t.Errorf("ParseANSILine(%q) emitted a raw ESC cell", tc.in)
					break
				}
			}
		})
	}
}

// TestYankLine_DeliversSanitizedTextToClipboard exercises the yank path
// end-to-end: a captured line carrying an OSC title and SGR styling is
// stripped down to its plain text before reaching the clipboard. Without
// the shared escape scanner, OSC payload bytes would have leaked into the
// destination shell on paste.
func TestYankLine_DeliversSanitizedTextToClipboard(t *testing.T) {
	bel := "\x07"
	rawLine := "hello\x1b]0;evil-title" + bel + "world\x1b[31m!\x1b[0m"

	var captured string
	ti := &TUI{
		doc: NewDocument([]string{rawLine}),
		cfg: config.Settings{
			CopyTarget: config.CopyTargetClipboard,
			ExitOnYank: false,
		},
		clipboardFunc: func(s string) error {
			captured = s
			return nil
		},
	}

	ti.yankLine()

	if want := "helloworld!"; captured != want {
		t.Errorf("clipboard received %q; want %q", captured, want)
	}
	if strings.ContainsRune(captured, '\x1b') {
		t.Errorf("clipboard contains raw ESC: %q", captured)
	}
	if strings.ContainsRune(captured, '\x07') {
		t.Errorf("clipboard contains raw BEL: %q", captured)
	}
}

// TestParseANSILine_SGRStillApplied confirms the inline-CSI refactor didn't
// regress SGR style application: a CSI [31m sequence still colors the
// following cell red.
func TestParseANSILine_SGRStillApplied(t *testing.T) {
	cells := ParseANSILine("\x1b[31mA\x1b[0mB")
	if len(cells) != 2 {
		t.Fatalf("expected 2 cells; got %d", len(cells))
	}
	if cells[0].Rune != 'A' || cells[0].Style.FgColor != 31 {
		t.Errorf("cells[0]: got rune=%q fg=%d, want rune='A' fg=31", cells[0].Rune, cells[0].Style.FgColor)
	}
	if cells[1].Rune != 'B' || cells[1].Style.FgColor != 0 {
		t.Errorf("cells[1]: got rune=%q fg=%d, want rune='B' fg=0", cells[1].Rune, cells[1].Style.FgColor)
	}
}
