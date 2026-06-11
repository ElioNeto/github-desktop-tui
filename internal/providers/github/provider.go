package github

import (
	"context"

	"github.com/ElioNeto/github-desktop-tui/internal/providers"
	"github.com/ElioNeto/github-desktop-tui/pkg/types"
)

// GitHubProvider implements providers.GitProvider for GitHub.
type GitHubProvider struct {
	client *Client
}

// NewProvider creates a new GitHub provider.
func NewProvider() *GitHubProvider {
	return &GitHubProvider{}
}

func (p *GitHubProvider) Name() string           { return "github" }
func (p *GitHubProvider) DisplayName() string    { return "GitHub" }
func (p *GitHubProvider) Icon() string           { return "" }

func (p *GitHubProvider) AuthType() providers.AuthType {
	return providers.AuthToken
}

func (p *GitHubProvider) IsAuthenticated() bool {
	return p.client != nil && p.client.HasToken()
}

func (p *GitHubProvider) Authenticate(ctx context.Context) (*providers.AuthResult, error) {
	// GitHub uses personal access tokens
	// In a real implementation, this would:
	// 1. Check env var GITHUB_TOKEN
	// 2. Check OS keychain
	// 3. Prompt the user for a token
	// 4. Validate the token with a test API call
	return &providers.AuthResult{
		Success: false,
		Error:   context.Canceled,
	}, nil
}

func (p *GitHubProvider) ListRepositories(ctx context.Context) ([]*types.Repository, error) {
	if p.client == nil {
		return nil, nil
	}
	return p.client.ListRepos(ctx)
}

func (p *GitHubProvider) GetRepository(ctx context.Context, owner, name string) (*types.Repository, error) {
	if p.client == nil {
		return nil, nil
	}
	return p.client.GetRepo(ctx, owner, name)
}

func (p *GitHubProvider) ListPullRequests(ctx context.Context, repo *types.Repository, state types.PullRequestState) ([]*types.PullRequest, error) {
	if p.client == nil {
		return nil, nil
	}
	return p.client.ListPRs(ctx, repo.Owner, repo.Name, state)
}

func (p *GitHubProvider) GetPullRequest(ctx context.Context, repo *types.Repository, id int) (*types.PullRequest, error) {
	if p.client == nil {
		return nil, nil
	}
	return p.client.GetPR(ctx, repo.Owner, repo.Name, id)
}

func (p *GitHubProvider) ListIssues(ctx context.Context, repo *types.Repository, state types.IssueState) ([]*types.Issue, error) {
	if p.client == nil {
		return nil, nil
	}
	return p.client.ListIssues(ctx, repo.Owner, repo.Name, state)
}

func (p *GitHubProvider) ListBranches(ctx context.Context, repo *types.Repository) ([]*types.Branch, error) {
	if p.client == nil {
		return nil, nil
	}
	return p.client.ListBranches(ctx, repo.Owner, repo.Name)
}

func (p *GitHubProvider) ListCommits(ctx context.Context, repo *types.Repository, opts *providers.CommitListOptions) ([]*types.Commit, error) {
	if p.client == nil {
		return nil, nil
	}
	return p.client.ListCommits(ctx, repo.Owner, repo.Name, opts)
}
