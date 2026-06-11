package providers

import (
	"context"

	"github.com/nicoddemus/github-desktop-tui/pkg/types"
)

// AuthType represents the authentication method used by a provider.
type AuthType string

const (
	AuthOAuth AuthType = "oauth"
	AuthToken AuthType = "token"
	AuthBasic AuthType = "basic"
	AuthSSH   AuthType = "ssh"
)

// AuthResult contains the result of an authentication attempt.
type AuthResult struct {
	Success     bool
	Token       string
	Username    string
	Error       error
}

// GitProvider defines the interface that all Git providers must implement.
type GitProvider interface {
	// Identity
	Name() string
	DisplayName() string
	Icon() string

	// Authentication
	AuthType() AuthType
	IsAuthenticated() bool
	Authenticate(ctx context.Context) (*AuthResult, error)

	// Repositories
	ListRepositories(ctx context.Context) ([]*types.Repository, error)
	GetRepository(ctx context.Context, owner, name string) (*types.Repository, error)

	// Pull Requests / Merge Requests
	ListPullRequests(ctx context.Context, repo *types.Repository, state types.PullRequestState) ([]*types.PullRequest, error)
	GetPullRequest(ctx context.Context, repo *types.Repository, id int) (*types.PullRequest, error)

	// Issues
	ListIssues(ctx context.Context, repo *types.Repository, state types.IssueState) ([]*types.Issue, error)

	// Branches
	ListBranches(ctx context.Context, repo *types.Repository) ([]*types.Branch, error)

	// Commits
	ListCommits(ctx context.Context, repo *types.Repository, opts *CommitListOptions) ([]*types.Commit, error)
}

// CommitListOptions provides filtering options for commit listing.
type CommitListOptions struct {
	Branch string // Branch name
	Limit  int    // Max commits to return
	Since  string // ISO 8601 date
	Until  string // ISO 8601 date
	Path   string // Filter by file path
}
