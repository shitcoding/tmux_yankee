package config

import (
	"strings"
	"testing"
)

// defaultOpts returns a CLIOptions with all required fields set to their defaults.
func defaultOpts() CLIOptions {
	return CLIOptions{
		PaneID:          "%1",
		Mode:            DefaultMode,
		ScrollbackLines: DefaultScrollbackLines,
		Theme:           DefaultTheme,
		ToggleModeKey:   DefaultToggleModeKey,
		WrapKey:         DefaultWrapKey,
		CopyTarget:      DefaultCopyTarget,
		ExitOnYank:      DefaultExitOnYank,
		StartPosition:   DefaultStartPosition,
		StatusBar:       DefaultStatusBar,
	}
}

func TestResolve_Defaults(t *testing.T) {
	opts := defaultOpts()
	cfg, err := Resolve(opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.PaneID != "%1" {
		t.Errorf("PaneID: got %q, want %q", cfg.PaneID, "%1")
	}
	if cfg.Mode != LineNumberModeHybrid {
		t.Errorf("Mode: got %q, want %q", cfg.Mode, LineNumberModeHybrid)
	}
	if cfg.ScrollbackLines != DefaultScrollbackLines {
		t.Errorf("ScrollbackLines: got %d, want %d", cfg.ScrollbackLines, DefaultScrollbackLines)
	}
	if cfg.ToggleModeKey != 'L' {
		t.Errorf("ToggleModeKey: got %q, want 'L'", cfg.ToggleModeKey)
	}
	if cfg.CopyTarget != CopyTargetBoth {
		t.Errorf("CopyTarget: got %q, want %q", cfg.CopyTarget, CopyTargetBoth)
	}
	if cfg.ExitOnYank != true {
		t.Errorf("ExitOnYank: got %v, want true", cfg.ExitOnYank)
	}
	if cfg.StartPosition != StartPositionBottom {
		t.Errorf("StartPosition: got %q, want %q", cfg.StartPosition, StartPositionBottom)
	}
}

func TestResolve_InvalidMode(t *testing.T) {
	opts := defaultOpts()
	opts.Mode = "foobar"
	_, err := Resolve(opts)
	if err == nil {
		t.Fatal("expected error for invalid mode")
	}
	if !strings.Contains(err.Error(), "foobar") {
		t.Errorf("error should mention bad value 'foobar', got: %v", err)
	}
}

func TestResolve_InvalidHexColor(t *testing.T) {
	opts := defaultOpts()
	opts.CursorFG = "red"
	_, err := Resolve(opts)
	if err == nil {
		t.Fatal("expected error for invalid hex color")
	}
	if !strings.Contains(err.Error(), "red") {
		t.Errorf("error should mention bad value 'red', got: %v", err)
	}
}

func TestResolve_MultiCharToggleKey(t *testing.T) {
	opts := defaultOpts()
	opts.ToggleModeKey = "LL"
	_, err := Resolve(opts)
	if err == nil {
		t.Fatal("expected error for multi-char toggle key")
	}
	if !strings.Contains(err.Error(), "LL") {
		t.Errorf("error should mention bad value 'LL', got: %v", err)
	}
}

func TestResolve_ScrollbackClamp(t *testing.T) {
	// Below minimum should clamp to MinScrollbackLines
	opts := defaultOpts()
	opts.ScrollbackLines = 50
	cfg, err := Resolve(opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.ScrollbackLines != MinScrollbackLines {
		t.Errorf("ScrollbackLines: got %d, want %d (clamped from 50)", cfg.ScrollbackLines, MinScrollbackLines)
	}

	// Above maximum should clamp to MaxScrollbackLines
	opts.ScrollbackLines = 300000
	cfg, err = Resolve(opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.ScrollbackLines != MaxScrollbackLines {
		t.Errorf("ScrollbackLines: got %d, want %d (clamped from 300000)", cfg.ScrollbackLines, MaxScrollbackLines)
	}
}

func TestResolve_ScrollbackValid(t *testing.T) {
	opts := defaultOpts()
	opts.ScrollbackLines = 5000
	cfg, err := Resolve(opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.ScrollbackLines != 5000 {
		t.Errorf("ScrollbackLines: got %d, want 5000", cfg.ScrollbackLines)
	}
}

func TestResolve_CopyTargetTmux(t *testing.T) {
	opts := defaultOpts()
	opts.CopyTarget = "tmux"
	cfg, err := Resolve(opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.CopyTarget != CopyTargetTmux {
		t.Errorf("CopyTarget: got %q, want %q", cfg.CopyTarget, CopyTargetTmux)
	}
}

func TestResolve_ExitOnYankOff(t *testing.T) {
	opts := defaultOpts()
	opts.ExitOnYank = "off"
	cfg, err := Resolve(opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.ExitOnYank {
		t.Errorf("ExitOnYank: got true, want false")
	}
}

func TestResolve_StartPositionTop(t *testing.T) {
	opts := defaultOpts()
	opts.StartPosition = "top"
	cfg, err := Resolve(opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.StartPosition != StartPositionTop {
		t.Errorf("StartPosition: got %q, want %q", cfg.StartPosition, StartPositionTop)
	}
}
