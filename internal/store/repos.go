package store

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

// TrackedRepo represents a repository being tracked.
type TrackedRepo struct {
	Path       string `json:"path"`
	Name       string `json:"name"`
	IsFavorite bool   `json:"is_favorite"`
	LastOpened string `json:"last_opened"` // ISO timestamp
}

// RepoManager manages tracked repositories with persistence.
type RepoManager struct {
	mu         sync.RWMutex
	repos      []*TrackedRepo
	configPath string
	filePath   string
}

// NewRepoManager creates a new RepoManager.
func NewRepoManager(configDir string) *RepoManager {
	rm := &RepoManager{
		repos:      make([]*TrackedRepo, 0),
		configPath: configDir,
		filePath:   filepath.Join(configDir, "repos.json"),
	}
	rm.Load()
	return rm
}

// Add adds a repository path to tracking.
func (rm *RepoManager) Add(path string) error {
	abs, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("caminho inválido: %w", err)
	}

	// Verify it's a git repo
	gitDir := filepath.Join(abs, ".git")
	if info, err := os.Stat(gitDir); err != nil || !info.IsDir() {
		// Check if .git is a file (worktree)
		if info, err := os.Stat(gitDir); err != nil || info.IsDir() {
			return fmt.Errorf("diretoria %s não é um repositório Git", abs)
		}
	}

	rm.mu.Lock()
	defer rm.mu.Unlock()

	// Check if already tracked
	for _, r := range rm.repos {
		if r.Path == abs {
			r.LastOpened = time.Now().Format(time.RFC3339)
			rm.save()
			return nil
		}
	}

	name := filepath.Base(abs)
	rm.repos = append(rm.repos, &TrackedRepo{
		Path:       abs,
		Name:       name,
		LastOpened: time.Now().Format(time.RFC3339),
	})
	rm.save()
	return nil
}

// Remove removes a repository from tracking.
func (rm *RepoManager) Remove(path string) {
	rm.mu.Lock()
	defer rm.mu.Unlock()
	for i, r := range rm.repos {
		if r.Path == path {
			rm.repos = append(rm.repos[:i], rm.repos[i+1:]...)
			break
		}
	}
	rm.save()
}

// List returns all tracked repos sorted by favorites first, then last opened.
func (rm *RepoManager) List() []*TrackedRepo {
	rm.mu.RLock()
	defer rm.mu.RUnlock()
	result := make([]*TrackedRepo, len(rm.repos))
	copy(result, rm.repos)
	sort.Slice(result, func(i, j int) bool {
		if result[i].IsFavorite != result[j].IsFavorite {
			return result[i].IsFavorite
		}
		return result[i].LastOpened > result[j].LastOpened
	})
	return result
}

// ToggleFavorite toggles the favorite status of a repo.
func (rm *RepoManager) ToggleFavorite(path string) {
	rm.mu.Lock()
	defer rm.mu.Unlock()
	for _, r := range rm.repos {
		if r.Path == path {
			r.IsFavorite = !r.IsFavorite
			break
		}
	}
	rm.save()
}

// Scan searches for Git repositories recursively from a root path.
func (rm *RepoManager) Scan(root string) ([]string, error) {
	var repos []string
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // skip inaccessible dirs
		}
		if info.IsDir() && info.Name() == ".git" {
			repos = append(repos, filepath.Dir(path))
			return filepath.SkipDir
		}
		// Skip hidden directories (but not .git itself)
		if info.IsDir() && strings.HasPrefix(info.Name(), ".") && info.Name() != "." {
			return filepath.SkipDir
		}
		// Skip node_modules, vendor, etc.
		if info.IsDir() && (info.Name() == "node_modules" || info.Name() == "vendor" || info.Name() == "target") {
			return filepath.SkipDir
		}
		return nil
	})
	if err != nil {
		return repos, fmt.Errorf("scan: %w", err)
	}
	return repos, nil
}

// Save persists the repo list to disk.
func (rm *RepoManager) save() {
	data, err := json.MarshalIndent(rm.repos, "", "  ")
	if err != nil {
		return
	}
	if err := os.MkdirAll(rm.configPath, 0700); err != nil {
		return
	}
	_ = os.WriteFile(rm.filePath, data, 0600)
}

// Load reads the repo list from disk.
func (rm *RepoManager) Load() {
	data, err := os.ReadFile(rm.filePath)
	if err != nil {
		return
	}
	rm.mu.Lock()
	defer rm.mu.Unlock()
	_ = json.Unmarshal(data, &rm.repos)
}
