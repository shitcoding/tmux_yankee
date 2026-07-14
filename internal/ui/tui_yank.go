package ui

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"

	"github.com/shitcoding/tmux_yankee/internal/config"
	vmode "github.com/shitcoding/tmux_yankee/internal/mode"
	"github.com/shitcoding/tmux_yankee/internal/selection"
)

// Yank / copy dispatch: sends the current selection to the tmux buffer and/or
// system clipboard. Extracted from tui.go (see internal/ui/tui_yank_test.go).

// dispatchCopy sends text to tmux buffer and/or system clipboard based on CopyTarget config.
func (t *TUI) dispatchCopy(text, caller string) {
	clipboardCopy := func(s string) error {
		if t.clipboardFunc != nil {
			return t.clipboardFunc(s)
		}
		return t.copyToClipboard(s)
	}
	tmuxCopy := func(s string) {
		if t.client == nil {
			return // no tmux client (e.g. demo mode) — skip the paste-buffer write
		}
		if err := t.client.SetBuffer(s); err != nil {
			fmt.Fprintf(os.Stderr, "%s: SetBuffer failed: %v\n", caller, err)
		}
	}

	switch t.cfg.CopyTarget {
	case config.CopyTargetTmux:
		tmuxCopy(text)
	case config.CopyTargetClipboard:
		if err := clipboardCopy(text); err != nil {
			fmt.Fprintf(os.Stderr, "%s: copyToClipboard failed: %v\n", caller, err)
		}
	default: // CopyTargetBoth or unset
		tmuxCopy(text)
		if err := clipboardCopy(text); err != nil {
			fmt.Fprintf(os.Stderr, "%s: copyToClipboard failed: %v\n", caller, err)
		}
	}
}

// yank extracts selected text, copies to clipboard and tmux buffer
// Returns true to quit TUI after yank
func (t *TUI) yank() bool {
	// Get current selection region from mode machine
	region := t.modeMachine.Region()

	// Only yank if there is an active selection
	if region.Kind == selection.KindNone {
		return false
	}

	// Extract selected text lazily (only accesses lines within the selection region).
	text, err := selection.ExtractRegionFromProvider(t.doc, region)
	if err != nil {
		fmt.Fprintf(os.Stderr, "yank: ExtractRegion failed: %v\n", err)
		return false
	}

	t.dispatchCopy(text, "yank")

	// Exit visual mode and return to Normal mode (vim behavior)
	pos := selection.Pos{Line: t.cursorLine, Col: t.cursorCol}
	t.modeMachine.Handle(vmode.EventEscape, pos)
	t.syncKeymapToMode()

	// ExitOnYank=true (default): exit TUI after yank
	// ExitOnYank=false: stay in TUI in Normal mode (selection already cleared above)
	if !t.cfg.ExitOnYank {
		return false
	}
	return true
}

// yankLine yanks the full content of the current cursor line (yy binding).
// Unlike yank(), it does not require an active visual selection.
func (t *TUI) yankLine() bool {
	if t.doc.LineCount() == 0 {
		return false
	}
	text := t.doc.Line(t.cursorLine)

	t.dispatchCopy(text, "yankLine")

	if !t.cfg.ExitOnYank {
		return false
	}
	return true
}

// copyToClipboard copies text to system clipboard via copy_stdin.sh.
// Resolution lives in copy_path.go and excludes any CWD-relative path.
func (t *TUI) copyToClipboard(text string) error {
	execPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("cannot determine executable path: %w", err)
	}

	scriptPath, err := resolveCopyScriptPath(execPath, statExists)
	if err != nil {
		return err
	}

	cmd := exec.Command(scriptPath)
	cmd.Stdin = bytes.NewBufferString(text)
	cmd.Stderr = os.Stderr // Show errors from script

	return cmd.Run()
}
