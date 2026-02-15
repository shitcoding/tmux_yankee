package tmux

import (
	"testing"
)

func TestNewClient(t *testing.T) {
	client := NewClient()
	if client == nil {
		t.Fatal("NewClient returned nil")
	}
}

// Note: CapturePane, GetFormatVar, etc. require a running tmux session
// These tests would need integration test setup with actual tmux
// For now, we verify the API contract exists

func TestClientHasRequiredMethods(t *testing.T) {
	client := NewClient()

	// Verify methods exist (compilation check)
	_ = client.CapturePane
	_ = client.GetFormatVar
	_ = client.GetHistorySize
	_ = client.GetScrollPosition
	_ = client.SetBuffer
}
