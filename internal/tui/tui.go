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
	RepoManager *store.RepoManager
	ThemeName   string
	ThemeDir    string
}

// Mode indica o modo atual da TUI.
type Mode int

const (
	ModeNormal Mode = iota
	ModeCommitMessage
	ModeAuthInput
	ModeAuthMethod
	ModeAddRemote
	ModeRepoAdd
)

// Model is the root Bubble Tea model.
type Model struct {
	store       *store.Store
	theme       *theme.Theme
	config      *config.Config
	registry    *providers.Registry
	authManager *auth.Manager
	gitOps      gitlocal.GitOperations
	repoManager *store.RepoManager

	keys keybindings.KeyMap

	layout  Layout
	focused PanelID

	leftView   ViewID
	centerView ViewID
	rightView  ViewID

	showHelp     bool
	showSearch   bool
	showStaging  bool
	showRemotes  bool
	showAuth     bool
	showTimeline bool
	showHistory  bool
	showRepoAdd  bool

	notification *NotificationMsg
	notifTimer   time.Time

	// F1.1: Spinner
	spinnerActive   bool
	spinnerOp       string
	spinnerChars    []string
	spinnerFrame    int
	spinnerTick     int

	// F1.1: Notification history
	notificationHistory []NotificationMsg
	historySelected     int

	width, height int
	ready         bool

	mode Mode

	// F1.4: Theme
	availableThemes  []string
	currentThemeIdx  int
	themeDir         string

	commitInput textinput.Model
	authInput   textinput.Model
	authProvider string

	remoteInput textinput.Model
	remoteName  string

	repoAddInput textinput.Model

	fileChanges     []*types.FileChange
	selectedFile    int
	commitMsg       string
	stagingSelected map[int]bool

	timelineCommits []*types.Commit
	timelineIndex   int

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

	rai := textinput.New()
	rai.Placeholder = "Caminho do repositório (ex: ~/projects/my-repo)"
	rai.Focus()
	rai.CharLimit = 300
	rai.Width = 50

	// Determine available themes
	availThemes := []string{"dark", "light"}
	themeIdx := 0
	for i, t := range availThemes {
		if t == opts.ThemeName {
			themeIdx = i
			break
		}
	}

	return &Model{
		store:       opts.Store,
		theme:       opts.Theme,
		config:      opts.Config,
		registry:    opts.Registry,
		authManager: opts.AuthManager,
		gitOps:      opts.GitOps,
		repoManager: opts.RepoManager,
		keys:        keybindings.DefaultKeyMap(),
		focused:     PanelLeft,
		leftView:    ViewRepositories,
		centerView:  ViewCommitLog,
		rightView:   ViewDetails,
		commitInput: ci,
		authInput:   ai,
		remoteInput: ri,
		repoAddInput: rai,
		fileChanges:     make([]*types.FileChange, 0),
		selectedFile:    -1,
		stagingSelected: make(map[int]bool),
		timelineCommits: make([]*types.Commit, 0),
		timelineIndex:   0,
		remoteList:      make([]*types.Remote, 0),
		notificationHistory: make([]NotificationMsg, 0),
		spinnerChars:        []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"},
		availableThemes:     availThemes,
		currentThemeIdx:     themeIdx,
		themeDir:            opts.ThemeDir,
	}
}

// Init initializes the model.
func (m Model) Init() tea.Cmd {
	return tea.Batch(
		tea.EnterAltScreen,
		m.waitForSize,
		m.loadCommits,
		m.loadGitStatus(),
		m.loadBranches(),
	)
}

func (m Model) waitForSize() tea.Msg {
	return tea.WindowSizeMsg{Width: m.width, Height: m.height}
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

	// F1.3: Mouse click to focus panel
	case tea.MouseMsg:
		if msg.Action == tea.MouseActionRelease && msg.Button == tea.MouseButtonLeft {
			if msg.X < m.layout.LeftWidth {
				m.focused = PanelLeft
			} else if msg.X < m.layout.LeftWidth+m.layout.CenterWidth {
				m.focused = PanelCenter
			} else {
				m.focused = PanelRight
			}
		}
		return m, nil

	// F1.1: Spinner tick
	case SpinnerTickMsg:
		if m.spinnerActive {
			m.spinnerTick++
			if m.spinnerTick%6 == 0 { // Slow down animation
				m.spinnerFrame = (m.spinnerFrame + 1) % len(m.spinnerChars)
			}
			return m, func() tea.Msg {
				time.Sleep(50 * time.Millisecond)
				return SpinnerTickMsg{}
			}
		}
		return m, nil

	case SpinnerStartMsg:
		m.spinnerActive = true
		m.spinnerOp = msg.Operation
		m.spinnerFrame = 0
		return m, func() tea.Msg {
			time.Sleep(50 * time.Millisecond)
			return SpinnerTickMsg{}
		}

	case SpinnerStopMsg:
		m.spinnerActive = false
		m.spinnerOp = ""
		return m, nil

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

	case RepoListLoadedMsg:
		m.store.Repositories.SetRepos(msg.Repos)
		return m, nil

	case RepoListErrorMsg:
		m.setNotification("error", msg.Err.Error())
		return m, nil

	case RepoSelectMsg:
		return m, nil

	case GitStatusMsg:
		m.fileChanges = msg.Changes
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

	case GitCommitMsg:
		m.setNotification("success", fmt.Sprintf("Commit realizado: %s", msg.Hash[:7]))
		m.commitMsg = ""
		m.commitInput.SetValue("")
		m.mode = ModeNormal
		m.stagingSelected = make(map[int]bool)
		return m, refreshGitStatus(m.gitOps)

	case GitCommitErrorMsg:
		m.setNotification("error", fmt.Sprintf("Erro: %v", msg.Err))
		m.mode = ModeNormal
		return m, nil

	case GitPushMsg:
		if msg.Success {
			m.setNotification("success", "Push realizado com sucesso")
		}
		return m, nil

	case GitPushErrorMsg:
		m.setNotification("error", msg.Err.Error())
		return m, nil

	case GitLogLoadedMsg:
		m.timelineCommits = msg.Commits
		m.timelineIndex = 0
		return m, nil

	case GitLogErrorMsg:
		m.setNotification("error", msg.Err.Error())
		return m, nil

	case RemotesLoadedMsg:
		m.remoteList = msg.Remotes
		return m, nil

	case RemotesErrorMsg:
		m.setNotification("error", msg.Err.Error())
		return m, nil

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

	case AuthCompleteMsg:
		if msg.Success {
			m.setNotification("success", "Autenticação realizada com sucesso")
		} else {
			errStr := "erro desconhecido"
			if msg.Error != nil {
				errStr = msg.Error.Error()
			}
			m.setNotification("error", fmt.Sprintf("Falha: %s", errStr))
		}
		m.mode = ModeNormal
		m.showAuth = false
		return m, nil

	case ProviderSwitchMsg:
		if err := m.registry.SetActive(msg.Provider); err != nil {
			return m, func() tea.Msg { return ErrorMsg{Err: err} }
		}
		m.store.Settings.SetActiveProvider(msg.Provider)
		m.setNotification("info", fmt.Sprintf("Provedor: %s", msg.Provider))
		return m, nil

	case GitBranchesLoadedMsg:
		m.store.Branches.SetBranches(msg.Branches)
		return m, nil

	case GitBranchesErrorMsg:
		m.setNotification("error", msg.Err.Error())
		return m, nil

	case GitCommitsLoadedMsg:
		m.store.Commits.SetCommits(msg.Commits)
		return m, nil

	case GitCommitsErrorMsg:
		m.setNotification("error", msg.Err.Error())
		return m, nil

	// F1.2: Repo management
	case RepoScanMsg:
		for _, p := range msg.Repos {
			_ = m.repoManager.Add(p)
		}
		m.setNotification("success", fmt.Sprintf("%d repositórios adicionados", len(msg.Repos)))
		return m, nil

	case RepoScanErrorMsg:
		m.setNotification("error", msg.Err.Error())
		return m, nil

	case RepoAddMsg:
		m.setNotification("success", fmt.Sprintf("Repositório adicionado: %s", msg.Name))
		return m, nil

	case RepoAddErrorMsg:
		m.setNotification("error", msg.Err.Error())
		return m, nil

	case ReposUpdatedMsg:
		m.setNotification("info", "Lista de repositórios atualizada")
		return m, nil
	}

	return m, nil
}

// ---------------------------------------------------------------------------
// Key handling
// ---------------------------------------------------------------------------

func (m Model) handleKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch m.mode {
	case ModeCommitMessage:
		return m.handleCommitInput(msg)
	case ModeAuthInput, ModeAuthMethod:
		return m.handleAuthInput(msg)
	case ModeAddRemote:
		return m.handleRemoteInput(msg)
	case ModeRepoAdd:
		return m.handleRepoAddInput(msg)
	}

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
		m.focused = PanelID((int(m.focused) + 1) % NumPanels)
		return m, nil

	case key.Matches(msg, m.keys.FocusPrev):
		m.focused = PanelID((int(m.focused) - 1 + NumPanels) % NumPanels)
		return m, nil

	case key.Matches(msg, m.keys.Enter):
		return m.handleEnter()

	case key.Matches(msg, m.keys.Escape):
		return m.handleEscape()

	case key.Matches(msg, m.keys.Up):
		return m.handleUp()

	case key.Matches(msg, m.keys.Down):
		return m.handleDown()

	case key.Matches(msg, m.keys.PageUp):
		return m.handlePageUp()

	case key.Matches(msg, m.keys.PageDown):
		return m.handlePageDown()

	case key.Matches(msg, m.keys.Refresh):
		return m, m.refreshAll()

	case key.Matches(msg, m.keys.Commit):
		if !m.showStaging && !m.showTimeline && !m.showRemotes {
			m.showStaging = true
			m.selectedFile = 0
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
		if m.centerView == ViewBranchList {
			m.centerView = ViewCommitLog
		} else {
			m.centerView = ViewBranchList
		}
		return m, m.loadBranches()

	case key.Matches(msg, m.keys.ProviderSwitch):
		m.showAuth = !m.showAuth
		m.mode = ModeAuthMethod
		if m.showAuth {
			// Set first provider as default
			providers := m.registry.List()
			if len(providers) > 0 {
				m.authProvider = providers[0]
			}
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

	// F1.3: Panel shortcuts
	case key.Matches(msg, m.keys.Panel1):
		m.focused = PanelLeft
		return m, nil

	case key.Matches(msg, m.keys.Panel2):
		m.focused = PanelCenter
		return m, nil

	case key.Matches(msg, m.keys.Panel3):
		m.focused = PanelRight
		return m, nil

	// F1.1: Notification history
	case key.Matches(msg, m.keys.History):
		m.showHistory = !m.showHistory
		return m, nil

	// F1.4: Theme toggle
	case key.Matches(msg, m.keys.ThemeToggle):
		return m.handleThemeToggle()

	// F1.2: Repo management (only when left panel is focused and no overlays open)
	case key.Matches(msg, m.keys.RepoAdd):
		if m.focused == PanelLeft && !m.showStaging && !m.showTimeline && !m.showRemotes && !m.showAuth {
			m.showRepoAdd = true
			m.mode = ModeAddRemote // reuse ModeAddRemote for repo input
			m.repoAddInput.Focus()
			return m, nil
		}
		return m, nil

	case key.Matches(msg, m.keys.RepoScan):
		if m.focused == PanelLeft {
			return m, m.scanRepos()
		}
		return m, nil

	case key.Matches(msg, m.keys.RepoRemove):
		if m.focused == PanelLeft {
			return m.handleRemoveRepo()
		}
		return m, nil

	case key.Matches(msg, m.keys.RepoFav):
		if m.focused == PanelLeft {
			m.handleFavRepo()
			return m, nil
		}
		return m, nil
	}

	return m, nil
}

// ---------------------------------------------------------------------------
// F1.4: Theme toggle
// ---------------------------------------------------------------------------

func (m Model) handleThemeToggle() (tea.Model, tea.Cmd) {
	m.currentThemeIdx = (m.currentThemeIdx + 1) % len(m.availableThemes)
	name := m.availableThemes[m.currentThemeIdx]

	// Try loading from theme dir first, fallback to built-in
	newTheme := theme.BuiltinTheme(name)
	if m.themeDir != "" {
		if loaded, err := theme.Load(m.themeDir + "/" + name + ".json"); err == nil {
			newTheme = loaded
		}
	}
	m.theme = newTheme

	// Persist preference
	m.config.ThemeName = name
	_ = m.config.Save()

	m.setNotification("info", fmt.Sprintf("Tema: %s", name))
	return m, nil
}

// ---------------------------------------------------------------------------
// F1.2: Repo management handlers
// ---------------------------------------------------------------------------

func (m Model) scanRepos() tea.Cmd {
	return func() tea.Msg {
		repos, err := m.repoManager.Scan(".")
		if err != nil {
			return RepoScanErrorMsg{Err: err}
		}
		if len(repos) == 0 {
			return RepoScanErrorMsg{Err: fmt.Errorf("nenhum repositório Git encontrado")}
		}
		for _, p := range repos {
			_ = m.repoManager.Add(p)
		}
		m.setNotification("success", fmt.Sprintf("%d repositórios adicionados", len(repos)))
		return RepoScanMsg{Repos: repos}
	}
}

func (m Model) handleRemoveRepo() (tea.Model, tea.Cmd) {
	repos := m.repoManager.List()
	if len(repos) == 0 {
		m.setNotification("warning", "Nenhum repositório para remover")
		return m, nil
	}
	idx := m.store.Repositories.SelectedIndex()
	if idx >= 0 && idx < len(repos) {
		m.repoManager.Remove(repos[idx].Path)
		m.setNotification("info", fmt.Sprintf("Removido: %s", repos[idx].Name))
	}
	return m, nil
}

func (m Model) handleFavRepo() {
	repos := m.repoManager.List()
	idx := m.store.Repositories.SelectedIndex()
	if idx >= 0 && idx < len(repos) {
		m.repoManager.ToggleFavorite(repos[idx].Path)
	}
}

// ---------------------------------------------------------------------------
// Input handlers
// ---------------------------------------------------------------------------

func (m Model) handleCommitInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEnter:
		msgText := strings.TrimSpace(m.commitInput.Value())
		if msgText == "" {
			m.setNotification("warning", "Mensagem não pode estar vazia")
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
		method := auth.AuthMethodDirect
		if m.mode == ModeAuthMethod {
			if strings.HasPrefix(value, "$") || strings.ToUpper(value) == value && len(value) > 3 {
				value = strings.TrimPrefix(value, "$")
				method = auth.AuthMethodEnvVar
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

func (m Model) handleRepoAddInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEnter:
		path := strings.TrimSpace(m.repoAddInput.Value())
		if path == "" {
			m.setNotification("warning", "Caminho não pode estar vazio")
			return m, nil
		}
		return m, m.executeRepoAdd(path)
	case tea.KeyEsc:
		m.mode = ModeNormal
		m.showRepoAdd = false
		m.repoAddInput.SetValue("")
		return m, nil
	default:
		var cmd tea.Cmd
		m.repoAddInput, cmd = m.repoAddInput.Update(msg)
		return m, cmd
	}
}

func (m Model) executeRepoAdd(path string) tea.Cmd {
	return func() tea.Msg {
		if err := m.repoManager.Add(path); err != nil {
			return RepoAddErrorMsg{Err: err}
		}
		m.mode = ModeNormal
		m.showRepoAdd = false
		m.repoAddInput.SetValue("")
		return RepoAddMsg{Path: path, Name: path}
	}
}

// ---------------------------------------------------------------------------
// Action handlers
// ---------------------------------------------------------------------------

func (m Model) handleEnter() (tea.Model, tea.Cmd) {
	if m.showStaging {
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
	switch {
	case m.showHelp:
		m.showHelp = false
	case m.showHistory:
		m.showHistory = false
	case m.showStaging:
		m.showStaging = false
	case m.showTimeline:
		m.showTimeline = false
	case m.showRemotes:
		m.showRemotes = false
	case m.showAuth:
		m.showAuth = false
		m.mode = ModeNormal
	case m.showRepoAdd:
		m.showRepoAdd = false
		m.mode = ModeNormal
	case m.showSearch:
		m.showSearch = false
	}
	return m, nil
}

func (m Model) handleUp() (tea.Model, tea.Cmd) {
	switch {
	case m.showStaging && m.selectedFile > 0:
		m.selectedFile--
	case m.showTimeline && m.timelineIndex > 0:
		m.timelineIndex--
	case m.centerView == ViewCommitLog:
		idx := m.store.Commits.SelectedIndex()
		if idx > 0 {
			m.store.Commits.Select(idx - 1)
		}
	}
	return m, nil
}

func (m Model) handleDown() (tea.Model, tea.Cmd) {
	switch {
	case m.showStaging && m.selectedFile < len(m.fileChanges)-1:
		m.selectedFile++
	case m.showTimeline && m.timelineIndex < len(m.timelineCommits)-1:
		m.timelineIndex++
	case m.centerView == ViewCommitLog:
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
	var paths []string
	for i, fc := range m.fileChanges {
		if m.stagingSelected[i] && !fc.Staged {
			paths = append(paths, fc.Path)
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
		return m, m.executeDiff(m.fileChanges[m.selectedFile].Path)
	}
	return m, nil
}

// ---------------------------------------------------------------------------
// Commands
// ---------------------------------------------------------------------------

func (m Model) refreshAll() tea.Cmd {
	return tea.Batch(m.loadGitStatus(), m.loadCommits, m.loadBranches(), m.loadTimeline(), m.loadRemotes())
}

func (m Model) loadCommits() tea.Msg {
	commits, err := m.gitOps.Log(context.TODO(), &gitlocal.LogOptions{Limit: 50})
	if err != nil {
		return GitCommitsErrorMsg{Err: err}
	}
	return GitCommitsLoadedMsg{Commits: commits}
}

func (m Model) loadGitStatus() tea.Cmd {
	return func() tea.Msg {
		changes, err := m.gitOps.Status(context.TODO())
		if err != nil {
			return GitStatusErrorMsg{Err: err}
		}
		return GitStatusMsg{Changes: changes}
	}
}

func (m Model) loadBranches() tea.Cmd {
	return func() tea.Msg {
		branches, err := m.gitOps.Branches(context.TODO())
		if err != nil {
			return GitBranchesErrorMsg{Err: err}
		}
		return GitBranchesLoadedMsg{Branches: branches}
	}
}

func (m Model) loadTimeline() tea.Cmd {
	return func() tea.Msg {
		commits, err := m.gitOps.Log(context.TODO(), &gitlocal.LogOptions{Limit: 50})
		if err != nil {
			return GitLogErrorMsg{Err: err}
		}
		return GitLogLoadedMsg{Commits: commits}
	}
}

func (m Model) loadRemotes() tea.Cmd {
	return func() tea.Msg {
		remotes, err := m.gitOps.Remotes(context.TODO())
		if err != nil {
			return RemotesErrorMsg{Err: err}
		}
		return RemotesLoadedMsg{Remotes: remotes}
	}
}

func (m Model) executeCommit(message string) tea.Cmd {
	return func() tea.Msg {
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
		if err := m.gitOps.Pull(context.TODO(), "origin", ""); err != nil {
			return ErrorMsg{Err: err}
		}
		return SuccessMsg{Message: "Pull realizado ✓"}
	}
}

func (m Model) executeDiff(path string) tea.Cmd {
	return func() tea.Msg {
		diff, err := m.gitOps.Diff(context.TODO(), path)
		if err != nil {
			return ErrorMsg{Err: err}
		}
		return GitDiffMsg{Diff: diff}
	}
}

func (m Model) executeAddRemote(name, url string) tea.Cmd {
	return func() tea.Msg {
		if err := m.gitOps.AddRemote(context.TODO(), name, url); err != nil {
			return ErrorMsg{Err: err}
		}
		m.mode = ModeNormal
		m.showRemotes = false
		return SuccessMsg{Message: fmt.Sprintf("Remote %s adicionado", name)}
	}
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func (m *Model) setNotification(level, message string) {
	msg := NotificationMsg{Level: level, Message: message}
	m.notification = &msg
	m.notifTimer = time.Now()
	// Add to history (max 50)
	m.notificationHistory = append(m.notificationHistory, msg)
	if len(m.notificationHistory) > 50 {
		m.notificationHistory = m.notificationHistory[1:]
	}
}

func refreshGitStatus(gitOps gitlocal.GitOperations) tea.Cmd {
	return func() tea.Msg {
		changes, err := gitOps.Status(context.TODO())
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
		return m.theme.TitleStyle.Render(" Carregando... ")
	}

	var b strings.Builder

	// ── Panel content with borders (F1.3) ──
	leftContent := m.renderLeftPanel()
	centerContent := m.renderCenterPanel()
	rightContent := m.renderRightPanel()

	// Apply panel styles
	leftStyle := m.theme.PanelStyle
	if m.focused == PanelLeft {
		leftStyle = m.theme.ActivePanelStyle
	}
	centerStyle := m.theme.PanelStyle
	if m.focused == PanelCenter {
		centerStyle = m.theme.ActivePanelStyle
	}
	rightStyle := m.theme.PanelStyle
	if m.focused == PanelRight {
		rightStyle = m.theme.ActivePanelStyle
	}

	leftContent = leftStyle.Width(m.layout.LeftWidth).Render(leftContent)
	centerContent = centerStyle.Width(m.layout.CenterWidth).Render(centerContent)
	rightContent = rightStyle.Width(m.layout.RightWidth).Render(rightContent)

	// ── Combine ──
	for line := 0; line < m.layout.PanelHeight-1; line++ {
		leftLine := getLine(leftContent, line, m.layout.LeftWidth)
		centerLine := getLine(centerContent, line, m.layout.CenterWidth)
		rightLine := getLine(rightContent, line, m.layout.RightWidth)
		b.WriteString(leftLine)
		b.WriteString(centerLine)
		b.WriteString(rightLine)
		if line < m.layout.PanelHeight-1 {
			b.WriteString("\n")
		}
	}

	// ── Status bar ──
	b.WriteString("\n")
	b.WriteString(m.renderStatusBar())
	b.WriteString("\n")
	b.WriteString(m.renderCommandBar())

	// ── Overlays ──
	if m.showHelp {
		b.WriteString("\n" + m.renderHelpOverlay())
	}
	if m.showHistory {
		b.WriteString("\n" + m.renderHistoryOverlay())
	}
	if m.showStaging {
		b.WriteString("\n" + m.renderStagingOverlay())
	}
	if m.showTimeline {
		b.WriteString("\n" + m.renderTimelineOverlay())
	}
	if m.showRemotes {
		b.WriteString("\n" + m.renderRemotesOverlay())
	}
	if m.showAuth {
		b.WriteString("\n" + m.renderAuthOverlay())
	}
	if m.showRepoAdd {
		b.WriteString("\n" + m.renderRepoAddOverlay())
	}
	if m.mode == ModeCommitMessage {
		b.WriteString("\n" + m.renderCommitInputOverlay())
	}
	if m.notification != nil && time.Since(m.notifTimer) < 4*time.Second {
		b.WriteString("\n" + m.renderNotification())
	}

	return b.String()
}

// ---------------------------------------------------------------------------
// Panel header renderer
// ---------------------------------------------------------------------------

func (m Model) renderPanelHeader(title string, isFocused bool) string {
	style := m.theme.DimmedStyle
	if isFocused {
		style = m.theme.ActivePanelTitleStyle
		title = "●" + title[1:]
	}
	// Pad to panel width
	return style.Render(title)
}

// ---------------------------------------------------------------------------
// Panel content
// ---------------------------------------------------------------------------

func (m Model) renderLeftPanel() string {
	var b strings.Builder

	// Provider + branch
	activeProv := m.store.Settings.ActiveProvider()
	b.WriteString(m.theme.BranchStyle.Render(" " + activeProv + " "))
	b.WriteString(m.theme.DimmedStyle.Render(m.store.Branches.Active()))
	b.WriteString("\n")

	// Separator
	b.WriteString(m.theme.DimmedStyle.Render(strings.Repeat("─", m.layout.LeftWidth)))
	b.WriteString("\n")

	// Repos from RepoManager (F1.2)
	titleStyle := m.theme.TitleStyle
	if m.focused == PanelLeft {
		titleStyle = m.theme.FocusedStyle
	}
	b.WriteString(titleStyle.Render(" Repos "))
	b.WriteString("\n")

	trackedRepos := m.repoManager.List()
	if len(trackedRepos) == 0 {
		b.WriteString(m.theme.DimmedStyle.Render("  (a: adicionar, A: scan)"))
		b.WriteString("\n")
	} else {
		for i, repo := range trackedRepos {
			cursor := " "
			if i == m.store.Repositories.SelectedIndex() {
				cursor = "▶"
			}
			style := m.theme.BaseStyle
			if i == m.store.Repositories.SelectedIndex() {
				style = m.theme.SelectedStyle
			}
			fav := " "
			if repo.IsFavorite {
				fav = m.theme.KeyStyle.Render("★")
			}
			b.WriteString(style.Render(fmt.Sprintf(" %s %s %s", cursor, fav, repo.Name)))
			b.WriteString("\n")
		}
	}

	// Providers
	b.WriteString("\n")
	b.WriteString(titleStyle.Render(" Providers "))
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
		line := fmt.Sprintf(" %s %s %s %s", mark, p.Icon(), p.DisplayName(), authMark)
		b.WriteString(m.theme.BaseStyle.Render(line))
		b.WriteString("\n")
	}

	// Quick keys
	b.WriteString("\n")
	b.WriteString(titleStyle.Render(" Keys "))
	b.WriteString("\n")
	keys := []struct{ k, d string }{
		{"c", "commit"}, {"s", "stage"}, {"p", "push"}, {"l", "pull"},
		{"b", "branch"}, {"/", "timeline"}, {"P", "auth"}, {"r", "refresh"},
		{"a", "add repo"}, {"A", "scan repos"}, {"x", "remove"}, {"f", "fav"},
		{"1-3", "panels"}, {"T", "theme"}, {"N", "history"},
	}
	for _, kv := range keys {
		b.WriteString(fmt.Sprintf("  %s  %s\n",
			m.theme.KeyStyle.Render(kv.k),
			m.theme.DimmedStyle.Render(kv.d)))
	}

	return b.String()
}

func (m Model) renderCenterPanel() string {
	var b strings.Builder
	switch m.centerView {
	case ViewCommitLog:
		b.WriteString(m.renderCommitLog())
	case ViewBranchList:
		b.WriteString(m.renderBranchList())
	default:
		b.WriteString(m.renderCommitLog())
	}
	return b.String()
}

func (m Model) renderCommitLog() string {
	var b strings.Builder

	activeBranch := m.store.Branches.Active()
	b.WriteString(m.theme.BranchStyle.Render(" ⎇ " + activeBranch + " "))
	b.WriteString("\n")

	commits := m.store.Commits.Commits()
	if len(commits) == 0 {
		b.WriteString("\n")
		b.WriteString(m.theme.DimmedStyle.Render("  Carregando commits..."))
		b.WriteString("\n")
		b.WriteString(m.theme.DimmedStyle.Render("  r: atualizar  /: timeline completa"))
		b.WriteString("\n")
	} else {
		b.WriteString("\n")
		maxCommits := m.layout.PanelHeight - 4
		if maxCommits > len(commits) {
			maxCommits = len(commits)
		}
		for i := 0; i < maxCommits; i++ {
			c := commits[i]
			style := m.theme.BaseStyle
			sel := m.store.Commits.SelectedIndex()
			cursor := " "
			if i == sel {
				style = m.theme.SelectedStyle
				cursor = "●"
			} else if i == len(commits)-1 || i == maxCommits-1 {
				cursor = "○"
			} else {
				cursor = "│"
			}

			timeStr := m.theme.DimmedStyle.Render(c.Timestamp.Format("02 Jan 15:04"))
			hash := m.theme.DimmedStyle.Render(c.ShortHash)
			msgHead := c.MessageHead
			available := m.layout.CenterWidth - 38
			if available < 10 {
				available = 10
			}
			if len(msgHead) > available {
				msgHead = msgHead[:available-1] + "…"
			}

			line := fmt.Sprintf(" %s %s %s %s", cursor, timeStr, hash, msgHead)
			b.WriteString(style.Render(line))
			if i < maxCommits-1 {
				b.WriteString("\n")
				// Graph continuation line (connecting commits)
				if i < len(commits)-1 {
					b.WriteString(m.theme.DimmedStyle.Render("  │"))
				}
			}
		}
		// Scroll hint
		if len(commits) > maxCommits {
			b.WriteString("\n" + m.theme.DimmedStyle.Render(
				fmt.Sprintf("  ↓ +%d commits  (/)", len(commits)-maxCommits)))
		}
	}

	return b.String()
}

func (m Model) renderBranchList() string {
	var b strings.Builder

	branches := m.store.Branches.Branches()
	if len(branches) == 0 {
		return m.theme.DimmedStyle.Render("  Nenhum branch encontrado")
	}

	maxBranches := m.layout.PanelHeight - 2
	if maxBranches > len(branches) {
		maxBranches = len(branches)
	}

	for i := 0; i < maxBranches; i++ {
		br := branches[i]
		style := m.theme.BaseStyle
		prefix := "  "
		if br.IsActive {
			style = m.theme.SelectedStyle
			prefix = "● "
		}
		remote := ""
		if br.IsRemote {
			remote = m.theme.DimmedStyle.Render(" [remote]")
		}
		ab := ""
		if br.Ahead > 0 || br.Behind > 0 {
			ab = m.theme.DimmedStyle.Render(fmt.Sprintf(" ↑%d↓%d", br.Ahead, br.Behind))
		}
		b.WriteString(style.Render(prefix + br.Name) + remote + ab)
		if i < maxBranches-1 {
			b.WriteString("\n")
		}
	}
	if len(branches) > maxBranches {
		b.WriteString("\n" + m.theme.DimmedStyle.Render(fmt.Sprintf("  ↓ +%d branches", len(branches)-maxBranches)))
	}

	return b.String()
}

func (m Model) renderRightPanel() string {
	var b strings.Builder

	// Git status summary
	if len(m.fileChanges) > 0 {
		staged := 0
		modified := 0
		added := 0
		deleted := 0
		untracked := 0
		for _, fc := range m.fileChanges {
			if fc.Staged {
				staged++
			}
			switch fc.Status {
			case types.FileStatusModified:
				modified++
			case types.FileStatusAdded:
				added++
			case types.FileStatusDeleted:
				deleted++
			case types.FileStatusUntracked:
				untracked++
			}
		}
		b.WriteString(m.theme.FocusedStyle.Render(" Status "))
		b.WriteString("\n")
		if staged > 0 {
			b.WriteString(fmt.Sprintf("  %s %d staged\n",
				m.theme.StagedStyle.Render("●"), staged))
		}
		if modified > 0 {
			b.WriteString(fmt.Sprintf("  %s %d modified\n",
				m.theme.StatusModified.Render("●"), modified))
		}
		if added > 0 {
			b.WriteString(fmt.Sprintf("  %s %d added\n",
				m.theme.StatusAdded.Render("●"), added))
		}
		if deleted > 0 {
			b.WriteString(fmt.Sprintf("  %s %d deleted\n",
				m.theme.StatusDeleted.Render("●"), deleted))
		}
		if untracked > 0 {
			b.WriteString(fmt.Sprintf("  %s %d untracked\n",
				m.theme.StatusUntracked.Render("●"), untracked))
		}
	}

	// Repository details
	repo := m.store.Repositories.Selected()
	if repo != nil {
		b.WriteString("\n")
		b.WriteString(m.theme.FocusedStyle.Render(" Repo "))
		b.WriteString("\n")
		b.WriteString(fmt.Sprintf("  %s\n", m.theme.BranchStyle.Render(repo.FullName)))
		b.WriteString(fmt.Sprintf("  %s\n", m.theme.DimmedStyle.Render(repo.URL)))
		if repo.Language != "" {
			b.WriteString(fmt.Sprintf("  Lang: %s\n", repo.Language))
		}
		if repo.Stars > 0 {
			b.WriteString(fmt.Sprintf("  ⭐ %d  🍴 %d\n", repo.Stars, repo.Forks))
		}
	}

	// Auth status
	b.WriteString("\n")
	b.WriteString(m.theme.FocusedStyle.Render(" Auth "))
	b.WriteString("\n")
	for _, p := range m.registry.All() {
		dot := m.theme.DimmedStyle.Render("○")
		if m.authManager.IsAuthenticated(p.Name()) {
			dot = m.theme.StagedStyle.Render("●")
		}
		b.WriteString(fmt.Sprintf("  %s %s\n", dot, p.DisplayName()))
	}

	return b.String()
}

// ---------------------------------------------------------------------------
// Status bar
// ---------------------------------------------------------------------------

func (m Model) renderStatusBar() string {
	provider := m.store.Settings.ActiveProvider()
	branch := m.store.Branches.Active()

	// Spinner indicator (F1.1)
	spinner := ""
	if m.spinnerActive {
		spinner = m.theme.WarningText.Render(" " + m.spinnerChars[m.spinnerFrame] + " " + m.spinnerOp + " ")
	}

	// Left section: provider + branch + spinner
	left := fmt.Sprintf("  %s  ⎇ %s  ", provider, branch)
	left += spinner
	if len(m.fileChanges) > 0 {
		staged := 0
		for _, fc := range m.fileChanges {
			if fc.Staged {
				staged++
			}
		}
		unstaged := len(m.fileChanges) - staged
		if staged > 0 {
			left += m.theme.StagedStyle.Render(fmt.Sprintf(" +%d", staged)) + " "
		}
		if unstaged > 0 {
			left += m.theme.UnstagedStyle.Render(fmt.Sprintf(" ~%d", unstaged)) + " "
		}
	}

	// Auth status
	providers := m.registry.List()
	for _, p := range providers {
		if m.authManager.IsAuthenticated(p) {
			left += m.theme.SuccessText.Render(" ✓" + p) + " "
		}
	}

	// Right section: time
	right := time.Now().Format(" 15:04 ")

	spaces := m.width - len(left) - len(right)
	if spaces < 1 {
		spaces = 1
	}
	return m.theme.StatusBarStyle.Render(left + strings.Repeat(" ", spaces) + right)
}

func (m Model) renderCommandBar() string {
	// Context-sensitive commands based on current state
	type cmd struct {
		key, desc string
	}
	var cmds []cmd

	if m.showStaging {
		cmds = []cmd{
			{"↑↓", "navegar"},
			{"s", "stage/unstage"},
			{"↵", "commitar"},
			{"p", "push"},
			{"d", "diff"},
			{"esc", "voltar"},
		}
	} else if m.showTimeline {
		cmds = []cmd{
			{"↑↓", "navegar"},
			{"esc", "fechar"},
		}
	} else if m.showHistory {
		cmds = []cmd{
			{"esc", "fechar"},
		}
	} else if m.showAuth {
		cmds = []cmd{
			{"↵", "confirmar"},
			{"esc", "cancelar"},
		}
	} else if m.showRemotes {
		cmds = []cmd{
			{"a", "adicionar"},
			{"esc", "fechar"},
		}
	} else if m.showRepoAdd {
		cmds = []cmd{
			{"↵", "adicionar"},
			{"esc", "cancelar"},
		}
	} else if m.mode == ModeCommitMessage {
		cmds = []cmd{
			{"↵", "commitar"},
			{"esc", "cancelar"},
		}
	} else if m.showHelp {
		cmds = []cmd{
			{"esc", "fechar"},
		}
	} else {
		// Global commands based on focused panel
		switch m.focused {
		case PanelLeft:
			cmds = []cmd{
				{"a", "add repo"},
				{"A", "scan"},
				{"x", "remove"},
				{"f", "fav"},
			}
		case PanelCenter:
			cmds = []cmd{
				{"↑↓", "navegar"},
				{"b", "branches"},
			}
		case PanelRight:
			cmds = []cmd{}
		}
		// Always show global commands
		globalCmds := []cmd{
			{"c", "commit"},
			{"p", "push"},
			{"l", "pull"},
			{"/", "timeline"},
			{"P", "auth"},
			{"r", "refresh"},
			{"1-3", "painel"},
			{"T", "tema"},
			{"N", "hist"},
			{"?", "ajuda"},
			{"q", "sair"},
		}
		cmds = append(cmds, globalCmds...)
	}

	// Render command bar
	var sb strings.Builder
	for i, c := range cmds {
		if i > 0 {
			sb.WriteString(" ")
		}
		sb.WriteString(m.theme.KeyStyle.Render(" " + c.key + " "))
		sb.WriteString(m.theme.DimmedStyle.Render(c.desc))
	}
	cmdStr := sb.String()

	// Pad to full width
	spaces := m.width - len(cmdStr)
	if spaces < 0 {
		spaces = 0
	}

	// Use a dimmed style for the command bar
	barStyle := lipgloss.NewStyle().
		Foreground(m.theme.Muted).
		Background(m.theme.Background).
		Padding(0, 2)

	return barStyle.Render(cmdStr + strings.Repeat(" ", spaces))
}

// ---------------------------------------------------------------------------
// Overlays
// ---------------------------------------------------------------------------

func (m Model) renderStagingOverlay() string {
	ovWidth := 66
	ovHeight := len(m.fileChanges) + 7
	if ovHeight > m.height-6 {
		ovHeight = m.height - 6
	}
	if ovHeight < 6 {
		ovHeight = 6
	}

	title := m.theme.OverlayTitle.Render(" Stage para Commit ")
	top := lipgloss.JoinHorizontal(lipgloss.Top,
		title,
		m.theme.DimmedStyle.Render(" ↑↓: nav  s: stage  ↵: commit  p: push  esc: sair "),
	)

	var inner strings.Builder
	if len(m.fileChanges) == 0 {
		inner.WriteString(m.theme.DimmedStyle.Render("  Nenhuma mudança no diretório de trabalho"))
	} else {
		start := 0
		maxVisible := ovHeight - 4
		if m.selectedFile > maxVisible-1 {
			start = m.selectedFile - maxVisible + 1
		}
		end := start + maxVisible
		if end > len(m.fileChanges) {
			end = len(m.fileChanges)
		}

		for i := start; i < end; i++ {
			fc := m.fileChanges[i]
			sel := m.stagingSelected[i]

			cursor := " "
			if i == m.selectedFile {
				cursor = "▶"
			}

			check := "[ ]"
			if sel {
				check = m.theme.StagedStyle.Render("[✓]")
			} else {
				check = m.theme.DimmedStyle.Render("[ ]")
			}

			// Color-code status
			var statusStyle lipgloss.Style
			switch fc.Status {
			case types.FileStatusModified:
				statusStyle = m.theme.StatusModified
			case types.FileStatusAdded:
				statusStyle = m.theme.StatusAdded
			case types.FileStatusDeleted:
				statusStyle = m.theme.StatusDeleted
			case types.FileStatusUntracked:
				statusStyle = m.theme.StatusUntracked
			case types.FileStatusRenamed:
				statusStyle = m.theme.StatusRenamed
			default:
				statusStyle = m.theme.DimmedStyle
			}

			path := fc.Path
			if len(path) > ovWidth-15 {
				path = path[:ovWidth-18] + "…"
			}

			line := fmt.Sprintf(" %s %s %s  %s",
				cursor, check, statusStyle.Render(fc.StatusShort()), path)

			style := m.theme.BaseStyle
			if i == m.selectedFile {
				style = m.theme.SelectedStyle
			}
			inner.WriteString(style.Render(line))
			if i < end-1 {
				inner.WriteString("\n")
			}
		}

		if len(m.fileChanges) > maxVisible {
			inner.WriteString("\n")
			inner.WriteString(m.theme.ScrollIndicator.Render(
				fmt.Sprintf("  ↓ %d de %d", end, len(m.fileChanges))))
		}
	}

	content := lipgloss.NewStyle().
		Width(ovWidth - 2).
		Render(top + "\n" + inner.String())

	return m.theme.OverlayBorder.
		Width(ovWidth).
		Render(centeredText(content, m.width, ovWidth))
}

func (m Model) renderTimelineOverlay() string {
	ovWidth := 76
	ovHeight := len(m.timelineCommits) + 4
	if ovHeight > m.height-6 {
		ovHeight = m.height - 6
	}
	if ovHeight < 5 {
		ovHeight = 5
	}

	title := m.theme.OverlayTitle.Render(" Timeline de Commits ")
	help := m.theme.DimmedStyle.Render(" ↑↓: navegar  esc: fechar ")

	var inner strings.Builder
	if len(m.timelineCommits) == 0 {
		inner.WriteString(m.theme.DimmedStyle.Render("  Nenhum commit encontrado"))
	} else {
		start := 0
		maxVisible := ovHeight - 3
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
				cursor = "●"
			}
			style := m.theme.BaseStyle
			if i == m.timelineIndex {
				style = m.theme.SelectedStyle
			}

			hash := m.theme.DimmedStyle.Render(c.ShortHash)
			timeStr := m.theme.DimmedStyle.Render(c.Timestamp.Format("02 Jan 2006 15:04"))
			author := c.Author
			if len(author) > 14 {
				author = author[:14]
			}
			msgHead := c.MessageHead
			avail := ovWidth - 45
			if avail < 5 {
				avail = 5
			}
			if len(msgHead) > avail {
				msgHead = msgHead[:avail-1] + "…"
			}

			line := fmt.Sprintf(" %s %s │ %s │ %-14s │ %s",
				cursor, timeStr, hash, author, msgHead)
			inner.WriteString(style.Render(line))
			if i < end-1 {
				inner.WriteString("\n")
			}
		}
	}

	content := lipgloss.NewStyle().Width(ovWidth - 2).Render(title + "\n" + help + "\n" + inner.String())
	return m.theme.OverlayBorder.Width(ovWidth).Render(centeredText(content, m.width, ovWidth))
}

func (m Model) renderRemotesOverlay() string {
	ovWidth := 60

	title := m.theme.OverlayTitle.Render(" Remotes ")
	help := m.theme.DimmedStyle.Render(" a: adicionar  esc: fechar ")

	var inner strings.Builder
	if len(m.remoteList) == 0 {
		inner.WriteString(m.theme.DimmedStyle.Render("  Nenhum remote configurado"))
	} else {
		for _, r := range m.remoteList {
			url := ""
			if len(r.URLs) > 0 {
				url = r.URLs[0]
				if len(url) > 40 {
					url = url[:40] + "…"
				}
			}
			line := fmt.Sprintf("  %s → %s", m.theme.BranchStyle.Render(r.Name), url)
			inner.WriteString(line + "\n")
		}
	}

	if m.mode == ModeAddRemote {
		inner.WriteString("\n")
		inner.WriteString(fmt.Sprintf("  Nome: %s\n", m.remoteName))
		inner.WriteString(fmt.Sprintf("  URL:  %s", m.remoteInput.View()))
	} else {
		inner.WriteString("\n  a: adicionar remote")
	}

	content := lipgloss.NewStyle().Width(ovWidth - 2).Render(title + "\n" + help + "\n" + inner.String())
	return m.theme.OverlayBorder.Width(ovWidth).Render(centeredText(content, m.width, ovWidth))
}

func (m Model) renderAuthOverlay() string {
	ovWidth := 55

	title := m.theme.OverlayTitle.Render(" Autenticação ")
	help := m.theme.DimmedStyle.Render(" ↵: confirmar  esc: fechar ")

	var inner strings.Builder
	for _, p := range m.registry.All() {
		dot := m.theme.DimmedStyle.Render("○")
		if m.authManager.IsAuthenticated(p.Name()) {
			dot = m.theme.StagedStyle.Render("●")
		}
		method := ""
		if m.authManager.HasTokenConfig(p.Name()) {
			if meth, err := m.authManager.GetMethod(p.Name()); err == nil {
				method = " [" + string(meth) + "]"
			}
		}
		line := fmt.Sprintf("  %s %s%s", dot, p.DisplayName(), method)
		inner.WriteString(line + "\n")
	}

	inner.WriteString("\n")
	inner.WriteString(fmt.Sprintf("  Provider: %s\n", m.authProvider))
	inner.WriteString(fmt.Sprintf("  Token: %s", m.authInput.View()))
	if m.mode == ModeAuthMethod {
		inner.WriteString("\n  💡 Dica: informe $NOME_VAR para env var")
	}

	content := lipgloss.NewStyle().Width(ovWidth - 2).Render(title + "\n" + help + "\n" + inner.String())
	return m.theme.OverlayBorder.Width(ovWidth).Render(centeredText(content, m.width, ovWidth))
}

func (m Model) renderCommitInputOverlay() string {
	ovWidth := 64

	title := m.theme.OverlayTitle.Render(" Mensagem do Commit ")

	var inner strings.Builder
	inner.WriteString("\n")
	inner.WriteString("  " + m.commitInput.View())
	inner.WriteString("\n\n")
	inner.WriteString(m.theme.DimmedStyle.Render("  ↵: confirmar  esc: cancelar"))

	content := lipgloss.NewStyle().Width(ovWidth - 2).Render(title + inner.String())
	return m.theme.OverlayBorder.Width(ovWidth).Render(centeredText(content, m.width, ovWidth))
}

func (m Model) renderNotification() string {
	if m.notification == nil {
		return ""
	}
	var style lipgloss.Style
	switch m.notification.Level {
	case "error":
		style = m.theme.ErrorText
	case "warning":
		style = m.theme.WarningText
	case "success":
		style = m.theme.SuccessText
	default:
		style = m.theme.InfoText
	}
	return "  " + style.Render(" " + m.notification.Message + " ")
}

func (m Model) renderHistoryOverlay() string {
	ovWidth := 66
	title := m.theme.OverlayTitle.Render(" Histórico de Notificações ")
	help := m.theme.DimmedStyle.Render(" esc: fechar ")

	var inner strings.Builder
	if len(m.notificationHistory) == 0 {
		inner.WriteString(m.theme.DimmedStyle.Render("  Nenhuma notificação no histórico"))
	} else {
		start := 0
		if len(m.notificationHistory) > 15 {
			start = len(m.notificationHistory) - 15
		}
		for i := start; i < len(m.notificationHistory); i++ {
			n := m.notificationHistory[i]
			var color lipgloss.Style
			switch n.Level {
			case "error":
				color = m.theme.ErrorText
			case "warning":
				color = m.theme.WarningText
			case "success":
				color = m.theme.SuccessText
			default:
				color = m.theme.InfoText
			}
			msg := n.Message
			if len(msg) > 50 {
				msg = msg[:50] + "…"
			}
			inner.WriteString(fmt.Sprintf("  %s %s\n", color.Render("▸"), msg))
		}
	}

	content := lipgloss.NewStyle().Width(ovWidth - 2).Render(title + "\n" + help + "\n" + inner.String())
	return m.theme.OverlayBorder.Width(ovWidth).Render(centeredText(content, m.width, ovWidth))
}

func (m Model) renderRepoAddOverlay() string {
	ovWidth := 60
	title := m.theme.OverlayTitle.Render(" Adicionar Repositório ")
	help := m.theme.DimmedStyle.Render(" ↵: confirmar  esc: cancelar ")

	var inner strings.Builder
	inner.WriteString("\n")
	inner.WriteString("  Caminho: " + m.repoAddInput.View() + "\n")
	inner.WriteString("\n")
	inner.WriteString(m.theme.DimmedStyle.Render("  Ex: ~/projects/my-repo ou /home/user/project"))

	content := lipgloss.NewStyle().Width(ovWidth - 2).Render(title + "\n" + help + "\n" + inner.String())
	return m.theme.OverlayBorder.Width(ovWidth).Render(centeredText(content, m.width, ovWidth))
}

func (m Model) renderHelpOverlay() string {
	title := m.theme.OverlayTitle.Render(" Ajuda - Teclas ")

	var inner strings.Builder
	keys := []struct{ k, d string }{
		{m.keys.Help.Help().Key, m.keys.Help.Help().Desc},
		{m.keys.Quit.Help().Key, m.keys.Quit.Help().Desc},
		{m.keys.Refresh.Help().Key, m.keys.Refresh.Help().Desc},
		{m.keys.FocusNext.Help().Key, m.keys.FocusNext.Help().Desc},
		{m.keys.FocusPrev.Help().Key, m.keys.FocusPrev.Help().Desc},
		{m.keys.Panel1.Help().Key, m.keys.Panel1.Help().Desc},
		{m.keys.Panel2.Help().Key, m.keys.Panel2.Help().Desc},
		{m.keys.Panel3.Help().Key, m.keys.Panel3.Help().Desc},
		{m.keys.Stage.Help().Key, m.keys.Stage.Help().Desc},
		{m.keys.Commit.Help().Key, m.keys.Commit.Help().Desc},
		{m.keys.Push.Help().Key, m.keys.Push.Help().Desc},
		{m.keys.Pull.Help().Key, m.keys.Pull.Help().Desc},
		{m.keys.Branch.Help().Key, m.keys.Branch.Help().Desc},
		{m.keys.Diff.Help().Key, m.keys.Diff.Help().Desc},
		{m.keys.ProviderSwitch.Help().Key, m.keys.ProviderSwitch.Help().Desc},
		{m.keys.Search.Help().Key, m.keys.Search.Help().Desc},
		{m.keys.ThemeToggle.Help().Key, m.keys.ThemeToggle.Help().Desc},
		{m.keys.History.Help().Key, m.keys.History.Help().Desc},
		{m.keys.RepoAdd.Help().Key, m.keys.RepoAdd.Help().Desc},
		{m.keys.RepoScan.Help().Key, m.keys.RepoScan.Help().Desc},
		{m.keys.RepoRemove.Help().Key, m.keys.RepoRemove.Help().Desc},
		{m.keys.RepoFav.Help().Key, m.keys.RepoFav.Help().Desc},
	}
	for _, kv := range keys {
		inner.WriteString(fmt.Sprintf("  %s  %s\n",
			m.theme.KeyStyle.Render(fmt.Sprintf("%-8s", kv.k)),
			kv.d))
	}
	inner.WriteString(m.theme.DimmedStyle.Render("  esc: fechar"))

	content := lipgloss.NewStyle().Width(44).Render(title + "\n" + inner.String())
	return m.theme.OverlayBorder.Width(46).Render(centeredText(content, m.width, 46))
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

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

// centeredText returns content padded to be centered within the given total width.
func centeredText(content string, termWidth, boxWidth int) string {
	padding := (termWidth - boxWidth) / 2
	if padding < 0 {
		padding = 0
	}
	pad := strings.Repeat(" ", padding)
	lines := strings.Split(content, "\n")
	for i, line := range lines {
		lines[i] = pad + line
	}
	return strings.Join(lines, "\n")
}
