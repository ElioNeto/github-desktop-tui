package app

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/nicoddemus/github-desktop-tui/internal/auth"
	"github.com/nicoddemus/github-desktop-tui/internal/config"
	gitlocal "github.com/nicoddemus/github-desktop-tui/internal/git"
	"github.com/nicoddemus/github-desktop-tui/internal/providers"
	githubprovider "github.com/nicoddemus/github-desktop-tui/internal/providers/github"
	"github.com/nicoddemus/github-desktop-tui/internal/store"
	"github.com/nicoddemus/github-desktop-tui/internal/tui"
	"github.com/nicoddemus/github-desktop-tui/internal/tui/theme"
)

// Run initializes and starts the TUI application.
func Run() error {
	// Parse theme flag
	themeName := ""
	args := os.Args[1:]
	for i, arg := range args {
		if arg == "--theme" && i+1 < len(args) {
			themeName = args[i+1]
			// Remove --theme and its value from args
			args = append(args[:i], args[i+2:]...)
			break
		}
	}

	// Get repo path from remaining args
	repoPath := "."
	if len(args) > 0 && args[0] != "" {
		repoPath = args[0]
	}

	// --- Configuração ---
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("carregar config: %w", err)
	}

	// Override theme from flag
	if themeName != "" {
		cfg.ThemeName = themeName
	}

	// --- Tema ---
	appDir, _ := os.UserConfigDir()
	appDir = filepath.Join(appDir, "github-desktop-tui")
	themeDir := filepath.Join(appDir, "themes")

	var th *theme.Theme
	if cfg.ThemeName != "" {
		th = theme.BuiltinTheme(cfg.ThemeName)
		// Try loading from theme dir
		themePath := filepath.Join(themeDir, cfg.ThemeName+".json")
		if loaded, err := theme.Load(themePath); err == nil {
			th = loaded
		}
	} else {
		th = theme.Default()
		if loaded, err := theme.Load(cfg.ThemeFile); err == nil {
			th = loaded
		}
	}

	// --- Store (estado central) ---
	st := store.New()
	st.Settings.SetConfig(cfg)

	// --- Registry de provedores Git ---
	registry := providers.NewRegistry()
	registry.Register(githubprovider.NewProvider())

	// --- Auth Manager ---
	authManager := auth.NewManager()

	// --- Git local ---
	gitOps := gitlocal.New(repoPath)

	// --- Repo Manager (F1.2) ---
	repoManager := store.NewRepoManager(appDir)

	// --- Contexto com cancelamento ---
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// --- Model raiz do Bubble Tea ---
	model := tui.New(tui.Options{
		Store:       st,
		Theme:       th,
		ThemeName:   cfg.ThemeName,
		ThemeDir:    themeDir,
		Config:      cfg,
		Registry:    registry,
		AuthManager: authManager,
		GitOps:      gitOps,
		RepoManager: repoManager,
	})

	// --- Bubble Tea program ---
	p := tea.NewProgram(
		model,
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
		tea.WithContext(ctx),
	)

	// --- Signal handling ---
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		cancel()
		p.Quit()
	}()

	// --- Iniciar TUI ---
	if _, err := p.Run(); err != nil {
		return fmt.Errorf("executar TUI: %w", err)
	}

	return nil
}
