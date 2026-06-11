package app

import "github.com/nicoddemus/github-desktop-tui/internal/config"

// ConfigLoadedMsg is sent after configuration is fully loaded.
type ConfigLoadedMsg struct {
	Config *config.Config
}
