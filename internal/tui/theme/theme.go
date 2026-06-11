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
	DimmedStyle    lipgloss.Style

	// Panel styles
	PanelStyle       lipgloss.Style
	ActivePanelStyle lipgloss.Style

	// Panel title bars
	PanelTitleStyle           lipgloss.Style
	ActivePanelTitleStyle     lipgloss.Style

	// Diff styles
	DiffAdded   lipgloss.Style
	DiffDeleted lipgloss.Style
	DiffHeader  lipgloss.Style

	// Source control styles
	FileTreeStyle    lipgloss.Style
	BranchStyle      lipgloss.Style
	TagStyle         lipgloss.Style
	StagedStyle      lipgloss.Style
	UnstagedStyle    lipgloss.Style

	// Status indicators
	StatusAdded   lipgloss.Style
	StatusDeleted lipgloss.Style
	StatusModified lipgloss.Style
	StatusUntracked lipgloss.Style
	StatusRenamed lipgloss.Style

	// Overlay styles
	OverlayBorder lipgloss.Style
	OverlayTitle  lipgloss.Style

	// Accent colors for status
	ScrollIndicator lipgloss.Style
	KeyStyle        lipgloss.Style

	// Notification styles
	SuccessText lipgloss.Style
	WarningText lipgloss.Style
	ErrorText   lipgloss.Style
	InfoText    lipgloss.Style
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

// initStyles creates all lipgloss styles derived from theme colors.
func (t *Theme) initStyles() {
	// --- Base ---
	t.BaseStyle = lipgloss.NewStyle().
		Foreground(t.Text).
		Background(t.Background)

	t.DimmedStyle = lipgloss.NewStyle().
		Foreground(t.Muted).
		Background(t.Background)

	t.FocusedStyle = lipgloss.NewStyle().
		Foreground(t.Accent).
		Bold(true)

	t.SelectedStyle = lipgloss.NewStyle().
		Foreground(t.Text).
		Background(lipgloss.Color("#5a3a3a"))

	// --- Titles ---
	t.TitleStyle = lipgloss.NewStyle().
		Foreground(t.Primary).
		Bold(true).
		Padding(0, 1)

	t.PanelTitleStyle = lipgloss.NewStyle().
		Foreground(t.Accent).
		Background(t.Background).
		Bold(true).
		Padding(0, 1).
		MarginRight(1)

	t.ActivePanelTitleStyle = lipgloss.NewStyle().
		Foreground(t.Background).
		Background(t.Accent).
		Bold(true).
		Padding(0, 1)

	// --- Panels ---
	t.PanelStyle = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(t.Muted).
		Background(t.Background).
		Foreground(t.Text).
		Padding(0, 1)

	t.ActivePanelStyle = lipgloss.NewStyle().
		Border(lipgloss.DoubleBorder()).
		BorderForeground(t.Accent).
		Background(t.Background).
		Foreground(t.Text).
		Padding(0, 1)

	// --- Status bar ---
	t.StatusBarStyle = lipgloss.NewStyle().
		Foreground(t.Background).
		Background(t.Muted).
		Padding(0, 2)

	// --- Diff ---
	t.DiffAdded = lipgloss.NewStyle().
		Foreground(t.Success).
		Background(t.Background).
		Padding(0, 1)

	t.DiffDeleted = lipgloss.NewStyle().
		Foreground(t.Error).
		Background(t.Background).
		Padding(0, 1)

	t.DiffHeader = lipgloss.NewStyle().
		Foreground(t.Info).
		Bold(true)

	// --- Source control ---
	t.FileTreeStyle = lipgloss.NewStyle().
		Foreground(t.Text).
		Background(t.Background).
		PaddingLeft(2)

	t.BranchStyle = lipgloss.NewStyle().
		Foreground(t.Info).
		Bold(true).
		Padding(0, 1)

	t.TagStyle = lipgloss.NewStyle().
		Foreground(t.Warning).
		Padding(0, 1)

	t.StagedStyle = lipgloss.NewStyle().
		Foreground(t.Success).
		Bold(true)

	t.UnstagedStyle = lipgloss.NewStyle().
		Foreground(t.Muted)

	// --- Status indicators ---
	t.StatusAdded = lipgloss.NewStyle().
		Foreground(t.Success).
		Bold(true)

	t.StatusDeleted = lipgloss.NewStyle().
		Foreground(t.Error).
		Bold(true)

	t.StatusModified = lipgloss.NewStyle().
		Foreground(t.Warning).
		Bold(true)

	t.StatusUntracked = lipgloss.NewStyle().
		Foreground(t.Muted)

	t.StatusRenamed = lipgloss.NewStyle().
		Foreground(t.Info).
		Bold(true)

	// --- Overlays ---
	t.OverlayBorder = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(t.Accent).
		Padding(1, 2).
		Background(lipgloss.Color("#2a1a1a"))

	t.OverlayTitle = lipgloss.NewStyle().
		Foreground(t.Background).
		Background(t.Accent).
		Bold(true).
		Padding(0, 2)

	// --- Misc ---
	t.ScrollIndicator = lipgloss.NewStyle().
		Foreground(t.Muted).
		Background(t.Surface)

	t.KeyStyle = lipgloss.NewStyle().
		Foreground(t.Warning).
		Bold(true)

	// --- Notifications ---
	t.SuccessText = lipgloss.NewStyle().
		Foreground(t.Success).
		Bold(true).
		Padding(0, 1)

	t.WarningText = lipgloss.NewStyle().
		Foreground(t.Warning).
		Bold(true).
		Padding(0, 1)

	t.ErrorText = lipgloss.NewStyle().
		Foreground(t.Error).
		Bold(true).
		Padding(0, 1)

	t.InfoText = lipgloss.NewStyle().
		Foreground(t.Info).
		Bold(true).
		Padding(0, 1)
}

// Load reads a theme from a JSON file and merges it over the defaults.
func Load(path string) (*Theme, error) {
	t := Default()

	data, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		return t, nil
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
		return t, nil
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
