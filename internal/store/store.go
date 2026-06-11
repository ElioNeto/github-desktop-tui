package store

import (
	"sync"

	"github.com/ElioNeto/github-desktop-tui/internal/config"
	"github.com/ElioNeto/github-desktop-tui/pkg/types"
)

// Store is the central application state container.
// It is safe for concurrent access.
type Store struct {
	mu sync.RWMutex

	Repositories *RepositoryStore
	Commits      *CommitStore
	Branches     *BranchStore
	Settings     *SettingsStore
}

// New creates a new Store with initialized sub-stores.
func New() *Store {
	return &Store{
		Repositories: NewRepositoryStore(),
		Commits:      NewCommitStore(),
		Branches:     NewBranchStore(),
		Settings:     NewSettingsStore(),
	}
}

// --- RepositoryStore ---

type RepositoryStore struct {
	mu       sync.RWMutex
	repos    []*types.Repository
	selected int // index of selected repo
}

func NewRepositoryStore() *RepositoryStore {
	return &RepositoryStore{
		repos:    make([]*types.Repository, 0),
		selected: -1,
	}
}

func (s *RepositoryStore) SetRepos(repos []*types.Repository) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.repos = repos
	if s.selected >= len(repos) {
		s.selected = len(repos) - 1
	}
}

func (s *RepositoryStore) Repos() []*types.Repository {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]*types.Repository, len(s.repos))
	copy(result, s.repos)
	return result
}

func (s *RepositoryStore) Selected() *types.Repository {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.selected < 0 || s.selected >= len(s.repos) {
		return nil
	}
	return s.repos[s.selected]
}

func (s *RepositoryStore) Select(index int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if index >= 0 && index < len(s.repos) {
		s.selected = index
	}
}

func (s *RepositoryStore) SelectedIndex() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.selected
}

// --- CommitStore ---

type CommitStore struct {
	mu       sync.RWMutex
	commits  []*types.Commit
	selected int
}

func NewCommitStore() *CommitStore {
	return &CommitStore{
		commits:  make([]*types.Commit, 0),
		selected: -1,
	}
}

func (s *CommitStore) SetCommits(commits []*types.Commit) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.commits = commits
	if s.selected >= len(commits) {
		s.selected = len(commits) - 1
	}
}

func (s *CommitStore) Commits() []*types.Commit {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]*types.Commit, len(s.commits))
	copy(result, s.commits)
	return result
}

func (s *CommitStore) Selected() *types.Commit {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.selected < 0 || s.selected >= len(s.commits) {
		return nil
	}
	return s.commits[s.selected]
}

func (s *CommitStore) Select(index int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if index >= 0 && index < len(s.commits) {
		s.selected = index
	}
}

func (s *CommitStore) SelectedIndex() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.selected
}

// --- BranchStore ---

type BranchStore struct {
	mu       sync.RWMutex
	branches []*types.Branch
	active   string
}

func NewBranchStore() *BranchStore {
	return &BranchStore{
		branches: make([]*types.Branch, 0),
		active:   "main",
	}
}

func (s *BranchStore) SetBranches(branches []*types.Branch) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.branches = branches
	for _, b := range branches {
		if b.IsActive {
			s.active = b.Name
			break
		}
	}
}

func (s *BranchStore) Branches() []*types.Branch {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]*types.Branch, len(s.branches))
	copy(result, s.branches)
	return result
}

func (s *BranchStore) Active() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.active
}

func (s *BranchStore) SetActive(name string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.active = name
}

// --- SettingsStore ---

type SettingsStore struct {
	mu      sync.RWMutex
	config  *config.Config
}

func NewSettingsStore() *SettingsStore {
	return &SettingsStore{
		config: config.Default(),
	}
}

func (s *SettingsStore) SetConfig(cfg *config.Config) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.config = cfg
}

func (s *SettingsStore) Config() *config.Config {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.config
}

func (s *SettingsStore) ActiveProvider() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.config.ActiveProvider
}

func (s *SettingsStore) SetActiveProvider(name string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.config.ActiveProvider = name
}
