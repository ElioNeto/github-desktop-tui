package types

// Branch represents a Git branch.
type Branch struct {
	Name       string `json:"name"`
	IsRemote   bool   `json:"is_remote"`
	IsActive   bool   `json:"is_active"`
	RemoteName string `json:"remote_name,omitempty"`
	CommitHash string `json:"commit_hash,omitempty"`
	Message    string `json:"message,omitempty"` // Latest commit message
	Ahead      int    `json:"ahead"`
	Behind     int    `json:"behind"`
}
