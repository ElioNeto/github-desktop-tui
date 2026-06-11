package github

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/nicoddemus/github-desktop-tui/internal/providers"
	"github.com/nicoddemus/github-desktop-tui/pkg/types"
)

const (
	defaultBaseURL = "https://api.github.com"
	defaultPerPage = 30
)

// Client is the GitHub API client.
type Client struct {
	baseURL string
	token   string
	http    *http.Client
}

// NewClient creates a new GitHub API client.
func NewClient(token, baseURL string) *Client {
	if baseURL == "" {
		baseURL = defaultBaseURL
	}

	return &Client{
		baseURL: baseURL,
		token:   token,
		http: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// HasToken reports whether the client has a valid token configured.
func (c *Client) HasToken() bool {
	return c.token != ""
}

// doRequest performs an authenticated HTTP request to the GitHub API.
func (c *Client) doRequest(ctx context.Context, method, path string, params url.Values, body io.Reader) (*http.Response, error) {
	u := fmt.Sprintf("%s%s", c.baseURL, path)
	if params != nil {
		u += "?" + params.Encode()
	}

	req, err := http.NewRequestWithContext(ctx, method, u, body)
	if err != nil {
		return nil, fmt.Errorf("criar requisição: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("User-Agent", "github-desktop-tui")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("executar requisição: %w", err)
	}

	if resp.StatusCode == http.StatusUnauthorized {
		return nil, fmt.Errorf("token inválido ou expirado")
	}

	if resp.StatusCode == http.StatusForbidden {
		return nil, fmt.Errorf("taxa de requisições excedida")
	}

	if resp.StatusCode >= 400 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, fmt.Errorf("erro GitHub API (%d): %s", resp.StatusCode, string(bodyBytes))
	}

	return resp, nil
}

// get performs a GET request and decodes the JSON response.
func (c *Client) get(ctx context.Context, path string, params url.Values, v interface{}) error {
	resp, err := c.doRequest(ctx, http.MethodGet, path, params, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if err := json.NewDecoder(resp.Body).Decode(v); err != nil {
		return fmt.Errorf("decodificar resposta: %w", err)
	}

	return nil
}

// ListRepos lists repositories for the authenticated user.
func (c *Client) ListRepos(ctx context.Context) ([]*types.Repository, error) {
	params := url.Values{}
	params.Set("per_page", strconv.Itoa(defaultPerPage))
	params.Set("sort", "updated")
	params.Set("type", "all")

	var ghRepos []githubRepo
	if err := c.get(ctx, "/user/repos", params, &ghRepos); err != nil {
		return nil, err
	}

	repos := make([]*types.Repository, len(ghRepos))
	for i, r := range ghRepos {
		repos[i] = r.toRepository()
	}

	return repos, nil
}

// GetRepo gets a single repository by owner and name.
func (c *Client) GetRepo(ctx context.Context, owner, name string) (*types.Repository, error) {
	path := fmt.Sprintf("/repos/%s/%s", url.PathEscape(owner), url.PathEscape(name))

	var ghRepo githubRepo
	if err := c.get(ctx, path, nil, &ghRepo); err != nil {
		return nil, err
	}

	return ghRepo.toRepository(), nil
}

// ListPRs lists pull requests for a repository.
func (c *Client) ListPRs(ctx context.Context, owner, repo string, state types.PullRequestState) ([]*types.PullRequest, error) {
	path := fmt.Sprintf("/repos/%s/%s/pulls", url.PathEscape(owner), url.PathEscape(repo))
	params := url.Values{}
	params.Set("per_page", strconv.Itoa(defaultPerPage))
	params.Set("state", string(state))
	params.Set("sort", "updated")
	params.Set("direction", "desc")

	var ghPRs []githubPR
	if err := c.get(ctx, path, params, &ghPRs); err != nil {
		return nil, err
	}

	prs := make([]*types.PullRequest, len(ghPRs))
	for i, pr := range ghPRs {
		prs[i] = pr.toPullRequest()
	}

	return prs, nil
}

// GetPR gets a single pull request by number.
func (c *Client) GetPR(ctx context.Context, owner, repo string, id int) (*types.PullRequest, error) {
	path := fmt.Sprintf("/repos/%s/%s/pulls/%d", url.PathEscape(owner), url.PathEscape(repo), id)

	var ghPR githubPR
	if err := c.get(ctx, path, nil, &ghPR); err != nil {
		return nil, err
	}

	return ghPR.toPullRequest(), nil
}

// ListIssues lists issues for a repository.
func (c *Client) ListIssues(ctx context.Context, owner, repo string, state types.IssueState) ([]*types.Issue, error) {
	path := fmt.Sprintf("/repos/%s/%s/issues", url.PathEscape(owner), url.PathEscape(repo))
	params := url.Values{}
	params.Set("per_page", strconv.Itoa(defaultPerPage))
	params.Set("state", string(state))
	params.Set("sort", "updated")
	params.Set("direction", "desc")

	var ghIssues []githubIssue
	if err := c.get(ctx, path, params, &ghIssues); err != nil {
		return nil, err
	}

	issues := make([]*types.Issue, 0, len(ghIssues))
	for _, gi := range ghIssues {
		if gi.PullRequest == nil {
			issues = append(issues, gi.toIssue())
		}
	}

	return issues, nil
}

// ListBranches lists branches for a repository.
func (c *Client) ListBranches(ctx context.Context, owner, repo string) ([]*types.Branch, error) {
	path := fmt.Sprintf("/repos/%s/%s/branches", url.PathEscape(owner), url.PathEscape(repo))
	params := url.Values{}
	params.Set("per_page", strconv.Itoa(defaultPerPage))

	var ghBranches []githubBranch
	if err := c.get(ctx, path, params, &ghBranches); err != nil {
		return nil, err
	}

	branches := make([]*types.Branch, len(ghBranches))
	for i, b := range ghBranches {
		branches[i] = b.toBranch()
	}

	return branches, nil
}

// ListCommits lists commits for a repository.
func (c *Client) ListCommits(ctx context.Context, owner, repo string, opts *providers.CommitListOptions) ([]*types.Commit, error) {
	path := fmt.Sprintf("/repos/%s/%s/commits", url.PathEscape(owner), url.PathEscape(repo))
	params := url.Values{}
	params.Set("per_page", strconv.Itoa(defaultPerPage))

	if opts != nil {
		if opts.Branch != "" {
			params.Set("sha", opts.Branch)
		}
		if opts.Limit > 0 {
			params.Set("per_page", strconv.Itoa(opts.Limit))
		}
		if opts.Since != "" {
			params.Set("since", opts.Since)
		}
		if opts.Until != "" {
			params.Set("until", opts.Until)
		}
		if opts.Path != "" {
			params.Set("path", opts.Path)
		}
	}

	var ghCommits []githubCommit
	if err := c.get(ctx, path, params, &ghCommits); err != nil {
		return nil, err
	}

	commits := make([]*types.Commit, len(ghCommits))
	for i, c := range ghCommits {
		commits[i] = c.toCommit()
	}

	return commits, nil
}
