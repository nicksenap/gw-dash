package grove

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeAgentJSON(t *testing.T, dir, sessionID string, fields map[string]any) {
	t.Helper()
	if fields == nil {
		fields = make(map[string]any)
	}
	if _, ok := fields["session_id"]; !ok {
		fields["session_id"] = sessionID
	}
	if _, ok := fields["status"]; !ok {
		fields["status"] = "IDLE"
	}
	data, _ := json.Marshal(fields)
	os.WriteFile(filepath.Join(dir, sessionID+".json"), data, 0644)
}

func TestScan_Empty(t *testing.T) {
	dir := t.TempDir()
	groveDir := filepath.Dir(dir)
	statusDir := filepath.Join(groveDir, StateDirName)
	os.MkdirAll(statusDir, 0755)

	agents, summary := Scan(groveDir)
	if len(agents) != 0 {
		t.Errorf("Scan() returned %d agents, want 0", len(agents))
	}
	if summary.Total != 0 {
		t.Errorf("summary.Total = %d, want 0", summary.Total)
	}
}

func TestScan_ReadsAgents(t *testing.T) {
	groveDir := t.TempDir()
	statusDir := filepath.Join(groveDir, StateDirName)
	os.MkdirAll(statusDir, 0755)

	writeAgentJSON(t, statusDir, "s1", map[string]any{
		"status":       "WORKING",
		"project_name": "proj-a",
		"cwd":          "/tmp/a",
	})
	writeAgentJSON(t, statusDir, "s2", map[string]any{
		"status":       "IDLE",
		"project_name": "proj-b",
		"cwd":          "/tmp/b",
	})

	agents, summary := Scan(groveDir)
	if len(agents) != 2 {
		t.Fatalf("Scan() returned %d agents, want 2", len(agents))
	}
	if summary.Total != 2 {
		t.Errorf("summary.Total = %d, want 2", summary.Total)
	}
	if summary.Working != 1 {
		t.Errorf("summary.Working = %d, want 1", summary.Working)
	}
	if summary.Idle != 1 {
		t.Errorf("summary.Idle = %d, want 1", summary.Idle)
	}
}

func TestScan_SetsDisplayName(t *testing.T) {
	groveDir := t.TempDir()
	statusDir := filepath.Join(groveDir, StateDirName)
	os.MkdirAll(statusDir, 0755)

	writeAgentJSON(t, statusDir, "s1", map[string]any{
		"project_name": "my-project",
	})
	writeAgentJSON(t, statusDir, "abcdef123456789", map[string]any{})

	agents, _ := Scan(groveDir)
	for _, a := range agents {
		if a.SessionID == "s1" && a.DisplayName != "my-project" {
			t.Errorf("DisplayName = %q, want %q", a.DisplayName, "my-project")
		}
		if a.SessionID == "abcdef123456789" && a.DisplayName != "abcdef123456" {
			t.Errorf("DisplayName = %q, want truncated session_id", a.DisplayName)
		}
	}
}

func TestScan_SortsAttentionFirst(t *testing.T) {
	groveDir := t.TempDir()
	statusDir := filepath.Join(groveDir, StateDirName)
	os.MkdirAll(statusDir, 0755)

	writeAgentJSON(t, statusDir, "idle1", map[string]any{
		"status":       "IDLE",
		"project_name": "aaa",
	})
	writeAgentJSON(t, statusDir, "perm1", map[string]any{
		"status":       "WAITING_PERMISSION",
		"project_name": "zzz",
	})

	agents, _ := Scan(groveDir)
	if len(agents) != 2 {
		t.Fatalf("expected 2 agents, got %d", len(agents))
	}
	// Attention agent should be first even though name sorts later
	if agents[0].Status != StatusWaitingPerm {
		t.Errorf("first agent status = %q, want WAITING_PERMISSION", agents[0].Status)
	}
}

func TestScan_SkipsInvalidJSON(t *testing.T) {
	groveDir := t.TempDir()
	statusDir := filepath.Join(groveDir, StateDirName)
	os.MkdirAll(statusDir, 0755)

	os.WriteFile(filepath.Join(statusDir, "bad.json"), []byte("not json{{{"), 0644)
	writeAgentJSON(t, statusDir, "good", map[string]any{"status": "IDLE"})

	agents, _ := Scan(groveDir)
	if len(agents) != 1 {
		t.Errorf("Scan() returned %d agents, want 1 (skipping invalid)", len(agents))
	}
}

func TestScan_NoStatusDir(t *testing.T) {
	groveDir := t.TempDir()
	agents, summary := Scan(groveDir)
	if len(agents) != 0 {
		t.Errorf("Scan() returned %d agents, want 0", len(agents))
	}
	if summary.Total != 0 {
		t.Errorf("summary.Total = %d, want 0", summary.Total)
	}
}

func TestCleanupStale_RemovesInvalidJSON(t *testing.T) {
	groveDir := t.TempDir()
	statusDir := filepath.Join(groveDir, StateDirName)
	os.MkdirAll(statusDir, 0755)

	os.WriteFile(filepath.Join(statusDir, "bad.json"), []byte("{invalid"), 0644)

	removed := CleanupStale(groveDir)
	if removed != 1 {
		t.Errorf("CleanupStale() removed %d, want 1", removed)
	}
}

func TestCleanupStale_RemovesDeadPID(t *testing.T) {
	groveDir := t.TempDir()
	statusDir := filepath.Join(groveDir, StateDirName)
	os.MkdirAll(statusDir, 0755)

	writeAgentJSON(t, statusDir, "dead", map[string]any{
		"pid": 999999999, // surely dead
	})

	removed := CleanupStale(groveDir)
	if removed != 1 {
		t.Errorf("CleanupStale() removed %d, want 1", removed)
	}
	if _, err := os.Stat(filepath.Join(statusDir, "dead.json")); !os.IsNotExist(err) {
		t.Error("dead.json should have been removed")
	}
}

func TestCleanupStale_KeepsAlive(t *testing.T) {
	groveDir := t.TempDir()
	statusDir := filepath.Join(groveDir, StateDirName)
	os.MkdirAll(statusDir, 0755)

	writeAgentJSON(t, statusDir, "alive", map[string]any{
		"pid": os.Getpid(), // current process
	})

	removed := CleanupStale(groveDir)
	if removed != 0 {
		t.Errorf("CleanupStale() removed %d, want 0", removed)
	}
}

func TestBucketAgents(t *testing.T) {
	agents := []*AgentState{
		{SessionID: "1", Status: StatusWorking},
		{SessionID: "2", Status: StatusWaitingPerm},
		{SessionID: "3", Status: StatusIdle},
		{SessionID: "4", Status: StatusError},
		{SessionID: "5", Status: StatusDone},
	}
	buckets := BucketAgents(agents)

	if len(buckets["active"]) != 1 {
		t.Errorf("active bucket = %d, want 1", len(buckets["active"]))
	}
	if len(buckets["attention"]) != 2 {
		t.Errorf("attention bucket = %d, want 2", len(buckets["attention"]))
	}
	if len(buckets["idle"]) != 1 {
		t.Errorf("idle bucket = %d, want 1", len(buckets["idle"]))
	}
	if len(buckets["done"]) != 1 {
		t.Errorf("done bucket = %d, want 1", len(buckets["done"]))
	}
}

func TestBucketAgents_UnknownStatusGoesToIdle(t *testing.T) {
	agents := []*AgentState{
		{SessionID: "1", Status: "UNKNOWN_STATUS"},
	}
	buckets := BucketAgents(agents)
	if len(buckets["idle"]) != 1 {
		t.Errorf("idle bucket = %d, want 1 for unknown status", len(buckets["idle"]))
	}
}

func TestMatchesFilter(t *testing.T) {
	agent := &AgentState{
		SessionID:   "abc123",
		DisplayName: "my-project",
		GitBranch:   "feat/cool-feature",
		LastTool:    "Bash",
		CWD:         "/home/user/repos/project",
		Status:      StatusWorking,
	}

	tests := []struct {
		query string
		want  bool
	}{
		{"my-project", true},
		{"MY-PROJECT", true}, // case insensitive
		{"cool-feature", true},
		{"Bash", true},
		{"repos/project", true},
		{"WORKING", true},
		{"nonexistent", false},
	}
	for _, tt := range tests {
		if got := MatchesFilter(agent, tt.query); got != tt.want {
			t.Errorf("MatchesFilter(%q) = %v, want %v", tt.query, got, tt.want)
		}
	}
}

func TestFilterAgents(t *testing.T) {
	agents := []*AgentState{
		{SessionID: "1", DisplayName: "proj-a", Status: StatusWorking},
		{SessionID: "2", DisplayName: "proj-b", Status: StatusIdle},
		{SessionID: "3", DisplayName: "other", Status: StatusWorking},
	}

	// Empty query returns all
	filtered := FilterAgents(agents, "")
	if len(filtered) != 3 {
		t.Errorf("FilterAgents('') = %d, want 3", len(filtered))
	}

	// Filter by name
	filtered = FilterAgents(agents, "proj")
	if len(filtered) != 2 {
		t.Errorf("FilterAgents('proj') = %d, want 2", len(filtered))
	}

	// Filter by status
	filtered = FilterAgents(agents, "idle")
	if len(filtered) != 1 {
		t.Errorf("FilterAgents('idle') = %d, want 1", len(filtered))
	}
}

func TestResolveWorkspace(t *testing.T) {
	groveDir := t.TempDir()

	// Write a state.json with workspaces
	wsPath := filepath.Join(groveDir, "workspaces", "feat-test")
	os.MkdirAll(wsPath, 0755)
	stateData, _ := json.Marshal([]map[string]any{
		{
			"name":   "feat-test",
			"path":   wsPath,
			"branch": "feat/test",
			"repos": []map[string]string{
				{"repo_name": "repo-a"},
				{"repo_name": "repo-b"},
			},
		},
	})
	os.WriteFile(filepath.Join(groveDir, "state.json"), stateData, 0644)

	agent := &AgentState{
		SessionID: "s1",
		CWD:       filepath.Join(wsPath, "repo-a"),
	}
	state, _ := LoadState(groveDir)
	resolveWorkspaceWithState(agent, state)

	if agent.WorkspaceName != "feat-test" {
		t.Errorf("WorkspaceName = %q, want %q", agent.WorkspaceName, "feat-test")
	}
	if agent.WorkspaceBranch != "feat/test" {
		t.Errorf("WorkspaceBranch = %q, want %q", agent.WorkspaceBranch, "feat/test")
	}
	if agent.DisplayName != "feat-test" {
		t.Errorf("DisplayName = %q, want %q", agent.DisplayName, "feat-test")
	}
	if len(agent.WorkspaceRepos) != 2 {
		t.Errorf("WorkspaceRepos = %d, want 2", len(agent.WorkspaceRepos))
	}
}

func TestResolveWorkspace_NoMatch(t *testing.T) {
	groveDir := t.TempDir()
	stateData, _ := json.Marshal([]map[string]any{
		{
			"name":   "ws1",
			"path":   "/some/other/path",
			"branch": "main",
		},
	})
	os.WriteFile(filepath.Join(groveDir, "state.json"), stateData, 0644)

	agent := &AgentState{
		SessionID:   "s1",
		CWD:         "/totally/different/path",
		DisplayName: "original",
	}
	state, _ := LoadState(groveDir)
	resolveWorkspaceWithState(agent, state)

	if agent.WorkspaceName != "" {
		t.Errorf("WorkspaceName = %q, want empty", agent.WorkspaceName)
	}
	if agent.DisplayName != "original" {
		t.Errorf("DisplayName changed to %q, should stay %q", agent.DisplayName, "original")
	}
}

func TestResolveWorkspace_EmptyCWD(t *testing.T) {
	agent := &AgentState{SessionID: "s1", CWD: ""}
	resolveWorkspaceWithState(agent, nil)
	if agent.WorkspaceName != "" {
		t.Errorf("WorkspaceName = %q, want empty for empty CWD", agent.WorkspaceName)
	}
}

func TestResetStalePermissions(t *testing.T) {
	groveDir := t.TempDir()
	statusDir := filepath.Join(groveDir, StateDirName)
	os.MkdirAll(statusDir, 0755)

	writeAgentJSON(t, statusDir, "perm-dead", map[string]any{
		"status": "WAITING_PERMISSION",
		"pid":    999999999,
	})
	writeAgentJSON(t, statusDir, "perm-alive", map[string]any{
		"status": "WAITING_PERMISSION",
		"pid":    os.Getpid(),
	})

	reset := ResetStalePermissions(groveDir)
	if reset != 1 {
		t.Errorf("ResetStalePermissions() = %d, want 1", reset)
	}

	// Dead one should be removed
	if _, err := os.Stat(filepath.Join(statusDir, "perm-dead.json")); !os.IsNotExist(err) {
		t.Error("perm-dead.json should have been removed")
	}
	// Alive one should remain
	if _, err := os.Stat(filepath.Join(statusDir, "perm-alive.json")); os.IsNotExist(err) {
		t.Error("perm-alive.json should still exist")
	}
}

func TestScan_IgnoresNonJSON(t *testing.T) {
	groveDir := t.TempDir()
	statusDir := filepath.Join(groveDir, StateDirName)
	os.MkdirAll(statusDir, 0755)

	// Write a non-JSON file and a directory
	os.WriteFile(filepath.Join(statusDir, "readme.txt"), []byte("hello"), 0644)
	os.MkdirAll(filepath.Join(statusDir, "subdir"), 0755)
	writeAgentJSON(t, statusDir, "real", map[string]any{"status": "IDLE"})

	agents, _ := Scan(groveDir)
	if len(agents) != 1 {
		t.Errorf("Scan() returned %d agents, want 1", len(agents))
	}
}

func TestScan_AllStatusTypes(t *testing.T) {
	groveDir := t.TempDir()
	statusDir := filepath.Join(groveDir, StateDirName)
	os.MkdirAll(statusDir, 0755)

	statuses := []string{"IDLE", "WORKING", "WAITING_PERMISSION", "WAITING_ANSWER", "ERROR", "DONE", "PROVISIONING"}
	for i, s := range statuses {
		id := strings.ToLower(s)
		writeAgentJSON(t, statusDir, id, map[string]any{
			"status":       s,
			"project_name": s,
		})
		_ = i
	}

	agents, summary := Scan(groveDir)
	if len(agents) != 7 {
		t.Errorf("Scan() returned %d agents, want 7", len(agents))
	}
	if summary.Total != 7 {
		t.Errorf("summary.Total = %d, want 7", summary.Total)
	}
}
