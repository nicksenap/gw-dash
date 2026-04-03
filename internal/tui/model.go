// Package tui implements the Bubble Tea TUI for the Grove dashboard.
package tui

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/nicksenap/gw-dash/internal/grove"
)

var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("10"))

	headerStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("4"))

	dimStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("8"))

	selectedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("14")).
			Bold(true)
)

// Model is the Bubble Tea model for the dashboard.
type Model struct {
	state    *grove.State
	groveDir string
	cursor   int
	width    int
	height   int
}

// NewModel creates a new dashboard model.
func NewModel(state *grove.State, groveDir string) Model {
	return Model{
		state:    state,
		groveDir: groveDir,
	}
}

type tickMsg time.Time

func tickCmd() tea.Cmd {
	return tea.Tick(2*time.Second, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func (m Model) Init() tea.Cmd {
	return tickCmd()
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "j", "down":
			if m.cursor < len(m.state.Workspaces)-1 {
				m.cursor++
			}
		case "k", "up":
			if m.cursor > 0 {
				m.cursor--
			}
		case "r":
			// Reload state
			state, err := grove.LoadState(m.groveDir)
			if err == nil {
				m.state = state
				if m.cursor >= len(m.state.Workspaces) {
					m.cursor = max(0, len(m.state.Workspaces)-1)
				}
			}
		}

	case tickMsg:
		// Periodic reload
		state, err := grove.LoadState(m.groveDir)
		if err == nil {
			m.state = state
			if m.cursor >= len(m.state.Workspaces) {
				m.cursor = max(0, len(m.state.Workspaces)-1)
			}
		}
		return m, tickCmd()

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	}

	return m, nil
}

func (m Model) View() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render("  Grove Dashboard"))
	b.WriteString("\n\n")

	if len(m.state.Workspaces) == 0 {
		b.WriteString(dimStyle.Render("  No workspaces. Create one with: gw create <branch>"))
		b.WriteString("\n")
	} else {
		// Header
		b.WriteString(headerStyle.Render(fmt.Sprintf("  %-20s %-30s %-5s", "WORKSPACE", "BRANCH", "REPOS")))
		b.WriteString("\n")

		for i, ws := range m.state.Workspaces {
			prefix := "  "
			style := lipgloss.NewStyle()
			if i == m.cursor {
				prefix = "▸ "
				style = selectedStyle
			}

			line := fmt.Sprintf("%s%-20s %-30s %-5d",
				prefix,
				truncate(ws.Name, 20),
				truncate(ws.Branch, 30),
				len(ws.Repos),
			)
			b.WriteString(style.Render(line))
			b.WriteString("\n")
		}
	}

	b.WriteString("\n")
	b.WriteString(dimStyle.Render("  j/k navigate • r refresh • q quit"))
	b.WriteString("\n")

	return b.String()
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n-1] + "…"
}
