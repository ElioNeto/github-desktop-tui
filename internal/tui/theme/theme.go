package theme

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/charmbracelet/lipgloss"
)

// Theme defines a dark professional desktop Git app style.
type Theme struct {
	// Core
	Background lipgloss.Color
	Surface    lipgloss.Color // panel backgrounds
	SurfaceAlt lipgloss.Color // alternate (e.g. selected row)
	Border     lipgloss.Color // panel borders
	Text       lipgloss.Color
	Muted      lipgloss.Color
	DimText    lipgloss.Color

	// Accents
	Orange    lipgloss.Color
	Blue      lipgloss.Color
	Green     lipgloss.Color
	Red       lipgloss.Color
	Yellow    lipgloss.Color
	Purple    lipgloss.Color
	Cyan      lipgloss.Color

	// Computed styles
	Base        lipgloss.Style
	BaseMuted   lipgloss.Style
	Dim         lipgloss.Style
	Selected    lipgloss.Style
	Accented    lipgloss.Style
	PanelBorder lipgloss.Style
	ActiveBorder lipgloss.Style

	ToolbarStyle lipgloss.Style
	StatusStyle  lipgloss.Style

	BranchLabel  lipgloss.Style
	TagLabel     lipgloss.Style
	BadgeAdded   lipgloss.Style
	BadgeDeleted lipgloss.Style
	BadgeModified lipgloss.Style

	OverlayBox  lipgloss.Style
	OverlayTitle lipgloss.Style

	SuccessText lipgloss.Style
	ErrorText   lipgloss.Style
	WarningText lipgloss.Style
	InfoText    lipgloss.Style
}

// NewDefault creates the dark professional theme.
func NewDefault() *Theme {
	t := &Theme{
		Background: lipgloss.Color("#0d1117"), // GitHub dark bg
		Surface:    lipgloss.Color("#161b22"), // panel bg
		SurfaceAlt: lipgloss.Color("#1c2333"), // selected/hover
		Border:     lipgloss.Color("#30363d"), // subtle border
		Text:       lipgloss.Color("#e6edf3"),
		Muted:      lipgloss.Color("#8b949e"),
		DimText:    lipgloss.Color("#484f58"),

		Orange: lipgloss.Color("#ef6c00"),
		Blue:   lipgloss.Color("#58a6ff"),
		Green:  lipgloss.Color("#3fb950"),
		Red:    lipgloss.Color("#f85149"),
		Yellow: lipgloss.Color("#d29922"),
		Purple: lipgloss.Color("#bc8cff"),
		Cyan:   lipgloss.Color("#39d2c0"),
	}
	t.initStyles()
	return t
}

// BuiltinTheme returns a built-in theme.
func BuiltinTheme(name string) *Theme {
	return NewDefault()
}

func (t *Theme) initStyles() {
	// Base text
	t.Base = lipgloss.NewStyle().Foreground(t.Text).Background(t.Background)
	t.BaseMuted = lipgloss.NewStyle().Foreground(t.Muted).Background(t.Background)
	t.Dim = lipgloss.NewStyle().Foreground(t.DimText).Background(t.Background)
	t.Selected = lipgloss.NewStyle().Foreground(t.Text).Background(t.SurfaceAlt)
	t.Accented = lipgloss.NewStyle().Foreground(t.Orange).Bold(true)

	// Panel borders
	t.PanelBorder = lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		BorderForeground(t.Border).
		Background(t.Surface).
		Foreground(t.Text).
		Padding(0, 1)

	t.ActiveBorder = lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		BorderForeground(t.Orange).
		Background(t.Surface).
		Foreground(t.Text).
		Padding(0, 1)

	// Toolbar
	t.ToolbarStyle = lipgloss.NewStyle().
		Background(t.Surface).
		Foreground(t.Text).
		Padding(0, 1)

	// Status bar
	t.StatusStyle = lipgloss.NewStyle().
		Background(t.Surface).
		Foreground(t.Muted).
		Padding(0, 2)

	// Labels
	t.BranchLabel = lipgloss.NewStyle().
		Foreground(t.Orange).
		Bold(true)

	t.TagLabel = lipgloss.NewStyle().
		Foreground(t.Yellow).
		Bold(true)

	// Badges
	t.BadgeAdded = lipgloss.NewStyle().Foreground(t.Green).Bold(true)
	t.BadgeDeleted = lipgloss.NewStyle().Foreground(t.Red).Bold(true)
	t.BadgeModified = lipgloss.NewStyle().Foreground(t.Yellow).Bold(true)

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

	// Notifications
	t.SuccessText = lipgloss.NewStyle().Foreground(t.Green).Bold(true)
	t.ErrorText = lipgloss.NewStyle().Foreground(t.Red).Bold(true)
	t.WarningText = lipgloss.NewStyle().Foreground(t.Yellow).Bold(true)
	t.InfoText = lipgloss.NewStyle().Foreground(t.Blue).Bold(true)
}

// Load reads theme from JSON.
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
