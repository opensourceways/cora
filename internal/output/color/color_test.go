package color

import (
	"strings"
	"testing"
)

func TestState(t *testing.T) {
	// Force-enable color for testing
	old := Enabled
	Enabled = true
	defer func() { Enabled = old }()

	tests := []struct {
		input string
		color string // expected ANSI color code prefix (empty = no color)
	}{
		{"open", "\033[32m"},
		{"Open", "\033[32m"},
		{"active", "\033[32m"},
		{"success", "\033[32m"},
		{"approved", "\033[32m"},
		{"closed", "\033[31m"},
		{"failed", "\033[31m"},
		{"rejected", "\033[31m"},
		{"error", "\033[31m"},
		{"merged", "\033[35m"},
		{"pending", "\033[33m"},
		{"waiting", "\033[33m"},
		{"reviewing", "\033[36m"},
		{"building", "\033[36m"},
		{"running", "\033[36m"},
		{"draft", "\033[90m"},
		// No color for unknown values.
		{"unknown", ""},
		{"", ""},
		{"  open  ", "\033[32m"}, // trimmed spaces
	}

	for _, tc := range tests {
		got := State(tc.input)
		if tc.color == "" {
			if got != tc.input && strings.TrimSpace(tc.input) == tc.input {
				t.Errorf("State(%q) = %q, want unchanged", tc.input, got)
			}
		} else {
			if !strings.HasPrefix(got, tc.color) {
				t.Errorf("State(%q) = %q, want prefix %q", tc.input, got, tc.color)
			}
			if !strings.Contains(got, strings.TrimSpace(tc.input)) {
				t.Errorf("State(%q) = %q, should contain original value", tc.input, got)
			}
		}
	}
}

func TestState_disabled(t *testing.T) {
	Enabled = false
	defer func() { Enabled = true }()

	if got := State("open"); got != "open" {
		t.Errorf("State should pass through when disabled, got %q", got)
	}
}
