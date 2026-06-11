package types

// GraphRow represents a single row in the visual commit graph output.
type GraphRow struct {
	// Graph is the ASCII graph prefix (lines, dots, branches).
	Graph string
	// Hash is the abbreviated commit hash.
	Hash string
	// Message is the first line of the commit message.
	Message string
	// Author is the commit author name.
	Author string
	// Time is the relative time string (e.g., "2h ago").
	Time string
	// Ref contains branch/tag references (e.g., "HEAD -> main, origin/main").
	Ref string
	// IsCommit is true if this line has a commit dot, false for continuation lines.
	IsCommit bool
}

// GraphColors defines the color palette for graph branch lines.
var GraphColors = []string{
	"#ef6c00", // Glint orange (primary)
	"#42a5f5", // blue
	"#66bb6a", // green
	"#ab47bc", // purple
	"#26c6da", // cyan
	"#ec407a", // pink
	"#ffa726", // amber
	"#8d6e63", // brown
	"#bdbdbd", // grey
	"#78909c", // blue-grey
	"#ef5350", // red
	"#7c4dff", // deep purple
}
