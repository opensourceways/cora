// Package color provides terminal-aware ANSI color support for output
// rendering. Colors are automatically disabled when stdout is not a
// terminal (e.g. piped to a file or another command).
package color

import (
	"os"
	"strings"
)

// Enabled controls whether ANSI color codes are emitted. Auto-detected
// from stdout at init time; call Disable() for --no-color override.
var Enabled = isTerminal()

func isTerminal() bool {
	fi, err := os.Stdout.Stat()
	if err != nil {
		return false
	}
	return (fi.Mode() & os.ModeCharDevice) != 0
}

// Disable turns off color output (e.g. for --no-color flag).
func Disable() { Enabled = false }

// ── ANSI escape codes ──────────────────────────────────────────────────────

const (
	reset   = "\033[0m"
	red     = "\033[31m"
	green   = "\033[32m"
	yellow  = "\033[33m"
	blue    = "\033[34m"
	magenta = "\033[35m"
	cyan    = "\033[36m"
	gray    = "\033[90m"
)

// State returns the value colored based on common state keywords.
// Used for fields like "state", "status", "phase", etc.
// Recognized values:
//
//	open, active, success, approved → green
//	closed, failed, rejected, error → red
//	merged                         → magenta
//	pending, waiting                → yellow
//	reviewing, building, running    → cyan
//	draft                           → gray
func State(val string) string {
	if !Enabled || val == "" {
		return val
	}
	lower := strings.ToLower(strings.TrimSpace(val))
	switch {
	case lower == "open", lower == "active",
		lower == "success", lower == "approved",
		lower == "passed", lower == "healthy":
		return green + val + reset

	case lower == "closed", lower == "failed",
		lower == "rejected", lower == "error",
		lower == "failure", lower == "unhealthy":
		return red + val + reset

	case lower == "merged":
		return magenta + val + reset

	case lower == "pending", lower == "waiting":
		return yellow + val + reset

	case lower == "reviewing", lower == "building",
		lower == "running", lower == "in_progress":
		return cyan + val + reset

	case lower == "draft":
		return gray + val + reset
	}
	return val
}
