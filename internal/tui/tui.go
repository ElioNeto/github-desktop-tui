package tui

import (
	"context"
	"fmt"
	"os/exec"
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
	ModeCreateBranch
	ModeRenameBranch
	ModeMergeBranch
	ModeDiffView
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

	// F2: File Tree Explorer
	fileTree *FileTree

	// Glint-inspired: commit graph rows
	graphRows   []*types.GraphRow
	graphColors map[int]lipgloss.Color // column -> color mapping

	commitInput textinput.Model
	authInput   textinput.Model
	authProvider string

	remoteInput textinput.Model
	remoteName  string

	repoAddInput    textinput.Model
	branchNameInput textinput.Model
	mergeBranchInput textinput.Model

	// Diff viewer (F2.1)
	diffContent    string
	diffLines      []string
	diffScrollPos  int
	diffFileName   string

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

	bni := textinput.New()
	bni.Placeholder = "Nome do novo branch"
	bni.Focus()
	bni.CharLimit = 100
	bni.Width = 40

	mbi := textinput.New()
	mbi.Placeholder = "Nome do branch para merge"
	mbi.Focus()
	mbi.CharLimit = 100
	mbi.Width = 40

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
		repoAddInput:     rai,
		branchNameInput:  bni,
		mergeBranchInput: mbi,
		diffLines:        make([]string, 0),
		fileChanges:      make([]*types.FileChange, 0),
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
		fileTree:            NewFileTree(opts.GitOps.Root(), opts.GitOps),
		graphRows:           make([]*types.GraphRow, 0),
		graphColors:         make(map[int]lipgloss.Color),
	}
}

// Init initializes the model.
func (m Model) Init() tea.Cmd {
	return tea.Batch(
		tea.EnterAltScreen,
		m.waitForSize,
		m.loadCommits,
		m.loadGitGraph(),
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

	case GitDiffMsg:
		m.mode = ModeDiffView
		m.diffContent = msg.Diff
		m.diffLines = strings.Split(msg.Diff, "\n")
		m.diffScrollPos = 0
		m.diffFileName = msg.FileName
		return m, nil

	case GitDiffErrorMsg:
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

	case GraphLogLoadedMsg:
		m.graphRows = msg.Rows
		m.buildGraphColors()
		return m, nil

	case GraphLogErrorMsg:
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

	// F2.2: Branch management
	case BranchCreatedMsg:
		m.setNotification("success", fmt.Sprintf("Branch criado: %s", msg.Name))
		m.mode = ModeNormal
		return m, m.loadBranches()

	case BranchCreateErrorMsg:
		m.setNotification("error", msg.Err.Error())
		m.mode = ModeNormal
		return m, nil

	case BranchDeletedMsg:
		m.setNotification("success", fmt.Sprintf("Branch deletado: %s", msg.Name))
		return m, m.loadBranches()

	case BranchDeleteErrorMsg:
		m.setNotification("error", msg.Err.Error())
		return m, nil

	case BranchMergedMsg:
		m.setNotification("success", fmt.Sprintf("Merge concluído: %s", msg.Branch))
		return m, tea.Batch(m.loadBranches(), m.loadCommits)

	case BranchMergeErrorMsg:
		m.setNotification("error", msg.Err.Error())
		return m, nil

	// F2.5: Cherry-pick / Revert
	case CherryPickMsg:
		m.setNotification("success", fmt.Sprintf("Cherry-pick: %s", msg.Hash[:7]))
		return m, tea.Batch(m.loadCommits, m.loadGitStatus())

	case CherryPickErrorMsg:
		m.setNotification("error", msg.Err.Error())
		return m, nil

	case RevertMsg:
		m.setNotification("success", fmt.Sprintf("Revertido: %s", msg.Hash[:7]))
		return m, tea.Batch(m.loadCommits, m.loadGitStatus())

	case RevertErrorMsg:
		m.setNotification("error", msg.Err.Error())
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
	case ModeCreateBranch:
		return m.handleCreateBranchInput(msg)
	case ModeRenameBranch:
		return m.handleRenameBranchInput(msg)
	case ModeMergeBranch:
		return m.handleMergeBranchInput(msg)
	case ModeDiffView:
		return m.handleDiffViewInput(msg)
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

	// F2.6: File tree explorer
	case key.Matches(msg, m.keys.FileTreeToggle):
		if m.rightView == ViewFileTree {
			m.rightView = ViewDetails
		} else {
			m.rightView = ViewFileTree
		}
		if m.rightView == ViewFileTree {
			return m, func() tea.Msg {
				if err := m.fileTree.Load(); err != nil {
					return ErrorMsg{Err: err}
				}
				return SuccessMsg{Message: "Árvore de arquivos carregada"}
			}
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

	// F2.2: Branch management (when center panel is focused and showing branches)
	case key.Matches(msg, m.keys.CreateBranch):
		if m.centerView == ViewBranchList || m.focused == PanelCenter {
			m.mode = ModeCreateBranch
			m.branchNameInput.SetValue("")
			m.branchNameInput.Focus()
			return m, nil
		}
		return m, nil

	case key.Matches(msg, m.keys.DeleteBranch):
		if m.centerView == ViewBranchList {
			return m.handleDeleteBranch()
		}
		return m, nil

	case key.Matches(msg, m.keys.MergeBranch):
		if m.centerView == ViewBranchList {
			m.mode = ModeMergeBranch
			m.mergeBranchInput.SetValue("")
			m.mergeBranchInput.Focus()
			return m, nil
		}
		return m, nil

	case key.Matches(msg, m.keys.RenameBranch):
		if m.centerView == ViewBranchList {
			m.mode = ModeRenameBranch
			m.branchNameInput.SetValue("")
			m.branchNameInput.Focus()
			return m, nil
		}
		return m, nil

	// F2.5: Cherry-pick / Revert (from commit log)
	case key.Matches(msg, m.keys.CherryPick):
		if m.centerView == ViewCommitLog && m.store.Commits.Selected() != nil {
			hash := m.store.Commits.Selected().Hash
			return m, m.executeCherryPick(hash)
		}
		return m, nil

	case key.Matches(msg, m.keys.Revert):
		if m.centerView == ViewCommitLog && m.store.Commits.Selected() != nil {
			hash := m.store.Commits.Selected().Hash
			return m, m.executeRevert(hash)
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
// F2: Branch input handlers
// ---------------------------------------------------------------------------

func (m Model) handleCreateBranchInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEnter:
		name := strings.TrimSpace(m.branchNameInput.Value())
		if name == "" {
			m.setNotification("warning", "Nome do branch não pode estar vazio")
			return m, nil
		}
		return m, m.executeCreateBranch(name)
	case tea.KeyEsc:
		m.mode = ModeNormal
		return m, nil
	default:
		var cmd tea.Cmd
		m.branchNameInput, cmd = m.branchNameInput.Update(msg)
		return m, cmd
	}
}

func (m Model) handleRenameBranchInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEnter:
		name := strings.TrimSpace(m.branchNameInput.Value())
		if name == "" {
			m.setNotification("warning", "Novo nome não pode estar vazio")
			return m, nil
		}
		return m, m.executeRenameBranch(name)
	case tea.KeyEsc:
		m.mode = ModeNormal
		return m, nil
	default:
		var cmd tea.Cmd
		m.branchNameInput, cmd = m.branchNameInput.Update(msg)
		return m, cmd
	}
}

func (m Model) handleMergeBranchInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEnter:
		branch := strings.TrimSpace(m.mergeBranchInput.Value())
		if branch == "" {
			m.setNotification("warning", "Nome do branch não pode estar vazio")
			return m, nil
		}
		return m, m.executeMerge(branch)
	case tea.KeyEsc:
		m.mode = ModeNormal
		return m, nil
	default:
		var cmd tea.Cmd
		m.mergeBranchInput, cmd = m.mergeBranchInput.Update(msg)
		return m, cmd
	}
}

func (m Model) handleDeleteBranch() (tea.Model, tea.Cmd) {
	branches := m.store.Branches.Branches()
	active := m.store.Branches.Active()
	for _, b := range branches {
		if b.IsActive {
			continue
		}
		if b.Name == active {
			continue
		}
		// Delete the first non-active branch
		return m, m.executeDeleteBranch(b.Name)
	}
	m.setNotification("warning", "Nenhum branch para deletar (além do ativo)")
	return m, nil
}

// ---------------------------------------------------------------------------
// F2: Diff viewer input
// ---------------------------------------------------------------------------

func (m Model) handleDiffViewInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEsc:
		m.mode = ModeNormal
		m.diffContent = ""
		m.diffLines = nil
		m.diffScrollPos = 0
		return m, nil
	case tea.KeyUp:
		if m.diffScrollPos > 0 {
			m.diffScrollPos--
		}
		return m, nil
	case tea.KeyDown:
		if m.diffScrollPos < len(m.diffLines)-1 {
			m.diffScrollPos++
		}
		return m, nil
	case tea.KeyPgUp:
		m.diffScrollPos -= m.height / 2
		if m.diffScrollPos < 0 {
			m.diffScrollPos = 0
		}
		return m, nil
	case tea.KeyPgDown:
		m.diffScrollPos += m.height / 2
		if m.diffScrollPos >= len(m.diffLines) {
			m.diffScrollPos = len(m.diffLines) - 1
		}
		return m, nil
	default:
		return m, nil
	}
}

// ---------------------------------------------------------------------------
// F2: Branch management commands
// ---------------------------------------------------------------------------

func (m Model) executeCreateBranch(name string) tea.Cmd {
	return func() tea.Msg {
		base := m.store.Branches.Active()
		ctx := context.TODO()
		if err := m.gitOps.CreateBranch(ctx, name, base); err != nil {
			return BranchCreateErrorMsg{Err: err}
		}
		m.mode = ModeNormal
		return BranchCreatedMsg{Name: name}
	}
}

func (m Model) executeDeleteBranch(name string) tea.Cmd {
	return func() tea.Msg {
		ctx := context.TODO()
		if err := m.gitOps.DeleteBranch(ctx, name, false); err != nil {
			return BranchDeleteErrorMsg{Err: err}
		}
		return BranchDeletedMsg{Name: name}
	}
}

func (m Model) executeMerge(branch string) tea.Cmd {
	return func() tea.Msg {
		ctx := context.TODO()
		if err := m.gitOps.Merge(ctx, branch); err != nil {
			return BranchMergeErrorMsg{Err: err}
		}
		m.mode = ModeNormal
		return BranchMergedMsg{Branch: branch}
	}
}

func (m Model) executeRenameBranch(name string) tea.Cmd {
	return func() tea.Msg {
		// Git doesn't have a native rename; use branch -m
		oldName := m.store.Branches.Active()
		ctx := context.TODO()
		// Create new, delete old
		if err := m.gitOps.CreateBranch(ctx, name, oldName); err != nil {
			return BranchCreateErrorMsg{Err: err}
		}
		if err := m.gitOps.DeleteBranch(ctx, oldName, false); err != nil {
			return BranchDeleteErrorMsg{Err: err}
		}
		if err := m.gitOps.Checkout(ctx, name); err != nil {
			return BranchCreateErrorMsg{Err: err}
		}
		m.mode = ModeNormal
		return BranchCreatedMsg{Name: name}
	}
}

// ---------------------------------------------------------------------------
// F2.5: Cherry-pick / Revert commands
// ---------------------------------------------------------------------------

func (m Model) executeCherryPick(hash string) tea.Cmd {
	return func() tea.Msg {
		ctx := context.TODO()
		// Use git cherry-pick via exec since go-git doesn't support it natively
		cmd := exec.CommandContext(ctx, "git", "-C", m.gitOps.Root(), "cherry-pick", hash)
		if out, err := cmd.CombinedOutput(); err != nil {
			return CherryPickErrorMsg{Err: fmt.Errorf("cherry-pick %s: %s: %w", hash[:7], strings.TrimSpace(string(out)), err)}
		}
		return CherryPickMsg{Hash: hash}
	}
}

func (m Model) executeRevert(hash string) tea.Cmd {
	return func() tea.Msg {
		ctx := context.TODO()
		cmd := exec.CommandContext(ctx, "git", "-C", m.gitOps.Root(), "revert", "--no-edit", hash)
		if out, err := cmd.CombinedOutput(); err != nil {
			return RevertErrorMsg{Err: fmt.Errorf("revert %s: %s: %w", hash[:7], strings.TrimSpace(string(out)), err)}
		}
		return RevertMsg{Hash: hash}
	}
}

// ---------------------------------------------------------------------------
// Action handlers
// ---------------------------------------------------------------------------

func (m Model) handleEnter() (tea.Model, tea.Cmd) {
	if m.rightView == ViewFileTree && m.focused == PanelRight {
		m.fileTree.ToggleExpand()
		return m, nil
	}
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
	case m.rightView == ViewFileTree && m.focused == PanelRight:
		m.fileTree.CursorUp()
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
	case m.rightView == ViewFileTree && m.focused == PanelRight:
		m.fileTree.CursorDown()
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
	if m.centerView == ViewCommitLog && m.store.Commits.Selected() != nil {
		return m, m.executeCommitDiff(m.store.Commits.Selected().Hash)
	}
	return m, nil
}

// ---------------------------------------------------------------------------
// Commands
// ---------------------------------------------------------------------------

func (m Model) refreshAll() tea.Cmd {
	return tea.Batch(
		m.loadGitStatus(),
		m.loadCommits,
		m.loadGitGraph(),
		m.loadBranches(),
		m.loadTimeline(),
		m.loadRemotes(),
		func() tea.Msg {
			if err := m.fileTree.Load(); err != nil {
				return ErrorMsg{Err: err}
			}
			return nil
		},
	)
}

func (m Model) loadCommits() tea.Msg {
	commits, err := m.gitOps.Log(context.TODO(), &gitlocal.LogOptions{Limit: 50})
	if err != nil {
		return GitCommitsErrorMsg{Err: err}
	}
	return GitCommitsLoadedMsg{Commits: commits}
}

func (m Model) loadGitGraph() tea.Cmd {
	return func() tea.Msg {
		rows, err := m.gitOps.GraphLog(context.TODO(), &gitlocal.LogOptions{Limit: 100})
		if err != nil {
			return GraphLogErrorMsg{Err: err}
		}
		return GraphLogLoadedMsg{Rows: rows}
	}
}

// buildGraphColors assigns a color to each column in the commit graph.
func (m *Model) buildGraphColors() {
	m.graphColors = make(map[int]lipgloss.Color)
	col := 0
	for _, row := range m.graphRows {
		if !row.IsCommit {
			continue
		}
		// Find the column of the * in the graph line
		for i, ch := range row.Graph {
			if ch == '*' {
				if _, ok := m.graphColors[i]; !ok {
					cIdx := col % len(types.GraphColors)
					m.graphColors[i] = lipgloss.Color(types.GraphColors[cIdx])
					col++
				}
				break
			}
		}
	}
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
		return GitDiffMsg{Diff: diff, FileName: path}
	}
}

func (m Model) executeCommitDiff(hash string) tea.Cmd {
	return func() tea.Msg {
		ctx := context.TODO()
		cmd := exec.CommandContext(ctx, "git", "-C", m.gitOps.Root(), "show", "--no-color", hash)
		out, err := cmd.Output()
		if err != nil {
			return ErrorMsg{Err: fmt.Errorf("mostrar commit %s: %w", hash[:7], err)}
		}
		return GitDiffMsg{Diff: string(out), FileName: hash[:7]}
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
		return m.theme.Base.Render(" Loading... ")
	}

	var b strings.Builder

	// ── Toolbar (top bar with repo info, actions, search) ──
	b.WriteString(m.renderToolbar())
	b.WriteString("\n")

	// ── Three-panel layout with borders ──
	// Each panel is rendered inside a NormalBorder box.
	// We render content for each panel, then wrap in bordered style.

	// Determine which border style to use per panel
	leftBorder := m.theme.PanelBorder
	centerBorder := m.theme.PanelBorder
	rightBorder := m.theme.PanelBorder
	if m.focused == PanelLeft {
		leftBorder = m.theme.ActiveBorder
	} else if m.focused == PanelCenter {
		centerBorder = m.theme.ActiveBorder
	} else if m.focused == PanelRight {
		rightBorder = m.theme.ActiveBorder
	}

	// Render each panel's content
	leftContent := m.renderLeftPanel()
	centerContent := m.renderCenterPanel()
	rightContent := m.renderRightPanel()

	// The bordered style adds 1 line top/bottom and 1 char left/right.
	// So the inner content height = PanelHeight - 2 (top/bottom border)
	// and inner width = panelWidth - 2 (left/right border).
	innerHeight := m.layout.PanelHeight - 2
	if innerHeight < 1 {
		innerHeight = 1
	}

	// Wrap each content in its bordered style at correct width
	leftWrapped := leftBorder.Width(m.layout.LeftWidth).Render(
		m.padContentLines(leftContent, m.layout.ContentWidth(PanelLeft), innerHeight))
	centerWrapped := centerBorder.Width(m.layout.CenterWidth).Render(
		m.padContentLines(centerContent, m.layout.ContentWidth(PanelCenter), innerHeight))
	rightWrapped := rightBorder.Width(m.layout.RightWidth).Render(
		m.padContentLines(rightContent, m.layout.ContentWidth(PanelRight), innerHeight))

	// Combine panels horizontally line by line
	// Each bordered panel has PanelHeight lines (top border + content + bottom border)
	for line := 0; line < m.layout.PanelHeight; line++ {
		leftLine := getLine(leftWrapped, line, m.layout.LeftWidth)
		centerLine := getLine(centerWrapped, line, m.layout.CenterWidth)
		rightLine := getLine(rightWrapped, line, m.layout.RightWidth)
		b.WriteString(leftLine)
		b.WriteString(centerLine)
		b.WriteString(rightLine)
		b.WriteString("\n")
	}

	// ── Status bar ──
	b.WriteString(m.renderStatusBar())

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
	if m.mode == ModeDiffView {
		b.WriteString("\n" + m.renderDiffViewerOverlay())
	}
	if m.mode == ModeCreateBranch {
		b.WriteString("\n" + m.renderBranchInputOverlay("Create Branch", m.branchNameInput.View(), "C"))
	}
	if m.mode == ModeRenameBranch {
		b.WriteString("\n" + m.renderBranchInputOverlay("Rename Branch", m.branchNameInput.View(), "R"))
	}
	if m.mode == ModeMergeBranch {
		b.WriteString("\n" + m.renderBranchInputOverlay("Merge Branch", m.mergeBranchInput.View(), "m"))
	}
	if m.notification != nil && time.Since(m.notifTimer) < 4*time.Second {
		b.WriteString("\n" + m.renderNotification())
	}

	return b.String()
}

// padContentLines pads content to exactly n lines of width w.
func (m Model) padContentLines(content string, width, height int) string {
	lines := strings.Split(content, "\n")
	var out strings.Builder
	for i := 0; i < height; i++ {
		if i < len(lines) {
			line := lines[i]
			if len(line) > width {
				line = line[:width]
			}
			out.WriteString(line)
			if pad := width - len(line); pad > 0 {
				out.WriteString(strings.Repeat(" ", pad))
			}
		} else {
			out.WriteString(strings.Repeat(" ", width))
		}
		if i < height-1 {
			out.WriteString("\n")
		}
	}
	return out.String()
}

// ---------------------------------------------------------------------------
// Toolbar (desktop-app style top bar)
// ---------------------------------------------------------------------------

func (m Model) renderToolbar() string {
	// Left: app icon + repo name
	repoName := "github-desktop-tui"
	if r := m.store.Repositories.Selected(); r != nil {
		repoName = r.FullName
	}

	branch := m.store.Branches.Active()

	// Build toolbar components
	var left strings.Builder
	left.WriteString(m.theme.Accented.Render(" ◉ "))
	left.WriteString(m.theme.Base.Render(repoName))
	left.WriteString(m.theme.BaseMuted.Render("  ⎇ "))
	left.WriteString(m.theme.BranchLabel.Render(branch))

	// Right: actions + search + config
	var right strings.Builder
	if m.spinnerActive {
		right.WriteString(m.theme.BaseMuted.Render(" " + m.spinnerChars[m.spinnerFrame] + " "))
	}
	right.WriteString(m.theme.BaseMuted.Render(" 🔍 "))
	right.WriteString(m.theme.Accented.Render(" ◆ "))

	// Combine: left [spaces] right
	toolbarStr := left.String() + right.String()
	avail := m.width - len(toolbarStr)
	if avail < 1 {
		avail = 1
	}
	spaces := strings.Repeat(" ", avail)
	return m.theme.ToolbarStyle.Render(left.String() + spaces + right.String())
}

// ---------------------------------------------------------------------------
// Panel content
// ---------------------------------------------------------------------------

// renderSectionHeader renders a clean section header like Glint's "Repositories"
func (m Model) renderSectionHeader(title string) string {
	return m.theme.Accented.Render(" " + title + " ")
}

func (m Model) renderLeftPanel() string {
	var b strings.Builder
	w := m.layout.ContentWidth(PanelLeft)

	b.WriteString(m.renderSectionHeader("Branches"))
	b.WriteString("\n")

	branches := m.store.Branches.Branches()
	if len(branches) == 0 && len(m.repoManager.List()) == 0 {
		b.WriteString(m.theme.BaseMuted.Render("  No branches  "))
		return b.String()
	}

	// Separate local from remote
	var local, remote []*types.Branch
	for _, br := range branches {
		if br.IsRemote {
			remote = append(remote, br)
		} else {
			local = append(local, br)
		}
	}

	contentHeight := m.layout.PanelHeight - 4
	maxItems := contentHeight / 2
	if maxItems < 3 {
		maxItems = 3
	}

	// ── Local section (collapsible-style header) ──
	b.WriteString(m.theme.Accented.Render(" ▼ Local"))
	b.WriteString("\n")
	shown := 0
	for _, br := range local {
		if shown >= maxItems {
			break
		}
		prefix := "   "
		if br.IsActive {
			prefix = m.theme.Accented.Render(" ● ")
		} else {
			prefix = m.theme.BaseMuted.Render("   ")
		}
		name := br.Name
		if len(name) > w-8 {
			name = name[:w-11] + "…"
		}
		ab := ""
		if br.Ahead > 0 || br.Behind > 0 {
			ab = m.theme.BaseMuted.Render(fmt.Sprintf(" ↑%d↓%d", br.Ahead, br.Behind))
		}
		style := m.theme.Base
		if br.IsActive {
			style = m.theme.Selected
		}
		b.WriteString(fmt.Sprintf("%s%s%s", prefix, style.Render(name), ab))
		b.WriteString("\n")
		shown++
	}
	if len(local) > maxItems {
		b.WriteString(m.theme.BaseMuted.Render(fmt.Sprintf("   ↓ %d more", len(local)-maxItems)))
		b.WriteString("\n")
	}

	// ── Remote section ──
	if len(remote) > 0 && shown < contentHeight-2 {
		b.WriteString(m.theme.BaseMuted.Render(" Remote"))
		b.WriteString("\n")
		remMax := contentHeight - shown - 3
		if remMax > len(remote) {
			remMax = len(remote)
		}
		for i := 0; i < remMax; i++ {
			br := remote[i]
			name := br.Name
			if len(name) > w-6 {
				name = name[:w-9] + "…"
			}
			b.WriteString(fmt.Sprintf("   %s\n", m.theme.BaseMuted.Render(name)))
			shown++
		}
	}

	// ── Tracking repos (like Submodules) ──
	repos := m.repoManager.List()
	if len(repos) > 0 && shown < contentHeight-2 {
		b.WriteString(m.theme.BaseMuted.Render(" Repositories"))
		b.WriteString("\n")
		repMax := contentHeight - shown - 3
		if repMax > len(repos) {
			repMax = len(repos)
		}
		selRepo := m.store.Repositories.SelectedIndex()
		for i := 0; i < repMax; i++ {
			r := repos[i]
			prefix := "   "
			style := m.theme.BaseMuted
			if i == selRepo {
				prefix = m.theme.Accented.Render(" ● ")
				style = m.theme.Selected
			}
			b.WriteString(fmt.Sprintf("%s%s\n", prefix, style.Render(r.Name)))
		}
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
	w := m.layout.ContentWidth(PanelCenter)

	// Branch header with active branch badge
	activeBranch := m.store.Branches.Active()
	b.WriteString(m.theme.Accented.Render(" ▼ " + activeBranch))
	branchCount := len(m.store.Branches.Branches())
	if branchCount > 0 {
		b.WriteString(m.theme.BaseMuted.Render(fmt.Sprintf("  %d branches", branchCount)))
	}
	b.WriteString("\n")

	if len(m.graphRows) == 0 {
		b.WriteString(m.theme.BaseMuted.Render("  No commits"))
		return b.String()
	}

	selIdx := m.store.Commits.SelectedIndex()
	if selIdx < 0 {
		selIdx = 0
	}
	maxRows := m.layout.PanelHeight - 4
	if maxRows < 3 {
		maxRows = 3
	}
	if maxRows > len(m.graphRows) {
		maxRows = len(m.graphRows)
	}

	// Scroll
	start := 0
	if selIdx >= maxRows {
		start = countGraphCommitsUpTo(m.graphRows, selIdx)
		if start > len(m.graphRows)-maxRows {
			start = len(m.graphRows) - maxRows
		}
	}
	end := start + maxRows
	if end > len(m.graphRows) {
		end = len(m.graphRows)
	}

	commitIdx := 0
	for i := start; i < end; i++ {
		row := m.graphRows[i]
		isSelected := false
		if row.IsCommit {
			if commitIdx == selIdx {
				isSelected = true
			}
			commitIdx++
		}

		// Graph part: colored dots and lines
		graphPart := m.renderGraphLine(row.Graph)

		// Spacing after graph
		rowStr := graphPart

		if row.IsCommit {
			// Commit data: hash message author time
			msg := row.Message
			msgWidth := w - len(row.Graph) - 28
			if msgWidth < 5 {
				msgWidth = 5
			}
			if len(msg) > msgWidth {
				msg = msg[:msgWidth-1] + "…"
			}
			hash := m.theme.BaseMuted.Render(row.Hash)
			author := m.theme.BaseMuted.Render(row.Author)
			timeStr := m.theme.BaseMuted.Render(row.Time)

			rowStr += fmt.Sprintf(" %s %s  %s  %s", hash, msg, author, timeStr)

			// Branch labels (like Glint's "main", "feature" badges)
			if row.Ref != "" {
				refs := parseRefs(row.Ref)
				for _, ref := range refs {
					if ref == "HEAD" {
						continue
					}
					label := m.theme.BranchLabel.Render(" " + ref + " ")
					// Only show if we have space
					if len(rowStr)+len(ref)+4 < w {
						rowStr += " " + label
					}
				}
			}
		}

		if isSelected {
			b.WriteString(m.theme.Selected.Render(rowStr))
		} else {
			b.WriteString(m.theme.Base.Render(rowStr))
		}

		if i < end-1 {
			b.WriteString("\n")
		}
	}

	// Scroll hint
	totalCommits := countCommitsInGraph(m.graphRows)
	if len(m.graphRows) > maxRows {
		b.WriteString(fmt.Sprintf("\n%s", m.theme.BaseMuted.Render(
			fmt.Sprintf("  ↓ %d commits", totalCommits))))
	}

	return b.String()
}

// parseRefs extracts branch/tag references from git log --graph --decorate output.
func parseRefs(refStr string) []string {
	if refStr == "" {
		return nil
	}
	var refs []string
	// Format: "HEAD -> main, origin/main, tag: v1.0"
	parts := strings.Split(refStr, ",")
	for _, p := range parts {
		p = strings.TrimSpace(p)
		// Remove "tag: " prefix
		p = strings.TrimPrefix(p, "tag: ")
		// Remove "HEAD -> " prefix
		p = strings.TrimPrefix(p, "HEAD -> ")
		if p != "" && p != "HEAD" {
			refs = append(refs, p)
		}
	}
	return refs
}

// renderGraphLine renders the ASCII graph prefix with colored columns.
func (m Model) renderGraphLine(graph string) string {
	if graph == "" {
		return ""
	}
	var result strings.Builder
	for i, ch := range graph {
		if ch == '*' {
			color, ok := m.graphColors[i]
			if !ok {
				color = m.theme.Orange
			}
			dotStyle := lipgloss.NewStyle().Foreground(color).Bold(true)
			result.WriteString(dotStyle.Render("●"))
		} else if ch == '|' || ch == '/' || ch == '\\' || ch == '_' {
			color, ok := m.graphColors[i]
			style := m.theme.BaseMuted
			if ok {
				style = lipgloss.NewStyle().Foreground(color)
			}
			result.WriteString(style.Render(string(ch)))
		} else {
			result.WriteString(string(ch))
		}
	}
	return result.String()
}

// countGraphCommitsUpTo counts how many commits exist up to a graph row index.
func countGraphCommitsUpTo(rows []*types.GraphRow, maxCommits int) int {
	count := 0
	for i, row := range rows {
		if row.IsCommit {
			count++
		}
		if count > maxCommits {
			return i
		}
	}
	return len(rows)
}

// countCommitsInGraph returns total number of commits in graph rows.
func countCommitsInGraph(rows []*types.GraphRow) int {
	count := 0
	for _, row := range rows {
		if row.IsCommit {
			count++
		}
	}
	return count
}

func (m Model) renderBranchList() string {
	// Reuse the sidebar branch rendering but as center panel content
	return m.renderLeftPanel()
}

func (m Model) renderRightPanel() string {
	if m.rightView == ViewFileTree {
		return m.fileTree.Render(m.layout.RightWidth, m.layout.PanelHeight, m.theme)
	}

	var b strings.Builder
	w := m.layout.ContentWidth(PanelRight)

	// ── Commit Details (the main purpose of the right panel) ──
	selCommit := m.store.Commits.Selected()
	if selCommit != nil {
		b.WriteString(m.renderSectionHeader("Commit"))
		b.WriteString("\n")

		// Hash (full)
		b.WriteString(m.theme.BaseMuted.Render("  Hash     "))
		b.WriteString(m.theme.Base.Render(selCommit.Hash[:min(12, len(selCommit.Hash))]))
		b.WriteString("\n")

		// Author
		author := selCommit.Author
		if len(author) > w-12 {
			author = author[:w-15] + "…"
		}
		b.WriteString(m.theme.BaseMuted.Render("  Author   "))
		b.WriteString(m.theme.Base.Render(author))
		b.WriteString("\n")

		// Date
		b.WriteString(m.theme.BaseMuted.Render("  Date     "))
		b.WriteString(m.theme.Base.Render(selCommit.Timestamp.Format("02 Jan 2006 15:04")))
		b.WriteString("\n")

		// Message
		b.WriteString(m.theme.BaseMuted.Render("  Message  "))
		msg := selCommit.MessageHead
		msgW := w - 12
		if msgW < 5 {
			msgW = 5
		}
		if len(msg) > msgW {
			msg = msg[:msgW-1] + "…"
		}
		b.WriteString(m.theme.Base.Render(msg))
		b.WriteString("\n")
	}

	// ── Changed Files (from selected commit or working tree) ──
	changes := m.fileChanges
	if len(changes) == 0 && selCommit != nil {
		// Show empty state
		b.WriteString("\n")
		b.WriteString(m.renderSectionHeader("Files"))
		b.WriteString("\n")
		b.WriteString(m.theme.BaseMuted.Render("  No files changed"))
	} else if len(changes) > 0 {
		b.WriteString("\n")
		staged := 0
		for _, fc := range changes {
			if fc.Staged {
				staged++
			}
		}
		summary := ""
		if staged > 0 {
			summary += m.theme.BadgeAdded.Render(fmt.Sprintf(" %d staged", staged))
		}
		unstaged := len(changes) - staged
		if unstaged > 0 {
			summary += m.theme.BaseMuted.Render(fmt.Sprintf(" %d changed", unstaged))
		}
		b.WriteString(m.renderSectionHeader("Files"))
		b.WriteString(summary)
		b.WriteString("\n")

		availH := m.layout.PanelHeight - 10
		if availH < 3 {
			availH = 3
		}
		if availH > len(changes) {
			availH = len(changes)
		}
		for i := 0; i < availH; i++ {
			fc := changes[i]
			badge := m.theme.BaseMuted.Render(" ")
			switch fc.Status {
			case types.FileStatusModified:
				badge = m.theme.BadgeModified.Render("M")
			case types.FileStatusAdded:
				badge = m.theme.BadgeAdded.Render("A")
			case types.FileStatusDeleted:
				badge = m.theme.BadgeDeleted.Render("D")
			case types.FileStatusUntracked:
				badge = m.theme.BaseMuted.Render("?")
			}
			path := fc.Path
			pathW := w - 6
			if pathW < 3 {
				pathW = 3
			}
			if len(path) > pathW {
				path = path[:pathW-1] + "…"
			}
			style := m.theme.Base
			if i == m.selectedFile {
				style = m.theme.Selected
				b.WriteString(fmt.Sprintf(" %s %s %s", m.theme.Accented.Render("▸"), badge, style.Render(path)))
			} else {
				b.WriteString(fmt.Sprintf("   %s %s", badge, style.Render(path)))
			}
			if i < availH-1 {
				b.WriteString("\n")
			}
		}
		if len(changes) > availH {
			b.WriteString(fmt.Sprintf("\n%s", m.theme.BaseMuted.Render(fmt.Sprintf("  ↓ %d more", len(changes)-availH))))
		}
	}

	return b.String()
}

// ---------------------------------------------------------------------------
// Status bar
// ---------------------------------------------------------------------------

func (m Model) renderStatusBar() string {
	branch := m.store.Branches.Active()

	// Left: repo state info
	changes := 0
	staged := 0
	for _, fc := range m.fileChanges {
		changes++
		if fc.Staged {
			staged++
		}
	}

	left := fmt.Sprintf(" %s", m.theme.BranchLabel.Render(branch))
	if staged > 0 {
		left += m.theme.BadgeAdded.Render(fmt.Sprintf(" +%d", staged))
	}
	if changes-staged > 0 {
		left += m.theme.BaseMuted.Render(fmt.Sprintf(" ~%d", changes-staged))
	}
	if m.spinnerActive {
		left += m.theme.Accented.Render(" " + m.spinnerChars[m.spinnerFrame] + " " + m.spinnerOp)
	}

	// Right: provider, auth, time
	right := ""
	for _, p := range m.registry.All() {
		if m.authManager.IsAuthenticated(p.Name()) {
			right += m.theme.BadgeAdded.Render(" ✓")
			break
		}
	}
	right += "  " + time.Now().Format("15:04")

	spaces := m.width - len(left) - len(right)
	if spaces < 1 {
		spaces = 1
	}
	return m.theme.StatusStyle.Render(left + strings.Repeat(" ", spaces) + right)
}

func (m Model) renderCommandBar() string {
	type cmd struct{ key, desc string }
	var cmds []cmd

	switch {
	case m.showStaging:
		cmds = []cmd{
			{"s", "stage"}, {"↵", "commit"}, {"d", "diff"},
			{"p", "push"}, {"esc", "back"},
		}
	case m.showTimeline, m.showHistory, m.showHelp:
		cmds = []cmd{{"↑↓", "nav"}, {"esc", "close"}}
	case m.showAuth, m.showRemotes, m.showRepoAdd:
		cmds = []cmd{{"↵", "ok"}, {"esc", "cancel"}}
	case m.mode == ModeCommitMessage:
		cmds = []cmd{{"↵", "commit"}, {"esc", "cancel"}}
	default:
		cmds = []cmd{
			{"c", "commit"}, {"p", "push"}, {"l", "pull"},
			{"d", "diff"}, {"b", "branch"}, {"t", "tree"},
			{"/", "log"}, {"y", "cherry"}, {"V", "revert"},
			{"r", "refresh"}, {"?", "help"},
		}
	}

	var sb strings.Builder
	for i, c := range cmds {
		if i > 0 {
			sb.WriteString("  ")
		}
		sb.WriteString(m.theme.Accented.Render(" " + c.key + " "))
		sb.WriteString(m.theme.BaseMuted.Render(c.desc))
	}
	cmdStr := sb.String()

	spaces := m.width - len(cmdStr)
	if spaces < 0 {
		spaces = 0
	}
	return m.theme.BaseMuted.Render(cmdStr + strings.Repeat(" ", spaces))
}

// ---------------------------------------------------------------------------
// F2: Diff viewer overlay
// ---------------------------------------------------------------------------

func (m Model) renderDiffViewerOverlay() string {
	ovWidth := m.width - 8
	if ovWidth > 120 {
		ovWidth = 120
	}
	ovHeight := m.height - 6
	if ovHeight < 10 {
		ovHeight = 10
	}

	title := m.theme.OverlayTitle.Render(" Diff: " + m.diffFileName + " ")
	help := m.theme.BaseMuted.Render(" ↑↓ scroll  pgup/pgdn page  esc close ")

	var inner strings.Builder
	maxLines := ovHeight - 4
	start := m.diffScrollPos
	end := start + maxLines
	if end > len(m.diffLines) {
		end = len(m.diffLines)
	}

	for i := start; i < end; i++ {
		line := m.diffLines[i]
		lineStyle := m.theme.Base

		if len(line) > 0 {
			switch line[0] {
			case '+':
				lineStyle = m.theme.BadgeAdded
			case '-':
				lineStyle = m.theme.BadgeDeleted
			case '@':
				lineStyle = m.theme.BadgeAdded
			case 'd':
				if strings.HasPrefix(line, "diff --git") {
					lineStyle = m.theme.Accented
				}
			}
		}

		disp := line
		if len(disp) > ovWidth-6 {
			disp = disp[:ovWidth-9] + "…"
		}
		inner.WriteString(lineStyle.Render(" " + disp))
		if i < end-1 {
			inner.WriteString("\n")
		}
	}

	// Scroll indicator
	scrollInfo := ""
	if len(m.diffLines) > maxLines {
		pct := int(float64(m.diffScrollPos) / float64(len(m.diffLines)-maxLines) * 100)
		scrollInfo = fmt.Sprintf("  %d%% (%d/%d)", pct, m.diffScrollPos, len(m.diffLines))
	}

	footer := title + "\n" + help + scrollInfo + "\n" + inner.String()
	content := lipgloss.NewStyle().Width(ovWidth - 2).Render(footer)
	return m.theme.OverlayBox.Width(ovWidth).Render(centeredText(content, m.width, ovWidth))
}

// ---------------------------------------------------------------------------
// F2: Branch input overlay (reusable for create, rename, merge)
// ---------------------------------------------------------------------------

func (m Model) renderBranchInputOverlay(title string, inputView string, key string) string {
	ovWidth := 50
	titleRendered := m.theme.OverlayTitle.Render(title)
	help := m.theme.BaseMuted.Render(" ↵: confirm  esc: cancel ")

	var inner strings.Builder
	inner.WriteString("\n")
	inner.WriteString("  " + inputView + "\n")
	inner.WriteString("\n")
	inner.WriteString(m.theme.BaseMuted.Render("  Press " + key + " or use the branch panel"))

	content := lipgloss.NewStyle().Width(ovWidth - 2).Render(titleRendered + "\n" + help + "\n" + inner.String())
	return m.theme.OverlayBox.Width(ovWidth).Render(centeredText(content, m.width, ovWidth))
}

// ---------------------------------------------------------------------------
// Overlays
// ---------------------------------------------------------------------------

func (m Model) renderStagingOverlay() string {
	ovWidth := 72
	ovHeight := len(m.fileChanges) + 7
	if ovHeight > m.height-6 {
		ovHeight = m.height - 6
	}
	if ovHeight < 6 {
		ovHeight = 6
	}

	title := m.theme.OverlayTitle.Render(" Stage Changes ")
	top := lipgloss.JoinHorizontal(lipgloss.Top,
		title,
		m.theme.BaseMuted.Render(" ↑↓ nav  s stage  ↵ commit  d diff  esc back "),
	)

	var inner strings.Builder
	if len(m.fileChanges) == 0 {
		inner.WriteString(m.theme.BaseMuted.Render("  Nenhuma mudança no diretório de trabalho"))
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
				check = m.theme.BadgeAdded.Render("[✓]")
			} else {
				check = m.theme.BaseMuted.Render("[ ]")
			}

			// Color-code status
			var statusStyle lipgloss.Style
			switch fc.Status {
			case types.FileStatusModified:
				statusStyle = m.theme.BadgeModified
			case types.FileStatusAdded:
				statusStyle = m.theme.BadgeAdded
			case types.FileStatusDeleted:
				statusStyle = m.theme.BadgeDeleted
			case types.FileStatusUntracked:
				statusStyle = m.theme.BaseMuted
			case types.FileStatusRenamed:
				statusStyle = m.theme.InfoText
			default:
				statusStyle = m.theme.BaseMuted
			}

			path := fc.Path
			if len(path) > ovWidth-15 {
				path = path[:ovWidth-18] + "…"
			}

			line := fmt.Sprintf(" %s %s %s  %s",
				cursor, check, statusStyle.Render(fc.StatusShort()), path)

			style := m.theme.Base
			if i == m.selectedFile {
				style = m.theme.Selected
			}
			inner.WriteString(style.Render(line))
			if i < end-1 {
				inner.WriteString("\n")
			}
		}

		if len(m.fileChanges) > maxVisible {
			inner.WriteString("\n")
			inner.WriteString(m.theme.Dim.Render(
				fmt.Sprintf("  ↓ %d de %d", end, len(m.fileChanges))))
		}
	}

	content := lipgloss.NewStyle().
		Width(ovWidth - 2).
		Render(top + "\n" + inner.String())

	return m.theme.OverlayBox.
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

	title := m.theme.OverlayTitle.Render(" Commit Timeline ")
	help := m.theme.BaseMuted.Render(" ↑↓ nav  esc close ")

	var inner strings.Builder
	if len(m.timelineCommits) == 0 {
		inner.WriteString(m.theme.BaseMuted.Render("  Nenhum commit encontrado"))
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
			style := m.theme.Base
			if i == m.timelineIndex {
				style = m.theme.Selected
			}

			hash := m.theme.BaseMuted.Render(c.ShortHash)
			timeStr := m.theme.BaseMuted.Render(c.Timestamp.Format("02 Jan 2006 15:04"))
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
	return m.theme.OverlayBox.Width(ovWidth).Render(centeredText(content, m.width, ovWidth))
}

func (m Model) renderRemotesOverlay() string {
	ovWidth := 60

	title := m.theme.OverlayTitle.Render(" Remotes ")
	help := m.theme.BaseMuted.Render(" a add  esc close ")

	var inner strings.Builder
	if len(m.remoteList) == 0 {
		inner.WriteString(m.theme.BaseMuted.Render("  Nenhum remote configurado"))
	} else {
		for _, r := range m.remoteList {
			url := ""
			if len(r.URLs) > 0 {
				url = r.URLs[0]
				if len(url) > 40 {
					url = url[:40] + "…"
				}
			}
			line := fmt.Sprintf("  %s → %s", m.theme.BranchLabel.Render(r.Name), url)
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
	return m.theme.OverlayBox.Width(ovWidth).Render(centeredText(content, m.width, ovWidth))
}

func (m Model) renderAuthOverlay() string {
	ovWidth := 55

	title := m.theme.OverlayTitle.Render(" Authentication ")
	help := m.theme.BaseMuted.Render(" ↵ confirm  esc close ")

	var inner strings.Builder
	for _, p := range m.registry.All() {
		dot := m.theme.BaseMuted.Render("○")
		if m.authManager.IsAuthenticated(p.Name()) {
			dot = m.theme.BadgeAdded.Render("●")
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
	return m.theme.OverlayBox.Width(ovWidth).Render(centeredText(content, m.width, ovWidth))
}

func (m Model) renderCommitInputOverlay() string {
	ovWidth := 64

	title := m.theme.OverlayTitle.Render(" Mensagem do Commit ")

	var inner strings.Builder
	inner.WriteString("\n")
	inner.WriteString("  " + m.commitInput.View())
	inner.WriteString("\n\n")
	inner.WriteString(m.theme.BaseMuted.Render("  ↵: confirmar  esc: cancelar"))

	content := lipgloss.NewStyle().Width(ovWidth - 2).Render(title + inner.String())
	return m.theme.OverlayBox.Width(ovWidth).Render(centeredText(content, m.width, ovWidth))
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
	return m.theme.Base.Render("  " + style.Render(" " + m.notification.Message + " "))
}

func (m Model) renderHistoryOverlay() string {
	ovWidth := 66
	title := m.theme.OverlayTitle.Render(" Histórico de Notificações ")
	help := m.theme.BaseMuted.Render(" esc: fechar ")

	var inner strings.Builder
	if len(m.notificationHistory) == 0 {
		inner.WriteString(m.theme.BaseMuted.Render("  Nenhuma notificação no histórico"))
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
	return m.theme.OverlayBox.Width(ovWidth).Render(centeredText(content, m.width, ovWidth))
}

func (m Model) renderRepoAddOverlay() string {
	ovWidth := 60
	title := m.theme.OverlayTitle.Render(" Adicionar Repositório ")
	help := m.theme.BaseMuted.Render(" ↵: confirmar  esc: cancelar ")

	var inner strings.Builder
	inner.WriteString("\n")
	inner.WriteString("  Caminho: " + m.repoAddInput.View() + "\n")
	inner.WriteString("\n")
	inner.WriteString(m.theme.BaseMuted.Render("  Ex: ~/projects/my-repo ou /home/user/project"))

	content := lipgloss.NewStyle().Width(ovWidth - 2).Render(title + "\n" + help + "\n" + inner.String())
	return m.theme.OverlayBox.Width(ovWidth).Render(centeredText(content, m.width, ovWidth))
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
		{m.keys.FileTreeToggle.Help().Key, m.keys.FileTreeToggle.Help().Desc},
	}
	for _, kv := range keys {
		inner.WriteString(fmt.Sprintf("  %s  %s\n",
			m.theme.Accented.Render(fmt.Sprintf("%-8s", kv.k)),
			kv.d))
	}
	inner.WriteString(m.theme.BaseMuted.Render("  esc: fechar"))

	content := lipgloss.NewStyle().Width(44).Render(title + "\n" + inner.String())
	return m.theme.OverlayBox.Width(46).Render(centeredText(content, m.width, 46))
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
