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

// TestCapturePaneArgs verifies that capturePaneArgs produces the correct tmux
// argument slice without requiring a live tmux session.
func TestCapturePaneArgs(t *testing.T) {
	tests := []struct {
		name           string
		paneID         string
		start          int
		end            int
		preserveColors bool
		wantSFlag      string // expected value after "-S"
		wantEFlag      bool   // whether "-E" should appear
	}{
		{
			name:           "negative start (scrollback 2000)",
			paneID:         "%1",
			start:          -2000,
			end:            -1,
			preserveColors: true,
			wantSFlag:      "-2000",
			wantEFlag:      false,
		},
		{
			name:           "negative start (scrollback 5000)",
			paneID:         "%2",
			start:          -5000,
			end:            -1,
			preserveColors: true,
			wantSFlag:      "-5000",
			wantEFlag:      false,
		},
		{
			name:           "zero start (full history)",
			paneID:         "%1",
			start:          0,
			end:            -1,
			preserveColors: false,
			wantSFlag:      "-",
			wantEFlag:      false,
		},
		{
			name:           "positive start with end",
			paneID:         "%1",
			start:          100,
			end:            200,
			preserveColors: false,
			wantSFlag:      "100",
			wantEFlag:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := capturePaneArgs(tt.paneID, tt.start, tt.end, tt.preserveColors)

			// Verify -S flag value
			sIdx := indexOf(args, "-S")
			if sIdx < 0 {
				t.Fatal("expected -S flag in args")
			}
			if sIdx+1 >= len(args) {
				t.Fatal("-S flag has no value")
			}
			if args[sIdx+1] != tt.wantSFlag {
				t.Errorf("-S value: got %q, want %q", args[sIdx+1], tt.wantSFlag)
			}

			// Verify -E flag presence
			eIdx := indexOf(args, "-E")
			if tt.wantEFlag && eIdx < 0 {
				t.Error("expected -E flag in args, not found")
			}
			if !tt.wantEFlag && eIdx >= 0 {
				t.Errorf("unexpected -E flag in args: %v", args)
			}

			// Verify -e flag presence for color preservation
			hasColorFlag := indexOf(args, "-e") >= 0
			if tt.preserveColors && !hasColorFlag {
				t.Error("expected -e flag for color preservation")
			}
			if !tt.preserveColors && hasColorFlag {
				t.Error("unexpected -e flag when preserveColors=false")
			}
		})
	}
}

// indexOf returns the first index of needle in slice, or -1 if not found.
func indexOf(slice []string, needle string) int {
	for i, s := range slice {
		if s == needle {
			return i
		}
	}
	return -1
}
