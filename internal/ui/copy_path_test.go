package ui

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCopyScriptCandidates_NoCWDEntry(t *testing.T) {
	// Regression guard: the helper must never include a bare relative path
	// like "scripts/copy_stdin.sh" — that resolves against CWD and lets a
	// malicious working directory hijack yank.
	got := copyScriptCandidates("/opt/yankee/bin/tmux-yankee")
	for _, p := range got {
		if !filepath.IsAbs(p) {
			t.Errorf("candidate %q is not absolute; CWD-relative paths are forbidden", p)
		}
	}
	for _, p := range got {
		if strings.HasPrefix(p, "scripts/") {
			t.Errorf("candidate %q is a bare CWD-relative path", p)
		}
	}
}

func TestResolveCopyScriptPath_PrefersBinDirCandidate(t *testing.T) {
	tmp := t.TempDir()
	binDir := filepath.Join(tmp, "bin")
	scriptDir := filepath.Join(tmp, "scripts")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(scriptDir, 0o755); err != nil {
		t.Fatal(err)
	}
	wantScript := filepath.Join(scriptDir, "copy_stdin.sh")
	if err := os.WriteFile(wantScript, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	got, err := resolveCopyScriptPath(filepath.Join(binDir, "tmux-yankee"), statExists)
	if err != nil {
		t.Fatalf("resolveCopyScriptPath: %v", err)
	}
	wantAbs, _ := filepath.Abs(wantScript)
	if got != wantAbs {
		t.Errorf("got %q, want %q", got, wantAbs)
	}
}

func TestResolveCopyScriptPath_FallsBackToSystemPath(t *testing.T) {
	// statFn pretends only /usr/local/bin/copy_stdin.sh exists.
	stat := func(p string) error {
		if p == "/usr/local/bin/copy_stdin.sh" {
			return nil
		}
		return errors.New("not found")
	}
	got, err := resolveCopyScriptPath("/opt/yankee/bin/tmux-yankee", stat)
	if err != nil {
		t.Fatalf("resolveCopyScriptPath: %v", err)
	}
	if got != "/usr/local/bin/copy_stdin.sh" {
		t.Errorf("got %q, want /usr/local/bin/copy_stdin.sh", got)
	}
}

func TestResolveCopyScriptPath_NoCWDFallback(t *testing.T) {
	// Even with a copy_stdin.sh present at <CWD>/scripts/copy_stdin.sh, the
	// resolver must never QUERY any path under CWD. We assert that property
	// directly by recording the paths passed to statFn — independent of what
	// happens to exist on the system running this test.
	tmp := t.TempDir()
	scriptDir := filepath.Join(tmp, "scripts")
	if err := os.MkdirAll(scriptDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(scriptDir, "copy_stdin.sh"), []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	origWD, _ := os.Getwd()
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	t.Cleanup(func() { os.Chdir(origWD) })

	var queried []string
	stat := func(p string) error {
		queried = append(queried, p)
		return errors.New("not found")
	}

	_, err := resolveCopyScriptPath("/opt/yankee/bin/tmux-yankee", stat)
	if err == nil {
		t.Fatal("expected error when no candidates exist; got nil")
	}

	cwd, _ := os.Getwd()
	for _, p := range queried {
		if p == cwd || strings.HasPrefix(p, cwd+string(filepath.Separator)) {
			t.Errorf("resolver queried a path under CWD %q: %q", cwd, p)
		}
	}
}

func TestResolveCopyScriptPath_RelativeExecPath_NoCWDCandidate(t *testing.T) {
	// If a caller passes a relative execPath, the binDir-derived candidate
	// must be omitted entirely. Absolutizing via CWD would otherwise land
	// under <CWD>/scripts/copy_stdin.sh.
	tmp := t.TempDir()
	origWD, _ := os.Getwd()
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	t.Cleanup(func() { os.Chdir(origWD) })

	got := copyScriptCandidates("bin/tmux-yankee")
	cwd, _ := os.Getwd()
	for _, p := range got {
		if !filepath.IsAbs(p) {
			t.Errorf("candidate %q is not absolute", p)
		}
		if p == cwd || strings.HasPrefix(p, cwd+string(filepath.Separator)) {
			t.Errorf("candidate %q is under CWD %q", p, cwd)
		}
	}
}

func TestResolveCopyScriptPath_EmptyExecPath_NoCWDCandidate(t *testing.T) {
	// Empty execPath (filepath.Dir(\"\") == \".\") must not yield a CWD
	// candidate either.
	got := copyScriptCandidates("")
	for _, p := range got {
		if !filepath.IsAbs(p) {
			t.Errorf("candidate %q is not absolute", p)
		}
	}
}

func TestResolveCopyScriptPath_NoneFound(t *testing.T) {
	stat := func(string) error { return errors.New("not found") }
	_, err := resolveCopyScriptPath("/opt/yankee/bin/tmux-yankee", stat)
	if err == nil {
		t.Fatal("expected error when no candidates exist")
	}
	if !strings.Contains(err.Error(), "copy_stdin.sh") {
		t.Errorf("error should mention copy_stdin.sh: %q", err)
	}
}
