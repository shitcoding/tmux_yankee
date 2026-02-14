package main

import (
	"flag"
	"fmt"
	"os"
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

	// TODO: Phase 3 will implement the TUI loop here
	fmt.Printf("tmux-yankee starting...\n")
	fmt.Printf("Pane: %s\n", *paneID)
	fmt.Printf("Mode: %s\n", *mode)
	fmt.Println("\nPress 'q' to exit (TUI not yet implemented)")

	// Wait for 'q' keystroke
	var input string
	fmt.Scanln(&input)
}
