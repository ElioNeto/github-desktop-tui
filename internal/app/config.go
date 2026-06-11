package app

import "github.com/ElioNeto/github-desktop-tui/internal/config"

// ConfigLoadedMsg is sent after configuration is fully loaded.
type ConfigLoadedMsg struct {
	Config *config.Config
}
