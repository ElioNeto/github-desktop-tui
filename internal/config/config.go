package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// Config holds all application configuration.
type Config struct {
	// ActiveProvider is the name of the currently active Git provider.
	ActiveProvider string `json:"active_provider"`

	// ThemeFile is the path to the theme JSON file.
	ThemeFile string `json:"theme_file"`

	// DataDir is the directory for storing app data (credentials, cache).
	DataDir string `json:"data_dir"`

	// KeybindingsFile is the path to custom keybindings.
	KeybindingsFile string `json:"keybindings_file,omitempty"`

	// Providers holds per-provider configuration.
	Providers map[string]ProviderConfig `json:"providers"`
}

// ProviderConfig holds configuration for a specific Git provider.
type ProviderConfig struct {
	Enabled bool   `json:"enabled"`
	BaseURL string `json:"base_url,omitempty"` // For self-hosted instances
	Token   string `json:"-"`                   // Never serialized
}

// Load reads configuration from the default location.
func Load() (*Config, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		configDir = "~/.config"
	}

	appDir := filepath.Join(configDir, "github-desktop-tui")
	cfgFile := filepath.Join(appDir, "config.json")

	cfg := Default()

	data, err := os.ReadFile(cfgFile)
	if err != nil {
		if os.IsNotExist(err) {
			// Create config directory and return defaults
			if err := os.MkdirAll(appDir, 0700); err != nil {
				return nil, fmt.Errorf("criar diretório de config: %w", err)
			}
			return cfg, nil
		}
		return nil, fmt.Errorf("ler config: %w", err)
	}

	if err := json.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parse config JSON: %w", err)
	}

	return cfg, nil
}

// Save writes the configuration to disk.
func (c *Config) Save() error {
	configDir, err := os.UserConfigDir()
	if err != nil {
		configDir = "~/.config"
	}

	appDir := filepath.Join(configDir, "github-desktop-tui")
	cfgFile := filepath.Join(appDir, "config.json")

	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Errorf("serializar config: %w", err)
	}

	if err := os.WriteFile(cfgFile, data, 0600); err != nil {
		return fmt.Errorf("escrever config: %w", err)
	}

	return nil
}
