package types

// FileStatus represents the status of a file in the working tree or staging area.
type FileStatus int

const (
	FileStatusUnmodified  FileStatus = iota // ' '
	FileStatusAdded                         // A
	FileStatusDeleted                       // D
	FileStatusModified                      // M
	FileStatusRenamed                       // R
	FileStatusCopied                        // C
	FileStatusUpdated                       // U (updated but unmerged)
	FileStatusUntracked                     // ?
	FileStatusIgnored                       // !
)

// String returns a human-readable description of the file status.
func (s FileStatus) String() string {
	switch s {
	case FileStatusUnmodified:
		return "unmodified"
	case FileStatusAdded:
		return "added"
	case FileStatusDeleted:
		return "deleted"
	case FileStatusModified:
		return "modified"
	case FileStatusRenamed:
		return "renamed"
	case FileStatusCopied:
		return "copied"
	case FileStatusUpdated:
		return "updated"
	case FileStatusUntracked:
		return "untracked"
	case FileStatusIgnored:
		return "ignored"
	default:
		return "unknown"
	}
}

// Short returns a single-character representation of the file status,
// matching the standard git status output.
func (s FileStatus) Short() string {
	switch s {
	case FileStatusUnmodified:
		return " "
	case FileStatusAdded:
		return "A"
	case FileStatusDeleted:
		return "D"
	case FileStatusModified:
		return "M"
	case FileStatusRenamed:
		return "R"
	case FileStatusCopied:
		return "C"
	case FileStatusUpdated:
		return "U"
	case FileStatusUntracked:
		return "?"
	case FileStatusIgnored:
		return "!"
	default:
		return "?"
	}
}

// ChangeStatus is a string-based status for backward compatibility.
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

// FileChange represents a single file change in the working tree or staging area.
type FileChange struct {
	// Path is the file path relative to the repository root.
	Path string `json:"path"`
	// OldPath is the previous path for renamed files.
	OldPath string `json:"old_path,omitempty"`
	// Status is the current status of the file in the working tree (int enum).
	Status FileStatus `json:"status"`
	// Staged indicates whether the file is currently staged for commit.
	Staged bool `json:"staged"`
	// StagedStatus is the status of the file in the staging area.
	StagedStatus FileStatus `json:"staged_status,omitempty"`
	// AddedLines is the number of added lines (from diff).
	AddedLines int `json:"added_lines,omitempty"`
	// DeletedLines is the number of deleted lines (from diff).
	DeletedLines int `json:"deleted_lines,omitempty"`
}

// StatusString returns a human-readable status string.
func (fc *FileChange) StatusString() string {
	if fc.Staged && fc.StagedStatus != FileStatusUnmodified {
		return fc.StagedStatus.String()
	}
	return fc.Status.String()
}

// StatusShort returns a single-character status indicator.
func (fc *FileChange) StatusShort() string {
	if fc.Staged && fc.StagedStatus != FileStatusUnmodified {
		return fc.StagedStatus.Short()
	}
	return fc.Status.Short()
}
