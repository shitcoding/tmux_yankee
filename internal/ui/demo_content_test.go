package ui

import (
	"strings"
	"testing"
)

func TestDemoPages_NonEmpty(t *testing.T) {
	pages := DemoPages()
	if len(pages) != 4 {
		t.Fatalf("DemoPages(): got %d pages, want 4", len(pages))
	}
	for i, page := range pages {
		if len(page) == 0 {
			t.Errorf("DemoPages()[%d]: empty page", i)
		}
	}
}

func TestDemoPageNames_MatchPages(t *testing.T) {
	if len(DemoPageNames) != len(DemoPages()) {
		t.Errorf("DemoPageNames length %d != DemoPages length %d", len(DemoPageNames), len(DemoPages()))
	}
}

func TestDemoShellSession_HasANSI(t *testing.T) {
	lines := DemoShellSession()
	found := false
	for _, line := range lines {
		if strings.Contains(line, "\x1b[") {
			found = true
			break
		}
	}
	if !found {
		t.Error("DemoShellSession: expected ANSI escape codes in output")
	}
}

func TestDemoCodeSnippet_HasANSI(t *testing.T) {
	lines := DemoCodeSnippet()
	found := false
	for _, line := range lines {
		if strings.Contains(line, "\x1b[") {
			found = true
			break
		}
	}
	if !found {
		t.Error("DemoCodeSnippet: expected ANSI escape codes in output")
	}
}

func TestDemoColorTestCard_HasANSI(t *testing.T) {
	lines := DemoColorTestCard()
	found := false
	for _, line := range lines {
		if strings.Contains(line, "\x1b[") {
			found = true
			break
		}
	}
	if !found {
		t.Error("DemoColorTestCard: expected ANSI escape codes in output")
	}
}

func TestDemoLongLines_HasLongLines(t *testing.T) {
	lines := DemoLongLines()
	foundLong := false
	for _, line := range lines {
		if len(line) > 150 {
			foundLong = true
			break
		}
	}
	if !foundLong {
		t.Error("DemoLongLines: expected at least one line > 150 bytes")
	}
}
