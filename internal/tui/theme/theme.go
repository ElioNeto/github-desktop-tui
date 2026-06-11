package theme

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/charmbracelet/lipgloss"
)

// Theme defines all colors used in the TUI.
type Theme struct {
	Primary    lipgloss.Color
	Background lipgloss.Color
	Surface    lipgloss.Color
	Text       lipgloss.Color
	Muted      lipgloss.Color
	Success    lipgloss.Color
	Warning    lipgloss.Color
	Error      lipgloss.Color
	Info       lipgloss.Color

	// Derived styles
	BaseStyle        lipgloss.Style
	FocusedStyle     lipgloss.Style
	TitleStyle       lipgloss.Style
	SelectedStyle    lipgloss.Style
	StatusBarStyle   lipgloss.Style
}

// ThemeJSON represents the serializable theme format.
type ThemeJSON struct {
	Defs map[string]string `json:"defs"`
}

// Default returns a dark theme inspired by One Dark Pro.
func Default() *Theme {
	t := &Theme{
		Primary:    lipgloss.Color("#0f3460"),
		Background: lipgloss.Color("#1a1a2e"),
		Surface:    lipgloss.Color("#16213e"),
		Text:       lipgloss.Color("#e0e0e0"),
		Muted:      lipgloss.Color("#a0a0a0"),
		Success:    lipgloss.Color("#4caf50"),
		Warning:    lipgloss.Color("#ff9800"),
		Error:      lipgloss.Color("#f44336"),
		Info:       lipgloss.Color("#2196f3"),
	}

	t.initStyles()
	return t
}

// Load reads a theme from a JSON file.
func Load(path string) (*Theme, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("ler tema: %w", err)
	}

	var tj ThemeJSON
	if err := json.Unmarshal(data, &tj); err != nil {
		return nil, fmt.Errorf("parse tema JSON: %w", err)
	}

	t := &Theme{}

	if v, ok := tj.Defs["bg"]; ok {
		t.Background = lipgloss.Color(v)
	}
	if v, ok := tj.Defs["surface"]; ok {
		t.Surface = lipgloss.Color(v)
	}
	if v, ok := tj.Defs["accent"]; ok {
		t.Primary = lipgloss.Color(v)
	}
	if v, ok := tj.Defs["text"]; ok {
		t.Text = lipgloss.Color(v)
	}
	if v, ok := tj.Defs["muted"]; ok {
		t.Muted = lipgloss.Color(v)
	}
	if v, ok := tj.Defs["success"]; ok {
		t.Success = lipgloss.Color(v)
	}
	if v, ok := tj.Defs["warning"]; ok {
		t.Warning = lipgloss.Color(v)
	}
	if v, ok := tj.Defs["error"]; ok {
		t.Error = lipgloss.Color(v)
	}
	if v, ok := tj.Defs["info"]; ok {
		t.Info = lipgloss.Color(v)
	}

	t.initStyles()
	return t, nil
}

func (t *Theme) initStyles() {
	t.BaseStyle = lipgloss.NewStyle().
		Foreground(t.Text).
		Background(t.Background)

	t.FocusedStyle = t.BaseStyle.Copy().
		BorderForeground(t.Primary)

	t.TitleStyle = lipgloss.NewStyle().
		Foreground(t.Text).
		Background(t.Primary).
		Bold(true)

	t.SelectedStyle = lipgloss.NewStyle().
		Foreground(t.Text).
		Background(t.Primary)

	t.StatusBarStyle = lipgloss.NewStyle().
		Foreground(t.Muted).
		Background(t.Surface)
}
