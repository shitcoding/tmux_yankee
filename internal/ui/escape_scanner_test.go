package ui

import (
	"strings"
	"testing"
)

// scanFixture is one fixture for a table-driven scanner test. raw is the
// input string; afterEscape is what the scanner should produce when applied
// repeatedly across the line (escape bytes dropped, everything else kept).
type scanFixture struct {
	name        string
	raw         string
	afterEscape string
}

func driveScanner(raw string) string {
	runes := []rune(raw)
	var b strings.Builder
	b.Grow(len(raw))
	i := 0
	for i < len(runes) {
		if runes[i] == '\x1b' {
			i = scanEscape(runes, i)
			continue
		}
		b.WriteRune(runes[i])
		i++
	}
	return b.String()
}

func TestScanEscape_Fixtures(t *testing.T) {
	st := "\x1b\\" // ST (String Terminator)
	bel := "\x07"
	esc := "\x1b"
	cases := []scanFixture{
		{"plain", "plain", "plain"},
		{"lone ESC at EOL", esc, ""},
		{"trailing lone ESC", "a" + esc, "a"},

		{"CSI SGR", "\x1b[31mred\x1b[0m", "red"},
		{"CSI with ~ terminator", "\x1b[1~X", "X"},
		{"CSI bracketed paste markers", "\x1b[200~text\x1b[201~Y", "textY"},
		{"CSI private mode set", "\x1b[?1049hX", "X"},
		{"CSI window manipulation t", "\x1b[2;3;4t Y", " Y"},
		{"unterminated CSI", "\x1b[" + strings.Repeat("1", 2000), ""},

		{"OSC + BEL", "\x1b]0;title" + bel + "after", "after"},
		{"OSC8 hyperlink", "\x1b]8;;https://x" + st + "hi\x1b]8;;" + st + "rest", "hirest"},
		{"DCS + ST", "\x1bPDCS" + st + "after", "after"},
		{"APC + ST", "\x1b_APC" + st + "X", "X"},
		{"PM + ST", "\x1b^PM" + st + "X", "X"},
		{"SOS + ST", "\x1bXSOS" + st + "X", "X"},
		{"unterminated OSC", "\x1b]0;" + strings.Repeat("Q", 2000) + "X", ""},

		{"SS2 + char", "\x1bNaX", "X"},
		{"SS3 + char", "\x1bObY", "Y"},
		{"SS2 + ESC must NOT swallow the next escape", "\x1bN\x1b[7mhostile", "hostile"},
		{"SS3 + ESC must NOT swallow the next escape", "\x1bO\x1b[31mY", "Y"},
		{"SS2 at EOL with no char", "\x1bN", ""},
		{"SS3 at EOL with no char", "\x1bO", ""},

		{"charset designation '('", "\x1b(0X", "X"},
		{"charset designation ')'", "\x1b)0Y", "Y"},
		{"charset designation '*'", "\x1b*BZ", "Z"},
		{"charset designation '+'", "\x1b+BW", "W"},
		{"charset designation '-' (96-char G1)", "\x1b-AX", "X"},
		{"charset designation '.' (96-char G2)", "\x1b.AY", "Y"},
		{"charset designation '/' (96-char G3)", "\x1b/AZ", "Z"},
		{"charset intro with no selector", "\x1b(", ""},

		{"DEC screen alignment test", "\x1b#8X", "X"},
		{"UTF-8 mode switch %G", "\x1b%GX", "X"},
		{"UTF-8 mode switch %@", "\x1b%@Y", "Y"},

		{"Fp 2-byte: save cursor", "\x1b7X", "X"},
		{"Fp 2-byte: keypad mode", "\x1b=X", "X"},
		{"Fs 2-byte: reset", "\x1bcX", "X"},
		{"Fe 2-byte: reverse index", "\x1bMX", "X"},

		{"sequential escapes around text", "A\x1b[31mB\x1b]0;t\x07C\x1b(BD", "ABCD"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := driveScanner(tc.raw)
			if got != tc.afterEscape {
				t.Errorf("driveScanner(%q) = %q, want %q", tc.raw, got, tc.afterEscape)
			}
		})
	}
}

func TestScanCSI_FinalByteRange(t *testing.T) {
	// Every byte in 0x40-0x7e is a valid CSI final. Spot-check a handful at
	// the boundaries so the helper doesn't accidentally narrow.
	finals := []rune{0x40, 0x41, 0x4d, 0x5a, 0x5b, 0x60, 0x7d, 0x7e}
	for _, f := range finals {
		raw := "\x1b[1" + string(f) + "X"
		got := driveScanner(raw)
		if got != "X" {
			t.Errorf("final=%#x: driveScanner(%q) = %q, want %q", f, raw, got, "X")
		}
	}
}

func TestScanStringControl_STAtBoundary(t *testing.T) {
	// Boundary check from Codex review: the ESC of ST sits at the LAST
	// position the bounded loop visits, and the trailing `\` is one past
	// the payload bound. The scanner must still recognize the terminator.
	//
	// `\x1b]` is the OSC introducer (2 runes); scanStringControl is called
	// with start=2. We then need payload of length stringControlMaxLen-1
	// so the ESC of ST lands at index 2+(maxLen-1) = maxLen+1, which is
	// end-1 — the last iteration. The trailing `\` is at maxLen+2 == end,
	// just past the payload bound but still within the line.
	payload := strings.Repeat("Q", stringControlMaxLen-1)
	raw := "\x1b]" + payload + "\x1b\\after"
	got := driveScanner(raw)
	if got != "after" {
		t.Errorf("ST at scan boundary: got %q, want %q", got, "after")
	}
}

func TestScanStringControl_BELAtBoundary(t *testing.T) {
	// BEL terminator at the very last position the bounded loop visits.
	payload := strings.Repeat("Q", stringControlMaxLen-1)
	raw := "\x1b]" + payload + "\x07after"
	got := driveScanner(raw)
	if got != "after" {
		t.Errorf("BEL at scan boundary: got %q, want %q", got, "after")
	}
}

func TestScanStringControl_OverflowDropsToEOL(t *testing.T) {
	// Payload exceeds stringControlMaxLen with NO terminator → drop to EOL.
	// The "X" after the bound must NOT survive (it sits inside the still-
	// unterminated OSC payload from the scanner's perspective).
	raw := "\x1b]0;" + strings.Repeat("Q", stringControlMaxLen+200) + "X"
	got := driveScanner(raw)
	if got != "" {
		t.Errorf("overflow without terminator should drop to EOL; got %q", got)
	}
}
