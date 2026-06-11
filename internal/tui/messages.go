package tui

import (
	"github.com/nicoddemus/github-desktop-tui/pkg/types"
)

// --- Mensagens de Navegação ---

type FocusChangeMsg struct {
	Panel PanelID
}

type ViewChangeMsg struct {
	Panel PanelID
	View  ViewID
}

// --- Mensagens de Repositório ---

type RepoSelectMsg struct {
	Repo *types.Repository
}

type RepoRefreshMsg struct{}

type RepoListLoadedMsg struct {
	Repos []*types.Repository
}

type RepoListErrorMsg struct {
	Err error
}

// --- Mensagens de Git ---

type GitStatusMsg struct {
	Changes []*types.FileChange
}

type GitCommitMsg struct {
	Hash string
}

type GitCommitErrorMsg struct {
	Err error
}

type GitPushMsg struct {
	Success bool
}

type GitPullMsg struct {
	Changes int
}

type GitBranchSwitchMsg struct {
	Branch string
}

type GitDiffMsg struct {
	Diff string
}

type GitDiffErrorMsg struct {
	Err error
}

// --- Mensagens de Provedor ---

type ProviderSwitchMsg struct {
	Provider string
}

type AuthRequiredMsg struct {
	Provider string
}

type AuthCompleteMsg struct {
	Success bool
	Error   error
}

// --- Mensagens de Sistema ---

type ErrorMsg struct {
	Err error
}

type SuccessMsg struct {
	Message string
}

type NotificationMsg struct {
	Level   string // "info", "warning", "error", "success"
	Message string
}

type TickMsg struct{}
