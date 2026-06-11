package types

import "time"

// PullRequestState represents the state of a pull/merge request.
type PullRequestState string

const (
	PRStateOpen   PullRequestState = "open"
	PRStateClosed PullRequestState = "closed"
	PRStateMerged PullRequestState = "merged"
	PRStateAll    PullRequestState = "all"
)

// PullRequest represents a pull request or merge request.
type PullRequest struct {
	ID          int64            `json:"id"`
	Number      int              `json:"number"`
	Title       string           `json:"title"`
	Description string           `json:"description"`
	State       PullRequestState `json:"state"`
	Author      string           `json:"author"`
	SourceBranch string          `json:"source_branch"`
	TargetBranch string          `json:"target_branch"`
	URL         string           `json:"url"`
	IsDraft     bool             `json:"is_draft"`
	CreatedAt   time.Time        `json:"created_at"`
	UpdatedAt   time.Time        `json:"updated_at"`
	Labels      []string         `json:"labels"`
}
