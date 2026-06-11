package types

import "time"

// Repository represents a Git repository from any provider.
type Repository struct {
	ID          int64     `json:"id"`
	Provider    string    `json:"provider"`
	Name        string    `json:"name"`
	Owner       string    `json:"owner"`
	FullName    string    `json:"full_name"`
	Description string    `json:"description"`
	URL         string    `json:"url"`
	CloneURL    string    `json:"clone_url"`
	Language    string    `json:"language"`
	DefaultBranch string  `json:"default_branch"`
	Private     bool      `json:"private"`
	Fork        bool      `json:"fork"`
	Stars       int       `json:"stars"`
	Forks       int       `json:"forks"`
	OpenIssues  int       `json:"open_issues"`
	Topics      []string  `json:"topics"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	PushedAt    time.Time `json:"pushed_at"`
}

// RepositoryList is a sortable slice of repositories.
type RepositoryList []*Repository

func (l RepositoryList) Len() int           { return len(l) }
func (l RepositoryList) Less(i, j int) bool { return l[i].FullName < l[j].FullName }
func (l RepositoryList) Swap(i, j int)      { l[i], l[j] = l[j], l[i] }
