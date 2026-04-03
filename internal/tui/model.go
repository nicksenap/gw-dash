// Package tui implements the Bubble Tea TUI for the Grove dashboard.
package tui

import (
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/nicksenap/gw-dash/internal/grove"
)

// Model is the Bubble Tea model for the dashboard.
type Model struct {
	groveDir string
	width    int
	height   int

	// Agent state
	agents   []*grove.AgentState
	filtered []*grove.AgentState
	summary  grove.StatusSummary
	buckets  map[string][]*grove.AgentState

	// Navigation
	cursorCol  int // which kanban column
	cursorCard int // which card within column

	// Search
	searching   bool
	searchQuery string
}

// NewModel creates a new dashboard model.
func NewModel(groveDir string) Model {
	return Model{
		groveDir: groveDir,
		buckets:  make(map[string][]*grove.AgentState),
	}
}

// --- Messages ---

type tickMsg time.Time
type cleanupMsg time.Time

func tickCmd() tea.Cmd {
	return tea.Tick(grove.StatePollInterval, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func cleanupCmd() tea.Cmd {
	return tea.Tick(grove.CleanupInterval, func(t time.Time) tea.Msg {
		return cleanupMsg(t)
	})
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(tickCmd(), cleanupCmd())
}

// --- Update ---
//
// tea.Model requires value receivers for Init/Update/View.
// All mutation happens on the value copy and is returned.

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m.handleKey(msg)

	case tickMsg:
		m.refresh()
		return m, tickCmd()

	case cleanupMsg:
		grove.CleanupStale(m.groveDir)
		grove.ResetStalePermissions(m.groveDir)
		return m, cleanupCmd()

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.refresh()
		return m, nil
	}

	return m, nil
}

// refresh scans agent state and rebuilds all derived data.
func (m *Model) refresh() {
	agents, summary := grove.Scan(m.groveDir)
	m.agents = agents
	m.summary = summary
	m.refilter()
}

// refilter rebuilds filtered/bucketed data and clamps cursor.
func (m *Model) refilter() {
	m.filtered = grove.FilterAgents(m.agents, m.searchQuery)
	m.buckets = grove.BucketAgents(m.filtered)

	// Clamp cursor to valid range
	cols := grove.KanbanColumns
	if m.cursorCol < 0 {
		m.cursorCol = 0
	}
	if m.cursorCol >= len(cols) {
		m.cursorCol = len(cols) - 1
	}
	colID := cols[m.cursorCol].ID
	cards := m.buckets[colID]
	if m.cursorCard >= len(cards) {
		m.cursorCard = max(0, len(cards)-1)
	}
}

// focusedAgent returns the agent under the cursor, or nil.
func (m Model) focusedAgent() *grove.AgentState {
	cols := grove.KanbanColumns
	if m.cursorCol < 0 || m.cursorCol >= len(cols) {
		return nil
	}
	colID := cols[m.cursorCol].ID
	cards := m.buckets[colID]
	if m.cursorCard < 0 || m.cursorCard >= len(cards) {
		return nil
	}
	return cards[m.cursorCard]
}

// --- Key handling ---

func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.searching {
		return m.handleSearchKey(msg)
	}

	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit

	case "j", "down":
		cols := grove.KanbanColumns
		colID := cols[m.cursorCol].ID
		cards := m.buckets[colID]
		if len(cards) > 0 && m.cursorCard < len(cards)-1 {
			m.cursorCard++
		}

	case "k", "up":
		if m.cursorCard > 0 {
			m.cursorCard--
		}

	case "h", "left":
		m.moveColumn(-1)

	case "l", "right":
		m.moveColumn(1)

	case "enter":
		if agent := m.focusedAgent(); agent != nil {
			grove.ZellijJumpToAgent(agent.ProjectName, agent.CWD)
		}

	case "y":
		if agent := m.focusedAgent(); agent != nil && agent.Status == grove.StatusWaitingPerm {
			if grove.ZellijJumpToAgent(agent.ProjectName, agent.CWD) {
				grove.ZellijApprove()
				grove.ZellijJumpToAgent("grove", "")
			}
		}

	case "n":
		if agent := m.focusedAgent(); agent != nil && agent.Status == grove.StatusWaitingPerm {
			if grove.ZellijJumpToAgent(agent.ProjectName, agent.CWD) {
				grove.ZellijDeny()
				grove.ZellijJumpToAgent("grove", "")
			}
		}

	case "r":
		m.refresh()

	case "/":
		m.searching = true
		m.searchQuery = ""
	}

	return m, nil
}

func (m *Model) moveColumn(dir int) {
	cols := grove.KanbanColumns
	numCols := len(cols)
	m.cursorCol = (m.cursorCol + dir + numCols) % numCols
	m.cursorCard = 0

	// Skip empty columns (at most one full loop)
	for range numCols {
		colID := cols[m.cursorCol].ID
		if len(m.buckets[colID]) > 0 {
			return
		}
		m.cursorCol = (m.cursorCol + dir + numCols) % numCols
	}
}

func (m Model) handleSearchKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.searching = false
		m.searchQuery = ""
		m.refilter()
	case "enter":
		m.searching = false
	case "backspace":
		if len(m.searchQuery) > 0 {
			m.searchQuery = m.searchQuery[:len(m.searchQuery)-1]
			m.refilter()
		}
	default:
		if len(msg.String()) == 1 {
			m.searchQuery += msg.String()
			m.refilter()
		}
	}
	return m, nil
}

// --- View ---

func (m Model) View() string {
	if m.width == 0 || m.height == 0 {
		return "Loading..."
	}

	var sections []string

	// Header
	header := renderHeader(m.summary, m.width)
	sections = append(sections, header)

	// Layout dimensions
	headerHeight := lipgloss.Height(header)
	statusBarHeight := 1
	boardHeight := m.height - headerHeight - statusBarHeight
	if boardHeight < 3 {
		boardHeight = 3
	}

	// Main split: board (2/3) + detail (1/3)
	boardWidth := m.width * 2 / 3
	detailWidth := m.width - boardWidth

	// Render kanban columns
	cols := grove.KanbanColumns
	colWidth := boardWidth / len(cols)
	var colViews []string
	for i, col := range cols {
		agents := m.buckets[col.ID]
		focusedIdx := -1
		if i == m.cursorCol {
			focusedIdx = m.cursorCard
		}
		colViews = append(colViews, renderColumn(col, agents, focusedIdx, i == m.cursorCol, colWidth, boardHeight))
	}
	board := lipgloss.JoinHorizontal(lipgloss.Top, colViews...)

	// Detail panel
	detail := renderDetail(m.focusedAgent(), detailWidth, boardHeight)

	// Join board + detail
	mainArea := lipgloss.JoinHorizontal(lipgloss.Top, board, detail)
	sections = append(sections, mainArea)

	// Status bar
	sections = append(sections, renderStatusBar(m.searching, m.searchQuery, m.width))

	return strings.Join(sections, "\n")
}
