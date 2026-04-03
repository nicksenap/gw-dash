package grove

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// ClaudeUsage holds Claude API usage data from the Usage Tracker cache.
type ClaudeUsage struct {
	Utilization int
	ResetsAt    string
	ProfileName string
	Stale       bool
}

const usageCacheFile = ".statusline-usage-cache"
const staleSeconds = 600 // 10 minutes

// ReadUsageCache reads usage data from the Claude Usage Tracker cache file.
func ReadUsageCache() *ClaudeUsage {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil
	}
	path := filepath.Join(home, ".claude", usageCacheFile)
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}

	vals := make(map[string]string)
	for _, line := range strings.Split(strings.TrimSpace(string(data)), "\n") {
		if idx := strings.Index(line, "="); idx >= 0 {
			k := strings.TrimSpace(line[:idx])
			v := strings.TrimSpace(line[idx+1:])
			vals[k] = v
		}
	}

	utilStr, ok := vals["UTILIZATION"]
	if !ok {
		return nil
	}
	util, err := strconv.Atoi(utilStr)
	if err != nil {
		return nil
	}

	ts, err := strconv.ParseInt(vals["TIMESTAMP"], 10, 64)
	if err != nil {
		ts = 0
	}
	stale := true
	if ts > 0 {
		stale = (time.Now().Unix() - ts) > staleSeconds
	}

	return &ClaudeUsage{
		Utilization: util,
		ResetsAt:    vals["RESETS_AT"],
		ProfileName: vals["PROFILE_NAME"],
		Stale:       stale,
	}
}

// ResetCountdown returns a human-friendly countdown to reset, e.g. "1h32m".
func (u *ClaudeUsage) ResetCountdown() string {
	if u.ResetsAt == "" {
		return ""
	}
	reset, err := time.Parse(time.RFC3339, u.ResetsAt)
	if err != nil {
		// Try without timezone
		reset, err = time.Parse("2006-01-02T15:04:05Z", u.ResetsAt)
		if err != nil {
			return ""
		}
	}
	secs := int(time.Until(reset).Seconds())
	if secs <= 0 {
		return "now"
	}
	if secs < 60 {
		return fmt.Sprintf("%ds", secs)
	}
	if secs < 3600 {
		return fmt.Sprintf("%dm", secs/60)
	}
	return fmt.Sprintf("%dh%02dm", secs/3600, (secs%3600)/60)
}

// Bar returns a 10-block progress bar using block characters.
func (u *ClaudeUsage) Bar() string {
	filled := int(math.Max(0, math.Min(10, float64(u.Utilization/10))))
	return strings.Repeat("▓", filled) + strings.Repeat("░", 10-filled)
}

// UsageColor returns a Gruvbox color based on utilization percentage.
func UsageColor(pct int) string {
	switch {
	case pct < 50:
		return ColorGreen
	case pct < 75:
		return ColorYellow
	case pct < 90:
		return ColorOrange
	default:
		return ColorRed
	}
}
