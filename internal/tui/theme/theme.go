package theme

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/charmbracelet/lipgloss"
)

// Theme defines the color palette and derived styles for the TUI.
type Theme struct {
	// Core colors
	Primary    lipgloss.Color
	Background lipgloss.Color
	Surface    lipgloss.Color
	Text       lipgloss.Color
	Muted      lipgloss.Color
	Success    lipgloss.Color
	Warning    lipgloss.Color
	Error      lipgloss.Color
	Info       lipgloss.Color
	Accent     lipgloss.Color

	// Base styles
	BaseStyle      lipgloss.Style
	FocusedStyle   lipgloss.Style
	TitleStyle     lipgloss.Style
	StatusBarStyle lipgloss.Style
	SelectedStyle  lipgloss.Style

	// Panel styles
	PanelStyle       lipgloss.Style
	ActivePanelStyle lipgloss.Style

	// Diff styles
	DiffAdded   lipgloss.Style
	DiffDeleted lipgloss.Style

	// Source control styles
	FileTreeStyle lipgloss.Style
	BranchStyle   lipgloss.Style
	TagStyle      lipgloss.Style

	// Focus/border style
	FocusStyle lipgloss.Style
}

// Default returns the default theme with the user's palette.
func Default() *Theme {
	t := &Theme{
		Background: lipgloss.Color("#382a2a"),
		Surface:    lipgloss.Color("#4a3a3a"),
		Primary:    lipgloss.Color("#ff3d3d"),
		Accent:     lipgloss.Color("#ff3d3d"),
		Text:       lipgloss.Color("#e5ebbc"),
		Muted:      lipgloss.Color("#8dc4b7"),
		Success:    lipgloss.Color("#8dc4b7"),
		Warning:    lipgloss.Color("#ff9d7d"),
		Error:      lipgloss.Color("#ff3d3d"),
		Info:       lipgloss.Color("#8dc4b7"),
	}
	t.initStyles()
	return t
}

// initStyles creates all lipgloss styles derived from the theme colors.
func (t *Theme) initStyles() {
	t.BaseStyle = lipgloss.NewStyle().
		Foreground(t.Text).
		Background(t.Background)

	t.FocusedStyle = lipgloss.NewStyle().
		Foreground(t.Accent).
		Bold(true)

	t.TitleStyle = lipgloss.NewStyle().
		Foreground(t.Primary).
		Bold(true).
		Padding(0, 1)

	t.SelectedStyle = lipgloss.NewStyle().
		Foreground(t.Text).
		Background(lipgloss.Color("#5a3a3a"))

	t.StatusBarStyle = lipgloss.NewStyle().
		Foreground(t.Background).
		Background(t.Muted).
		Padding(0, 1)

	t.PanelStyle = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(t.Muted).
		Background(t.Surface).
		Foreground(t.Text).
		Padding(1, 2)

	t.ActivePanelStyle = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(t.Accent).
		Background(t.Surface).
		Foreground(t.Text).
		Padding(1, 2)

	// Diff styles
	t.DiffAdded = lipgloss.NewStyle().
		Foreground(t.Success).
		Background(t.Background).
		Padding(0, 1)

	t.DiffDeleted = lipgloss.NewStyle().
		Foreground(t.Error).
		Background(t.Background).
		Padding(0, 1)

	// File tree style
	t.FileTreeStyle = lipgloss.NewStyle().
		Foreground(t.Text).
		Background(t.Background).
		PaddingLeft(2)

	// Branch style
	t.BranchStyle = lipgloss.NewStyle().
		Foreground(t.Info).
		Background(t.Background).
		Bold(true).
		Padding(0, 1).
		MarginRight(1)

	// Tag style
	t.TagStyle = lipgloss.NewStyle().
		Foreground(t.Warning).
		Background(t.Background).
		Padding(0, 1).
		MarginRight(1)

	// Focus style — used for indicating which panel/view is active
	t.FocusStyle = lipgloss.NewStyle().
		Border(lipgloss.DoubleBorder()).
		BorderForeground(t.Accent).
		Padding(1, 2)
}

// Load reads a theme from a JSON file and merges it over the defaults.
func Load(path string) (*Theme, error) {
	t := Default()

	data, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		return t, nil // Return defaults if file not found
	}

	var raw struct {
		BG      string `json:"bg"`
		Surface string `json:"surface"`
		Accent  string `json:"accent"`
		Text    string `json:"text"`
		Muted   string `json:"muted"`
		Success string `json:"success"`
		Warning string `json:"warning"`
		Error   string `json:"error"`
		Info    string `json:"info"`
	}

	if err := json.Unmarshal(data, &raw); err != nil {
		return t, nil // Return defaults on parse error
	}

	if raw.BG != "" {
		t.Background = lipgloss.Color(raw.BG)
	}
	if raw.Surface != "" {
		t.Surface = lipgloss.Color(raw.Surface)
	}
	if raw.Accent != "" {
		t.Accent = lipgloss.Color(raw.Accent)
		t.Primary = lipgloss.Color(raw.Accent)
	}
	if raw.Text != "" {
		t.Text = lipgloss.Color(raw.Text)
	}
	if raw.Muted != "" {
		t.Muted = lipgloss.Color(raw.Muted)
	}
	if raw.Success != "" {
		t.Success = lipgloss.Color(raw.Success)
	}
	if raw.Warning != "" {
		t.Warning = lipgloss.Color(raw.Warning)
	}
	if raw.Error != "" {
		t.Error = lipgloss.Color(raw.Error)
	}
	if raw.Info != "" {
		t.Info = lipgloss.Color(raw.Info)
	}

	t.initStyles()
	return t, nil
}
