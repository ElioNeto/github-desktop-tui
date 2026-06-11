package types

import "time"

// Commit represents a Git commit.
type Commit struct {
	Hash        string    `json:"hash"`
	ShortHash   string    `json:"short_hash"`
	Author      string    `json:"author"`
	AuthorEmail string    `json:"author_email"`
	Message     string    `json:"message"`
	MessageHead string    `json:"message_head"` // First line of message
	Timestamp   time.Time `json:"timestamp"`
	Branch      string    `json:"branch"`
	Refs        []string  `json:"refs"` // branches/tags pointing to this commit
	ParentCount int       `json:"parent_count"`
}
