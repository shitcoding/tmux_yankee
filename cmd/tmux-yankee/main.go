package main

import (
	"context"
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

	var tui *ui.TUI

	if cfg.Demo {
		// Demo mode: skip tmux, use synthetic demo content
		pages := ui.DemoPages()
		tui = ui.NewDemoTUI(cfg, pages, ui.DemoPageNames)
	} else {
		// Normal mode: capture pane content from tmux
		client := tmux.NewClient(context.Background())
		content, err := client.CapturePane(cfg.PaneID, -cfg.ScrollbackLines, -1, true)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error capturing pane: %v\n", err)
			os.Exit(1)
		}
		content = trimTrailingEmptyLines(content)
		tui = ui.NewTUI(cfg, content)
	}

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
		// Signal received — tell TUI to exit and wait for Run() to return.
		// This ensures restoreTerminal() runs via deferred cleanup in Run().
		tui.Stop()
		<-errChan
	}
}
