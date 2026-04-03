package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/nicksenap/gw-dash/internal/grove"
)

// idleAgo formats idle seconds as a human-readable age string.
func idleAgo(seconds float64) string {
	if seconds < 5 {
		return ""
	}
	if seconds < 60 {
		return fmt.Sprintf("%ds", int(seconds))
	}
	if seconds < 3600 {
		return fmt.Sprintf("%dm", int(seconds)/60)
	}
	return fmt.Sprintf("%dh", int(seconds)/3600)
}

// renderHeader renders the top header bar with summary and usage.
func renderHeader(summary grove.StatusSummary, width int) string {
	var parts []string
	parts = append(parts, lipgloss.NewStyle().Bold(true).Foreground(fg).Render("⚎ gw dash"))
	parts = append(parts, "  ")

	if summary.Total == 0 {
		parts = append(parts, lipgloss.NewStyle().Foreground(grey).Render("No agents"))
	} else {
		parts = append(parts, lipgloss.NewStyle().Foreground(grey).Render(fmt.Sprintf("agents: %d", summary.Total)))
		parts = append(parts, "  ")
		if summary.Working > 0 {
			parts = append(parts, lipgloss.NewStyle().Foreground(green).Render(fmt.Sprintf(">>>%d", summary.Working)))
			parts = append(parts, "  ")
		}
		if summary.WaitingPerm > 0 {
			parts = append(parts, lipgloss.NewStyle().Bold(true).Foreground(red).Render(fmt.Sprintf("[!]%d", summary.WaitingPerm)))
			parts = append(parts, "  ")
		}
		if summary.WaitingAnswer > 0 {
			parts = append(parts, lipgloss.NewStyle().Bold(true).Foreground(yellow).Render(fmt.Sprintf("[?]%d", summary.WaitingAnswer)))
			parts = append(parts, "  ")
		}
		if summary.Error > 0 {
			parts = append(parts, lipgloss.NewStyle().Foreground(orange).Render(fmt.Sprintf("[X]%d", summary.Error)))
			parts = append(parts, "  ")
		}
		if summary.Idle > 0 {
			parts = append(parts, lipgloss.NewStyle().Foreground(grey).Render(fmt.Sprintf("---%d", summary.Idle)))
		}
	}

	// Claude usage
	usage := grove.ReadUsageCache()
	if usage != nil {
		uColor := lipgloss.Color(grove.UsageColor(usage.Utilization))
		staleStr := ""
		if usage.Stale {
			staleStr = lipgloss.NewStyle().Foreground(grey).Render(" stale")
		}
		resetStr := ""
		if cd := usage.ResetCountdown(); cd != "" {
			resetStr = " → " + cd
		}
		parts = append(parts, "  ")
		parts = append(parts, lipgloss.NewStyle().Foreground(grey).Render("│"))
		parts = append(parts, "  ")
		parts = append(parts, lipgloss.NewStyle().Foreground(grey).Render("usage: "))
		parts = append(parts, lipgloss.NewStyle().Foreground(uColor).Render(
			fmt.Sprintf("%d%% %s%s", usage.Utilization, usage.Bar(), resetStr),
		))
		parts = append(parts, staleStr)
	}

	line := strings.Join(parts, "")
	return headerStyle.Width(width).Render(line)
}

// renderCard renders a single agent task card.
func renderCard(agent *grove.AgentState, focused bool, width int) string {
	sd, ok := grove.StatusDisplayMap[agent.Status]
	if !ok {
		sd = grove.StatusDisplay{Color: grove.ColorGrey, Label: "?"}
	}
	sColor := lipgloss.Color(sd.Color)

	var lines []string

	// Line 1: Name + status badge
	name := agent.DisplayName
	if name == "" && len(agent.SessionID) > 12 {
		name = agent.SessionID[:12]
	} else if name == "" {
		name = agent.SessionID
	}
	line1 := lipgloss.NewStyle().Bold(true).Foreground(fg).Render(name) +
		"  " + lipgloss.NewStyle().Foreground(sColor).Render(sd.Label)
	lines = append(lines, line1)

	// Line 2: Branch + tool info
	var line2Parts []string
	if agent.GitBranch != "" {
		line2Parts = append(line2Parts, lipgloss.NewStyle().Foreground(aqua).Render(agent.GitBranch))
	}
	if agent.LastTool != "" {
		ago := idleAgo(agent.IdleSeconds())
		toolStr := agent.LastTool
		if ago != "" {
			toolStr += " " + lipgloss.NewStyle().Foreground(grey).Render(ago)
		}
		line2Parts = append(line2Parts, toolStr)
	}
	if len(line2Parts) > 0 {
		lines = append(lines, strings.Join(line2Parts, "  "))
	}

	// Line 3: Counts + sparkline
	var meta []string
	if agent.ToolCount > 0 {
		meta = append(meta, lipgloss.NewStyle().Foreground(grey).Render(fmt.Sprintf("%d tools", agent.ToolCount)))
	}
	if agent.ErrorCount > 0 {
		meta = append(meta, lipgloss.NewStyle().Foreground(orange).Render(fmt.Sprintf("%d err", agent.ErrorCount)))
	}
	if agent.SubagentCount > 0 {
		meta = append(meta, lipgloss.NewStyle().Foreground(aqua).Render(fmt.Sprintf("+%d sub", agent.SubagentCount)))
	}
	if uptime := agent.Uptime(); uptime != "" {
		meta = append(meta, lipgloss.NewStyle().Foreground(grey).Render(uptime))
	}
	if spark := agent.Sparkline(); spark != "" {
		meta = append(meta, lipgloss.NewStyle().Foreground(green).Render(spark))
	}
	if len(meta) > 0 {
		lines = append(lines, strings.Join(meta, "  "))
	}

	// Line 4: Prompt snippet
	if agent.InitialPrompt != "" {
		prompt := strings.ReplaceAll(agent.InitialPrompt, "\n", " ")
		if len(prompt) > 60 {
			prompt = prompt[:60]
		}
		lines = append(lines, lipgloss.NewStyle().Foreground(grey).Render(prompt))
	}

	// Special states
	if agent.Status == grove.StatusWaitingPerm && agent.ToolRequestSummary != nil {
		summary := strings.SplitN(*agent.ToolRequestSummary, "\n", 2)[0]
		if len(summary) > 60 {
			summary = summary[:60]
		}
		lines = append(lines,
			lipgloss.NewStyle().Foreground(red).Render("PERM: "+agent.LastTool)+" "+
				lipgloss.NewStyle().Foreground(grey).Render(summary))
	}

	if agent.Status == grove.StatusError && agent.LastError != "" {
		errMsg := strings.ReplaceAll(agent.LastError, "\n", " ")
		if len(errMsg) > 60 {
			errMsg = errMsg[:60]
		}
		lines = append(lines, lipgloss.NewStyle().Foreground(orange).Render(errMsg))
	}

	if agent.NotificationMessage != nil && *agent.NotificationMessage != "" {
		msg := *agent.NotificationMessage
		if len(msg) > 60 {
			msg = msg[:60]
		}
		lines = append(lines, lipgloss.NewStyle().Foreground(purple).Render(msg))
	}

	content := strings.Join(lines, "\n")

	style := cardStyle
	if focused {
		style = cardFocusedStyle.BorderForeground(sColor)
	}
	style = style.Width(width - 2)

	return style.Render(content)
}

// renderColumn renders a kanban column with its cards.
func renderColumn(col grove.KanbanColumn, agents []*grove.AgentState, focusedIdx int, isActiveCol bool, width, height int) string {
	// Column title line (rendered inside the box, below the border)
	title := col.Title
	if len(agents) > 0 {
		title = fmt.Sprintf("%s (%d)", col.Title, len(agents))
	}
	titleLine := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#85A598")).
		Padding(0, 1).
		Render(title)

	var cardViews []string
	for i, agent := range agents {
		cardViews = append(cardViews, renderCard(agent, i == focusedIdx, width-2))
	}

	// Title + cards
	var contentParts []string
	contentParts = append(contentParts, titleLine)
	if len(cardViews) > 0 {
		contentParts = append(contentParts, strings.Join(cardViews, "\n"))
	}
	content := strings.Join(contentParts, "\n")

	style := columnStyle
	if isActiveCol {
		style = columnActiveStyle
	}

	return style.Width(width - 2).Height(height - 2).Render(content)
}

// renderDetail renders the detail panel for the selected agent.
func renderDetail(agent *grove.AgentState, width, height int) string {
	// Title line
	detailTitle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#85A598")).
		Padding(0, 1).
		Render("Detail")

	if agent == nil {
		content := detailTitle + "\n\n" +
			lipgloss.NewStyle().Foreground(grey).Padding(0, 1).Render("No agent selected")
		return detailPaneStyle.Width(width - 2).Height(height - 2).Render(content)
	}

	sd, ok := grove.StatusDisplayMap[agent.Status]
	if !ok {
		sd = grove.StatusDisplay{Color: grove.ColorGrey, Label: "?"}
	}
	sColor := lipgloss.Color(sd.Color)

	var lines []string
	lines = append(lines, detailTitle)
	lines = append(lines, "")

	// Title + status
	lines = append(lines, lipgloss.NewStyle().Bold(true).Foreground(fg).Padding(0, 1).Render(agent.DisplayName)+
		"  "+lipgloss.NewStyle().Foreground(sColor).Render(sd.Label))
	lines = append(lines, "")

	pad := lipgloss.NewStyle().Padding(0, 1)

	// Workspace or CWD
	if agent.WorkspaceName != "" {
		lines = append(lines, pad.Render(
			lipgloss.NewStyle().Foreground(grey).Render("workspace: ")+
				lipgloss.NewStyle().Foreground(aqua).Render(agent.WorkspaceName)))
		if len(agent.WorkspaceRepos) > 0 {
			lines = append(lines, pad.Render(
				lipgloss.NewStyle().Foreground(grey).Render("repos:     ")+
					strings.Join(agent.WorkspaceRepos, ", ")))
		}
	} else if agent.CWD != "" {
		lines = append(lines, pad.Render(
			lipgloss.NewStyle().Foreground(grey).Render("cwd:    ")+agent.CWD))
	}

	if agent.GitBranch != "" {
		dirty := ""
		if agent.GitDirtyCount > 0 {
			dirty = fmt.Sprintf(" (%d dirty)", agent.GitDirtyCount)
		}
		lines = append(lines, pad.Render(
			lipgloss.NewStyle().Foreground(grey).Render("branch: ")+
				lipgloss.NewStyle().Foreground(aqua).Render(agent.GitBranch)+dirty))
	}

	if agent.Model != "" {
		modelStr := agent.Model
		if agent.PermissionMode != "" && agent.PermissionMode != "default" {
			modelStr += "  " + lipgloss.NewStyle().Foreground(yellow).Render(agent.PermissionMode)
		}
		lines = append(lines, pad.Render(
			lipgloss.NewStyle().Foreground(grey).Render("model:  ")+modelStr))
	}

	if uptime := agent.Uptime(); uptime != "" {
		sourceTag := ""
		if agent.SessionSource != "" && agent.SessionSource != "startup" {
			sourceTag = " " + lipgloss.NewStyle().Foreground(aqua).Render("("+agent.SessionSource+")")
		}
		lines = append(lines, pad.Render(
			lipgloss.NewStyle().Foreground(grey).Render("uptime: ")+uptime+sourceTag))
	}

	lines = append(lines, "")
	lines = append(lines, pad.Render(
		lipgloss.NewStyle().Foreground(grey).Render("tools:  ")+fmt.Sprintf("%d", agent.ToolCount)+
			lipgloss.NewStyle().Foreground(grey).Render("    errors: ")+fmt.Sprintf("%d", agent.ErrorCount)+
			lipgloss.NewStyle().Foreground(grey).Render("    subs: ")+fmt.Sprintf("%d", agent.SubagentCount)))

	if agent.LastTool != "" {
		idle := agent.IdleSeconds()
		var ago string
		if idle < 60 {
			ago = fmt.Sprintf("%ds ago", int(idle))
		} else if idle < 3600 {
			ago = fmt.Sprintf("%dm ago", int(idle)/60)
		} else {
			ago = fmt.Sprintf("%dh ago", int(idle)/3600)
		}
		lines = append(lines, pad.Render(
			lipgloss.NewStyle().Foreground(grey).Render("last:   ")+agent.LastTool+" ("+ago+")"))
	}

	if spark := agent.Sparkline(); spark != "" {
		lines = append(lines, pad.Render(
			lipgloss.NewStyle().Foreground(grey).Render("activity: ")+
				lipgloss.NewStyle().Foreground(green).Render(spark)))
	}

	if agent.CompactCount > 0 {
		trigger := ""
		if agent.CompactTrigger != "" {
			trigger = " (" + agent.CompactTrigger + ")"
		}
		lines = append(lines, pad.Render(lipgloss.NewStyle().Foreground(yellow).Render(
			fmt.Sprintf("compacted: %dx%s", agent.CompactCount, trigger))))
	}

	if len(agent.ActiveSubagents) > 0 {
		lines = append(lines, pad.Render(
			lipgloss.NewStyle().Foreground(grey).Render("agents:  ")+
				lipgloss.NewStyle().Foreground(aqua).Render(strings.Join(agent.ActiveSubagents, ", "))))
	}

	// Initial prompt
	if agent.InitialPrompt != "" {
		lines = append(lines, "")
		prompt := strings.ReplaceAll(agent.InitialPrompt, "\n", " ")
		if len(prompt) > 120 {
			prompt = prompt[:120]
		}
		lines = append(lines, pad.Render(
			lipgloss.NewStyle().Foreground(grey).Render("prompt: ")+prompt))
	}

	// Last message
	if agent.LastMessage != "" && agent.Status == grove.StatusIdle {
		lines = append(lines, "")
		msg := strings.ReplaceAll(agent.LastMessage, "\n", " ")
		if len(msg) > 200 {
			msg = msg[:200]
		}
		lines = append(lines, pad.Render(
			lipgloss.NewStyle().Foreground(grey).Render("last reply: ")+msg))
	}

	// Last error
	if agent.LastError != "" && agent.Status == grove.StatusError {
		lines = append(lines, "")
		errStr := agent.LastError
		if len(errStr) > 200 {
			errStr = errStr[:200]
		}
		lines = append(lines, pad.Render(
			lipgloss.NewStyle().Foreground(red).Render("error: ")+errStr))
	}

	// Permission request detail
	if agent.Status == grove.StatusWaitingPerm && agent.ToolRequestSummary != nil {
		lines = append(lines, "")
		lines = append(lines, pad.Render(lipgloss.NewStyle().Bold(true).Foreground(red).Render("Permission Request")))
		lines = append(lines, pad.Render(lipgloss.NewStyle().Bold(true).Render("Tool: ")+agent.LastTool))
		lines = append(lines, "")
		summaryLines := strings.Split(*agent.ToolRequestSummary, "\n")
		for i, line := range summaryLines {
			if i >= 10 {
				break
			}
			if strings.HasPrefix(line, "+ ") {
				lines = append(lines, pad.Render(lipgloss.NewStyle().Foreground(green).Render(line)))
			} else if strings.HasPrefix(line, "- ") {
				lines = append(lines, pad.Render(lipgloss.NewStyle().Foreground(red).Render(line)))
			} else if strings.HasPrefix(line, "$ ") {
				lines = append(lines, pad.Render(lipgloss.NewStyle().Foreground(yellow).Render(line)))
			} else {
				lines = append(lines, pad.Render(line))
			}
		}
	}

	// Notification message
	if agent.NotificationMessage != nil && *agent.NotificationMessage != "" {
		lines = append(lines, "")
		lines = append(lines, pad.Render(
			lipgloss.NewStyle().Foreground(purple).Render("Notification: ")+*agent.NotificationMessage))
	}

	content := strings.Join(lines, "\n")
	return detailPaneStyle.Width(width - 2).Height(height - 2).Render(content)
}

// renderStatusBar renders the bottom status bar.
func renderStatusBar(searching bool, searchQuery string, width int) string {
	if searching {
		return statusBarStyle.Width(width).Render("/" + searchQuery + "█")
	}
	return statusBarStyle.Width(width).Render(
		lipgloss.NewStyle().Foreground(grey).Render("q") + " quit  " +
			lipgloss.NewStyle().Foreground(grey).Render("h/l") + " columns  " +
			lipgloss.NewStyle().Foreground(grey).Render("j/k") + " cards  " +
			lipgloss.NewStyle().Foreground(grey).Render("enter") + " jump  " +
			lipgloss.NewStyle().Foreground(grey).Render("y/n") + " approve/deny  " +
			lipgloss.NewStyle().Foreground(grey).Render("r") + " refresh  " +
			lipgloss.NewStyle().Foreground(grey).Render("/") + " search",
	)
}
