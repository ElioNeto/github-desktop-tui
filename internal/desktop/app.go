package desktop

import (
	"context"
	"image"
	"image/color"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"

	"github.com/nicoddemus/github-desktop-tui/internal/auth"
	"github.com/nicoddemus/github-desktop-tui/internal/config"
	gitlocal "github.com/nicoddemus/github-desktop-tui/internal/git"
	"github.com/nicoddemus/github-desktop-tui/internal/providers"
	"github.com/nicoddemus/github-desktop-tui/internal/store"
	"github.com/nicoddemus/github-desktop-tui/pkg/types"
)

// AppState holds shared state for all UI components.
type AppState struct {
	mu     sync.RWMutex
	store  *store.Store
	gitOps gitlocal.GitOperations

	repos       []*store.TrackedRepo
	selectedRepo int
	branches    []*types.Branch
	commits     []*types.Commit
	graphRows   []*types.GraphRow
	changes     []*types.FileChange
	selectedCommit int

	// Callbacks for UI refresh
	onRefresh func()
}

func (s *AppState) LoadData() {
	ctx := context.TODO()

	// Load branches
	if branches, err := s.gitOps.Branches(ctx); err == nil {
		s.mu.Lock()
		s.branches = branches
		s.mu.Unlock()
	}

	// Load commits
	if commits, err := s.gitOps.Log(ctx, &gitlocal.LogOptions{Limit: 100}); err == nil {
		s.mu.Lock()
		s.commits = commits
		s.mu.Unlock()
	}

	// Load graph
	if rows, err := s.gitOps.GraphLog(ctx, &gitlocal.LogOptions{Limit: 100}); err == nil {
		s.mu.Lock()
		s.graphRows = rows
		s.mu.Unlock()
	}

	// Load status
	if changes, err := s.gitOps.Status(ctx); err == nil {
		s.mu.Lock()
		s.changes = changes
		s.mu.Unlock()
	}

	if s.onRefresh != nil {
		s.onRefresh()
	}
}

// Run starts the desktop application.
func Run(cfg *config.Config) error {
	// Initialize backend
	gitOps := gitlocal.New(".")

	st := store.New()
	st.Settings.SetConfig(cfg)

	_ = auth.NewManager()
	_ = providers.NewRegistry()

	// App state
	state := &AppState{
		store:     st,
		gitOps:    gitOps,
		repos:     make([]*store.TrackedRepo, 0),
		branches:  make([]*types.Branch, 0),
		commits:   make([]*types.Commit, 0),
		graphRows: make([]*types.GraphRow, 0),
		changes:   make([]*types.FileChange, 0),
	}

	// Load initial data
	state.LoadData()

	// Create Fyne app
	a := app.NewWithID("github-desktop-tui")
	a.Settings().SetTheme(&desktopTheme{})

	w := a.NewWindow("GitHub Desktop TUI")
	w.Resize(fyne.NewSize(1200, 800))

	// Build UI
	sidebar := NewSidebar(state)
	centerPanel := NewCenterPanel(state)
	details := NewDetailsPanel(state)

	appLayout := container.NewBorder(
		NewToolbar(state, func() { state.LoadData() }),
		nil, nil, nil,
		container.NewHSplit(
			container.NewHSplit(
				sidebar,
				centerPanel,
			),
			details,
		),
	)

	// Store refresh callback
	state.onRefresh = func() {
		sidebar.Refresh()
		centerPanel.Refresh()
		details.Refresh()
	}

	w.SetContent(appLayout)

	// Periodic refresh
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		for range ticker.C {
			state.LoadData()
		}
	}()

	w.ShowAndRun()
	return nil
}

// desktopTheme implements fyne.Theme for a dark Git-app look.
type desktopTheme struct{}

func (t *desktopTheme) Color(c fyne.ThemeColorName, v fyne.ThemeVariant) color.Color {
	switch c {
	case theme.ColorNameBackground:
		return color.RGBA{13, 17, 23, 255} // #0d1117
	case theme.ColorNameInputBackground:
		return color.RGBA{22, 27, 34, 255} // #161b22
	case theme.ColorNameButton:
		return color.RGBA{22, 27, 34, 255}
	case theme.ColorNamePrimary:
		return color.RGBA{239, 108, 0, 255} // #ef6c00
	case theme.ColorNameForeground:
		return color.RGBA{230, 237, 243, 255} // #e6edf3
	case theme.ColorNameDisabled:
		return color.RGBA{139, 148, 158, 255} // #8b949e
	case theme.ColorNameMenuBackground:
		return color.RGBA{22, 27, 34, 255}
	case theme.ColorNameSelection:
		return color.RGBA{239, 108, 0, 80} // orange with alpha
	case theme.ColorNameHover:
		return color.RGBA{48, 54, 61, 255} // #30363d
	case theme.ColorNameScrollBar:
		return color.RGBA{48, 54, 61, 255}
	default:
		return color.RGBA{13, 17, 23, 255}
	}
}

func (t *desktopTheme) Font(s fyne.TextStyle) fyne.Resource {
	return theme.DefaultTheme().Font(s)
}

func (t *desktopTheme) Icon(n fyne.ThemeIconName) fyne.Resource {
	return theme.DefaultTheme().Icon(n)
}

func (t *desktopTheme) Size(s fyne.ThemeSizeName) float32 {
	switch s {
	case theme.SizeNameText:
		return 13
	case theme.SizeNamePadding:
		return 4
	case theme.SizeNameInnerPadding:
		return 8
	case theme.SizeNameScrollBar:
		return 10
	case theme.SizeNameScrollBarSmall:
		return 6
	default:
		return theme.DefaultTheme().Size(s)
	}
}

// colorFromHex creates a color from a hex string.
func colorFromHex(hex string) color.Color {
	if len(hex) < 6 {
		return color.RGBA{0, 0, 0, 255}
	}
	hex = hex[1:] // remove #
	r := hexToByte(hex[0:2])
	g := hexToByte(hex[2:4])
	b := hexToByte(hex[4:6])
	return color.RGBA{r, g, b, 255}
}

func hexToByte(s string) uint8 {
	val := uint8(0)
	for _, c := range s {
		val *= 16
		switch {
		case c >= '0' && c <= '9':
			val += uint8(c - '0')
		case c >= 'a' && c <= 'f':
			val += uint8(c - 'a' + 10)
		case c >= 'A' && c <= 'F':
			val += uint8(c - 'A' + 10)
		}
	}
	return val
}

var branchColors = []color.Color{
	color.RGBA{239, 108, 0, 255},   // orange
	color.RGBA{88, 166, 255, 255},   // blue
	color.RGBA{63, 185, 80, 255},    // green
	color.RGBA{188, 140, 255, 255},  // purple
	color.RGBA{57, 210, 192, 255},   // cyan
	color.RGBA{210, 153, 34, 255},   // yellow
	color.RGBA{248, 81, 73, 255},    // red
}

// imageToResource converts an image to a fyne.Resource for icons.
func imageToResource(img image.Image) fyne.Resource {
	// Simple wrapper - in production use fyne.NewStaticResource
	return nil
}
