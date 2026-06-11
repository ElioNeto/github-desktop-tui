package git

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/transport"
	"github.com/go-git/go-git/v5/plumbing/transport/http"

	"github.com/nicoddemus/github-desktop-tui/pkg/types"
)

// ---------------------------------------------------------------------------
// GitOperations interface
// ---------------------------------------------------------------------------

// GitOperations defines the interface for local Git operations.
type GitOperations interface {
	// Root returns the absolute path to the repository root.
	Root() string

	// Status operations
	Status(ctx context.Context) ([]*types.FileChange, error)
	Diff(ctx context.Context, path string) (string, error)
	StagedDiff(ctx context.Context) (string, error)

	// Staging operations
	Stage(ctx context.Context, paths ...string) error
	Unstage(ctx context.Context, paths ...string) error
	Discard(ctx context.Context, paths ...string) error

	// Commit operations
	Commit(ctx context.Context, message string) (string, error)

	// Remote synchronization
	Push(ctx context.Context, remote, branch string, force bool) error
	Pull(ctx context.Context, remote, branch string) error
	Fetch(ctx context.Context, remote string) error

	// History
	Log(ctx context.Context, opts *LogOptions) ([]*types.Commit, error)

	// Branch operations
	Branches(ctx context.Context) ([]*types.Branch, error)
	Checkout(ctx context.Context, branch string) error
	CreateBranch(ctx context.Context, name, base string) error
	DeleteBranch(ctx context.Context, name string, force bool) error
	Merge(ctx context.Context, branch string) error
	CurrentBranch(ctx context.Context) (string, error)

	// Remote management
	Remotes(ctx context.Context) ([]*types.Remote, error)
	AddRemote(ctx context.Context, name, url string) error
	RemoveRemote(ctx context.Context, name string) error
	SetRemoteURL(ctx context.Context, name, url string) error
}

// LogOptions configures the Log operation.
type LogOptions struct {
	Branch string     // Branch name to filter
	Limit  int        // Max commits to return (0 = no limit)
	Since  *time.Time // Filter commits after this time
	Until  *time.Time // Filter commits before this time
	Path   string     // Filter by file path
}

// ---------------------------------------------------------------------------
// LocalGit implementation
// ---------------------------------------------------------------------------

// LocalGit implements GitOperations using go-git v5 with shell fallbacks.
type LocalGit struct {
	repoPath string
}

// New creates a new LocalGit instance for the repository at repoPath.
func New(repoPath string) *LocalGit {
	return &LocalGit{repoPath: repoPath}
}

// errStopIteration is a sentinel to stop early from ForEach callbacks.
var errStopIteration = errors.New("stop iteration")

// ---------------------------------------------------------------------------
// Repository access helpers
// ---------------------------------------------------------------------------

func (g *LocalGit) openRepo() (*git.Repository, error) {
	repo, err := git.PlainOpen(g.repoPath)
	if err != nil {
		return nil, fmt.Errorf("abrir repositório em %s: %w", g.repoPath, err)
	}
	return repo, nil
}

func (g *LocalGit) openWorktree() (*git.Worktree, *git.Repository, error) {
	repo, err := g.openRepo()
	if err != nil {
		return nil, nil, err
	}
	w, err := repo.Worktree()
	if err != nil {
		return nil, nil, fmt.Errorf("abrir worktree: %w", err)
	}
	return w, repo, nil
}

// getAuth returns authentication from environment variables.
func (g *LocalGit) getAuth(token string) transport.AuthMethod {
	if token != "" {
		return &http.TokenAuth{Token: token}
	}
	// Fallback to environment variables
	for _, env := range []string{"GIT_TOKEN", "GITHUB_TOKEN", "GH_TOKEN", "GITLAB_TOKEN"} {
		if v := os.Getenv(env); v != "" {
			return &http.TokenAuth{Token: v}
		}
	}
	return nil
}

func (g *LocalGit) getAuthorInfo() *object.Signature {
	name := g.gitConfig("user.name")
	email := g.gitConfig("user.email")
	if name == "" {
		name = "User"
	}
	if email == "" {
		email = "user@example.com"
	}
	return &object.Signature{
		Name:  name,
		Email: email,
		When:  time.Now(),
	}
}

func (g *LocalGit) gitConfig(key string) string {
	cmd := exec.Command("git", "-C", g.repoPath, "config", key)
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

// ---------------------------------------------------------------------------
// Status mapping helpers
// ---------------------------------------------------------------------------

func gitStatusCodeToFileStatus(s git.StatusCode) types.FileStatus {
	switch s {
	case git.Unmodified:
		return types.FileStatusUnmodified
	case git.Added:
		return types.FileStatusAdded
	case git.Deleted:
		return types.FileStatusDeleted
	case git.Modified:
		return types.FileStatusModified
	case git.Renamed:
		return types.FileStatusRenamed
	case git.Copied:
		return types.FileStatusCopied
	case git.UpdatedButUnmerged:
		return types.FileStatusUpdated
	case git.Untracked:
		return types.FileStatusUntracked
	default:
		return types.FileStatusUnmodified
	}
}

// ---------------------------------------------------------------------------
// Root
// ---------------------------------------------------------------------------

func (g *LocalGit) Root() string {
	return g.repoPath
}

// ---------------------------------------------------------------------------
// Status
// ---------------------------------------------------------------------------

func (g *LocalGit) Status(ctx context.Context) ([]*types.FileChange, error) {
	w, _, err := g.openWorktree()
	if err != nil {
		return nil, err
	}

	status, err := w.Status()
	if err != nil {
		return nil, fmt.Errorf("obter status: %w", err)
	}

	var changes []*types.FileChange
	for path, fs := range status {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		change := &types.FileChange{
			Path:         path,
			Status:       gitStatusCodeToFileStatus(fs.Worktree),
			Staged:       fs.Staging != git.Unmodified,
			StagedStatus: gitStatusCodeToFileStatus(fs.Staging),
		}
		changes = append(changes, change)
	}

	sort.Slice(changes, func(i, j int) bool {
		return changes[i].Path < changes[j].Path
	})

	return changes, nil
}

// ---------------------------------------------------------------------------
// Diff
// ---------------------------------------------------------------------------

func (g *LocalGit) Diff(ctx context.Context, path string) (string, error) {
	args := []string{"-C", g.repoPath, "diff", "--no-color"}
	if path != "" {
		args = append(args, "--", path)
	}
	cmd := exec.CommandContext(ctx, "git", args...)
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("executar git diff: %w", err)
	}
	return string(out), nil
}

func (g *LocalGit) StagedDiff(ctx context.Context) (string, error) {
	cmd := exec.CommandContext(ctx, "git", "-C", g.repoPath, "diff", "--cached", "--no-color")
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("executar git diff --cached: %w", err)
	}
	return string(out), nil
}

// ---------------------------------------------------------------------------
// Staging
// ---------------------------------------------------------------------------

func (g *LocalGit) Stage(ctx context.Context, paths ...string) error {
	w, _, err := g.openWorktree()
	if err != nil {
		return err
	}
	for _, path := range paths {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		_, err := w.Add(path)
		if err != nil {
			return fmt.Errorf("adicionar %s ao stage: %w", path, err)
		}
	}
	return nil
}

func (g *LocalGit) Unstage(ctx context.Context, paths ...string) error {
	args := append([]string{"-C", g.repoPath, "reset", "HEAD", "--"}, paths...)
	cmd := exec.CommandContext(ctx, "git", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("remover do stage: %s: %w", strings.TrimSpace(string(out)), err)
	}
	return nil
}

func (g *LocalGit) Discard(ctx context.Context, paths ...string) error {
	// For untracked files, use clean; for tracked, use checkout
	for _, path := range paths {
		args := []string{"-C", g.repoPath, "checkout", "--", path}
		cmd := exec.CommandContext(ctx, "git", args...)
		if err := cmd.Run(); err != nil {
			// Try git clean for untracked
			cleanArgs := []string{"-C", g.repoPath, "clean", "-fd", "--", path}
			cleanCmd := exec.CommandContext(ctx, "git", cleanArgs...)
			if cleanErr := cleanCmd.Run(); cleanErr != nil {
				return fmt.Errorf("descartar alterações em %s: checkout: %v, clean: %v", path, err, cleanErr)
			}
		}
	}
	return nil
}

// ---------------------------------------------------------------------------
// Commit
// ---------------------------------------------------------------------------

func (g *LocalGit) Commit(ctx context.Context, message string) (string, error) {
	w, _, err := g.openWorktree()
	if err != nil {
		return "", err
	}

	author := g.getAuthorInfo()
	hash, err := w.Commit(message, &git.CommitOptions{
		Author: author,
	})
	if err != nil {
		return "", fmt.Errorf("realizar commit: %w", err)
	}
	return hash.String(), nil
}

// ---------------------------------------------------------------------------
// Push / Pull / Fetch
// ---------------------------------------------------------------------------

func (g *LocalGit) Push(ctx context.Context, remote, branch string, force bool) error {
	repo, err := g.openRepo()
	if err != nil {
		return err
	}

	if branch == "" {
		branch, err = g.CurrentBranch(ctx)
		if err != nil {
			return fmt.Errorf("obter branch atual: %w", err)
		}
	}

	spec := fmt.Sprintf("refs/heads/%s:refs/heads/%s", branch, branch)
	if force {
		spec = "+" + spec
	}

	opts := &git.PushOptions{
		RemoteName: remote,
		RefSpecs:   []config.RefSpec{config.RefSpec(spec)},
		Auth:       g.getAuth(""),
	}

	if err := repo.PushContext(ctx, opts); err != nil {
		return fmt.Errorf("fazer push para %s/%s: %w", remote, branch, err)
	}
	return nil
}

func (g *LocalGit) Pull(ctx context.Context, remote, branch string) error {
	w, _, err := g.openWorktree()
	if err != nil {
		return err
	}

	if branch == "" {
		branch, err = g.CurrentBranch(ctx)
		if err != nil {
			return fmt.Errorf("obter branch atual: %w", err)
		}
	}
	if remote == "" {
		remote = "origin"
	}

	if err := w.PullContext(ctx, &git.PullOptions{
		RemoteName:    remote,
		ReferenceName: plumbing.ReferenceName("refs/heads/" + branch),
		Auth:          g.getAuth(""),
	}); err != nil && !errors.Is(err, git.NoErrAlreadyUpToDate) {
		return fmt.Errorf("fazer pull de %s/%s: %w", remote, branch, err)
	}
	return nil
}

func (g *LocalGit) Fetch(ctx context.Context, remote string) error {
	repo, err := g.openRepo()
	if err != nil {
		return err
	}
	if remote == "" {
		remote = "origin"
	}

	if err := repo.FetchContext(ctx, &git.FetchOptions{
		RemoteName: remote,
		Auth:       g.getAuth(""),
	}); err != nil && !errors.Is(err, git.NoErrAlreadyUpToDate) {
		return fmt.Errorf("fazer fetch de %s: %w", remote, err)
	}
	return nil
}

// ---------------------------------------------------------------------------
// Log (timeline)
// ---------------------------------------------------------------------------

func (g *LocalGit) Log(ctx context.Context, opts *LogOptions) ([]*types.Commit, error) {
	repo, err := g.openRepo()
	if err != nil {
		return nil, err
	}

	var logOpts git.LogOptions
	if opts != nil && opts.Branch != "" {
		ref, err := repo.Reference(plumbing.ReferenceName("refs/heads/"+opts.Branch), true)
		if err != nil {
			ref, err = repo.Reference(plumbing.ReferenceName("refs/remotes/"+opts.Branch), true)
			if err != nil {
				return nil, fmt.Errorf("branch %q não encontrada: %w", opts.Branch, err)
			}
		}
		logOpts.From = ref.Hash()
	}

	iter, err := repo.Log(&logOpts)
	if err != nil {
		return nil, fmt.Errorf("obter log de commits: %w", err)
	}
	defer iter.Close()

	var commits []*types.Commit
	limit := 0
	if opts != nil {
		limit = opts.Limit
	}

	_ = iter.ForEach(func(c *object.Commit) error {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if opts != nil {
			if limit > 0 && len(commits) >= limit {
				return errStopIteration
			}
			if opts.Since != nil && c.Committer.When.Before(*opts.Since) {
				return nil
			}
			if opts.Until != nil && c.Committer.When.After(*opts.Until) {
				return nil
			}
			if opts.Path != "" && !commitModifiesPath(c, opts.Path) {
				return nil
			}
		}

		commits = append(commits, types.NewCommit(
			c.Hash.String(),
			strings.TrimSpace(c.Message),
			c.Author.Name,
			c.Author.Email,
			c.Author.When,
			len(c.ParentHashes),
		))
		return nil
	})

	if err != nil && !errors.Is(err, errStopIteration) {
		return nil, fmt.Errorf("iterar commits: %w", err)
	}

	return commits, nil
}

// commitModifiesPath checks whether a commit modifies the given file path.
func commitModifiesPath(c *object.Commit, path string) bool {
	tree, err := c.Tree()
	if err != nil {
		return false
	}

	if len(c.ParentHashes) == 0 {
		// Root commit: check if file exists in this tree
		_, err := tree.FindEntry(path)
		return err == nil
	}

	parent, err := c.Parent(0)
	if err != nil {
		return false
	}

	parentTree, err := parent.Tree()
	if err != nil {
		return false
	}

	changes, err := parentTree.Diff(tree)
	if err != nil {
		return false
	}

	for _, ch := range changes {
		if ch.From.Name == path || ch.To.Name == path {
			return true
		}
	}
	return false
}

// ---------------------------------------------------------------------------
// Branches
// ---------------------------------------------------------------------------

func (g *LocalGit) Branches(ctx context.Context) ([]*types.Branch, error) {
	repo, err := g.openRepo()
	if err != nil {
		return nil, err
	}

	head, err := repo.Head()
	if err != nil {
		return nil, fmt.Errorf("obter HEAD: %w", err)
	}
	currentBranch := head.Name().Short()

	var branches []*types.Branch
	seenLocals := make(map[string]bool)

	// List local branches
	branchIter, err := repo.Branches()
	if err != nil {
		return nil, fmt.Errorf("listar branches locais: %w", err)
	}
	defer branchIter.Close()

	_ = branchIter.ForEach(func(ref *plumbing.Reference) error {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		name := ref.Name().Short()
		seenLocals[name] = true

		remoteName := g.trackingRemote(name)
		var ahead, behind int
		if remoteName != "" {
			ahead, behind = g.calcAheadBehind(ctx, name, remoteName)
		}

		branches = append(branches, &types.Branch{
			Name:       name,
			IsActive:   name == currentBranch,
			IsRemote:   false,
			RemoteName: remoteName,
			Ahead:      ahead,
			Behind:     behind,
		})
		return nil
	})

	// List remote branches without local counterpart
	refIter, err := repo.References()
	if err != nil {
		return nil, fmt.Errorf("listar referências: %w", err)
	}
	defer refIter.Close()

	_ = refIter.ForEach(func(ref *plumbing.Reference) error {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if !ref.Name().IsRemote() {
			return nil
		}

		fullName := ref.Name().Short()
		localName := fullName[strings.Index(fullName, "/")+1:]
		if seenLocals[localName] {
			return nil
		}

		branches = append(branches, &types.Branch{
			Name:     fullName,
			IsActive: false,
			IsRemote: true,
		})
		return nil
	})

	sort.Slice(branches, func(i, j int) bool {
		if branches[i].IsActive {
			return true
		}
		if branches[j].IsActive {
			return false
		}
		return branches[i].Name < branches[j].Name
	})

	return branches, nil
}

func (g *LocalGit) trackingRemote(branch string) string {
	cmd := exec.Command("git", "-C", g.repoPath, "config", "--get", "branch."+branch+".remote")
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

func (g *LocalGit) calcAheadBehind(ctx context.Context, branch, remote string) (int, int) {
	rangeSpec := fmt.Sprintf("refs/remotes/%s/%s...refs/heads/%s", remote, branch, branch)
	cmd := exec.CommandContext(ctx, "git", "-C", g.repoPath, "rev-list", "--count", "--left-right", rangeSpec)
	out, err := cmd.Output()
	if err != nil {
		return 0, 0
	}
	parts := strings.Split(strings.TrimSpace(string(out)), "\t")
	if len(parts) == 2 {
		behind, _ := strconv.Atoi(parts[0])
		ahead, _ := strconv.Atoi(parts[1])
		return ahead, behind
	}
	return 0, 0
}

// ---------------------------------------------------------------------------
// Checkout / Create / Delete / Merge
// ---------------------------------------------------------------------------

func (g *LocalGit) Checkout(ctx context.Context, branch string) error {
	w, _, err := g.openWorktree()
	if err != nil {
		return err
	}
	return w.Checkout(&git.CheckoutOptions{
		Branch: plumbing.ReferenceName("refs/heads/" + branch),
		Force:  false,
	})
}

func (g *LocalGit) CreateBranch(ctx context.Context, name, base string) error {
	repo, err := g.openRepo()
	if err != nil {
		return err
	}

	baseRef, err := repo.Reference(plumbing.ReferenceName("refs/heads/"+base), true)
	if err != nil {
		baseRef, err = repo.Reference(plumbing.ReferenceName("refs/remotes/"+base), true)
		if err != nil {
			return fmt.Errorf("base %q não encontrada: %w", base, err)
		}
	}

	newRef := plumbing.NewHashReference(
		plumbing.ReferenceName("refs/heads/"+name),
		baseRef.Hash(),
	)
	if err := repo.Storer.SetReference(newRef); err != nil {
		return fmt.Errorf("criar branch: %w", err)
	}

	w, err := repo.Worktree()
	if err != nil {
		return fmt.Errorf("abrir worktree: %w", err)
	}
	return w.Checkout(&git.CheckoutOptions{
		Branch: plumbing.ReferenceName("refs/heads/" + name),
	})
}

func (g *LocalGit) DeleteBranch(ctx context.Context, name string, force bool) error {
	repo, err := g.openRepo()
	if err != nil {
		return err
	}

	ref := plumbing.ReferenceName("refs/heads/" + name)
	if _, err := repo.Reference(ref, true); err != nil {
		return fmt.Errorf("branch %q não encontrada: %w", name, err)
	}

	if !force {
		cmd := exec.CommandContext(ctx, "git", "-C", g.repoPath, "branch", "--merged")
		out, err := cmd.Output()
		if err == nil {
			merged := strings.Split(strings.TrimSpace(string(out)), "\n")
			found := false
			for _, line := range merged {
				line = strings.TrimSpace(strings.TrimPrefix(line, "* "))
				if line == name {
					found = true
					break
				}
			}
			if !found {
				return fmt.Errorf("branch %q não está totalmente mesclada; use force para deletar", name)
			}
		}
	}

	if err := repo.Storer.RemoveReference(ref); err != nil {
		return fmt.Errorf("deletar branch %q: %w", name, err)
	}
	_ = repo.DeleteBranch(name)
	return nil
}

func (g *LocalGit) Merge(ctx context.Context, branch string) error {
	cmd := exec.CommandContext(ctx, "git", "-C", g.repoPath, "merge", "--no-edit", branch)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("mesclar branch %q: %s: %w", branch, strings.TrimSpace(string(out)), err)
	}
	return nil
}

func (g *LocalGit) CurrentBranch(ctx context.Context) (string, error) {
	repo, err := g.openRepo()
	if err != nil {
		return "", err
	}
	head, err := repo.Head()
	if err != nil {
		return "", fmt.Errorf("obter HEAD: %w", err)
	}
	if !head.Name().IsBranch() {
		return head.Hash().String()[:8], nil // detached HEAD
	}
	return head.Name().Short(), nil
}

// ---------------------------------------------------------------------------
// Remote management
// ---------------------------------------------------------------------------

func (g *LocalGit) Remotes(ctx context.Context) ([]*types.Remote, error) {
	repo, err := g.openRepo()
	if err != nil {
		return nil, err
	}

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	remotes, err := repo.Remotes()
	if err != nil {
		return nil, fmt.Errorf("listar remotes: %w", err)
	}

	result := make([]*types.Remote, 0, len(remotes))
	for _, r := range remotes {
		cfg := r.Config()
		urls := make([]string, len(cfg.URLs))
		copy(urls, cfg.URLs)
		fetchURLs := make([]string, len(cfg.URLs))
		copy(fetchURLs, cfg.URLs)

		result = append(result, &types.Remote{
			Name:      cfg.Name,
			URLs:      urls,
			FetchURLs: fetchURLs,
		})
	}
	return result, nil
}

func (g *LocalGit) AddRemote(ctx context.Context, name, url string) error {
	repo, err := g.openRepo()
	if err != nil {
		return err
	}
	_, err = repo.CreateRemote(&config.RemoteConfig{
		Name: name,
		URLs: []string{url},
	})
	if err != nil {
		return fmt.Errorf("adicionar remote %q: %w", name, err)
	}
	return nil
}

func (g *LocalGit) RemoveRemote(ctx context.Context, name string) error {
	repo, err := g.openRepo()
	if err != nil {
		return err
	}
	if err := repo.DeleteRemote(name); err != nil {
		return fmt.Errorf("remover remote %q: %w", name, err)
	}
	return nil
}

func (g *LocalGit) SetRemoteURL(ctx context.Context, name, url string) error {
	repo, err := g.openRepo()
	if err != nil {
		return err
	}

	remote, err := repo.Remote(name)
	if err != nil {
		return fmt.Errorf("remote %q não encontrado: %w", name, err)
	}

	if err := repo.DeleteRemote(name); err != nil {
		return fmt.Errorf("remover remote %q: %w", name, err)
	}

	_, err = repo.CreateRemote(&config.RemoteConfig{
		Name:  name,
		URLs:  []string{url},
		Fetch: remote.Config().Fetch,
	})
	if err != nil {
		return fmt.Errorf("recriar remote %q: %w", name, err)
	}
	return nil
}

// ---------------------------------------------------------------------------
// Compile-time interface check
// ---------------------------------------------------------------------------

var _ GitOperations = (*LocalGit)(nil)
