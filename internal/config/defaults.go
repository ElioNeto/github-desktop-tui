package config

import (
	"os"
	"path/filepath"
)

// Default returns a Config with sensible defaults.
func Default() *Config {
	configDir, err := os.UserConfigDir()
	if err != nil {
		configDir = "~/.config"
	}

	appDir := filepath.Join(configDir, "github-desktop-tui")

	return &Config{
		ActiveProvider: "github",
		ThemeName:      "dark",
		ThemeFile:      filepath.Join(appDir, "theme.json"),
		DataDir:        filepath.Join(appDir, "data"),
		Providers: map[string]ProviderConfig{
			"github": {
				Enabled: true,
				BaseURL: "https://api.github.com",
			},
			"gitlab": {
				Enabled: false,
				BaseURL: "https://gitlab.com/api/v4",
			},
			"bitbucket": {
				Enabled: false,
				BaseURL: "https://api.bitbucket.org/2.0",
			},
			"gitea": {
				Enabled: false,
				BaseURL: "",
			},
		},
	}
}
