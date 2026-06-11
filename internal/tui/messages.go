package tui

import (
	"github.com/nicoddemus/github-desktop-tui/internal/store"
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
	Diff     string
	FileName string
}

type GitDiffErrorMsg struct {
	Err error
}

type GitStatusErrorMsg struct {
	Err error
}

type GitPushErrorMsg struct {
	Err error
}

type GitLogLoadedMsg struct {
	Commits []*types.Commit
}

type GitLogErrorMsg struct {
	Err error
}

type GitCommitsLoadedMsg struct {
	Commits []*types.Commit
}

type GitCommitsErrorMsg struct {
	Err error
}

type GitBranchesLoadedMsg struct {
	Branches []*types.Branch
}

type GitBranchesErrorMsg struct {
	Err error
}

type RemotesLoadedMsg struct {
	Remotes []*types.Remote
}

type RemotesErrorMsg struct {
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

// --- F1.1: Spinner messages ---

type SpinnerTickMsg struct{}

type SpinnerStartMsg struct {
	Operation string
}

type SpinnerStopMsg struct{}

// --- F1.2: Repo management messages ---

type RepoScanMsg struct {
	Repos []string
}

type RepoScanErrorMsg struct {
	Err error
}

type RepoAddMsg struct {
	Path string
	Name string
}

type RepoAddErrorMsg struct {
	Err error
}

type RepoRemoveMsg struct {
	Path string
}

type ReposUpdatedMsg struct {
	Repos []*store.TrackedRepo
}

// --- F2.2: Branch management ---

type BranchCreatedMsg struct {
	Name string
}

type BranchCreateErrorMsg struct {
	Err error
}

type BranchDeletedMsg struct {
	Name string
}

type BranchDeleteErrorMsg struct {
	Err error
}

type BranchMergedMsg struct {
	Branch string
}

type BranchMergeErrorMsg struct {
	Err error
}

// --- F2.5: Cherry-pick / Revert ---

type CherryPickMsg struct {
	Hash string
}

type CherryPickErrorMsg struct {
	Err error
}

type RevertMsg struct {
	Hash string
}

type RevertErrorMsg struct {
	Err error
}
