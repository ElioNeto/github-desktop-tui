package git

import (
	"context"

	"github.com/nicoddemus/github-desktop-tui/pkg/types"
)

// GitOperations defines the interface for local Git operations.
type GitOperations interface {
	// Status returns the current working tree status.
	Status(ctx context.Context) ([]*types.FileChange, error)

	// Diff returns the diff for a specific file or all unstaged changes.
	Diff(ctx context.Context, path string) (string, error)

	// StagedDiff returns the diff of staged changes.
	StagedDiff(ctx context.Context) (string, error)

	// Stage adds files to the staging area.
	Stage(ctx context.Context, paths ...string) error

	// Unstage removes files from the staging area.
	Unstage(ctx context.Context, paths ...string) error

	// Discard reverts unstaged changes.
	Discard(ctx context.Context, paths ...string) error

	// Commit creates a new commit.
	Commit(ctx context.Context, message string) (string, error)

	// Push pushes commits to the remote.
	Push(ctx context.Context, remote, branch string, force bool) error

	// Pull pulls changes from the remote.
	Pull(ctx context.Context, remote, branch string) error

	// Fetch fetches changes from the remote.
	Fetch(ctx context.Context, remote string) error

	// Log returns the commit history.
	Log(ctx context.Context, opts *LogOptions) ([]*types.Commit, error)

	// Branches lists all local and remote branches.
	Branches(ctx context.Context) ([]*types.Branch, error)

	// Checkout switches to a branch.
	Checkout(ctx context.Context, branch string) error

	// CreateBranch creates a new branch.
	CreateBranch(ctx context.Context, name, base string) error

	// DeleteBranch deletes a branch.
	DeleteBranch(ctx context.Context, name string, force bool) error

	// Merge merges a branch into the current branch.
	Merge(ctx context.Context, branch string) error

	// CurrentBranch returns the name of the current branch.
	CurrentBranch(ctx context.Context) (string, error)

	// Root returns the repository root path.
	Root() string
}

// LogOptions provides filtering options for commit log.
type LogOptions struct {
	Branch string
	Limit  int
	Since  string
	Until  string
	Path   string
	Author string
}

// LocalGit implements GitOperations using go-git or git CLI.
type LocalGit struct {
	repoPath string
}

// New creates a new LocalGit instance for the given repository path.
func New(repoPath string) *LocalGit {
	return &LocalGit{repoPath: repoPath}
}

// Root returns the repository root directory.
func (g *LocalGit) Root() string {
	return g.repoPath
}
