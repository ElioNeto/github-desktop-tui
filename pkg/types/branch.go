package types

// Branch represents a Git branch.
type Branch struct {
	// Name is the short name of the branch (e.g., "main").
	Name string `json:"name"`
	// IsRemote indicates whether this is a remote-tracking branch.
	IsRemote bool `json:"is_remote"`
	// IsActive indicates whether this is the currently checked-out branch.
	IsActive bool `json:"is_active"`
	// RemoteName is the name of the upstream remote, if any.
	RemoteName string `json:"remote_name,omitempty"`
	// CommitHash is the hash of the commit this branch points to.
	CommitHash string `json:"commit_hash,omitempty"`
	// Message is the latest commit message (for display).
	Message string `json:"message,omitempty"`
	// Ahead is the number of commits the local branch is ahead of its remote.
	Ahead int `json:"ahead"`
	// Behind is the number of commits the local branch is behind its remote.
	Behind int `json:"behind"`
}
