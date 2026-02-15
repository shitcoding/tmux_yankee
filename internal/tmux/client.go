package tmux

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

// Client wraps tmux command execution
type Client struct{}

// NewClient creates a new tmux client
func NewClient() *Client {
	return &Client{}
}

// CapturePane captures the content of a tmux pane
// start: starting line (0-based), use 0 for beginning
// end: ending line, use -1 for end of history
func (c *Client) CapturePane(paneID string, start, end int) ([]string, error) {
	args := []string{"capture-pane", "-p", "-t", paneID}

	if start > 0 {
		args = append(args, "-S", strconv.Itoa(start))
	} else {
		// Capture full history
		args = append(args, "-S", "-")
	}

	if end >= 0 {
		args = append(args, "-E", strconv.Itoa(end))
	}

	cmd := exec.Command("tmux", args...)
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

// GetFormatVar queries a tmux format variable
func (c *Client) GetFormatVar(paneID, formatVar string) (string, error) {
	cmd := exec.Command("tmux", "display-message", "-p", "-t", paneID, formatVar)
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

// SetBuffer sets the tmux paste buffer
func (c *Client) SetBuffer(text string) error {
	cmd := exec.Command("tmux", "set-buffer", text)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("set-buffer failed: %w", err)
	}
	return nil
}
