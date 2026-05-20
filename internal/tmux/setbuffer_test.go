package tmux

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// installFakeTmux writes a fake tmux script at <dir>/tmux that records its
// argv to <logDir>/argv (newline-separated) and its stdin to <logDir>/stdin.
// Returns the dir that should be prepended to PATH.
func installFakeTmux(t *testing.T, logDir string) string {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("fake-tmux uses /bin/sh shebang; skip on Windows")
	}
	binDir := t.TempDir()
	script := `#!/bin/sh
# Record argv (one per line, newline-terminated).
for a in "$@"; do
    printf '%s\n' "$a"
done > "` + logDir + `/argv"
# Record stdin bytes verbatim.
cat > "` + logDir + `/stdin"
`
	path := filepath.Join(binDir, "tmux")
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatalf("write fake tmux: %v", err)
	}
	return binDir
}

func withPath(t *testing.T, prepend string) {
	t.Helper()
	orig := os.Getenv("PATH")
	if err := os.Setenv("PATH", prepend+string(os.PathListSeparator)+orig); err != nil {
		t.Fatalf("set PATH: %v", err)
	}
	t.Cleanup(func() { os.Setenv("PATH", orig) })
}

func readFile(t *testing.T, p string) string {
	t.Helper()
	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatalf("read %s: %v", p, err)
	}
	return string(b)
}

func TestSetBuffer_UsesLoadBufferStdin(t *testing.T) {
	logDir := t.TempDir()
	binDir := installFakeTmux(t, logDir)
	withPath(t, binDir)

	c := NewClient(context.Background())
	if err := c.SetBuffer("hello world"); err != nil {
		t.Fatalf("SetBuffer: %v", err)
	}

	argv := strings.TrimRight(readFile(t, filepath.Join(logDir, "argv")), "\n")
	gotArgs := strings.Split(argv, "\n")
	wantArgs := []string{"load-buffer", "-"}
	if !sliceEqual(gotArgs, wantArgs) {
		t.Errorf("argv: got %v, want %v", gotArgs, wantArgs)
	}

	stdin := readFile(t, filepath.Join(logDir, "stdin"))
	if stdin != "hello world" {
		t.Errorf("stdin: got %q, want %q", stdin, "hello world")
	}
}

func TestSetBuffer_LargePayload(t *testing.T) {
	logDir := t.TempDir()
	binDir := installFakeTmux(t, logDir)
	withPath(t, binDir)

	payload := strings.Repeat("A", 1<<20) // 1 MiB
	c := NewClient(context.Background())
	if err := c.SetBuffer(payload); err != nil {
		t.Fatalf("SetBuffer: %v", err)
	}

	stdin := readFile(t, filepath.Join(logDir, "stdin"))
	if len(stdin) != len(payload) {
		t.Fatalf("stdin size: got %d, want %d", len(stdin), len(payload))
	}
	if stdin != payload {
		t.Errorf("stdin contents differ from payload")
	}
}

func TestSetBuffer_EmptyPayload(t *testing.T) {
	logDir := t.TempDir()
	binDir := installFakeTmux(t, logDir)
	withPath(t, binDir)

	c := NewClient(context.Background())
	if err := c.SetBuffer(""); err != nil {
		t.Fatalf("SetBuffer: %v", err)
	}

	stdin := readFile(t, filepath.Join(logDir, "stdin"))
	if stdin != "" {
		t.Errorf("stdin: got %q, want empty", stdin)
	}
}

func TestSetBuffer_UTF8AndControlChars(t *testing.T) {
	logDir := t.TempDir()
	binDir := installFakeTmux(t, logDir)
	withPath(t, binDir)

	// Multi-byte UTF-8 + newline + tab. Skip NUL: argv cannot carry NUL but
	// stdin can. We don't include NUL here because the old argv-based impl
	// would have failed on it; we want a fixture both impls could in principle
	// transport, so the assertion is about *what bytes were delivered*.
	payload := "привет\tмир\nλ"
	c := NewClient(context.Background())
	if err := c.SetBuffer(payload); err != nil {
		t.Fatalf("SetBuffer: %v", err)
	}

	stdin := readFile(t, filepath.Join(logDir, "stdin"))
	if stdin != payload {
		t.Errorf("stdin: got %q, want %q", stdin, payload)
	}
}

func sliceEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
