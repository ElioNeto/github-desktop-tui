package types

// ChangeStatus represents the status of a file change.
type ChangeStatus string

const (
	ChangeAdded     ChangeStatus = "added"
	ChangeModified  ChangeStatus = "modified"
	ChangeDeleted   ChangeStatus = "deleted"
	ChangeRenamed   ChangeStatus = "renamed"
	ChangeCopied    ChangeStatus = "copied"
	ChangeUnmerged  ChangeStatus = "unmerged"
	ChangeUntracked ChangeStatus = "untracked"
)

// FileChange represents a single file change in the working tree.
type FileChange struct {
	Path         string       `json:"path"`
	OldPath      string       `json:"old_path,omitempty"` // For renames
	Status       ChangeStatus `json:"status"`
	Staged       bool         `json:"staged"`
	AddedLines   int          `json:"added_lines"`
	DeletedLines int          `json:"deleted_lines"`
}
