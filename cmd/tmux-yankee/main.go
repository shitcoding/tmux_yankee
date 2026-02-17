package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/shitcoding/tmux_yankee/internal/config"
	"github.com/shitcoding/tmux_yankee/internal/tmux"
	"github.com/shitcoding/tmux_yankee/internal/ui"
)

// trimTrailingEmptyLines removes empty lines from the end of content
func trimTrailingEmptyLines(lines []string) []string {
	for len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	return lines
}

func main() {
	var opts config.CLIOptions
	config.RegisterFlags(flag.CommandLine, &opts)
	flag.Parse()

	cfg, err := config.Resolve(opts)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}

	// Setup signal handling for clean exit
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Create tmux client
	client := tmux.NewClient()

	// Capture pane content with ANSI color codes preserved.
	// Capture scrollback: negative value is the -S flag passed to tmux capture-pane
	content, err := client.CapturePane(cfg.PaneID, -cfg.ScrollbackLines, -1, true)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error capturing pane: %v\n", err)
		os.Exit(1)
	}

	// Trim trailing empty lines (common in scrollback buffers)
	content = trimTrailingEmptyLines(content)

	// Create TUI
	tui := ui.NewTUI(cfg.PaneID, content, string(cfg.Mode))

	// Run TUI in goroutine
	errChan := make(chan error, 1)
	go func() {
		errChan <- tui.Run()
	}()

	// Wait for TUI exit or signal
	select {
	case err := <-errChan:
		if err != nil {
			fmt.Fprintf(os.Stderr, "TUI error: %v\n", err)
			os.Exit(1)
		}
	case <-sigChan:
		// Signal received, exit cleanly
		// TUI cleanup happens via defer in tui.Run()
	}
}
