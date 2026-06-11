package theme

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/charmbracelet/lipgloss"
)

// Theme defines the color palette and derived styles for the TUI.
// Designed to match Glint's clean Material Design aesthetic.
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

	// Base
	BaseStyle    lipgloss.Style
	DimmedStyle  lipgloss.Style
	MutedStyle   lipgloss.Style
	SelectedStyle lipgloss.Style
	FocusedStyle lipgloss.Style

	// Panels (no borders — Glint style)
	PanelSeparator lipgloss.Style
	ActivePanelMarker lipgloss.Style

	// Source control
	BranchStyle lipgloss.Style
	StagedStyle lipgloss.Style
	UnstagedStyle lipgloss.Style

	// Status badges
	StatusAdded    lipgloss.Style
	StatusDeleted  lipgloss.Style
	StatusModified lipgloss.Style
	StatusUntracked lipgloss.Style
	StatusRenamed  lipgloss.Style

	// Diff
	DiffAdded   lipgloss.Style
	DiffDeleted lipgloss.Style
	DiffHeader  lipgloss.Style

	// Status bar
	StatusBarStyle lipgloss.Style

	// Overlays
	OverlayBorder lipgloss.Style
	OverlayTitle  lipgloss.Style

	// Notifications
	SuccessText lipgloss.Style
	WarningText lipgloss.Style
	ErrorText   lipgloss.Style
	InfoText    lipgloss.Style

	// Misc
	ScrollIndicator lipgloss.Style
	KeyStyle        lipgloss.Style
	GraphStyle      lipgloss.Style
}

// BuiltinTheme returns one of the built-in themes by name.
func BuiltinTheme(name string) *Theme {
	switch name {
	case "light":
		return Light()
	default:
		return Default()
	}
}

// Light returns the light theme — Glint's clean white + orange aesthetic.
func Light() *Theme {
	t := &Theme{
		Background: lipgloss.Color("#ffffff"),
		Surface:    lipgloss.Color("#f8f9fa"),
		Primary:    lipgloss.Color("#ef6c00"), // Glint signature orange
		Accent:     lipgloss.Color("#ef6c00"),
		Text:       lipgloss.Color("#1a1a2e"),
		Muted:      lipgloss.Color("#8e8ea0"),
		Success:    lipgloss.Color("#2e7d32"),
		Warning:    lipgloss.Color("#e65100"),
		Error:      lipgloss.Color("#d32f2f"),
		Info:       lipgloss.Color("#1565c0"),
	}
	t.initStyles()
	return t
}

// Default is the dark variant — still Glint-inspired but for dark terminals.
func Default() *Theme {
	t := &Theme{
		Background: lipgloss.Color("#1a1a2e"),
		Surface:    lipgloss.Color("#22223a"),
		Primary:    lipgloss.Color("#ff8a3d"), // Warmer orange for dark bg
		Accent:     lipgloss.Color("#ff8a3d"),
		Text:       lipgloss.Color("#e2e2e8"),
		Muted:      lipgloss.Color("#6c6c8a"),
		Success:    lipgloss.Color("#66bb6a"),
		Warning:    lipgloss.Color("#ffa726"),
		Error:      lipgloss.Color("#ef5350"),
		Info:       lipgloss.Color("#42a5f5"),
	}
	t.initStyles()
	return t
}

// initStyles creates all lipgloss styles derived from theme colors.
// Glint-inspired: clean, minimal, no borders, lots of whitespace.
func (t *Theme) initStyles() {
	// ── Base ──
	t.BaseStyle = lipgloss.NewStyle().
		Foreground(t.Text).
		Background(t.Background)

	t.DimmedStyle = lipgloss.NewStyle().
		Foreground(t.Muted).
		Background(t.Background)

	t.MutedStyle = lipgloss.NewStyle().
		Foreground(t.Muted).
		Background(t.Background)

	t.SelectedStyle = lipgloss.NewStyle().
		Foreground(t.Text).
		Background(t.Surface).
		Padding(0, 1)

	t.FocusedStyle = lipgloss.NewStyle().
		Foreground(t.Accent).
		Bold(true)

	// ── Panel separator (thin vertical line between panels, Glint-style) ──
	t.PanelSeparator = lipgloss.NewStyle().
		Foreground(lipgloss.Color(t.Muted)).
		Background(t.Background).
		Padding(0, 0)

	t.ActivePanelMarker = lipgloss.NewStyle().
		Foreground(t.Accent).
		Background(t.Background).
		Padding(0, 1)

	// ── Source control ──
	t.BranchStyle = lipgloss.NewStyle().
		Foreground(t.Accent).
		Bold(true)

	t.StagedStyle = lipgloss.NewStyle().
		Foreground(t.Success).
		Bold(true)

	t.UnstagedStyle = lipgloss.NewStyle().
		Foreground(t.Muted)

	// ── Status badges (A/M/D/?) ──
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

	// ── Diff ──
	t.DiffAdded = lipgloss.NewStyle().
		Foreground(t.Success).
		Background(t.Background)

	t.DiffDeleted = lipgloss.NewStyle().
		Foreground(t.Error).
		Background(t.Background)

	t.DiffHeader = lipgloss.NewStyle().
		Foreground(t.Info).
		Bold(true)

	// ── Status bar (thin, no borders, Glint minimal) ──
	t.StatusBarStyle = lipgloss.NewStyle().
		Foreground(t.Muted).
		Background(t.Background).
		Padding(0, 2)

	// ── Overlays ──
	t.OverlayBorder = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(t.Muted).
		Padding(1, 2).
		Background(t.Background)

	t.OverlayTitle = lipgloss.NewStyle().
		Foreground(t.Background).
		Background(t.Accent).
		Bold(true).
		Padding(0, 2)

	// ── Misc ──
	t.ScrollIndicator = lipgloss.NewStyle().
		Foreground(t.Muted).
		Background(t.Background)

	t.KeyStyle = lipgloss.NewStyle().
		Foreground(t.Accent).
		Bold(true)

	t.GraphStyle = lipgloss.NewStyle().
		Foreground(t.Muted).
		Background(t.Background)

	// ── Notifications ──
	t.SuccessText = lipgloss.NewStyle().
		Foreground(t.Success).
		Bold(true)

	t.WarningText = lipgloss.NewStyle().
		Foreground(t.Warning).
		Bold(true)

	t.ErrorText = lipgloss.NewStyle().
		Foreground(t.Error).
		Bold(true)

	t.InfoText = lipgloss.NewStyle().
		Foreground(t.Info).
		Bold(true)
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
