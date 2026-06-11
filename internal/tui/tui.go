package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/nicoddemus/github-desktop-tui/internal/config"
	"github.com/nicoddemus/github-desktop-tui/internal/providers"
	"github.com/nicoddemus/github-desktop-tui/internal/store"
	"github.com/nicoddemus/github-desktop-tui/internal/tui/keybindings"
	"github.com/nicoddemus/github-desktop-tui/internal/tui/theme"
)

// Options holds the dependencies for creating a new TUI model.
type Options struct {
	Store    *store.Store
	Theme    *theme.Theme
	Config   *config.Config
	Registry *providers.Registry
}

// Model is the root Bubble Tea model for the application.
type Model struct {
	// Dependencies
	store    *store.Store
	theme    *theme.Theme
	config   *config.Config
	registry *providers.Registry
	keys     keybindings.KeyMap

	// Layout
	layout   Layout
	focused  PanelID

	// Which view is active in each panel
	leftView   ViewID
	centerView ViewID
	rightView  ViewID

	// Overlay state
	showHelp   bool
	showSearch bool

	// Notifications
	notification *NotificationMsg
	notifTimer   time.Time

	// Terminal dimensions
	width  int
	height int

	// Ready flag — true after first resize
	ready bool
}

// New creates a new Model with the given options.
func New(opts Options) *Model {
	return &Model{
		store:    opts.Store,
		theme:    opts.Theme,
		config:   opts.Config,
		registry: opts.Registry,
		keys:     keybindings.DefaultKeyMap(),
		focused:  PanelLeft,
		leftView: ViewRepositories,
		centerView: ViewCommitLog,
		rightView:  ViewDetails,
	}
}

// Init initializes the model and returns initial commands.
func (m Model) Init() tea.Cmd {
	return tea.Batch(
		tea.EnterAltScreen,
		m.waitForSize,
	)
}

func (m Model) waitForSize() tea.Msg {
	return tea.WindowSizeMsg{
		Width:  m.width,
		Height: m.height,
	}
}

// Update handles messages and returns the updated model.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	// --- Terminal resize ---
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.layout = CalculateLayout(msg.Width, msg.Height)
		m.ready = true
		return m, nil

	// --- Key presses ---
	case tea.KeyMsg:
		return m.handleKeyMsg(msg)

	// --- Focus change ---
	case FocusChangeMsg:
		m.focused = msg.Panel
		return m, nil

	// --- View change ---
	case ViewChangeMsg:
		switch msg.Panel {
		case PanelLeft:
			m.leftView = msg.View
		case PanelCenter:
			m.centerView = msg.View
		case PanelRight:
			m.rightView = msg.View
		}
		return m, nil

	// --- Repo list loaded ---
	case RepoListLoadedMsg:
		m.store.Repositories.SetRepos(msg.Repos)
		return m, nil

	// --- Repo list error ---
	case RepoListErrorMsg:
		m.notification = &NotificationMsg{
			Level:   "error",
			Message: msg.Err.Error(),
		}
		m.notifTimer = time.Now()
		return m, nil

	// --- Repo selection ---
	case RepoSelectMsg:
		return m, nil

	// --- Error ---
	case ErrorMsg:
		m.notification = &NotificationMsg{
			Level:   "error",
			Message: msg.Err.Error(),
		}
		m.notifTimer = time.Now()
		return m, nil

	// --- Success ---
	case SuccessMsg:
		m.notification = &NotificationMsg{
			Level:   "success",
			Message: msg.Message,
		}
		m.notifTimer = time.Now()
		return m, nil

	// --- Notification ---
	case NotificationMsg:
		m.notification = &msg
		m.notifTimer = time.Now()
		return m, nil

	// --- Provider switch ---
	case ProviderSwitchMsg:
		if err := m.registry.SetActive(msg.Provider); err != nil {
			return m, func() tea.Msg {
				return ErrorMsg{Err: err}
			}
		}
		m.store.Settings.SetActiveProvider(msg.Provider)
		m.notification = &NotificationMsg{
			Level:   "info",
			Message: fmt.Sprintf("Provedor alterado para: %s", msg.Provider),
		}
		m.notifTimer = time.Now()
		return m, nil

	case AuthCompleteMsg:
		if msg.Success {
			m.notification = &NotificationMsg{
				Level:   "success",
				Message: "Autenticação realizada com sucesso",
			}
		} else {
			m.notification = &NotificationMsg{
				Level:   "error",
				Message: fmt.Sprintf("Falha na autenticação: %v", msg.Error),
			}
		}
		m.notifTimer = time.Now()
		return m, nil

	case GitCommitMsg:
		m.notification = &NotificationMsg{
			Level:   "success",
			Message: fmt.Sprintf("Commit realizado: %s", msg.Hash[:7]),
		}
		m.notifTimer = time.Now()
		return m, nil

	case GitCommitErrorMsg:
		m.notification = &NotificationMsg{
			Level:   "error",
			Message: fmt.Sprintf("Erro no commit: %v", msg.Err),
		}
		m.notifTimer = time.Now()
		return m, nil
	}

	return m, nil
}

// handleKeyMsg processes keyboard input.
func (m Model) handleKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Global keybindings
	switch {
	case key.Matches(msg, m.keys.Quit):
		return m, tea.Quit

	case key.Matches(msg, m.keys.Help):
		m.showHelp = !m.showHelp
		return m, nil

	case key.Matches(msg, m.keys.FocusNext):
		m.focused = PanelID((int(m.focused) + 1) % NumPanels)
		return m, nil

	case key.Matches(msg, m.keys.FocusPrev):
		m.focused = PanelID((int(m.focused) - 1 + NumPanels) % NumPanels)
		return m, nil
	}

	return m, nil
}

// View renders the entire TUI.
func (m Model) View() string {
	if !m.ready {
		return "Carregando..."
	}

	// Build the three panels
	var b strings.Builder

	// Render panels side by side
	leftPanel := m.renderLeftPanel()
	centerPanel := m.renderCenterPanel()
	rightPanel := m.renderRightPanel()

	// Combine panels
	for line := 0; line < m.layout.PanelHeight; line++ {
		leftLine := getLine(leftPanel, line, m.layout.LeftWidth)
		centerLine := getLine(centerPanel, line, m.layout.CenterWidth)
		rightLine := getLine(rightPanel, line, m.layout.RightWidth)

		b.WriteString(leftLine)
		b.WriteString(centerLine)
		b.WriteString(rightLine)
		b.WriteString("\n")
	}

	// Status bar
	b.WriteString(m.renderStatusBar())

	// Overlays
	if m.showHelp {
		b.WriteString(m.renderHelpOverlay())
	}

	return b.String()
}

// getLine extracts a line from a multi-line string, padded to width.
func getLine(s string, line, width int) string {
	lines := strings.Split(s, "\n")
	if line >= len(lines) {
		return strings.Repeat(" ", width)
	}
	l := lines[line]
	if len(l) > width {
		return l[:width]
	}
	return l + strings.Repeat(" ", width-len(l))
}

// renderLeftPanel renders the explorer panel (left).
func (m Model) renderLeftPanel() string {
	var b strings.Builder

	// Title
	title := " Repositórios "
	b.WriteString(m.theme.TitleStyle.Render(title))
	b.WriteString("\n")

	// Repository list
	repos := m.store.Repositories.Repos()
	selected := m.store.Repositories.SelectedIndex()

	if len(repos) == 0 {
		b.WriteString("  Nenhum repositório\n")
	} else {
		for i, repo := range repos {
			cursor := " "
			if i == selected {
				cursor = "▶"
			}
			style := m.theme.BaseStyle
			if i == selected {
				style = m.theme.SelectedStyle
			}
			line := fmt.Sprintf(" %s %s/%s", cursor, repo.Owner, repo.Name)
			b.WriteString(style.Render(line))
			b.WriteString("\n")
		}
	}

	// Provider list
	b.WriteString("\n")
	b.WriteString(m.theme.TitleStyle.Render(" Provedores "))
	b.WriteString("\n")

	for _, p := range m.registry.All() {
		mark := "○"
		if p.Name() == m.store.Settings.ActiveProvider() {
			mark = "●"
		}
		b.WriteString(fmt.Sprintf(" %s %s %s\n", mark, p.Icon(), p.DisplayName()))
	}

	return b.String()
}

// renderCenterPanel renders the content panel (center).
func (m Model) renderCenterPanel() string {
	var b strings.Builder

	switch m.centerView {
	case ViewCommitLog:
		b.WriteString(m.renderCommitLog())
	case ViewBranchList:
		b.WriteString("Lista de Branches\n")
	case ViewDiffViewer:
		b.WriteString("Visualizador de Diff\n")
	default:
		b.WriteString("Visualizador de Commits\n")
	}

	return b.String()
}

// renderCommitLog renders the commit history view.
func (m Model) renderCommitLog() string {
	var b strings.Builder

	// Branch indicator
	activeBranch := m.store.Branches.Active()
	b.WriteString(m.theme.TitleStyle.Render(fmt.Sprintf(" Branch: %s ", activeBranch)))
	b.WriteString("\n\n")

	// Commit list
	commits := m.store.Commits.Commits()
	if len(commits) == 0 {
		b.WriteString("  Nenhum commit encontrado\n")
		b.WriteString("  Pressione 'r' para atualizar\n")
	} else {
		selected := m.store.Commits.Selected()
		for _, c := range commits {
			hash := c.ShortHash
			if len(hash) > 7 {
				hash = hash[:7]
			}

			style := m.theme.BaseStyle
			cursor := " "
			if selected != nil && c.Hash == selected.Hash {
				style = m.theme.SelectedStyle
				cursor = "▶"
			}

			timeStr := c.Timestamp.Format("2006-01-02 15:04")
			msgHead := c.MessageHead
			if len(msgHead) > 50 {
				msgHead = msgHead[:50] + "..."
			}

			line := fmt.Sprintf(" %s %s %s  %s", cursor, timeStr, hash, msgHead)
			b.WriteString(style.Render(line))
			b.WriteString("\n")
		}
	}

	return b.String()
}

// renderRightPanel renders the details panel (right).
func (m Model) renderRightPanel() string {
	var b strings.Builder

	repo := m.store.Repositories.Selected()
	if repo == nil {
		b.WriteString("  Nenhum repositório\n  selecionado\n")
		return b.String()
	}

	b.WriteString(m.theme.TitleStyle.Render(" Detalhes "))
	b.WriteString("\n\n")
	b.WriteString(fmt.Sprintf("  Nome: %s\n", repo.FullName))
	b.WriteString(fmt.Sprintf("  Dono: %s\n", repo.Owner))
	b.WriteString(fmt.Sprintf("  URL:  %s\n", repo.URL))
	b.WriteString(fmt.Sprintf("  Lang: %s\n", repo.Language))
	b.WriteString(fmt.Sprintf("  Stars: %d\n", repo.Stars))
	b.WriteString(fmt.Sprintf("  Forks: %d\n", repo.Forks))

	return b.String()
}

// renderStatusBar renders the bottom status bar.
func (m Model) renderStatusBar() string {
	activeProvider := m.store.Settings.ActiveProvider()
	activeBranch := m.store.Branches.Active()

	left := fmt.Sprintf(" %s | branch: %s ", activeProvider, activeBranch)
	right := time.Now().Format(" 15:04 ")

	// Calculate padding
	spaces := m.width - len(left) - len(right)
	if spaces < 1 {
		spaces = 1
	}

	bar := m.theme.StatusBarStyle.Render(left + strings.Repeat(" ", spaces) + right)
	return bar
}

// renderHelpOverlay renders the help screen overlay.
func (m Model) renderHelpOverlay() string {
	return fmt.Sprintf(`
╔══════════════════════════════════════════╗
║              Ajuda - Teclas             ║
╠══════════════════════════════════════════╣
║ %-36s ║
║ %-36s ║
║ %-36s ║
║ %-36s ║
║ %-36s ║
║ %-36s ║
║ %-36s ║
║ %-36s ║
║ %-36s ║
║ %-36s ║
║ %-36s ║
║ %-36s ║
║ %-36s ║
╚══════════════════════════════════════════╝`,
		fmt.Sprintf("%s  %s", m.keys.Help.Help().Key, m.keys.Help.Help().Desc),
		fmt.Sprintf("%s  %s", m.keys.Quit.Help().Key, m.keys.Quit.Help().Desc),
		fmt.Sprintf("%s  %s", m.keys.Refresh.Help().Key, m.keys.Refresh.Help().Desc),
		fmt.Sprintf("%s  %s", m.keys.FocusNext.Help().Key, m.keys.FocusNext.Help().Desc),
		fmt.Sprintf("%s  %s", m.keys.FocusPrev.Help().Key, m.keys.FocusPrev.Help().Desc),
		fmt.Sprintf("%s  %s", m.keys.Search.Help().Key, m.keys.Search.Help().Desc),
		fmt.Sprintf("%s  %s", m.keys.Diff.Help().Key, m.keys.Diff.Help().Desc),
		fmt.Sprintf("%s  %s", m.keys.Stage.Help().Key, m.keys.Stage.Help().Desc),
		fmt.Sprintf("%s  %s", m.keys.Commit.Help().Key, m.keys.Commit.Help().Desc),
		fmt.Sprintf("%s  %s", m.keys.Push.Help().Key, m.keys.Push.Help().Desc),
		fmt.Sprintf("%s  %s", m.keys.Pull.Help().Key, m.keys.Pull.Help().Desc),
		fmt.Sprintf("%s  %s", m.keys.Branch.Help().Key, m.keys.Branch.Help().Desc),
		fmt.Sprintf("%s  %s", m.keys.ProviderSwitch.Help().Key, m.keys.ProviderSwitch.Help().Desc),
	)
}
