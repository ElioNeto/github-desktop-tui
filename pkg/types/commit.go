package types

import "time"

// Commit represents a single Git commit.
type Commit struct {
	// Hash is the full SHA commit hash.
	Hash string `json:"hash"`
	// ShortHash is the abbreviated hash (7 chars).
	ShortHash string `json:"short_hash"`
	// Message is the full commit message.
	Message string `json:"message"`
	// MessageHead is the first line of the commit message.
	MessageHead string `json:"message_head"`
	// Author is the name of the commit author.
	Author string `json:"author"`
	// AuthorEmail is the email of the commit author.
	AuthorEmail string `json:"author_email"`
	// Timestamp is when the commit was authored.
	Timestamp time.Time `json:"timestamp"`
	// Branch is the branch this commit belongs to.
	Branch string `json:"branch,omitempty"`
	// Refs are branch/tag names pointing to this commit.
	Refs []string `json:"refs,omitempty"`
	// ParentCount is the number of parent commits.
	ParentCount int `json:"parent_count"`
}

// NewCommit creates a Commit from a raw message and hash.
func NewCommit(hash, message, author, email string, timestamp time.Time, parentCount int) *Commit {
	msgHead := message
	if len(message) > 80 {
		for i := 0; i < len(message) && i < 80; i++ {
			if message[i] == '\n' {
				msgHead = message[:i]
				break
			}
		}
	}

	shortHash := hash
	if len(hash) > 7 {
		shortHash = hash[:7]
	}

	return &Commit{
		Hash:        hash,
		ShortHash:   shortHash,
		Message:     message,
		MessageHead: msgHead,
		Author:      author,
		AuthorEmail: email,
		Timestamp:   timestamp,
		ParentCount: parentCount,
	}
}
