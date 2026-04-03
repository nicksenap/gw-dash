package tui

import "github.com/charmbracelet/lipgloss"

// Gruvbox dark palette.
var (
	green  = lipgloss.Color("#b8bb26")
	aqua   = lipgloss.Color("#8ec07c")
	red    = lipgloss.Color("#fb4934")
	yellow = lipgloss.Color("#fabd2f")
	grey   = lipgloss.Color("#928374")
	fg     = lipgloss.Color("#fbf1c7")
	orange = lipgloss.Color("#fe8019")
	purple = lipgloss.Color("#d3869b")
	panel  = lipgloss.Color("#504945")
)

// Common styles.
var (
	headerStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(fg).
			Padding(0, 1)

	statusBarStyle = lipgloss.NewStyle().
			Foreground(grey).
			Padding(0, 1)

	columnStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(panel).
			Padding(0, 0)

	columnActiveStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#85A598")).
			Padding(0, 0)

	cardStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(panel).
			Padding(0, 1).
			Width(0) // set dynamically

	cardFocusedStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#85A598")).
			Padding(0, 1).
			Width(0)

	detailPaneStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(panel).
			Padding(0, 1)

)
