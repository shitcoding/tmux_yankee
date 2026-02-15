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

	// Capture pane content
	content, err := client.CapturePane(*paneID, 0, -1)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error capturing pane: %v\n", err)
		os.Exit(1)
	}

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
