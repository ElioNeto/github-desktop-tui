package tui

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/nicoddemus/github-desktop-tui/internal/auth"
	"github.com/nicoddemus/github-desktop-tui/internal/config"
	gitlocal "github.com/nicoddemus/github-desktop-tui/internal/git"
	"github.com/nicoddemus/github-desktop-tui/internal/providers"
	"github.com/nicoddemus/github-desktop-tui/internal/store"
	"github.com/nicoddemus/github-desktop-tui/internal/tui/keybindings"
	"github.com/nicoddemus/github-desktop-tui/internal/tui/theme"
	"github.com/nicoddemus/github-desktop-tui/pkg/types"
)

// Options holds the dependencies for creating a new TUI model.
type Options struct {
	Store       *store.Store
	Theme       *theme.Theme
	Config      *config.Config
	Registry    *providers.Registry
	AuthManager *auth.Manager
	GitOps      gitlocal.GitOperations
}

// Mode indica o modo atual da TUI (navegação, input, etc).
type Mode int

const (
	ModeNormal Mode = iota
	ModeCommitMessage
	ModeAuthInput
	ModeAuthMethod
	ModeAddRemote
)

// Model is the root Bubble Tea model for the application.
type Model struct {
	// Dependencies
	store       *store.Store
	theme       *theme.Theme
	config      *config.Config
	registry    *providers.Registry
	authManager *auth.Manager
	gitOps      gitlocal.GitOperations

	keys keybindings.KeyMap

	// Layout
	layout  Layout
	focused PanelID

	// Which view is active in each panel
	leftView   ViewID
	centerView ViewID
	rightView  ViewID

	// Overlay state
	showHelp     bool
	showSearch   bool
	showStaging  bool
	showRemotes  bool
	showAuth     bool
	showTimeline bool

	// Notification
	notification *NotificationMsg
	notifTimer   time.Time

	// Terminal dimensions
	width  int
	height int

	// Ready flag
	ready bool

	// Mode (normal, typing commit message, etc)
	mode Mode

	// Text input for commit message
	commitInput textinput.Model

	// Text input for auth token
	authInput    textinput.Model
	authProvider string

	// Text input for remote URL
	remoteInput textinput.Model
	remoteName  string

	// Git status cache
	fileChanges []*types.FileChange
	selectedFile int
	commitMsg    string

	// Staging selected files (indices)
	stagingSelected map[int]bool

	// Timeline mode
	timelineCommits []*types.Commit
	timelineIndex   int

	// Remotes list
	remoteList []*types.Remote
}

// New creates a new Model with the given options.
func New(opts Options) *Model {
	ci := textinput.New()
	ci.Placeholder = "Mensagem do commit..."
	ci.Focus()
	ci.CharLimit = 200
	ci.Width = 60

	ai := textinput.New()
	ai.Placeholder = "Token ou nome da variável de ambiente"
	ai.Focus()
	ai.CharLimit = 200
	ai.Width = 50
	ai.EchoMode = textinput.EchoPassword

	ri := textinput.New()
	ri.Placeholder = "URL do remote (ex: https://github.com/user/repo.git)"
	ri.Focus()
	ri.CharLimit = 300
	ri.Width = 60

	return &Model{
		store:       opts.Store,
		theme:       opts.Theme,
		config:      opts.Config,
		registry:    opts.Registry,
		authManager: opts.AuthManager,
		gitOps:      opts.GitOps,

		keys: keybindings.DefaultKeyMap(),

		focused:  PanelLeft,
		leftView: ViewRepositories,
		centerView: ViewCommitLog,
		rightView:  ViewDetails,

		commitInput:  ci,
		authInput:    ai,
		remoteInput:  ri,

		fileChanges:     make([]*types.FileChange, 0),
		selectedFile:    -1,
		stagingSelected: make(map[int]bool),

		timelineCommits: make([]*types.Commit, 0),
		timelineIndex:   0,

		remoteList: make([]*types.Remote, 0),
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

// ---------------------------------------------------------------------------
// Update
// ---------------------------------------------------------------------------

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.layout = CalculateLayout(msg.Width, msg.Height)
		m.ready = true
		return m, nil

	case tea.KeyMsg:
		return m.handleKeyMsg(msg)

	// --- Focus/View changes ---
	case FocusChangeMsg:
		m.focused = msg.Panel
		return m, nil

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

	// --- Repo events ---
	case RepoListLoadedMsg:
		m.store.Repositories.SetRepos(msg.Repos)
		return m, nil

	case RepoListErrorMsg:
		m.setNotification("error", msg.Err.Error())
		return m, nil

	case RepoSelectMsg:
		return m, nil

	// --- Git status ---
	case GitStatusMsg:
		m.fileChanges = msg.Changes
		// Reset staging selection
		m.stagingSelected = make(map[int]bool)
		for i, fc := range m.fileChanges {
			if fc.Staged {
				m.stagingSelected[i] = true
			}
		}
		return m, nil

	case GitStatusErrorMsg:
		m.setNotification("error", msg.Err.Error())
		return m, nil

	// --- Git commit ---
	case GitCommitMsg:
		m.setNotification("success", fmt.Sprintf("Commit realizado: %s", msg.Hash[:7]))
		m.commitMsg = ""
		m.commitInput.SetValue("")
		m.mode = ModeNormal
		m.stagingSelected = make(map[int]bool)
		return m, refreshGitStatus(m.gitOps)

	case GitCommitErrorMsg:
		m.setNotification("error", fmt.Sprintf("Erro no commit: %v", msg.Err))
		m.mode = ModeNormal
		return m, nil

	// --- Git push ---
	case GitPushMsg:
		if msg.Success {
			m.setNotification("success", "Push realizado com sucesso")
		}
		return m, nil

	case GitPushErrorMsg:
		m.setNotification("error", msg.Err.Error())
		return m, nil

	// --- Git log loaded ---
	case GitLogLoadedMsg:
		m.timelineCommits = msg.Commits
		m.timelineIndex = 0
		return m, nil

	case GitLogErrorMsg:
		m.setNotification("error", msg.Err.Error())
		return m, nil

	// --- Remotes loaded ---
	case RemotesLoadedMsg:
		m.remoteList = msg.Remotes
		return m, nil

	case RemotesErrorMsg:
		m.setNotification("error", msg.Err.Error())
		return m, nil

	// --- Error/Success ---
	case ErrorMsg:
		m.setNotification("error", msg.Err.Error())
		return m, nil

	case SuccessMsg:
		m.setNotification("success", msg.Message)
		return m, nil

	case NotificationMsg:
		m.notification = &msg
		m.notifTimer = time.Now()
		return m, nil

	// --- Auth events ---
	case AuthCompleteMsg:
		if msg.Success {
			m.setNotification("success", "Autenticação realizada com sucesso")
		} else {
			errStr := "erro desconhecido"
			if msg.Error != nil {
				errStr = msg.Error.Error()
			}
			m.setNotification("error", fmt.Sprintf("Falha na autenticação: %s", errStr))
		}
		m.mode = ModeNormal
		m.showAuth = false
		return m, nil

	// --- Provider switch ---
	case ProviderSwitchMsg:
		if err := m.registry.SetActive(msg.Provider); err != nil {
			return m, func() tea.Msg { return ErrorMsg{Err: err} }
		}
		m.store.Settings.SetActiveProvider(msg.Provider)
		m.setNotification("info", fmt.Sprintf("Provedor: %s", msg.Provider))
		return m, nil

	// --- Branches loaded ---
	case GitBranchesLoadedMsg:
		m.store.Branches.SetBranches(msg.Branches)
		return m, nil

	case GitBranchesErrorMsg:
		m.setNotification("error", msg.Err.Error())
		return m, nil
	}

	return m, nil
}

// ---------------------------------------------------------------------------
// handleKeyMsg
// ---------------------------------------------------------------------------

func (m Model) handleKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// --- Input modes ---
	switch m.mode {
	case ModeCommitMessage:
		return m.handleCommitInput(msg)
	case ModeAuthInput, ModeAuthMethod:
		return m.handleAuthInput(msg)
	case ModeAddRemote:
		return m.handleRemoteInput(msg)
	}

	// --- Global keybindings ---
	switch {
	case key.Matches(msg, m.keys.Quit):
		if m.showHelp {
			m.showHelp = false
			return m, nil
		}
		return m, tea.Quit

	case key.Matches(msg, m.keys.Help):
		m.showHelp = !m.showHelp
		return m, nil

	case key.Matches(msg, m.keys.FocusNext):
		if m.showStaging || m.showTimeline || m.showRemotes || m.showAuth || m.showSearch {
			return m, nil
		}
		m.focused = PanelID((int(m.focused) + 1) % NumPanels)
		return m, nil

	case key.Matches(msg, m.keys.FocusPrev):
		if m.showStaging || m.showTimeline || m.showRemotes || m.showAuth || m.showSearch {
			return m, nil
		}
		m.focused = PanelID((int(m.focused) - 1 + NumPanels) % NumPanels)
		return m, nil

	case key.Matches(msg, m.keys.Enter):
		return m.handleEnter()

	case key.Matches(msg, m.keys.Escape):
		return m.handleEscape()

	// --- Navigation within panels ---
	case key.Matches(msg, m.keys.Up):
		return m.handleUp()

	case key.Matches(msg, m.keys.Down):
		return m.handleDown()

	case key.Matches(msg, m.keys.PageUp):
		return m.handlePageUp()

	case key.Matches(msg, m.keys.PageDown):
		return m.handlePageDown()

	// --- Actions ---
	case key.Matches(msg, m.keys.Refresh):
		return m, m.refreshAll()

	case key.Matches(msg, m.keys.Commit):
		if !m.showStaging && !m.showTimeline && !m.showRemotes {
			m.showStaging = true
			return m, m.loadGitStatus()
		}
		return m, nil

	case key.Matches(msg, m.keys.Stage):
		if m.showStaging {
			return m.handleStageToggle()
		}
		return m, nil

	case key.Matches(msg, m.keys.Push):
		if m.showStaging {
			return m.handlePush()
		}
		return m, nil

	case key.Matches(msg, m.keys.Pull):
		return m.handlePull()

	case key.Matches(msg, m.keys.Branch):
		// Toggle branch list view in center panel
		if m.centerView == ViewBranchList {
			m.centerView = ViewCommitLog
		} else {
			m.centerView = ViewBranchList
		}
		return m, m.loadBranches()

	case key.Matches(msg, m.keys.ProviderSwitch):
		m.showAuth = !m.showAuth
		if m.showAuth {
			m.mode = ModeAuthMethod
		} else {
			m.mode = ModeNormal
		}
		return m, nil

	case key.Matches(msg, m.keys.Diff):
		return m.handleDiff()

	case key.Matches(msg, m.keys.Search):
		m.showTimeline = !m.showTimeline
		if m.showTimeline {
			return m, m.loadTimeline()
		}
		return m, nil
	}

	return m, nil
}

// ---------------------------------------------------------------------------
// Input handlers
// ---------------------------------------------------------------------------

func (m Model) handleCommitInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEnter:
		msgText := strings.TrimSpace(m.commitInput.Value())
		if msgText == "" {
			m.setNotification("warning", "Mensagem do commit não pode estar vazia")
			return m, nil
		}
		m.commitMsg = msgText
		return m, m.executeCommit(msgText)

	case tea.KeyEsc:
		m.mode = ModeNormal
		m.commitInput.SetValue("")
		return m, nil

	default:
		var cmd tea.Cmd
		m.commitInput, cmd = m.commitInput.Update(msg)
		return m, cmd
	}
}

func (m Model) handleAuthInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEnter:
		value := strings.TrimSpace(m.authInput.Value())
		if value == "" {
			m.setNotification("warning", "Valor não pode estar vazio")
			return m, nil
		}
		// Determine method based on mode
		method := auth.AuthMethodDirect
		if m.mode == ModeAuthMethod {
			// Check if it looks like an env var name
			if strings.HasPrefix(value, "$") {
				value = strings.TrimPrefix(value, "$")
				method = auth.AuthMethodEnvVar
			} else if strings.Contains(value, "_TOKEN") || strings.Contains(value, "_KEY") ||
				strings.ToUpper(value) == value && len(value) > 3 {
				method = auth.AuthMethodEnvVar
			} else {
				method = auth.AuthMethodDirect
			}
		}
		m.authManager.SetToken(m.authProvider, method, value)
		m.setNotification("success", "Token configurado para "+m.authProvider)
		m.mode = ModeNormal
		m.showAuth = false
		return m, nil

	case tea.KeyEsc:
		m.mode = ModeNormal
		m.showAuth = false
		return m, nil

	default:
		var cmd tea.Cmd
		m.authInput, cmd = m.authInput.Update(msg)
		return m, cmd
	}
}

func (m Model) handleRemoteInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEnter:
		url := strings.TrimSpace(m.remoteInput.Value())
		if url == "" {
			m.setNotification("warning", "URL não pode estar vazia")
			return m, nil
		}
		name := m.remoteName
		if name == "" {
			name = "origin"
		}
		return m, m.executeAddRemote(name, url)

	case tea.KeyEsc:
		m.mode = ModeNormal
		m.showRemotes = false
		return m, nil

	default:
		var cmd tea.Cmd
		m.remoteInput, cmd = m.remoteInput.Update(msg)
		return m, cmd
	}
}

// ---------------------------------------------------------------------------
// Action handlers
// ---------------------------------------------------------------------------

func (m Model) handleEnter() (tea.Model, tea.Cmd) {
	// If in staging mode, start commit
	if m.showStaging {
		// Check if anything is staged
		hasStaged := false
		for _, v := range m.stagingSelected {
			if v {
				hasStaged = true
				break
			}
		}
		if !hasStaged {
			m.setNotification("warning", "Nada selecionado para commit")
			return m, nil
		}
		m.mode = ModeCommitMessage
		m.commitInput.Focus()
		return m, nil
	}
	return m, nil
}

func (m Model) handleEscape() (tea.Model, tea.Cmd) {
	if m.showHelp {
		m.showHelp = false
	} else if m.showStaging {
		m.showStaging = false
	} else if m.showTimeline {
		m.showTimeline = false
	} else if m.showRemotes {
		m.showRemotes = false
	} else if m.showAuth {
		m.showAuth = false
		m.mode = ModeNormal
	} else if m.showSearch {
		m.showSearch = false
	}
	return m, nil
}

func (m Model) handleUp() (tea.Model, tea.Cmd) {
	if m.showStaging {
		if m.selectedFile > 0 {
			m.selectedFile--
		}
	} else if m.showTimeline {
		if m.timelineIndex > 0 {
			m.timelineIndex--
		}
	} else if m.centerView == ViewCommitLog {
		idx := m.store.Commits.SelectedIndex()
		if idx > 0 {
			m.store.Commits.Select(idx - 1)
		}
	} else if m.centerView == ViewBranchList {
		// Branch navigation handled by store selection
	}
	return m, nil
}

func (m Model) handleDown() (tea.Model, tea.Cmd) {
	if m.showStaging {
		if m.selectedFile < len(m.fileChanges)-1 {
			m.selectedFile++
		}
	} else if m.showTimeline {
		if m.timelineIndex < len(m.timelineCommits)-1 {
			m.timelineIndex++
		}
	} else if m.centerView == ViewCommitLog {
		idx := m.store.Commits.SelectedIndex()
		if idx < len(m.store.Commits.Commits())-1 {
			m.store.Commits.Select(idx + 1)
		}
	}
	return m, nil
}

func (m Model) handlePageUp() (tea.Model, tea.Cmd) {
	if m.showStaging {
		m.selectedFile -= 10
		if m.selectedFile < 0 {
			m.selectedFile = 0
		}
	}
	return m, nil
}

func (m Model) handlePageDown() (tea.Model, tea.Cmd) {
	if m.showStaging {
		m.selectedFile += 10
		if m.selectedFile >= len(m.fileChanges) {
			m.selectedFile = len(m.fileChanges) - 1
		}
	}
	return m, nil
}

func (m Model) handleStageToggle() (tea.Model, tea.Cmd) {
	if m.selectedFile >= 0 && m.selectedFile < len(m.fileChanges) {
		m.stagingSelected[m.selectedFile] = !m.stagingSelected[m.selectedFile]
	}
	return m, nil
}

func (m Model) handlePush() (tea.Model, tea.Cmd) {
	if m.fileChanges == nil {
		return m, nil
	}
	// First stage selected files, then push
	var paths []string
	fc := m.fileChanges
	for i := range fc {
		if m.stagingSelected[i] && !fc[i].Staged {
			paths = append(paths, fc[i].Path)
		}
	}
	if len(paths) > 0 {
		return m, m.executeStage(paths)
	}
	return m, nil
}

func (m Model) handlePull() (tea.Model, tea.Cmd) {
	return m, m.executePull()
}

func (m Model) handleDiff() (tea.Model, tea.Cmd) {
	if m.showStaging && m.selectedFile >= 0 && m.selectedFile < len(m.fileChanges) {
		path := m.fileChanges[m.selectedFile].Path
		return m, m.executeDiff(path)
	}
	return m, nil
}

// ---------------------------------------------------------------------------
// Commands (tea.Cmd that return tea.Msg)
// ---------------------------------------------------------------------------

func (m Model) refreshAll() tea.Cmd {
	return tea.Batch(
		m.loadGitStatus(),
		m.loadBranches(),
		m.loadTimeline(),
		m.loadRemotes(),
	)
}

func (m Model) loadGitStatus() tea.Cmd {
	return func() tea.Msg {
		ctx := context.TODO()
		changes, err := m.gitOps.Status(ctx)
		if err != nil {
			return GitStatusErrorMsg{Err: err}
		}
		return GitStatusMsg{Changes: changes}
	}
}

func (m Model) loadBranches() tea.Cmd {
	return func() tea.Msg {
		ctx := context.TODO()
		branches, err := m.gitOps.Branches(ctx)
		if err != nil {
			return GitBranchesErrorMsg{Err: err}
		}
		return GitBranchesLoadedMsg{Branches: branches}
	}
}

func (m Model) loadTimeline() tea.Cmd {
	return func() tea.Msg {
		ctx := context.TODO()
		commits, err := m.gitOps.Log(ctx, &gitlocal.LogOptions{Limit: 50})
		if err != nil {
			return GitLogErrorMsg{Err: err}
		}
		return GitLogLoadedMsg{Commits: commits}
	}
}

func (m Model) loadRemotes() tea.Cmd {
	return func() tea.Msg {
		ctx := context.TODO()
		remotes, err := m.gitOps.Remotes(ctx)
		if err != nil {
			return RemotesErrorMsg{Err: err}
		}
		return RemotesLoadedMsg{Remotes: remotes}
	}
}

func (m Model) executeCommit(message string) tea.Cmd {
	return func() tea.Msg {
		// Stage selected files first
		ctx := context.TODO()
		var paths []string
		for i, staged := range m.stagingSelected {
			if staged && !m.fileChanges[i].Staged {
				paths = append(paths, m.fileChanges[i].Path)
			}
		}
		if len(paths) > 0 {
			if err := m.gitOps.Stage(ctx, paths...); err != nil {
				return GitCommitErrorMsg{Err: fmt.Errorf("stage: %w", err)}
			}
		}

		hash, err := m.gitOps.Commit(ctx, message)
		if err != nil {
			return GitCommitErrorMsg{Err: err}
		}
		return GitCommitMsg{Hash: hash}
	}
}

func (m Model) executeStage(paths []string) tea.Cmd {
	return func() tea.Msg {
		ctx := context.TODO()
		if err := m.gitOps.Stage(ctx, paths...); err != nil {
			return GitStatusErrorMsg{Err: err}
		}
		changes, err := m.gitOps.Status(ctx)
		if err != nil {
			return GitStatusErrorMsg{Err: err}
		}
		return GitStatusMsg{Changes: changes}
	}
}

func (m Model) executePull() tea.Cmd {
	return func() tea.Msg {
		ctx := context.TODO()
		if err := m.gitOps.Pull(ctx, "origin", ""); err != nil {
			return ErrorMsg{Err: err}
		}
		return SuccessMsg{Message: "Pull realizado com sucesso"}
	}
}

func (m Model) executeDiff(path string) tea.Cmd {
	return func() tea.Msg {
		ctx := context.TODO()
		diff, err := m.gitOps.Diff(ctx, path)
		if err != nil {
			return ErrorMsg{Err: err}
		}
		return GitDiffMsg{Diff: diff}
	}
}

func (m Model) executeAddRemote(name, url string) tea.Cmd {
	return func() tea.Msg {
		ctx := context.TODO()
		if err := m.gitOps.AddRemote(ctx, name, url); err != nil {
			return ErrorMsg{Err: err}
		}
		m.mode = ModeNormal
		m.showRemotes = false
		return SuccessMsg{Message: fmt.Sprintf("Remote %s adicionado: %s", name, url)}
	}
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func (m *Model) setNotification(level, message string) {
	m.notification = &NotificationMsg{Level: level, Message: message}
	m.notifTimer = time.Now()
}

func refreshGitStatus(gitOps gitlocal.GitOperations) tea.Cmd {
	return func() tea.Msg {
		ctx := context.TODO()
		changes, err := gitOps.Status(ctx)
		if err != nil {
			return GitStatusErrorMsg{Err: err}
		}
		return GitStatusMsg{Changes: changes}
	}
}

// ---------------------------------------------------------------------------
// View
// ---------------------------------------------------------------------------

func (m Model) View() string {
	if !m.ready {
		return "Carregando..."
	}

	var b strings.Builder

	// Render panels
	leftPanel := m.renderLeftPanel()
	centerPanel := m.renderCenterPanel()
	rightPanel := m.renderRightPanel()

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
	if m.showStaging {
		b.WriteString(m.renderStagingOverlay())
	}
	if m.showTimeline {
		b.WriteString(m.renderTimelineOverlay())
	}
	if m.showRemotes {
		b.WriteString(m.renderRemotesOverlay())
	}
	if m.showAuth {
		b.WriteString(m.renderAuthOverlay())
	}
	if m.mode == ModeCommitMessage {
		b.WriteString(m.renderCommitInputOverlay())
	}
	if m.notification != nil && time.Since(m.notifTimer) < 4*time.Second {
		b.WriteString(m.renderNotification())
	}

	return b.String()
}

// ---------------------------------------------------------------------------
// Panel renderers
// ---------------------------------------------------------------------------

func (m Model) renderLeftPanel() string {
	var b strings.Builder

	// Title
	b.WriteString(m.theme.TitleStyle.Render(" Repositórios "))
	b.WriteString("\n")

	repos := m.store.Repositories.Repos()
	if len(repos) == 0 {
		b.WriteString("  Nenhum repositório\n")
	} else {
		for i, repo := range repos {
			cursor := " "
			if i == m.store.Repositories.SelectedIndex() {
				cursor = "▶"
			}
			style := m.theme.BaseStyle
			if i == m.store.Repositories.SelectedIndex() {
				style = m.theme.SelectedStyle
			}
			b.WriteString(style.Render(fmt.Sprintf(" %s %s/%s", cursor, repo.Owner, repo.Name)))
			b.WriteString("\n")
		}
	}

	b.WriteString("\n")
	b.WriteString(m.theme.TitleStyle.Render(" Provedores "))
	b.WriteString("\n")

	for _, p := range m.registry.All() {
		mark := "○"
		if p.Name() == m.store.Settings.ActiveProvider() {
			mark = "●"
		}
		authMark := " "
		if m.authManager.IsAuthenticated(p.Name()) {
			authMark = "✓"
		}
		b.WriteString(fmt.Sprintf(" %s %s %s %s\n", mark, p.Icon(), p.DisplayName(), authMark))
	}

	// Quick status
	b.WriteString("\n")
	b.WriteString(m.theme.TitleStyle.Render(" Atalhos "))
	b.WriteString("\n")
	b.WriteString("  c: commit\n")
	b.WriteString("  s: stage/unstage\n")
	b.WriteString("  p: push\n")
	b.WriteString("  l: pull\n")
	b.WriteString("  b: branches\n")
	b.WriteString("  /: timeline\n")
	b.WriteString("  P: auth\n")
	b.WriteString("  ?: ajuda\n")

	return b.String()
}

func (m Model) renderCenterPanel() string {
	var b strings.Builder

	switch m.centerView {
	case ViewCommitLog:
		b.WriteString(m.renderCommitLog())
	case ViewBranchList:
		b.WriteString(m.renderBranchList())
	case ViewDiffViewer:
		b.WriteString("Diff Viewer\n")
	default:
		b.WriteString(m.renderCommitLog())
	}

	return b.String()
}

func (m Model) renderCommitLog() string {
	var b strings.Builder

	activeBranch := m.store.Branches.Active()
	b.WriteString(m.theme.TitleStyle.Render(fmt.Sprintf(" Branch: %s ", activeBranch)))
	b.WriteString("\n\n")

	commits := m.store.Commits.Commits()
	if len(commits) == 0 {
		b.WriteString("  Nenhum commit encontrado\n")
		b.WriteString("  Pressione 'r' para atualizar, '/' para timeline\n")
	} else {
		for i, c := range commits {
			hash := c.ShortHash
			if len(hash) > 7 {
				hash = hash[:7]
			}
			style := m.theme.BaseStyle
			sel := m.store.Commits.SelectedIndex()
			cursor := " "
			if i == sel {
				style = m.theme.SelectedStyle
				cursor = "▶"
			}
			timeStr := c.Timestamp.Format("2006-01-02 15:04")
			msgHead := c.MessageHead
			if len(msgHead) > 45 {
				msgHead = msgHead[:45] + "..."
			}
			b.WriteString(style.Render(fmt.Sprintf(" %s %s %s  %s", cursor, timeStr, hash, msgHead)))
			b.WriteString("\n")
		}
	}

	return b.String()
}

func (m Model) renderBranchList() string {
	var b strings.Builder
	b.WriteString(m.theme.TitleStyle.Render(" Branches "))
	b.WriteString("\n\n")

	branches := m.store.Branches.Branches()
	if len(branches) == 0 {
		b.WriteString("  Nenhum branch encontrado\n")
		return b.String()
	}

	for _, br := range branches {
		cursor := " "
		style := m.theme.BaseStyle
		if br.IsActive {
			cursor = "▶"
			style = m.theme.SelectedStyle
		}
		remote := ""
		if br.IsRemote {
			remote = " [remote]"
		}
		ab := ""
		if br.Ahead > 0 || br.Behind > 0 {
			ab = fmt.Sprintf(" ↑%d↓%d", br.Ahead, br.Behind)
		}
		b.WriteString(style.Render(fmt.Sprintf(" %s %s%s%s", cursor, br.Name, remote, ab)))
		b.WriteString("\n")
	}

	return b.String()
}

func (m Model) renderRightPanel() string {
	var b strings.Builder

	repo := m.store.Repositories.Selected()
	if repo == nil {
		b.WriteString("  Nenhum repositório\n  selecionado\n")
		b.WriteString("\n  Pressione 'r' para\n  status do diretório\n")
		return b.String()
	}

	b.WriteString(m.theme.TitleStyle.Render(" Detalhes "))
	b.WriteString("\n\n")
	b.WriteString(fmt.Sprintf("  Nome: %s\n", repo.FullName))
	b.WriteString(fmt.Sprintf("  Dono: %s\n", repo.Owner))
	b.WriteString(fmt.Sprintf("  Lang: %s\n", repo.Language))
	b.WriteString(fmt.Sprintf("  Stars: %d\n", repo.Stars))
	b.WriteString(fmt.Sprintf("  Forks: %d\n", repo.Forks))

	// File changes summary
	if len(m.fileChanges) > 0 {
		b.WriteString("\n")
		b.WriteString(m.theme.TitleStyle.Render(" Mudanças "))
		b.WriteString("\n")
		staged := 0
		unstaged := 0
		for _, fc := range m.fileChanges {
			if fc.Staged {
				staged++
			} else {
				unstaged++
			}
		}
		b.WriteString(fmt.Sprintf("  Stage:    %d\n", staged))
		b.WriteString(fmt.Sprintf("  Unstage:  %d\n", unstaged))
		b.WriteString(fmt.Sprintf("  Total:    %d\n", len(m.fileChanges)))
	}

	// Auth status
	b.WriteString("\n")
	b.WriteString(m.theme.TitleStyle.Render(" Autenticação "))
	b.WriteString("\n")
	for _, p := range m.registry.All() {
		status := "○"
		if m.authManager.IsAuthenticated(p.Name()) {
			status = "●"
		}
		b.WriteString(fmt.Sprintf("  %s %s\n", status, p.DisplayName()))
	}

	return b.String()
}

func (m Model) renderStatusBar() string {
	activeProvider := m.store.Settings.ActiveProvider()
	activeBranch := m.store.Branches.Active()

	left := fmt.Sprintf(" %s | %s ", activeProvider, activeBranch)
	if len(m.fileChanges) > 0 {
		staged := 0
		for _, fc := range m.fileChanges {
			if fc.Staged {
				staged++
			}
		}
		left += fmt.Sprintf("| +%d ~%d ", staged, len(m.fileChanges)-staged)
	}

	right := time.Now().Format(" 15:04 ")
	spaces := m.width - len(left) - len(right)
	if spaces < 1 {
		spaces = 1
	}
	return m.theme.StatusBarStyle.Render(left + strings.Repeat(" ", spaces) + right)
}

// ---------------------------------------------------------------------------
// Overlays
// ---------------------------------------------------------------------------

func (m Model) renderStagingOverlay() string {
	var b strings.Builder

	// Calculate overlay dimensions
	ovWidth := 60
	ovHeight := len(m.fileChanges) + 8
	if ovHeight > m.height-4 {
		ovHeight = m.height - 4
	}
	if len(m.fileChanges) == 0 {
		ovHeight = 6
	}

	// Draw overlay
	b.WriteString("\n")
	b.WriteString(strings.Repeat(" ", (m.width-ovWidth)/2))
	b.WriteString("╔" + strings.Repeat("═", ovWidth-2) + "╗\n")

	title := " Stage para Commit "
	padding := (ovWidth - len(title) - 2) / 2
	b.WriteString(strings.Repeat(" ", (m.width-ovWidth)/2))
	b.WriteString("║" + strings.Repeat(" ", padding) + title + strings.Repeat(" ", ovWidth-2-len(title)-padding) + "║\n")

	if len(m.fileChanges) == 0 {
		b.WriteString(strings.Repeat(" ", (m.width-ovWidth)/2))
		b.WriteString("║" + strings.Repeat(" ", ovWidth-2) + "║\n")
		b.WriteString(strings.Repeat(" ", (m.width-ovWidth)/2))
		b.WriteString("║  Nenhuma mudança no diretório                       ║\n")
	} else {
		b.WriteString(strings.Repeat(" ", (m.width-ovWidth)/2))
		b.WriteString("╠═" + " Status │ Arquivo" + strings.Repeat(" ", ovWidth-20) + "╣\n")

		start := 0
		if m.selectedFile > 0 && m.selectedFile >= ovHeight-6 {
			start = m.selectedFile - (ovHeight - 7)
		}

		end := start + ovHeight - 6
		if end > len(m.fileChanges) {
			end = len(m.fileChanges)
		}

		for i := start; i < end; i++ {
			fc := m.fileChanges[i]
			selected := m.stagingSelected[i]
			mark := " "
			if selected {
				mark = "✓"
			}
			cursor := " "
			if i == m.selectedFile {
				cursor = "▶"
			}
			statusChar := fc.StatusShort()
			statusStyle := m.theme.BaseStyle
			if fc.Status == types.FileStatusModified || fc.StagedStatus == types.FileStatusModified {
				statusStyle = m.theme.DiffAdded
			} else if fc.Status == types.FileStatusDeleted || fc.StagedStatus == types.FileStatusDeleted {
				statusStyle = m.theme.DiffDeleted
			}
			path := fc.Path
			if len(path) > 45 {
				path = path[:45] + "..."
			}

			line := fmt.Sprintf(" %s[%s] %s %s", cursor, mark, statusStyle.Render(statusChar), path)
			b.WriteString(strings.Repeat(" ", (m.width-ovWidth)/2))
			b.WriteString("║" + line + strings.Repeat(" ", ovWidth-2-len(line)) + "║\n")
		}
	}

	// Controls
	b.WriteString(strings.Repeat(" ", (m.width-ovWidth)/2))
	b.WriteString("╠═" + strings.Repeat("═", ovWidth-4) + "╣\n")
	b.WriteString(strings.Repeat(" ", (m.width-ovWidth)/2))
	b.WriteString(fmt.Sprintf("║  ↑↓: navegar  s: stage/unstage  enter: commitar  p: push  esc: voltar%s║\n",
		strings.Repeat(" ", 10)))
	b.WriteString(strings.Repeat(" ", (m.width-ovWidth)/2))
	b.WriteString("╚" + strings.Repeat("═", ovWidth-2) + "╝")

	return b.String()
}

func (m Model) renderTimelineOverlay() string {
	var b strings.Builder
	ovWidth := 70
	ovHeight := len(m.timelineCommits) + 6
	if ovHeight > m.height-4 {
		ovHeight = m.height - 4
	}
	if len(m.timelineCommits) == 0 {
		ovHeight = 5
	}

	b.WriteString("\n")
	b.WriteString(strings.Repeat(" ", (m.width-ovWidth)/2))
	b.WriteString("╔" + strings.Repeat("═", ovWidth-2) + "╗\n")

	title := " Timeline de Commits "
	padding := (ovWidth - len(title) - 2) / 2
	b.WriteString(strings.Repeat(" ", (m.width-ovWidth)/2))
	b.WriteString("║" + strings.Repeat(" ", padding) + title + strings.Repeat(" ", ovWidth-2-len(title)-padding) + "║\n")

	if len(m.timelineCommits) == 0 {
		b.WriteString(strings.Repeat(" ", (m.width-ovWidth)/2))
		b.WriteString("║  Nenhum commit encontrado                            ║\n")
	} else {
		b.WriteString(strings.Repeat(" ", (m.width-ovWidth)/2))
		b.WriteString("╠═ Data/Hora    │ Hash   │ Autor           │ Mensagem" + strings.Repeat(" ", ovWidth-45) + "╣\n")

		start := 0
		maxVisible := ovHeight - 6
		if m.timelineIndex > maxVisible-1 {
			start = m.timelineIndex - maxVisible + 1
		}
		end := start + maxVisible
		if end > len(m.timelineCommits) {
			end = len(m.timelineCommits)
		}

		for i := start; i < end; i++ {
			c := m.timelineCommits[i]
			cursor := " "
			if i == m.timelineIndex {
				cursor = "▶"
			}
			hash := c.ShortHash
			timeStr := c.Timestamp.Format("2006-01-02 15:04")
			msgHead := c.MessageHead
			if len(msgHead) > 30 {
				msgHead = msgHead[:30] + ".."
			}
			author := c.Author
			if len(author) > 15 {
				author = author[:15]
			}
			line := fmt.Sprintf(" %s %s │ %s │ %-15s │ %s", cursor, timeStr, hash, author, msgHead)
			b.WriteString(strings.Repeat(" ", (m.width-ovWidth)/2))
			b.WriteString("║" + line + strings.Repeat(" ", ovWidth-2-len(line)) + "║\n")
		}
	}

	b.WriteString(strings.Repeat(" ", (m.width-ovWidth)/2))
	b.WriteString("╚" + strings.Repeat("═", ovWidth-2) + "╝")
	return b.String()
}

func (m Model) renderRemotesOverlay() string {
	var b strings.Builder
	ovWidth := 60
	ovHeight := len(m.remoteList) + 8
	if ovHeight > m.height-4 {
		ovHeight = m.height - 4
	}

	b.WriteString("\n")
	b.WriteString(strings.Repeat(" ", (m.width-ovWidth)/2))
	b.WriteString("╔" + strings.Repeat("═", ovWidth-2) + "╗\n")

	title := " Gerenciar Remotes "
	padding := (ovWidth - len(title) - 2) / 2
	b.WriteString(strings.Repeat(" ", (m.width-ovWidth)/2))
	b.WriteString("║" + strings.Repeat(" ", padding) + title + strings.Repeat(" ", ovWidth-2-len(title)-padding) + "║\n")

	if len(m.remoteList) == 0 {
		b.WriteString(strings.Repeat(" ", (m.width-ovWidth)/2))
		b.WriteString("║  Nenhum remote configurado                           ║\n")
	} else {
		for _, r := range m.remoteList {
			url := ""
			if len(r.URLs) > 0 {
				url = r.URLs[0]
				if len(url) > 40 {
					url = url[:40] + "..."
				}
			}
			line := fmt.Sprintf("  %s: %s", r.Name, url)
			b.WriteString(strings.Repeat(" ", (m.width-ovWidth)/2))
			b.WriteString("║" + line + strings.Repeat(" ", ovWidth-2-len(line)) + "║\n")
		}
	}

	b.WriteString(strings.Repeat(" ", (m.width-ovWidth)/2))
	b.WriteString("╠" + strings.Repeat("═", ovWidth-2) + "╣\n")
	if m.mode == ModeAddRemote {
		b.WriteString(strings.Repeat(" ", (m.width-ovWidth)/2))
		b.WriteString("║ Nome: " + m.remoteName + strings.Repeat(" ", ovWidth-14-len(m.remoteName)) + "║\n")
		b.WriteString(strings.Repeat(" ", (m.width-ovWidth)/2))
		b.WriteString("║ URL:  " + m.remoteInput.View() + strings.Repeat(" ", ovWidth-16-len(m.remoteInput.View())) + "║\n")
	}
	b.WriteString(strings.Repeat(" ", (m.width-ovWidth)/2))
	b.WriteString("╚" + strings.Repeat("═", ovWidth-2) + "╝")
	return b.String()
}

func (m Model) renderAuthOverlay() string {
	var b strings.Builder
	ovWidth := 55

	b.WriteString("\n")
	b.WriteString(strings.Repeat(" ", (m.width-ovWidth)/2))
	b.WriteString("╔" + strings.Repeat("═", ovWidth-2) + "╗\n")

	title := " Autenticação "
	padding := (ovWidth - len(title) - 2) / 2
	b.WriteString(strings.Repeat(" ", (m.width-ovWidth)/2))
	b.WriteString("║" + strings.Repeat(" ", padding) + title + strings.Repeat(" ", ovWidth-2-len(title)-padding) + "║\n")

	// Provider list
	for _, p := range m.registry.All() {
		status := "○"
		if m.authManager.IsAuthenticated(p.Name()) {
			status = "●"
		}
		line := fmt.Sprintf("  %s %s", status, p.DisplayName())

		// Show configured method
		if m.authManager.HasTokenConfig(p.Name()) {
			if method, err := m.authManager.GetMethod(p.Name()); err == nil {
				line += fmt.Sprintf(" [%s]", method)
			}
		}
		b.WriteString(strings.Repeat(" ", (m.width-ovWidth)/2))
		b.WriteString("║" + line + strings.Repeat(" ", ovWidth-2-len(line)) + "║\n")
	}

	b.WriteString(strings.Repeat(" ", (m.width-ovWidth)/2))
	b.WriteString("╠" + strings.Repeat("═", ovWidth-2) + "╣\n")

	// Token input
	providerLabel := fmt.Sprintf(" Provider: %s ", m.authProvider)
	b.WriteString(strings.Repeat(" ", (m.width-ovWidth)/2))
	b.WriteString("║" + providerLabel + strings.Repeat(" ", ovWidth-2-len(providerLabel)) + "║\n")

	authInputLine := " Token: " + m.authInput.View()
	b.WriteString(strings.Repeat(" ", (m.width-ovWidth)/2))
	remaining := ovWidth - 2 - len(authInputLine)
	if remaining < 0 {
		remaining = 0
	}
	b.WriteString("║" + authInputLine + strings.Repeat(" ", remaining) + "║\n")

	// Instructions
	var instr string
	if m.mode == ModeAuthMethod {
		instr = "  Digite o token ou nome da env var ($NOME)"
	} else {
		instr = "  Enter: confirmar   Esc: cancelar"
	}
	b.WriteString(strings.Repeat(" ", (m.width-ovWidth)/2))
	b.WriteString("║" + instr + strings.Repeat(" ", ovWidth-2-len(instr)) + "║\n")

	b.WriteString(strings.Repeat(" ", (m.width-ovWidth)/2))
	b.WriteString("╚" + strings.Repeat("═", ovWidth-2) + "╝")
	return b.String()
}

func (m Model) renderCommitInputOverlay() string {
	var b strings.Builder

	b.WriteString("\n")
	b.WriteString(strings.Repeat(" ", (m.width-60)/2))
	b.WriteString("╔" + strings.Repeat("═", 58) + "╗\n")
	b.WriteString(strings.Repeat(" ", (m.width-60)/2))
	b.WriteString("║  Mensagem do Commit:" + strings.Repeat(" ", 39) + "║\n")
	b.WriteString(strings.Repeat(" ", (m.width-60)/2))
	b.WriteString("║  " + m.commitInput.View())
	// Calculate padding for input
	inputLen := len(m.commitInput.View()) + 4
	paddingLen := 58 - inputLen
	if paddingLen < 0 {
		paddingLen = 0
	}
	b.WriteString(strings.Repeat(" ", paddingLen) + "║\n")
	b.WriteString(strings.Repeat(" ", (m.width-60)/2))
	b.WriteString("║  Enter: confirmar   Esc: cancelar" + strings.Repeat(" ", 26) + "║\n")
	b.WriteString(strings.Repeat(" ", (m.width-60)/2))
	b.WriteString("╚" + strings.Repeat("═", 58) + "╝")

	return b.String()
}

func (m Model) renderNotification() string {
	if m.notification == nil {
		return ""
	}
	color := m.theme.Info
	switch m.notification.Level {
	case "error":
		color = m.theme.Error
	case "warning":
		color = m.theme.Warning
	case "success":
		color = m.theme.Success
	}
	style := lipgloss.NewStyle().Foreground(color).Bold(true).Padding(0, 1)
	return "\n" + style.Render(" " + m.notification.Message + " ")
}

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
		fmt.Sprintf("%s  %s", m.keys.Stage.Help().Key, m.keys.Stage.Help().Desc),
		fmt.Sprintf("%s  %s", m.keys.Commit.Help().Key, m.keys.Commit.Help().Desc),
		fmt.Sprintf("%s  %s", m.keys.Push.Help().Key, m.keys.Push.Help().Desc),
		fmt.Sprintf("%s  %s", m.keys.Pull.Help().Key, m.keys.Pull.Help().Desc),
		fmt.Sprintf("%s  %s", m.keys.Branch.Help().Key, m.keys.Branch.Help().Desc),
		fmt.Sprintf("%s  %s", m.keys.Diff.Help().Key, m.keys.Diff.Help().Desc),
		fmt.Sprintf("%s  %s", m.keys.ProviderSwitch.Help().Key, m.keys.ProviderSwitch.Help().Desc),
		fmt.Sprintf("%s  %s", m.keys.Search.Help().Key, m.keys.Search.Help().Desc),
	)
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


