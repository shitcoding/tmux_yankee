package tmux

import (
	"context"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

const cmdTimeout = 10 * time.Second

// Client wraps tmux command execution
type Client struct {
	ctx context.Context
}

// NewClient creates a new tmux client. The context is used as a parent
// for per-command timeouts (10s) and allows external cancellation.
func NewClient(ctx context.Context) *Client {
	return &Client{ctx: ctx}
}

// command creates an exec.Cmd with a 10-second timeout derived from
// the client's parent context.
func (c *Client) command(name string, args ...string) (*exec.Cmd, context.CancelFunc) {
	ctx, cancel := context.WithTimeout(c.ctx, cmdTimeout)
	return exec.CommandContext(ctx, name, args...), cancel
}

// capturePaneArgs builds the argument slice for a tmux capture-pane command.
// Extracted for unit testing without requiring a live tmux session.
func capturePaneArgs(paneID string, start, end int, preserveColors bool) []string {
	args := []string{"capture-pane", "-p", "-t", paneID}

	// Add -e flag to preserve ANSI escape sequences
	if preserveColors {
		args = append(args, "-e")
	}

	if start != 0 {
		// Negative start means relative to current position (e.g. -2000 = 2000 lines back)
		// Positive start means absolute line offset
		args = append(args, "-S", strconv.Itoa(start))
	} else {
		// start == 0: capture full history
		args = append(args, "-S", "-")
	}

	if end >= 0 {
		args = append(args, "-E", strconv.Itoa(end))
	}

	return args
}

// CapturePane captures the content of a tmux pane
// start: starting line; negative value limits scrollback (e.g. -2000 = last 2000 lines),
// 0 captures full history, positive is an absolute line offset.
// end: ending line, use -1 for end of history
// preserveColors: if true, includes ANSI escape sequences via -e flag
func (c *Client) CapturePane(paneID string, start, end int, preserveColors bool) ([]string, error) {
	args := capturePaneArgs(paneID, start, end, preserveColors)

	cmd, cancel := c.command("tmux", args...)
	defer cancel()
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("capture-pane failed: %w", err)
	}

	// Split into lines, preserving empty lines
	content := strings.Split(string(output), "\n")

	// Remove trailing empty line if present
	if len(content) > 0 && content[len(content)-1] == "" {
		content = content[:len(content)-1]
	}

	return content, nil
}

// GetFormatVar queries a tmux format variable.
// NOTE: Only called with hardcoded format strings — do not pass user input.
func (c *Client) GetFormatVar(paneID, formatVar string) (string, error) {
	cmd, cancel := c.command("tmux", "display-message", "-p", "-t", paneID, formatVar)
	defer cancel()
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("display-message failed: %w", err)
	}

	return strings.TrimSpace(string(output)), nil
}

// GetHistorySize returns the pane's history size
func (c *Client) GetHistorySize(paneID string) (int, error) {
	val, err := c.GetFormatVar(paneID, "#{history_size}")
	if err != nil {
		return 0, err
	}

	size, err := strconv.Atoi(val)
	if err != nil {
		return 0, fmt.Errorf("invalid history_size: %w", err)
	}

	return size, nil
}

// GetScrollPosition returns the pane's scroll position
func (c *Client) GetScrollPosition(paneID string) (int, error) {
	val, err := c.GetFormatVar(paneID, "#{scroll_position}")
	if err != nil {
		return 0, err
	}

	pos, err := strconv.Atoi(val)
	if err != nil {
		return 0, fmt.Errorf("invalid scroll_position: %w", err)
	}

	return pos, nil
}

// SetBuffer sets the tmux paste buffer. Text is delivered via stdin
// (load-buffer -) to avoid ARG_MAX limits on large selections and to keep
// the buffer contents out of process argv (visible in ps).
func (c *Client) SetBuffer(text string) error {
	cmd, cancel := c.command("tmux", "load-buffer", "-")
	defer cancel()
	cmd.Stdin = strings.NewReader(text)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("load-buffer failed: %w", err)
	}
	return nil
}
