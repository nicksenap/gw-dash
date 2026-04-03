package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/nicksenap/gw-dash/internal/grove"
	"github.com/nicksenap/gw-dash/internal/tui"
)

var Version = "dev"

func main() {
	if len(os.Args) > 1 && os.Args[1] == "--version" {
		fmt.Printf("gw-dash %s\n", Version)
		os.Exit(0)
	}

	groveDir := grove.Dir()

	m := tui.NewModel(groveDir)
	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %s\n", err)
		os.Exit(1)
	}
}
