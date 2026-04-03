package grove

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestAgentState_NeedsAttention(t *testing.T) {
	tests := []struct {
		status AgentStatus
		want   bool
	}{
		{StatusWaitingPerm, true},
		{StatusWaitingAnswer, true},
		{StatusError, true},
		{StatusWorking, false},
		{StatusIdle, false},
		{StatusDone, false},
		{StatusProvisioning, false},
	}
	for _, tt := range tests {
		agent := &AgentState{SessionID: "s1", Status: tt.status}
		if got := agent.NeedsAttention(); got != tt.want {
			t.Errorf("NeedsAttention(%s) = %v, want %v", tt.status, got, tt.want)
		}
	}
}

func TestAgentState_Sparkline(t *testing.T) {
	agent := &AgentState{
		SessionID:       "s1",
		ActivityHistory: []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 8},
	}
	spark := agent.Sparkline()
	if len([]rune(spark)) != 10 {
		t.Errorf("sparkline length = %d, want 10", len([]rune(spark)))
	}
	// First char should be space (value 0)
	chars := []rune(spark)
	if chars[0] != ' ' {
		t.Errorf("sparkline[0] = %q, want space", chars[0])
	}
}

func TestAgentState_SparklineEmpty(t *testing.T) {
	agent := &AgentState{SessionID: "s1", ActivityHistory: []int{}}
	if spark := agent.Sparkline(); spark != "" {
		t.Errorf("sparkline = %q, want empty", spark)
	}
}

func TestAgentState_Uptime(t *testing.T) {
	// Test with no started_at
	agent := &AgentState{SessionID: "s1"}
	if got := agent.Uptime(); got != "" {
		t.Errorf("Uptime() = %q, want empty", got)
	}

	// Test with recent start
	agent.StartedAt = time.Now().UTC().Add(-30 * time.Second).Format(time.RFC3339)
	uptime := agent.Uptime()
	if uptime == "" {
		t.Error("Uptime() should not be empty for recent start")
	}

	// Test with hour-old start
	agent.StartedAt = time.Now().UTC().Add(-2 * time.Hour).Format(time.RFC3339)
	uptime = agent.Uptime()
	if uptime == "" || uptime[0] != '2' {
		t.Errorf("Uptime() = %q, expected to start with '2'", uptime)
	}
}

func TestAgentState_IdleSeconds(t *testing.T) {
	agent := &AgentState{SessionID: "s1"}
	if got := agent.IdleSeconds(); got != 0 {
		t.Errorf("IdleSeconds() = %v, want 0 for empty LastEventTime", got)
	}

	agent.LastEventTime = time.Now().UTC().Add(-10 * time.Second).Format(time.RFC3339)
	idle := agent.IdleSeconds()
	if idle < 9 || idle > 12 {
		t.Errorf("IdleSeconds() = %v, expected ~10", idle)
	}
}

func TestLoadAgentState(t *testing.T) {
	dir := t.TempDir()
	state := map[string]any{
		"session_id":       "abc",
		"status":           "WORKING",
		"cwd":              "/tmp/proj",
		"project_name":     "proj",
		"tool_count":       5,
		"error_count":      1,
		"activity_history": []int{0, 0, 0, 0, 0, 0, 0, 0, 1, 2},
	}
	data, _ := json.Marshal(state)
	path := filepath.Join(dir, "abc.json")
	os.WriteFile(path, data, 0644)

	agent, err := LoadAgentState(path)
	if err != nil {
		t.Fatalf("LoadAgentState() error = %v", err)
	}
	if agent.SessionID != "abc" {
		t.Errorf("SessionID = %q, want %q", agent.SessionID, "abc")
	}
	if agent.Status != StatusWorking {
		t.Errorf("Status = %q, want %q", agent.Status, StatusWorking)
	}
	if agent.ToolCount != 5 {
		t.Errorf("ToolCount = %d, want 5", agent.ToolCount)
	}
	if agent.ErrorCount != 1 {
		t.Errorf("ErrorCount = %d, want 1", agent.ErrorCount)
	}
}

func TestLoadAgentState_InvalidJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.json")
	os.WriteFile(path, []byte("not json{{{"), 0644)

	_, err := LoadAgentState(path)
	if err == nil {
		t.Error("LoadAgentState() should return error for invalid JSON")
	}
}

func TestLoadAgentState_UnknownStatus(t *testing.T) {
	dir := t.TempDir()
	data, _ := json.Marshal(map[string]any{
		"session_id": "s1",
		"status":     "BANANA",
	})
	path := filepath.Join(dir, "s1.json")
	os.WriteFile(path, data, 0644)

	agent, err := LoadAgentState(path)
	if err != nil {
		t.Fatalf("LoadAgentState() error = %v", err)
	}
	if agent.Status != StatusIdle {
		t.Errorf("Status = %q, want %q for unknown status", agent.Status, StatusIdle)
	}
}

func TestLoadAgentState_MissingFile(t *testing.T) {
	_, err := LoadAgentState("/nonexistent/path.json")
	if err == nil {
		t.Error("LoadAgentState() should return error for missing file")
	}
}

func TestLoadAgentState_DefaultActivityHistory(t *testing.T) {
	dir := t.TempDir()
	data, _ := json.Marshal(map[string]any{
		"session_id": "s1",
		"status":     "IDLE",
	})
	path := filepath.Join(dir, "s1.json")
	os.WriteFile(path, data, 0644)

	agent, err := LoadAgentState(path)
	if err != nil {
		t.Fatalf("LoadAgentState() error = %v", err)
	}
	if len(agent.ActivityHistory) != 10 {
		t.Errorf("ActivityHistory length = %d, want 10", len(agent.ActivityHistory))
	}
}

func TestStatusSummary_FromAgents(t *testing.T) {
	agents := []*AgentState{
		{SessionID: "1", Status: StatusWorking},
		{SessionID: "2", Status: StatusWorking},
		{SessionID: "3", Status: StatusIdle},
		{SessionID: "4", Status: StatusWaitingPerm},
		{SessionID: "5", Status: StatusError},
	}
	s := NewStatusSummary(agents)
	if s.Total != 5 {
		t.Errorf("Total = %d, want 5", s.Total)
	}
	if s.Working != 2 {
		t.Errorf("Working = %d, want 2", s.Working)
	}
	if s.Idle != 1 {
		t.Errorf("Idle = %d, want 1", s.Idle)
	}
	if s.WaitingPerm != 1 {
		t.Errorf("WaitingPerm = %d, want 1", s.WaitingPerm)
	}
	if s.Error != 1 {
		t.Errorf("Error = %d, want 1", s.Error)
	}
}

func TestStatusSummary_StatusLineEmpty(t *testing.T) {
	s := StatusSummary{}
	if got := s.StatusLine(); got != "no agents" {
		t.Errorf("StatusLine() = %q, want %q", got, "no agents")
	}
}

func TestStatusSummary_StatusLine(t *testing.T) {
	s := StatusSummary{Total: 3, Working: 2, Idle: 1}
	line := s.StatusLine()
	if !contains(line, "W:2") {
		t.Errorf("StatusLine() = %q, missing W:2", line)
	}
	if !contains(line, "I:1") {
		t.Errorf("StatusLine() = %q, missing I:1", line)
	}
}

func TestIsPIDAlive(t *testing.T) {
	if IsPIDAlive(0) {
		t.Error("IsPIDAlive(0) should be false")
	}
	if IsPIDAlive(-1) {
		t.Error("IsPIDAlive(-1) should be false")
	}
	if IsPIDAlive(999999999) {
		t.Error("IsPIDAlive(999999999) should be false")
	}
	// Current process should be alive
	if !IsPIDAlive(os.Getpid()) {
		t.Error("IsPIDAlive(self) should be true")
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(s) > 0 && containsStr(s, sub))
}

func containsStr(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
