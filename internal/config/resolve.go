package config

// Resolve validates opts and returns a fully populated Settings, or an error.
func Resolve(opts CLIOptions) (Settings, error) {
	if err := Validate(opts); err != nil {
		return Settings{}, err
	}

	// Clamp ScrollbackLines to [MinScrollbackLines, MaxScrollbackLines]
	scrollback := opts.ScrollbackLines
	if scrollback < MinScrollbackLines {
		scrollback = MinScrollbackLines
	}
	if scrollback > MaxScrollbackLines {
		scrollback = MaxScrollbackLines
	}

	// Parse bool strings
	statusIndicator := opts.StatusIndicator == "on"
	exitOnYank := opts.ExitOnYank == "on"

	// Parse ToggleModeKey to byte
	toggleKey := opts.ToggleModeKey[0]

	return Settings{
		PaneID:          opts.PaneID,
		Mode:            LineNumberMode(opts.Mode),
		ScrollbackLines: scrollback,
		// Palette is zero-value for now; theme resolution comes in commit 4
		Palette:         Palette{},
		StatusIndicator: statusIndicator,
		ToggleModeKey:   toggleKey,
		CopyTarget:      CopyTarget(opts.CopyTarget),
		ExitOnYank:      exitOnYank,
		StartPosition:   StartPosition(opts.StartPosition),
	}, nil
}
