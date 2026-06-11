package github

import (
	"time"

	"github.com/nicoddemus/github-desktop-tui/pkg/types"
)

// GitHub API response types

type githubRepo struct {
	ID          int64     `json:"id"`
	Name        string    `json:"name"`
	FullName    string    `json:"full_name"`
	Owner       struct {
		Login string `json:"login"`
	} `json:"owner"`
	Description  string    `json:"description"`
	HTMLURL      string    `json:"html_url"`
	CloneURL     string    `json:"clone_url"`
	Language     string    `json:"language"`
	DefaultBranch string   `json:"default_branch"`
	Private      bool      `json:"private"`
	Fork         bool      `json:"fork"`
	StargazersCount int    `json:"stargazers_count"`
	ForksCount    int      `json:"forks_count"`
	OpenIssuesCount int    `json:"open_issues_count"`
	Topics       []string  `json:"topics"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
	PushedAt     time.Time `json:"pushed_at"`
}

func (r *githubRepo) toRepository() *types.Repository {
	return &types.Repository{
		ID:            r.ID,
		Provider:      "github",
		Name:          r.Name,
		Owner:         r.Owner.Login,
		FullName:      r.FullName,
		Description:   r.Description,
		URL:           r.HTMLURL,
		CloneURL:      r.CloneURL,
		Language:      r.Language,
		DefaultBranch: r.DefaultBranch,
		Private:       r.Private,
		Fork:          r.Fork,
		Stars:         r.StargazersCount,
		Forks:         r.ForksCount,
		OpenIssues:    r.OpenIssuesCount,
		Topics:        r.Topics,
		CreatedAt:     r.CreatedAt,
		UpdatedAt:     r.UpdatedAt,
		PushedAt:      r.PushedAt,
	}
}

type githubPR struct {
	ID          int64      `json:"id"`
	Number      int        `json:"number"`
	Title       string     `json:"title"`
	Body        string     `json:"body"`
	State       string     `json:"state"`
	User        struct {
		Login string `json:"login"`
	} `json:"user"`
	Head        struct {
		Ref string `json:"ref"`
	} `json:"head"`
	Base        struct {
		Ref string `json:"ref"`
	} `json:"base"`
	HTMLURL     string     `json:"html_url"`
	Draft       bool       `json:"draft"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	Labels      []struct {
		Name string `json:"name"`
	} `json:"labels"`
}

func (pr *githubPR) toPullRequest() *types.PullRequest {
	state := types.PRStateOpen
	switch pr.State {
	case "closed":
		if pr.State == "closed" {
			state = types.PRStateClosed
		}
		// Note: GitHub's API doesn't distinguish merged via this endpoint
		// A separate check would be needed
	case "open":
		state = types.PRStateOpen
	}

	labels := make([]string, len(pr.Labels))
	for i, l := range pr.Labels {
		labels[i] = l.Name
	}

	return &types.PullRequest{
		ID:           pr.ID,
		Number:       pr.Number,
		Title:        pr.Title,
		Description:  pr.Body,
		State:        state,
		Author:       pr.User.Login,
		SourceBranch: pr.Head.Ref,
		TargetBranch: pr.Base.Ref,
		URL:          pr.HTMLURL,
		IsDraft:      pr.Draft,
		CreatedAt:    pr.CreatedAt,
		UpdatedAt:    pr.UpdatedAt,
		Labels:       labels,
	}
}

type githubIssue struct {
	ID          int64      `json:"id"`
	Number      int        `json:"number"`
	Title       string     `json:"title"`
	Body        string     `json:"body"`
	State       string     `json:"state"`
	User        struct {
		Login string `json:"login"`
	} `json:"user"`
	HTMLURL     string     `json:"html_url"`
	PullRequest *struct{}  `json:"pull_request,omitempty"`
	Labels      []struct {
		Name string `json:"name"`
	} `json:"labels"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

func (gi *githubIssue) toIssue() *types.Issue {
	state := types.IssueStateOpen
	switch gi.State {
	case "closed":
		state = types.IssueStateClosed
	case "open":
		state = types.IssueStateOpen
	}

	labels := make([]string, len(gi.Labels))
	for i, l := range gi.Labels {
		labels[i] = l.Name
	}

	return &types.Issue{
		ID:          gi.ID,
		Number:      gi.Number,
		Title:       gi.Title,
		Description: gi.Body,
		State:       state,
		Author:      gi.User.Login,
		URL:         gi.HTMLURL,
		IsPullRequest: gi.PullRequest != nil,
		Labels:     labels,
		CreatedAt:  gi.CreatedAt,
		UpdatedAt:  gi.UpdatedAt,
	}
}

var _ = (*githubIssue).toIssue // avoid unused function warning

type githubBranch struct {
	Name      string `json:"name"`
	Commit    struct {
		SHA string `json:"sha"`
	} `json:"commit"`
	Protected bool `json:"protected"`
}

func (b *githubBranch) toBranch() *types.Branch {
	return &types.Branch{
		Name:       b.Name,
		IsRemote:   false,
		IsActive:   false,
		CommitHash: b.Commit.SHA,
	}
}

type githubCommit struct {
	SHA    string `json:"sha"`
	Commit struct {
		Author struct {
			Name  string    `json:"name"`
			Email string    `json:"email"`
			Date  time.Time `json:"date"`
		} `json:"author"`
		Message string `json:"message"`
	} `json:"commit"`
	Parents []struct {
		SHA string `json:"sha"`
	} `json:"parents"`
}

func (c *githubCommit) toCommit() *types.Commit {
	msg := c.Commit.Message
	msgHead := msg
	if len(msg) > 80 {
		// Take first line only
		for i := 0; i < len(msg) && i < 80; i++ {
			if msg[i] == '\n' {
				msgHead = msg[:i]
				break
			}
		}
	}

	return &types.Commit{
		Hash:        c.SHA,
		ShortHash:   c.SHA[:7],
		Author:      c.Commit.Author.Name,
		AuthorEmail: c.Commit.Author.Email,
		Message:     msg,
		MessageHead: msgHead,
		Timestamp:   c.Commit.Author.Date,
		ParentCount: len(c.Parents),
	}
}
