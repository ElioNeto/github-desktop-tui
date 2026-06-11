package tui

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/nicoddemus/github-desktop-tui/internal/git"
	"github.com/nicoddemus/github-desktop-tui/internal/tui/theme"
	"github.com/nicoddemus/github-desktop-tui/pkg/types"
)

// ---------------------------------------------------------------------------
// F2: File Tree Explorer
// ---------------------------------------------------------------------------

// FileNode represents a single node (file or directory) in the file tree.
type FileNode struct {
	Name       string
	Path       string
	IsDir      bool
	Depth      int
	Expanded   bool
	HasChanges bool
	Status     types.FileStatus
	Children   []*FileNode
}

// FileTree manages the file tree state for the right panel.
type FileTree struct {
	rootPath string
	gitOps   git.GitOperations

	nodes    []*FileNode // flattened visible nodes
	cursor   int
	expanded map[string]bool // track expanded directories by abs path
	loaded   bool

	// Cached git status for marking files
	fileStatuses map[string]types.FileStatus
}

// NewFileTree creates a new FileTree for the given repository path.
func NewFileTree(rootPath string, gitOps git.GitOperations) *FileTree {
	return &FileTree{
		rootPath:     rootPath,
		gitOps:       gitOps,
		nodes:        make([]*FileNode, 0),
		expanded:     make(map[string]bool),
		fileStatuses: make(map[string]types.FileStatus),
	}
}

// Load builds the file tree from the filesystem and git status.
func (ft *FileTree) Load() error {
	ft.fileStatuses = make(map[string]types.FileStatus)

	// Get git status for file markers
	changes, err := ft.gitOps.Status(context.TODO())
	if err == nil {
		for _, fc := range changes {
			ft.fileStatuses[fc.Path] = fc.Status
		}
	}

	// Build tree from root path
	rootNode, err := ft.buildTree(ft.rootPath, 0)
	if err != nil {
		return fmt.Errorf("build file tree: %w", err)
	}

	// Flatten for display
	ft.nodes = make([]*FileNode, 0)
	ft.flattenNode(rootNode, 0)
	ft.loaded = true

	if ft.cursor >= len(ft.nodes) {
		ft.cursor = 0
	}

	return nil
}

// buildTree recursively walks the directory and builds FileNode entries.
func (ft *FileTree) buildTree(dirPath string, depth int) (*FileNode, error) {
	info, err := os.Stat(dirPath)
	if err != nil {
		return nil, err
	}

	relPath, _ := filepath.Rel(ft.rootPath, dirPath)
	if relPath == "." {
		relPath = ""
	}

	name := info.Name()
	if relPath == "" {
		name = filepath.Base(ft.rootPath)
	}

	node := &FileNode{
		Name:     name,
		Path:     relPath,
		IsDir:    info.IsDir(),
		Depth:    depth,
		Expanded: ft.expanded[dirPath],
	}

	if !info.IsDir() {
		// File - check git status
		if relPath != "" {
			if status, ok := ft.fileStatuses[relPath]; ok {
				node.HasChanges = true
				node.Status = status
			}
		}
		return node, nil
	}

	// Directory - read children
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return node, nil // skip inaccessible dirs
	}

	// Sort: dirs first, then alphabetical
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].IsDir() != entries[j].IsDir() {
			return entries[i].IsDir()
		}
		return strings.ToLower(entries[i].Name()) < strings.ToLower(entries[j].Name())
	})

	for _, entry := range entries {
		// Skip hidden files/dirs (starting with .)
		if strings.HasPrefix(entry.Name(), ".") && entry.Name() != "." {
			continue
		}
		// Skip common dirs
		if entry.IsDir() {
			switch entry.Name() {
			case "node_modules", "vendor", "target", ".git", "__pycache__", ".cache":
				continue
			}
		}

		childPath := filepath.Join(dirPath, entry.Name())
		child, err := ft.buildTree(childPath, depth+1)
		if err != nil {
			continue
		}
		// If directory has changes, propagate up
		if child.HasChanges {
			node.HasChanges = true
		}
		node.Children = append(node.Children, child)
	}

	return node, nil
}

// flattenNode flattens the tree into a visible slice based on expand state.
func (ft *FileTree) flattenNode(node *FileNode, depth int) {
	// Create the display node
	displayNode := &FileNode{
		Name:       node.Name,
		Path:       node.Path,
		IsDir:      node.IsDir,
		Depth:      node.Depth,
		Expanded:   node.Expanded,
		HasChanges: node.HasChanges,
		Status:     node.Status,
	}
	ft.nodes = append(ft.nodes, displayNode)

	// If directory is expanded, add children
	if node.IsDir && node.Expanded {
		for _, child := range node.Children {
			ft.flattenNode(child, depth+1)
		}
	}
}

// ToggleExpand toggles the expanded state of the directory at the cursor.
func (ft *FileTree) ToggleExpand() {
	if ft.cursor < 0 || ft.cursor >= len(ft.nodes) {
		return
	}
	node := ft.nodes[ft.cursor]
	if !node.IsDir {
		return
	}

	absPath := filepath.Join(ft.rootPath, node.Path)
	ft.expanded[absPath] = !ft.expanded[absPath]

	// Reload the tree
	_ = ft.Load()
}

// CursorUp moves the cursor up.
func (ft *FileTree) CursorUp() {
	if ft.cursor > 0 {
		ft.cursor--
	}
}

// CursorDown moves the cursor down.
func (ft *FileTree) CursorDown() {
	if ft.cursor < len(ft.nodes)-1 {
		ft.cursor++
	}
}

// Selected returns the currently selected FileNode, or nil.
func (ft *FileTree) Selected() *FileNode {
	if ft.cursor < 0 || ft.cursor >= len(ft.nodes) {
		return nil
	}
	return ft.nodes[ft.cursor]
}

// Render renders the file tree into a string.
func (ft *FileTree) Render(width, height int, th *theme.Theme) string {
	var b strings.Builder

	if !ft.loaded || len(ft.nodes) == 0 {
		return th.BaseMuted.Render("  (t: carregar árvore)")
	}

	// Header
	b.WriteString(th.Accented.Render(" Files "))
	b.WriteString("\n")

	// Separator
	b.WriteString(th.BaseMuted.Render(strings.Repeat("─", width)))
	b.WriteString("\n")

	// Calculate visible range
	maxNodes := height - 4
	if maxNodes > len(ft.nodes) {
		maxNodes = len(ft.nodes)
	}

	start := 0
	if ft.cursor >= maxNodes {
		start = ft.cursor - maxNodes + 1
	}
	end := start + maxNodes
	if end > len(ft.nodes) {
		end = len(ft.nodes)
	}

	for i := start; i < end; i++ {
		node := ft.nodes[i]

		// Indentation
		indent := strings.Repeat("  ", node.Depth)

		// Icon
		icon := " "
		if node.IsDir {
			if node.Expanded {
				icon = "▼"
			} else {
				icon = "▶"
			}
		}

		// Status indicator
		statusStr := ""
		statusStyle := th.Base
		if node.HasChanges {
			switch node.Status {
			case types.FileStatusModified:
				statusStr = th.BadgeModified.Render("M")
				statusStyle = th.BadgeModified
			case types.FileStatusAdded:
				statusStr = th.BadgeAdded.Render("A")
				statusStyle = th.BadgeAdded
			case types.FileStatusDeleted:
				statusStr = th.BadgeDeleted.Render("D")
				statusStyle = th.BadgeDeleted
			case types.FileStatusUntracked:
				statusStr = th.BaseMuted.Render("?")
				statusStyle = th.BaseMuted
			default:
				// Propagated from children
				statusStr = th.BaseMuted.Render("~")
			}
		} else if node.IsDir {
			statusStr = " "
		} else {
			statusStr = " "
		}

		// Cursor
		cursor := " "
		style := statusStyle
		if i == ft.cursor {
			cursor = th.Accented.Render("▸")
			style = th.Selected
		}

		// Truncate name if too long
		name := node.Name
		avail := width - node.Depth*2 - 5
		if avail < 5 {
			avail = 5
		}
		if len(name) > avail {
			name = name[:avail-1] + "…"
		}

		line := fmt.Sprintf(" %s %s %s%s%s",
			cursor,
			statusStr,
			indent,
			icon,
			style.Render(" "+name),
		)
		b.WriteString(line)
		if i < end-1 {
			b.WriteString("\n")
		}
	}

	// Scroll indicator
	if len(ft.nodes) > maxNodes {
		b.WriteString("\n")
		b.WriteString(th.Dim.Render(
			fmt.Sprintf("  ↓ %d/%d  ↑↓: nav  ↵: expandir", end, len(ft.nodes))))
	}

	return b.String()
}

// Init loads the initial tree.
func (ft *FileTree) Init() error {
	return ft.Load()
}

// SetGitOps updates the GitOperations reference.
func (ft *FileTree) SetGitOps(gitOps git.GitOperations) {
	ft.gitOps = gitOps
}
