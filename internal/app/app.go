package app

import (
	"context"
	"fmt"
	"os"
	"os/signal"
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
	// --- Configuração ---
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("carregar config: %w", err)
	}

	// --- Tema ---
	th, err := theme.Load(cfg.ThemeFile)
	if err != nil {
		th = theme.Default()
	}

	// --- Store (estado central) ---
	st := store.New()
	st.Settings.SetConfig(cfg)

	// --- Registry de provedores Git ---
	registry := providers.NewRegistry()
	registry.Register(githubprovider.NewProvider())

	// --- Auth Manager ---
	authManager := auth.NewManager()

	// --- Git local (abre diretório atual ou vazio) ---
	repoPath := "."
	if len(os.Args) > 1 {
		repoPath = os.Args[1]
	}
	gitOps := gitlocal.New(repoPath)

	// --- Contexto com cancelamento para shutdown graceful ---
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// --- Model raiz do Bubble Tea ---
	model := tui.New(tui.Options{
		Store:       st,
		Theme:       th,
		Config:      cfg,
		Registry:    registry,
		AuthManager: authManager,
		GitOps:      gitOps,
	})

	// --- Bubble Tea program ---
	p := tea.NewProgram(
		model,
		tea.WithAltScreen(),       // Tela alternativa (terminal limpo)
		tea.WithMouseCellMotion(), // Suporte a mouse
		tea.WithContext(ctx),
	)

	// --- Signal handling para shutdown graceful ---
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		cancel()
		p.Quit()
	}()

	// --- Iniciar a TUI (bloqueante) ---
	if _, err := p.Run(); err != nil {
		return fmt.Errorf("executar TUI: %w", err)
	}

	return nil
}
