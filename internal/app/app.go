package app

import (
	"github.com/nicoddemus/github-desktop-tui/internal/config"
	"github.com/nicoddemus/github-desktop-tui/internal/desktop"
)

// Run starts the desktop application.
func Run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	return desktop.Run(cfg)
}
