package grove

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"syscall"
	"time"
)

// AgentStatus represents the current state of a Claude Code agent.
type AgentStatus string

const (
	StatusProvisioning    AgentStatus = "PROVISIONING"
	StatusIdle            AgentStatus = "IDLE"
	StatusWorking         AgentStatus = "WORKING"
	StatusWaitingPerm     AgentStatus = "WAITING_PERMISSION"
	StatusWaitingAnswer   AgentStatus = "WAITING_ANSWER"
	StatusError           AgentStatus = "ERROR"
	StatusDone            AgentStatus = "DONE"
)

// AttentionStatuses are statuses that require user attention.
var AttentionStatuses = map[AgentStatus]bool{
	StatusWaitingPerm:   true,
	StatusWaitingAnswer: true,
	StatusError:         true,
}

// StatusDisplay holds the color and short label for a status.
type StatusDisplay struct {
	Color string
	Label string
}

// Gruvbox dark palette.
const (
	ColorGreen  = "#b8bb26"
	ColorAqua   = "#8ec07c"
	ColorRed    = "#fb4934"
	ColorYellow = "#fabd2f"
	ColorGrey   = "#928374"
	ColorFG     = "#fbf1c7"
	ColorOrange = "#fe8019"
	ColorPurple = "#d3869b"
	ColorBG     = "#282828"
	ColorBGLight = "#3c3836"
)

var StatusDisplayMap = map[AgentStatus]StatusDisplay{
	StatusProvisioning:  {ColorAqua, "PROV"},
	StatusIdle:          {ColorGrey, "IDLE"},
	StatusWorking:       {ColorGreen, "WORK"},
	StatusWaitingPerm:   {ColorRed, "PERM"},
	StatusWaitingAnswer: {ColorYellow, "WAIT"},
	StatusError:         {ColorOrange, "ERR"},
	StatusDone:          {ColorGreen, "DONE"},
}

// SparkChars are braille-based sparkline characters (0–8).
const SparkChars = " ▁▂▃▄▅▆▇█"

// AgentState is the state of a single Claude Code agent session.
type AgentState struct {
	SessionID           string      `json:"session_id"`
	Status              AgentStatus `json:"status"`
	CWD                 string      `json:"cwd"`
	ProjectName         string      `json:"project_name"`
	Model               string      `json:"model"`
	StartedAt           string      `json:"started_at"`
	LastEvent           string      `json:"last_event"`
	LastEventTime       string      `json:"last_event_time"`
	LastTool            string      `json:"last_tool"`
	ToolCount           int         `json:"tool_count"`
	ErrorCount          int         `json:"error_count"`
	SubagentCount       int         `json:"subagent_count"`
	CompactCount        int         `json:"compact_count"`
	PID                 int         `json:"pid"`
	GitBranch           string      `json:"git_branch"`
	GitDirtyCount       int         `json:"git_dirty_count"`
	NotificationMessage *string     `json:"notification_message"`
	ToolRequestSummary  *string     `json:"tool_request_summary"`
	ActivityHistory     []int       `json:"activity_history"`
	ZellijSession       string      `json:"zellij_session"`
	PermissionMode      string      `json:"permission_mode"`
	SessionSource       string      `json:"session_source"`
	LastMessage         string      `json:"last_message"`
	LastError           string      `json:"last_error"`
	InitialPrompt       string      `json:"initial_prompt"`
	CompactTrigger      string      `json:"compact_trigger"`
	ActiveSubagents     []string    `json:"active_subagents"`

	// Resolved by manager (not persisted)
	DisplayName     string   `json:"-"`
	WorkspaceName   string   `json:"-"`
	WorkspaceBranch string   `json:"-"`
	WorkspaceRepos  []string `json:"-"`
}

// NeedsAttention returns true if the agent needs user attention.
func (a *AgentState) NeedsAttention() bool {
	return AttentionStatuses[a.Status]
}

// Uptime returns a human-readable uptime string.
func (a *AgentState) Uptime() string {
	if a.StartedAt == "" {
		return ""
	}
	start, err := time.Parse(time.RFC3339, a.StartedAt)
	if err != nil {
		// Try alternate formats
		start, err = time.Parse("2006-01-02T15:04:05Z", a.StartedAt)
		if err != nil {
			return ""
		}
	}
	secs := int(time.Since(start).Seconds())
	if secs < 60 {
		return fmt.Sprintf("%ds", secs)
	}
	if secs < 3600 {
		return fmt.Sprintf("%dm", secs/60)
	}
	return fmt.Sprintf("%dh%dm", secs/3600, (secs%3600)/60)
}

// IdleSeconds returns how many seconds since the last event.
func (a *AgentState) IdleSeconds() float64 {
	if a.LastEventTime == "" {
		return 0
	}
	last, err := time.Parse(time.RFC3339, a.LastEventTime)
	if err != nil {
		last, err = time.Parse("2006-01-02T15:04:05Z", a.LastEventTime)
		if err != nil {
			return 0
		}
	}
	return time.Since(last).Seconds()
}

// Sparkline returns a sparkline string from the activity history.
func (a *AgentState) Sparkline() string {
	if len(a.ActivityHistory) == 0 {
		return ""
	}
	chars := []rune(SparkChars)
	mx := 0
	for _, v := range a.ActivityHistory {
		if v > mx {
			mx = v
		}
	}
	if mx == 0 {
		mx = 1
	}
	var sb strings.Builder
	for _, v := range a.ActivityHistory {
		idx := v * 8 / mx
		if idx > 8 {
			idx = 8
		}
		sb.WriteRune(chars[idx])
	}
	return sb.String()
}

// LoadAgentState reads an agent state from a JSON file.
func LoadAgentState(path string) (*AgentState, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var agent AgentState
	if err := json.Unmarshal(data, &agent); err != nil {
		return nil, err
	}

	// Default status
	if agent.Status == "" {
		agent.Status = StatusIdle
	}

	// Validate status
	switch agent.Status {
	case StatusProvisioning, StatusIdle, StatusWorking,
		StatusWaitingPerm, StatusWaitingAnswer, StatusError, StatusDone:
		// valid
	default:
		agent.Status = StatusIdle
	}

	// Default activity history
	if agent.ActivityHistory == nil {
		agent.ActivityHistory = make([]int, 10)
	}

	return &agent, nil
}

// StatusSummary holds aggregate counts across all agents.
type StatusSummary struct {
	Total        int
	Working      int
	Idle         int
	WaitingPerm  int
	WaitingAnswer int
	Error        int
}

// NewStatusSummary creates a summary from a list of agents.
func NewStatusSummary(agents []*AgentState) StatusSummary {
	s := StatusSummary{Total: len(agents)}
	for _, a := range agents {
		switch a.Status {
		case StatusWorking:
			s.Working++
		case StatusIdle:
			s.Idle++
		case StatusWaitingPerm:
			s.WaitingPerm++
		case StatusWaitingAnswer:
			s.WaitingAnswer++
		case StatusError:
			s.Error++
		}
	}
	return s
}

// StatusLine returns a compact status line string.
func (s StatusSummary) StatusLine() string {
	var parts []string
	if s.Working > 0 {
		parts = append(parts, fmt.Sprintf("W:%d", s.Working))
	}
	if s.WaitingPerm > 0 {
		parts = append(parts, fmt.Sprintf("[!]:%d", s.WaitingPerm))
	}
	if s.WaitingAnswer > 0 {
		parts = append(parts, fmt.Sprintf("?:%d", s.WaitingAnswer))
	}
	if s.Error > 0 {
		parts = append(parts, fmt.Sprintf("E:%d", s.Error))
	}
	if s.Idle > 0 {
		parts = append(parts, fmt.Sprintf("I:%d", s.Idle))
	}
	if len(parts) == 0 {
		return "no agents"
	}
	return strings.Join(parts, " ")
}

// IsPIDAlive checks if a process with the given PID is alive.
func IsPIDAlive(pid int) bool {
	if pid <= 0 {
		return false
	}
	proc, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	// On Unix, FindProcess always succeeds. Use signal 0 to probe.
	err = proc.Signal(syscall.Signal(0))
	return err == nil
}

// KanbanColumn defines a kanban column.
type KanbanColumn struct {
	ID       string
	Title    string
	Statuses map[AgentStatus]bool
}

// KanbanColumns defines the kanban column layout.
var KanbanColumns = []KanbanColumn{
	{"active", "Active", map[AgentStatus]bool{StatusWorking: true, StatusProvisioning: true}},
	{"attention", "Attention", map[AgentStatus]bool{StatusWaitingPerm: true, StatusWaitingAnswer: true, StatusError: true}},
	{"idle", "Idle", map[AgentStatus]bool{StatusIdle: true}},
	{"done", "Done", map[AgentStatus]bool{StatusDone: true}},
}

// Poll intervals.
const (
	StatePollInterval = 500 * time.Millisecond
	CleanupInterval   = 30 * time.Second
	StaleTimeout      = 1800 // 30 minutes in seconds
	StateDirName      = "status"
)
