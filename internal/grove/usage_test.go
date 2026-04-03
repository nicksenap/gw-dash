package grove

import (
	"testing"
)

func TestClaudeUsage_Bar(t *testing.T) {
	tests := []struct {
		util int
		want string
	}{
		{0, "░░░░░░░░░░"},
		{45, "▓▓▓▓░░░░░░"},
		{100, "▓▓▓▓▓▓▓▓▓▓"},
		{-5, "░░░░░░░░░░"},
	}
	for _, tt := range tests {
		usage := &ClaudeUsage{Utilization: tt.util}
		if got := usage.Bar(); got != tt.want {
			t.Errorf("Bar(%d) = %q, want %q", tt.util, got, tt.want)
		}
		if barLen := len([]rune(usage.Bar())); barLen != 10 {
			t.Errorf("Bar(%d) rune length = %d, want 10", tt.util, barLen)
		}
	}
}

func TestClaudeUsage_ResetCountdown_Empty(t *testing.T) {
	usage := &ClaudeUsage{}
	if got := usage.ResetCountdown(); got != "" {
		t.Errorf("ResetCountdown() = %q, want empty", got)
	}
}

func TestClaudeUsage_ResetCountdown_Past(t *testing.T) {
	usage := &ClaudeUsage{ResetsAt: "2020-01-01T00:00:00Z"}
	if got := usage.ResetCountdown(); got != "now" {
		t.Errorf("ResetCountdown() = %q, want %q", got, "now")
	}
}

func TestClaudeUsage_ResetCountdown_Invalid(t *testing.T) {
	usage := &ClaudeUsage{ResetsAt: "not-a-date"}
	if got := usage.ResetCountdown(); got != "" {
		t.Errorf("ResetCountdown() = %q, want empty for invalid date", got)
	}
}

func TestUsageColor(t *testing.T) {
	tests := []struct {
		pct  int
		want string
	}{
		{0, ColorGreen},
		{49, ColorGreen},
		{50, ColorYellow},
		{74, ColorYellow},
		{75, ColorOrange},
		{89, ColorOrange},
		{90, ColorRed},
		{100, ColorRed},
	}
	for _, tt := range tests {
		if got := UsageColor(tt.pct); got != tt.want {
			t.Errorf("UsageColor(%d) = %q, want %q", tt.pct, got, tt.want)
		}
	}
}
