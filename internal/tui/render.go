package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/nicksenap/gw-dash/internal/grove"
)

// truncate safely truncates a string to n runes (not bytes).
func truncate(s string, n int) string {
	runes := []rune(s)
	if len(runes) <= n {
		return s
	}
	return string(runes[:n])
}

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
	parts = append(parts, boldFG.Render("\u26a1\ufe0e gw dash"))
	parts = append(parts, "  ")

	if summary.Total == 0 {
		parts = append(parts, dimGrey.Render("No agents"))
	} else {
		parts = append(parts, dimGrey.Render(fmt.Sprintf("agents: %d", summary.Total)))
		parts = append(parts, "  ")
		if summary.Working > 0 {
			parts = append(parts, fgGreen.Render(fmt.Sprintf(">>>%d", summary.Working)))
			parts = append(parts, "  ")
		}
		if summary.WaitingPerm > 0 {
			parts = append(parts, boldRed.Render(fmt.Sprintf("[!]%d", summary.WaitingPerm)))
			parts = append(parts, "  ")
		}
		if summary.WaitingAnswer > 0 {
			parts = append(parts, boldYellow.Render(fmt.Sprintf("[?]%d", summary.WaitingAnswer)))
			parts = append(parts, "  ")
		}
		if summary.Error > 0 {
			parts = append(parts, fgOrange.Render(fmt.Sprintf("[X]%d", summary.Error)))
			parts = append(parts, "  ")
		}
		if summary.Idle > 0 {
			parts = append(parts, dimGrey.Render(fmt.Sprintf("---%d", summary.Idle)))
		}
	}

	// Claude usage
	usage := grove.ReadUsageCache()
	if usage != nil {
		uColor := lipgloss.Color(grove.UsageColor(usage.Utilization))
		staleStr := ""
		if usage.Stale {
			staleStr = dimGrey.Render(" stale")
		}
		resetStr := ""
		if cd := usage.ResetCountdown(); cd != "" {
			resetStr = " → " + cd
		}
		parts = append(parts, "  ")
		parts = append(parts, dimGrey.Render("│"))
		parts = append(parts, "  ")
		parts = append(parts, dimGrey.Render("usage: "))
		parts = append(parts, lipgloss.NewStyle().Foreground(uColor).Render(
			fmt.Sprintf("%d%% %s%s", usage.Utilization, usage.Bar(), resetStr),
		))
		parts = append(parts, staleStr)
	}

	line := strings.Join(parts, "")
	// MaxWidth prevents wrapping — truncates instead of creating a second line
	return headerStyle.MaxWidth(width).Render(line)
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
	if name == "" && len([]rune(agent.SessionID)) > 12 {
		name = truncate(agent.SessionID, 12)
	} else if name == "" {
		name = agent.SessionID
	}
	line1 := boldFG.Render(name) +
		"  " + lipgloss.NewStyle().Foreground(sColor).Render(sd.Label)
	lines = append(lines, line1)

	// Line 2: Branch + tool info
	var line2Parts []string
	if agent.GitBranch != "" {
		line2Parts = append(line2Parts, fgAqua.Render(agent.GitBranch))
	}
	if agent.LastTool != "" {
		ago := idleAgo(agent.IdleSeconds())
		toolStr := agent.LastTool
		if ago != "" {
			toolStr += " " + dimGrey.Render(ago)
		}
		line2Parts = append(line2Parts, toolStr)
	}
	if len(line2Parts) > 0 {
		lines = append(lines, strings.Join(line2Parts, "  "))
	}

	// Line 3: Counts + sparkline
	var meta []string
	if agent.ToolCount > 0 {
		meta = append(meta, dimGrey.Render(fmt.Sprintf("%d tools", agent.ToolCount)))
	}
	if agent.ErrorCount > 0 {
		meta = append(meta, fgOrange.Render(fmt.Sprintf("%d err", agent.ErrorCount)))
	}
	if agent.SubagentCount > 0 {
		meta = append(meta, fgAqua.Render(fmt.Sprintf("+%d sub", agent.SubagentCount)))
	}
	if uptime := agent.Uptime(); uptime != "" {
		meta = append(meta, dimGrey.Render(uptime))
	}
	if spark := agent.Sparkline(); spark != "" {
		meta = append(meta, fgGreen.Render(spark))
	}
	if len(meta) > 0 {
		lines = append(lines, strings.Join(meta, "  "))
	}

	// Line 4: Prompt snippet
	if agent.InitialPrompt != "" {
		prompt := strings.ReplaceAll(agent.InitialPrompt, "\n", " ")
		prompt = truncate(prompt, 60)
		lines = append(lines, dimGrey.Render(prompt))
	}

	// Special states
	if agent.Status == grove.StatusWaitingPerm && agent.ToolRequestSummary != nil {
		summary := strings.SplitN(*agent.ToolRequestSummary, "\n", 2)[0]
		summary = truncate(summary, 60)
		lines = append(lines,
			fgRed.Render("PERM: "+agent.LastTool)+" "+
				dimGrey.Render(summary))
	}

	if agent.Status == grove.StatusError && agent.LastError != "" {
		errMsg := strings.ReplaceAll(agent.LastError, "\n", " ")
		errMsg = truncate(errMsg, 60)
		lines = append(lines, fgOrange.Render(errMsg))
	}

	if agent.NotificationMessage != nil && *agent.NotificationMessage != "" {
		msg := *agent.NotificationMessage
		msg = truncate(msg, 60)
		lines = append(lines, fgPurple.Render(msg))
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
	titleLine := colTitle.Render(title)

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
	detailTitle := colTitle.Render("Detail")

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

	pad := padded

	// Workspace or CWD
	if agent.WorkspaceName != "" {
		lines = append(lines, pad.Render(
			dimGrey.Render("workspace: ")+
				fgAqua.Render(agent.WorkspaceName)))
		if len(agent.WorkspaceRepos) > 0 {
			lines = append(lines, pad.Render(
				dimGrey.Render("repos:     ")+
					strings.Join(agent.WorkspaceRepos, ", ")))
		}
	} else if agent.CWD != "" {
		lines = append(lines, pad.Render(
			dimGrey.Render("cwd:    ")+agent.CWD))
	}

	if agent.GitBranch != "" {
		dirty := ""
		if agent.GitDirtyCount > 0 {
			dirty = fmt.Sprintf(" (%d dirty)", agent.GitDirtyCount)
		}
		lines = append(lines, pad.Render(
			dimGrey.Render("branch: ")+
				fgAqua.Render(agent.GitBranch)+dirty))
	}

	if agent.Model != "" {
		modelStr := agent.Model
		if agent.PermissionMode != "" && agent.PermissionMode != "default" {
			modelStr += "  " + fgYellow.Render(agent.PermissionMode)
		}
		lines = append(lines, pad.Render(
			dimGrey.Render("model:  ")+modelStr))
	}

	if uptime := agent.Uptime(); uptime != "" {
		sourceTag := ""
		if agent.SessionSource != "" && agent.SessionSource != "startup" {
			sourceTag = " " + fgAqua.Render("("+agent.SessionSource+")")
		}
		lines = append(lines, pad.Render(
			dimGrey.Render("uptime: ")+uptime+sourceTag))
	}

	lines = append(lines, "")
	lines = append(lines, pad.Render(
		dimGrey.Render("tools:  ")+fmt.Sprintf("%d", agent.ToolCount)+
			dimGrey.Render("    errors: ")+fmt.Sprintf("%d", agent.ErrorCount)+
			dimGrey.Render("    subs: ")+fmt.Sprintf("%d", agent.SubagentCount)))

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
			dimGrey.Render("last:   ")+agent.LastTool+" ("+ago+")"))
	}

	if spark := agent.Sparkline(); spark != "" {
		lines = append(lines, pad.Render(
			dimGrey.Render("activity: ")+
				fgGreen.Render(spark)))
	}

	if agent.CompactCount > 0 {
		trigger := ""
		if agent.CompactTrigger != "" {
			trigger = " (" + agent.CompactTrigger + ")"
		}
		lines = append(lines, pad.Render(fgYellow.Render(
			fmt.Sprintf("compacted: %dx%s", agent.CompactCount, trigger))))
	}

	if len(agent.ActiveSubagents) > 0 {
		lines = append(lines, pad.Render(
			dimGrey.Render("agents:  ")+
				fgAqua.Render(strings.Join(agent.ActiveSubagents, ", "))))
	}

	// Initial prompt
	if agent.InitialPrompt != "" {
		lines = append(lines, "")
		prompt := strings.ReplaceAll(agent.InitialPrompt, "\n", " ")
		prompt = truncate(prompt, 120)
		lines = append(lines, pad.Render(
			dimGrey.Render("prompt: ")+prompt))
	}

	// Last message
	if agent.LastMessage != "" && agent.Status == grove.StatusIdle {
		lines = append(lines, "")
		msg := strings.ReplaceAll(agent.LastMessage, "\n", " ")
		msg = truncate(msg, 200)
		lines = append(lines, pad.Render(
			dimGrey.Render("last reply: ")+msg))
	}

	// Last error
	if agent.LastError != "" && agent.Status == grove.StatusError {
		lines = append(lines, "")
		errStr := agent.LastError
		errStr = truncate(errStr, 200)
		lines = append(lines, pad.Render(
			fgRed.Render("error: ")+errStr))
	}

	// Permission request detail
	if agent.Status == grove.StatusWaitingPerm && agent.ToolRequestSummary != nil {
		lines = append(lines, "")
		lines = append(lines, pad.Render(boldRed.Render("Permission Request")))
		lines = append(lines, pad.Render(lipgloss.NewStyle().Bold(true).Render("Tool: ")+agent.LastTool))
		lines = append(lines, "")
		summaryLines := strings.Split(*agent.ToolRequestSummary, "\n")
		for i, line := range summaryLines {
			if i >= 10 {
				break
			}
			if strings.HasPrefix(line, "+ ") {
				lines = append(lines, pad.Render(fgGreen.Render(line)))
			} else if strings.HasPrefix(line, "- ") {
				lines = append(lines, pad.Render(fgRed.Render(line)))
			} else if strings.HasPrefix(line, "$ ") {
				lines = append(lines, pad.Render(fgYellow.Render(line)))
			} else {
				lines = append(lines, pad.Render(line))
			}
		}
	}

	// Notification message
	if agent.NotificationMessage != nil && *agent.NotificationMessage != "" {
		lines = append(lines, "")
		lines = append(lines, pad.Render(
			fgPurple.Render("Notification: ")+*agent.NotificationMessage))
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
		dimGrey.Render("q") + " quit  " +
			dimGrey.Render("h/l") + " columns  " +
			dimGrey.Render("j/k") + " cards  " +
			dimGrey.Render("enter") + " jump  " +
			dimGrey.Render("y/n") + " approve/deny  " +
			dimGrey.Render("r") + " refresh  " +
			dimGrey.Render("/") + " search",
	)
}
