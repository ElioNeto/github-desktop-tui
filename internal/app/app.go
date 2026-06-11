package app

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	gitlocal "github.com/ElioNeto/github-desktop-tui/internal/git"
	"github.com/ElioNeto/github-desktop-tui/internal/store"
	"github.com/ElioNeto/github-desktop-tui/internal/tui"
)

// Run starts the TUI application.
func Run() error {
	gitOps := gitlocal.New(".")
	st := store.New()

	p := tea.NewProgram(tui.New(gitOps, st), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		return fmt.Errorf("TUI error: %w", err)
	}
	return nil
}

// Keep os import used
var _ = os.DevNull
