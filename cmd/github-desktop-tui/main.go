package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	gitlocal "github.com/nicoddemus/github-desktop-tui/internal/git"
	"github.com/nicoddemus/github-desktop-tui/internal/store"
	"github.com/nicoddemus/github-desktop-tui/internal/tui"
)

func main() {
	// Initialize git backend
	gitOps := gitlocal.New(".")
	st := store.New()

	// Create and start the TUI
	model := tui.New(gitOps, st)
	p := tea.NewProgram(model, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

// Ensure lipgloss is used (referenced in tui package)
var _ = lipgloss.NewStyle
