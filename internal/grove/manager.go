package grove

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Scan reads all agent state files and returns agents + summary.
func Scan(groveDir string) ([]*AgentState, StatusSummary) {
	statusDir := filepath.Join(groveDir, StateDirName)
	entries, err := os.ReadDir(statusDir)
	if err != nil {
		return nil, StatusSummary{}
	}

	var agents []*AgentState
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		path := filepath.Join(statusDir, entry.Name())
		agent, err := LoadAgentState(path)
		if err != nil {
			continue
		}
		// Set display name: project_name or truncated session_id
		if agent.ProjectName != "" {
			agent.DisplayName = agent.ProjectName
		} else if len(agent.SessionID) > 12 {
			agent.DisplayName = agent.SessionID[:12]
		} else {
			agent.DisplayName = agent.SessionID
		}
		resolveWorkspace(agent, groveDir)
		agents = append(agents, agent)
	}

	// Sort: needs_attention first, then by display name
	sort.Slice(agents, func(i, j int) bool {
		ai, aj := agents[i], agents[j]
		if ai.NeedsAttention() != aj.NeedsAttention() {
			return ai.NeedsAttention()
		}
		return strings.ToLower(ai.DisplayName) < strings.ToLower(aj.DisplayName)
	})

	return agents, NewStatusSummary(agents)
}

// resolveWorkspace enriches an agent with Grove workspace info if CWD is inside a workspace.
func resolveWorkspace(agent *AgentState, groveDir string) {
	if agent.CWD == "" {
		return
	}
	state, err := LoadState(groveDir)
	if err != nil {
		return
	}

	for _, ws := range state.Workspaces {
		if ws.Path == "" {
			continue
		}
		// Check if agent CWD is inside this workspace path
		if strings.HasPrefix(agent.CWD, ws.Path+"/") || agent.CWD == ws.Path {
			agent.WorkspaceName = ws.Name
			agent.WorkspaceBranch = ws.Branch
			for _, r := range ws.Repos {
				agent.WorkspaceRepos = append(agent.WorkspaceRepos, r.RepoName)
			}
			agent.DisplayName = ws.Name
			return
		}
	}
}

// CleanupStale removes state files for dead/stale sessions. Returns count removed.
func CleanupStale(groveDir string) int {
	statusDir := filepath.Join(groveDir, StateDirName)
	entries, err := os.ReadDir(statusDir)
	if err != nil {
		return 0
	}

	removed := 0
	seenPIDs := map[int]string{} // pid -> path

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		path := filepath.Join(statusDir, entry.Name())
		agent, err := LoadAgentState(path)
		if err != nil {
			os.Remove(path)
			removed++
			continue
		}

		// PID-based cleanup: remove if process is dead
		if agent.PID > 0 && !IsPIDAlive(agent.PID) {
			os.Remove(path)
			removed++
			continue
		}

		// Time-based cleanup: remove if idle too long and no PID
		if agent.PID == 0 && agent.IdleSeconds() > StaleTimeout {
			os.Remove(path)
			removed++
			continue
		}

		// PID dedup: keep the one with higher tool_count
		if agent.PID > 0 {
			if otherPath, ok := seenPIDs[agent.PID]; ok {
				otherAgent, err := LoadAgentState(otherPath)
				if err == nil && otherAgent.ToolCount >= agent.ToolCount {
					os.Remove(path)
					removed++
					continue
				} else {
					os.Remove(otherPath)
					removed++
				}
			}
			seenPIDs[agent.PID] = path
		}
	}

	return removed
}

// ResetStalePermissions clears WAITING_PERMISSION for dead processes.
func ResetStalePermissions(groveDir string) int {
	statusDir := filepath.Join(groveDir, StateDirName)
	entries, err := os.ReadDir(statusDir)
	if err != nil {
		return 0
	}

	reset := 0
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		path := filepath.Join(statusDir, entry.Name())
		agent, err := LoadAgentState(path)
		if err != nil {
			continue
		}
		if agent.Status == StatusWaitingPerm && agent.PID > 0 && !IsPIDAlive(agent.PID) {
			os.Remove(path)
			reset++
		}
	}

	return reset
}

// BucketAgents distributes agents into kanban column buckets.
func BucketAgents(agents []*AgentState) map[string][]*AgentState {
	buckets := make(map[string][]*AgentState)
	for _, col := range KanbanColumns {
		buckets[col.ID] = nil
	}

	for _, agent := range agents {
		placed := false
		for _, col := range KanbanColumns {
			if col.Statuses[agent.Status] {
				buckets[col.ID] = append(buckets[col.ID], agent)
				placed = true
				break
			}
		}
		if !placed {
			buckets["idle"] = append(buckets["idle"], agent)
		}
	}

	return buckets
}

// MatchesFilter checks if an agent matches a search query.
func MatchesFilter(agent *AgentState, query string) bool {
	q := strings.ToLower(query)
	name := agent.DisplayName
	if name == "" {
		name = agent.SessionID
	}
	return strings.Contains(strings.ToLower(name), q) ||
		strings.Contains(strings.ToLower(agent.GitBranch), q) ||
		strings.Contains(strings.ToLower(agent.LastTool), q) ||
		strings.Contains(strings.ToLower(agent.CWD), q) ||
		strings.Contains(strings.ToLower(string(agent.Status)), q)
}

// FilterAgents returns agents matching a search query.
func FilterAgents(agents []*AgentState, query string) []*AgentState {
	if query == "" {
		return agents
	}
	var filtered []*AgentState
	for _, a := range agents {
		if MatchesFilter(a, query) {
			filtered = append(filtered, a)
		}
	}
	return filtered
}

// LoadWorkspaceState reads the workspace state file and returns workspace names for quick lookup.
func LoadWorkspaceState(groveDir string) map[string]*Workspace {
	state, err := LoadState(groveDir)
	if err != nil {
		return nil
	}
	m := make(map[string]*Workspace)
	for i := range state.Workspaces {
		ws := &state.Workspaces[i]
		m[ws.Name] = ws
	}
	return m
}

