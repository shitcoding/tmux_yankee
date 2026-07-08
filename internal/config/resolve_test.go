package config

import (
	"strings"
	"testing"

	"github.com/shitcoding/tmux_yankee/internal/keymap"
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
		WrapMode:        DefaultWrapMode,
		Mouse:           DefaultMouse,
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

func TestResolve_ModeSpecificBindings(t *testing.T) {
	opts := defaultOpts()
	opts.Bindings = "H=first_nonblank"    // shared override
	opts.NormalBindings = "H=line_start"  // normal override wins
	opts.VisualBindings = "g-g=last_line" // visual prefix override

	cfg, err := Resolve(opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Normal mode: H should be line_start (normal override wins over shared)
	nm := cfg.ModeKeymap.ForMode(false)
	if a, ok := nm.Lookup(keymap.Key('H')); !ok || a != keymap.ActionLineStart {
		t.Errorf("normal H: got (%q, %v), want line_start", a, ok)
	}

	// Visual mode: H should be first_nonblank (shared override, no visual override for H)
	vm := cfg.ModeKeymap.ForMode(true)
	if a, ok := vm.Lookup(keymap.Key('H')); !ok || a != keymap.ActionFirstNonBlank {
		t.Errorf("visual H: got (%q, %v), want first_nonblank", a, ok)
	}

	// Visual mode: gg should be last_line (visual prefix override)
	if a, ok := vm.LookupPrefix('g', 'g'); !ok || a != keymap.ActionLastLine {
		t.Errorf("visual gg: got (%q, %v), want last_line", a, ok)
	}

	// Normal mode: gg should be first_line (default, no normal override)
	if a, ok := nm.LookupPrefix('g', 'g'); !ok || a != keymap.ActionFirstLine {
		t.Errorf("normal gg: got (%q, %v), want first_line", a, ok)
	}
}

func TestResolve_InvalidNormalBindings(t *testing.T) {
	opts := defaultOpts()
	opts.NormalBindings = "H=bogus_action"
	_, err := Resolve(opts)
	if err == nil {
		t.Fatal("expected error for invalid normal binding")
	}
	if !strings.Contains(err.Error(), "nbindings") {
		t.Errorf("error should mention nbindings: %v", err)
	}
}

func TestResolve_InvalidVisualBindings(t *testing.T) {
	opts := defaultOpts()
	opts.VisualBindings = "x=not_an_action"
	_, err := Resolve(opts)
	if err == nil {
		t.Fatal("expected error for invalid visual binding")
	}
	if !strings.Contains(err.Error(), "vbindings") {
		t.Errorf("error should mention vbindings: %v", err)
	}
}

func TestResolve_InvalidWrapMode(t *testing.T) {
	opts := defaultOpts()
	opts.WrapMode = "auto"
	_, err := Resolve(opts)
	if err == nil {
		t.Fatal("expected error for invalid wrap-mode")
	}
	if !strings.Contains(err.Error(), "wrap-mode") {
		t.Errorf("error should mention wrap-mode: %v", err)
	}
}

func TestResolve_InvalidFlashMinChars(t *testing.T) {
	opts := defaultOpts()
	opts.FlashMinChars = "abc"
	_, err := Resolve(opts)
	if err == nil {
		t.Fatal("expected error for invalid flash-min-chars")
	}
	if !strings.Contains(err.Error(), "flash-min-chars") {
		t.Errorf("error should mention flash-min-chars: %v", err)
	}
}

func TestResolve_InvalidFlashJumpPos(t *testing.T) {
	opts := defaultOpts()
	opts.FlashJumpPos = "bogus"
	_, err := Resolve(opts)
	if err == nil {
		t.Fatal("expected error for invalid flash-jump-pos")
	}
	if !strings.Contains(err.Error(), "flash-jump-pos") {
		t.Errorf("error should mention flash-jump-pos: %v", err)
	}
}

func TestResolve_InvalidFlashAltJumpPos(t *testing.T) {
	opts := defaultOpts()
	opts.FlashAltJumpPos = "bogus"
	_, err := Resolve(opts)
	if err == nil {
		t.Fatal("expected error for invalid flash-alt-jump-pos")
	}
	if !strings.Contains(err.Error(), "flash-alt-jump-pos") {
		t.Errorf("error should mention flash-alt-jump-pos: %v", err)
	}
}

func TestResolve_InvalidFlashOnOff(t *testing.T) {
	opts := defaultOpts()
	opts.Flash = "onn"
	_, err := Resolve(opts)
	if err == nil {
		t.Fatal("expected error for invalid flash value")
	}
	if !strings.Contains(err.Error(), "flash") || !strings.Contains(err.Error(), "onn") {
		t.Errorf("error should mention flash + bad value, got: %v", err)
	}
}

func TestResolve_InvalidFlashFTOnOff(t *testing.T) {
	opts := defaultOpts()
	opts.FlashFT = "yes"
	_, err := Resolve(opts)
	if err == nil {
		t.Fatal("expected error for invalid flash-ft value")
	}
	if !strings.Contains(err.Error(), "flash-ft") || !strings.Contains(err.Error(), "yes") {
		t.Errorf("error should mention flash-ft + bad value, got: %v", err)
	}
}

func TestResolve_InvalidFlashColors(t *testing.T) {
	colors := []struct {
		field string
		set   func(*CLIOptions)
	}{
		{"flash-label-fg", func(o *CLIOptions) { o.FlashLabelFG = "red" }},
		{"flash-label-bg", func(o *CLIOptions) { o.FlashLabelBG = "blue" }},
		{"flash-match-fg", func(o *CLIOptions) { o.FlashMatchFG = "#xxx" }},
		{"flash-match-bg", func(o *CLIOptions) { o.FlashMatchBG = "rgb(1,2,3)" }},
		{"flash-backdrop", func(o *CLIOptions) { o.FlashBackdrop = "transparent" }},
	}
	for _, tc := range colors {
		t.Run(tc.field, func(t *testing.T) {
			opts := defaultOpts()
			tc.set(&opts)
			_, err := Resolve(opts)
			if err == nil {
				t.Fatalf("expected error for bad %s color", tc.field)
			}
			if !strings.Contains(err.Error(), tc.field) {
				t.Errorf("error should mention %s, got: %v", tc.field, err)
			}
		})
	}
}

func TestResolve_ValidFlashColorsAndToggles(t *testing.T) {
	opts := defaultOpts()
	opts.Flash = "on"
	opts.FlashFT = "off"
	opts.FlashLabelFG = "#ff00ff"
	opts.FlashLabelBG = "#000000"
	opts.FlashMatchFG = "#abcdef"
	opts.FlashMatchBG = "#123456"
	opts.FlashBackdrop = "#808080"
	if _, err := Resolve(opts); err != nil {
		t.Fatalf("valid flash settings rejected: %v", err)
	}
}
