package types

import "time"

// IssueState represents the state of an issue.
type IssueState string

const (
	IssueStateOpen   IssueState = "open"
	IssueStateClosed IssueState = "closed"
	IssueStateAll    IssueState = "all"
)

// Issue represents an issue from any provider.
type Issue struct {
	ID          int64      `json:"id"`
	Number      int        `json:"number"`
	Title       string     `json:"title"`
	Description string     `json:"description"`
	State       IssueState `json:"state"`
	Author      string     `json:"author"`
	URL         string     `json:"url"`
	IsPullRequest bool    `json:"is_pull_request"`
	Labels      []string   `json:"labels"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}
