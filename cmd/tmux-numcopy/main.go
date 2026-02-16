package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

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
	// CLI flags
	paneID := flag.String("pane", "", "Target tmux pane ID")
	mode := flag.String("mode", "hybrid", "Line number mode (absolute, relative, hybrid)")

	flag.Parse()

	// Validate required flags
	if *paneID == "" {
		fmt.Fprintln(os.Stderr, "Error: --pane is required")
		flag.Usage()
		os.Exit(1)
	}

	// Validate mode
	switch *mode {
	case "absolute", "relative", "hybrid":
		// Valid modes
	default:
		fmt.Fprintf(os.Stderr, "Error: invalid mode %q (must be absolute, relative, or hybrid)\n", *mode)
		os.Exit(1)
	}

	// Setup signal handling for clean exit
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Create tmux client
	client := tmux.NewClient()

	// Capture pane content with ANSI color codes preserved
	// Use -2000 to capture last 2000 lines (recent history only, not entire scrollback)
	content, err := client.CapturePane(*paneID, -2000, -1, true)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error capturing pane: %v\n", err)
		os.Exit(1)
	}

	// Trim trailing empty lines (common in scrollback buffers)
	content = trimTrailingEmptyLines(content)

	// Create TUI
	tui := ui.NewTUI(*paneID, content, *mode)

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
