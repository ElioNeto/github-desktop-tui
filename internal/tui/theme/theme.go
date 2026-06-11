package theme

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/charmbracelet/lipgloss"
)

// Theme defines a dark professional desktop Git app style — terminal-native.
type Theme struct {
	// Core
	Background lipgloss.Color
	Surface    lipgloss.Color
	SurfaceAlt lipgloss.Color
	Border     lipgloss.Color
	Text       lipgloss.Color
	Muted      lipgloss.Color
	Dim        lipgloss.Color

	// Accents
	Orange lipgloss.Color
	Blue   lipgloss.Color
	Green  lipgloss.Color
	Red    lipgloss.Color
	Yellow lipgloss.Color

	// Computed styles
	Base       lipgloss.Style
	BaseMuted  lipgloss.Style
	DimStyle   lipgloss.Style
	Selected   lipgloss.Style
	Accented   lipgloss.Style

	Toolbar    lipgloss.Style
	Status     lipgloss.Style

	Branch     lipgloss.Style
	BadgeAdd   lipgloss.Style
	BadgeDel   lipgloss.Style
	BadgeMod   lipgloss.Style

	OverlayBox    lipgloss.Style
	OverlayTitle  lipgloss.Style

	SuccessText lipgloss.Style
	ErrorText   lipgloss.Style
	WarningText lipgloss.Style
	InfoText    lipgloss.Style
}

// NewDefault creates the dark professional theme.
func NewDefault() *Theme {
	t := &Theme{
		Background: lipgloss.Color("#0d1117"),
		Surface:    lipgloss.Color("#161b22"),
		SurfaceAlt: lipgloss.Color("#1c2533"),
		Border:     lipgloss.Color("#21262d"),
		Text:       lipgloss.Color("#e6edf3"),
		Muted:      lipgloss.Color("#8b949e"),
		Dim:        lipgloss.Color("#484f58"),

		Orange: lipgloss.Color("#ef6c00"),
		Blue:   lipgloss.Color("#58a6ff"),
		Green:  lipgloss.Color("#3fb950"),
		Red:    lipgloss.Color("#f85149"),
		Yellow: lipgloss.Color("#d29922"),
	}
	t.initStyles()
	return t
}

func BuiltinTheme(name string) *Theme { return NewDefault() }

func (t *Theme) initStyles() {
	t.Base = lipgloss.NewStyle().Foreground(t.Text).Background(t.Background)
	t.BaseMuted = lipgloss.NewStyle().Foreground(t.Muted).Background(t.Background)
	t.DimStyle = lipgloss.NewStyle().Foreground(t.Dim).Background(t.Background)
	t.Selected = lipgloss.NewStyle().Foreground(t.Text).Background(t.SurfaceAlt)
	t.Accented = lipgloss.NewStyle().Foreground(t.Orange).Bold(true)

	// Toolbar: full width, accent bg, white text
	t.Toolbar = lipgloss.NewStyle().
		Foreground(t.Text).
		Background(t.Orange).
		Padding(0, 1)

	// Status bar: muted bg
	t.Status = lipgloss.NewStyle().
		Foreground(t.Muted).
		Background(t.Surface).
		Padding(0, 1)

	// Labels
	t.Branch = lipgloss.NewStyle().Foreground(t.Orange).Bold(true)
	t.BadgeAdd = lipgloss.NewStyle().Foreground(t.Green).Bold(true)
	t.BadgeDel = lipgloss.NewStyle().Foreground(t.Red).Bold(true)
	t.BadgeMod = lipgloss.NewStyle().Foreground(t.Yellow).Bold(true)

	// Overlays
	t.OverlayBox = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(t.Border).
		Padding(1, 2).
		Background(t.Surface)

	t.OverlayTitle = lipgloss.NewStyle().
		Foreground(t.Background).
		Background(t.Orange).
		Bold(true).
		Padding(0, 2)

	t.SuccessText = lipgloss.NewStyle().Foreground(t.Green).Bold(true)
	t.ErrorText = lipgloss.NewStyle().Foreground(t.Red).Bold(true)
	t.WarningText = lipgloss.NewStyle().Foreground(t.Yellow).Bold(true)
	t.InfoText = lipgloss.NewStyle().Foreground(t.Blue).Bold(true)
}

func Load(path string) (*Theme, error) {
	t := NewDefault()
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
		t.Orange = lipgloss.Color(raw.Accent)
	}
	if raw.Text != "" {
		t.Text = lipgloss.Color(raw.Text)
	}
	if raw.Muted != "" {
		t.Muted = lipgloss.Color(raw.Muted)
	}
	if raw.Success != "" {
		t.Green = lipgloss.Color(raw.Success)
	}
	if raw.Warning != "" {
		t.Yellow = lipgloss.Color(raw.Warning)
	}
	if raw.Error != "" {
		t.Red = lipgloss.Color(raw.Error)
	}
	if raw.Info != "" {
		t.Blue = lipgloss.Color(raw.Info)
	}
	t.initStyles()
	return t, nil
}
