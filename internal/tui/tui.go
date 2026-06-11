package tui

import (
	"context"
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	gitlocal "github.com/nicoddemus/github-desktop-tui/internal/git"
	"github.com/nicoddemus/github-desktop-tui/internal/store"
	"github.com/nicoddemus/github-desktop-tui/pkg/types"
)

// focusArea identifies which panel has focus.
type focusArea int

const (
	focusSidebar focusArea = iota
	focusHistory
	focusDetails
)

// model is the root Bubble Tea model.
type model struct {
	width, height int
	focus         focusArea
	ready         bool

	gitOps  gitlocal.GitOperations
	store   *store.Store
	repos   []*store.TrackedRepo
	repoIdx int

	branches []*types.Branch
	commits  []*types.Commit
	graph    []*types.GraphRow
	changes  []*types.FileChange
	selFile  int

	detailCommit int // index of selected commit in history
}

// New creates a new TUI model.
func New(gitOps gitlocal.GitOperations, st *store.Store) tea.Model {
	return newModel(gitOps, st)
}

func newModel(gitOps gitlocal.GitOperations, st *store.Store) model {
	return model{
		gitOps:       gitOps,
		store:        st,
		repos:        make([]*store.TrackedRepo, 0),
		branches:     make([]*types.Branch, 0),
		commits:      make([]*types.Commit, 0),
		graph:        make([]*types.GraphRow, 0),
		changes:      make([]*types.FileChange, 0),
		detailCommit: -1,
		selFile:      -1,
		focus:        focusHistory,
	}
}

// ── Messages ──

type dataLoadedMsg struct {
	branches []*types.Branch
	commits  []*types.Commit
	graph    []*types.GraphRow
	changes  []*types.FileChange
}

type errMsg struct{ err error }

// ── Init ──

func (m model) Init() tea.Cmd {
	return tea.Batch(
		tea.EnterAltScreen,
		m.loadData(),
	)
}

func (m model) loadData() tea.Cmd {
	return func() tea.Msg {
		ctx := context.TODO()
		branches, _ := m.gitOps.Branches(ctx)
		commits, _ := m.gitOps.Log(ctx, &gitlocal.LogOptions{Limit: 100})
		graph, _ := m.gitOps.GraphLog(ctx, &gitlocal.LogOptions{Limit: 100})
		changes, _ := m.gitOps.Status(ctx)
		return dataLoadedMsg{branches, commits, graph, changes}
	}
}

// ── Update ──

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.ready = true
		return m, nil

	case dataLoadedMsg:
		m.branches = msg.branches
		m.commits = msg.commits
		m.graph = msg.graph
		m.changes = msg.changes
		if len(m.commits) > 0 && m.detailCommit < 0 {
			m.detailCommit = 0
		}
		return m, nil

	case errMsg:
		return m, nil

	case tea.KeyMsg:
		return m.handleKeyMsg(msg)
	}

	return m, nil
}

// ── Key handling ──

func (m model) handleKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit

	case "tab":
		m.focus = (m.focus + 1) % 3
		return m, nil

	case "shift+tab":
		m.focus = (m.focus - 1 + 3) % 3
		return m, nil

	case "r":
		return m, m.loadData()

	// Navigation
	case "up", "k":
		return m.navigateUp()
	case "down", "j":
		return m.navigateDown()

	// Git operations
	case "c":
		return m, m.doCommit()
	case "p":
		return m, m.doPush()
	case "l":
		return m, m.doPull()
	case "d":
		return m, m.doDiff()
	}

	return m, nil
}

func (m model) navigateUp() (tea.Model, tea.Cmd) {
	switch m.focus {
	case focusHistory:
		if m.detailCommit > 0 {
			m.detailCommit--
		}
	case focusSidebar:
		if m.repoIdx > 0 {
			m.repoIdx--
		}
	}
	return m, nil
}

func (m model) navigateDown() (tea.Model, tea.Cmd) {
	switch m.focus {
	case focusHistory:
		if m.detailCommit < len(m.commits)-1 {
			m.detailCommit++
		}
	case focusSidebar:
		if m.repoIdx < len(m.repos)-1 {
			m.repoIdx++
		}
	}
	return m, nil
}

func (m model) doCommit() tea.Cmd {
	return func() tea.Msg {
		ctx := context.TODO()
		// Stage all tracked changes
		for _, fc := range m.changes {
			if !fc.Staged {
				if err := m.gitOps.Stage(ctx, fc.Path); err != nil {
					return errMsg{err}
				}
			}
		}
		_, err := m.gitOps.Commit(ctx, "feat: WIP")
		if err != nil {
			return errMsg{err}
		}
		// Reload
		return m.loadData()()
	}
}

func (m model) doPush() tea.Cmd {
	return func() tea.Msg {
		ctx := context.TODO()
		branch := ""
		for _, b := range m.branches {
			if b.IsActive {
				branch = b.Name
				break
			}
		}
		if branch == "" {
			branch = "main"
		}
		if err := m.gitOps.Push(ctx, "origin", branch, false); err != nil {
			return errMsg{err}
		}
		return m.loadData()()
	}
}

func (m model) doPull() tea.Cmd {
	return func() tea.Msg {
		ctx := context.TODO()
		if err := m.gitOps.Pull(ctx, "origin", ""); err != nil {
			return errMsg{err}
		}
		return m.loadData()()
	}
}

func (m model) doDiff() tea.Cmd {
	return func() tea.Msg {
		// Dummy for now
		return nil
	}
}

// ── View ──

func (m model) View() string {
	if !m.ready {
		return lipgloss.NewStyle().Foreground(lipgloss.Color("#8B949E")).Render("Loading...")
	}

	// Layout dimensions
	sidebarW := max(26, m.width/5)
	detailsW := max(32, m.width/4)
	centerW := m.width - sidebarW - detailsW - 4

	// Panel styles
	active := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#7AA2F7")).
		Padding(0, 1).
		Width(m.width)

	inactive := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#3B4252")).
		Padding(0, 1).
		Width(m.width)

	// With fixed height, panels fill the available space
	panelH := m.height - 3 // header + 1 padding

	buildPanel := func(w int, isActive bool, content string) string {
		s := inactive.Width(w).Height(panelH)
		if isActive {
			s = active.Width(w).Height(panelH)
		}
		return s.Render(content)
	}

	// Build content
	leftContent := renderSidebar(m)
	centerContent := renderHistory(m)
	rightContent := renderDetails(m)

	// Header bar
	header := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#C5C8C6")).
		Background(lipgloss.Color("#1F2430")).
		Padding(0, 2).
		Width(m.width).
		Render(fmt.Sprintf(" git-tui  •  tab:focus  ↑↓:nav  c:commit  p:push  l:pull  d:diff  r:refresh  q:quit "))

	// Assemble panels
	left := buildPanel(sidebarW, m.focus == focusSidebar, leftContent)
	center := buildPanel(centerW, m.focus == focusHistory, centerContent)
	right := buildPanel(detailsW, m.focus == focusDetails, rightContent)

	body := lipgloss.JoinHorizontal(lipgloss.Top, left, center, right)

	return lipgloss.JoinVertical(lipgloss.Left, header, body)
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// ── Panel renderers ──

func renderSidebar(m model) string {
	var b strings.Builder

	b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#7AA2F7")).Bold(true).Render(" Branches "))
	b.WriteString("\n\n")

	for _, br := range m.branches {
		if br.IsActive {
			mark := "●"
			style := lipgloss.NewStyle().Foreground(lipgloss.Color("#7AA2F7")).Bold(true)
			ab := ""
			if br.Ahead > 0 || br.Behind > 0 {
				ab = fmt.Sprintf("  ↑%d↓%d", br.Ahead, br.Behind)
			}
			b.WriteString(fmt.Sprintf("  %s  %s%s\n", style.Render(mark), style.Render(br.Name), lipgloss.NewStyle().Foreground(lipgloss.Color("#565F89")).Render(ab)))
		}
	}

	for _, br := range m.branches {
		if !br.IsActive {
			mark := "○"
			style := lipgloss.NewStyle().Foreground(lipgloss.Color("#8B949E"))
			b.WriteString(fmt.Sprintf("  %s  %s\n", style.Render(mark), style.Render(br.Name)))
		}
	}

	b.WriteString("\n")
	b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#7AA2F7")).Bold(true).Render(" Changes "))
	b.WriteString("\n\n")

	if len(m.changes) == 0 {
		b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#565F89")).Render("  Clean working tree"))
	} else {
		staged := 0
		for _, fc := range m.changes {
			if fc.Staged {
				staged++
			}
		}
		b.WriteString(fmt.Sprintf("  %s\n", lipgloss.NewStyle().Foreground(lipgloss.Color("#8B949E")).Render(fmt.Sprintf("%d files (%d staged)", len(m.changes), staged))))
		for _, fc := range m.changes {
			badge := " "
			badgeStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#565F89"))
			switch fc.Status {
			case types.FileStatusModified:
				badge = "M"
				badgeStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#E5C07B"))
			case types.FileStatusAdded:
				badge = "A"
				badgeStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#98C379"))
			case types.FileStatusDeleted:
				badge = "D"
				badgeStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#E06C75"))
			case types.FileStatusUntracked:
				badge = "?"
				badgeStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#565F89"))
			}
			b.WriteString(fmt.Sprintf("  %s %s\n", badgeStyle.Render(badge), lipgloss.NewStyle().Foreground(lipgloss.Color("#C5C8C6")).Render(fc.Path)))
		}
	}

	return b.String()
}

func renderHistory(m model) string {
	var b strings.Builder

	// Branch header
	activeBranch := "main"
	for _, br := range m.branches {
		if br.IsActive {
			activeBranch = br.Name
			break
		}
	}
	b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#7AA2F7")).Bold(true).Render(" " + activeBranch + " "))
	b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#565F89")).Render(fmt.Sprintf("%d commits", len(m.commits))))
	b.WriteString("\n\n")

	if len(m.graph) == 0 {
		b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#565F89")).Render("  No commits"))
		return b.String()
	}

	// Graph colors per column
	graphColors := []string{"#7AA2F7", "#98C379", "#E5C07B", "#C678DD", "#56B6C2", "#D19A66", "#E06C75"}
	colColor := make(map[int]string)
	colIdx := 0

	// Show up to panel height - 3 lines
	maxRows := m.height - 6
	commitIdx := 0
	start := 0
	if m.detailCommit >= maxRows {
		start = countGraphUpTo(m.graph, m.detailCommit)
	}
	end := start + maxRows
	if end > len(m.graph) {
		end = len(m.graph)
	}

	for i := start; i < end; i++ {
		row := m.graph[i]

		// Graph part
		graphStr := ""
		for j, ch := range row.Graph {
			if ch == '*' {
				if _, ok := colColor[j]; !ok {
					colColor[j] = graphColors[colIdx%len(graphColors)]
					colIdx++
				}
				dot := lipgloss.NewStyle().Foreground(lipgloss.Color(colColor[j])).Bold(true).Render("●")
				graphStr += dot
			} else if ch == '|' || ch == '/' || ch == '\\' || ch == '_' {
				c := lipgloss.Color("#565F89")
				if color, ok := colColor[j]; ok {
					c = lipgloss.Color(color)
				}
				graphStr += lipgloss.NewStyle().Foreground(c).Render(string(ch))
			} else if ch == ' ' {
				graphStr += " "
			} else {
				graphStr += string(ch)
			}
		}

		// Pad graph to consistent width
		for len(graphStr) < 8 {
			graphStr += " "
		}

		if row.IsCommit {
			// Build commit line
			line := graphStr
			hash := lipgloss.NewStyle().Foreground(lipgloss.Color("#565F89")).Render(row.Hash)
			msg := row.Message
			avail := m.width/5*3 - len(graphStr) - 30
			if avail < 5 {
				avail = 5
			}
			if len(msg) > avail {
				msg = msg[:avail-1] + "…"
			}
			author := lipgloss.NewStyle().Foreground(lipgloss.Color("#565F89")).Render(row.Author)
			time := lipgloss.NewStyle().Foreground(lipgloss.Color("#565F89")).Render(row.Time)
			line += fmt.Sprintf(" %s %s  %s  %s", hash, msg, author, time)

			// Selected highlight
			if commitIdx == m.detailCommit {
				b.WriteString(lipgloss.NewStyle().Background(lipgloss.Color("#2C3E50")).Render(line))
			} else {
				b.WriteString(line)
			}
			commitIdx++
		} else {
			// Continuation line (just graph)
			b.WriteString(graphStr)
		}

		b.WriteString("\n")
	}

	// Scroll hint
	if len(m.graph) > maxRows {
		b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#565F89")).Render(fmt.Sprintf("  ↓ %d more", len(m.graph)-end)))
	}

	return b.String()
}

func renderDetails(m model) string {
	var b strings.Builder

	if m.detailCommit < 0 || m.detailCommit >= len(m.commits) {
		b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#565F89")).Render(" No commit selected"))
		return b.String()
	}

	c := m.commits[m.detailCommit]
	label := lipgloss.NewStyle().Foreground(lipgloss.Color("#565F89"))
	value := lipgloss.NewStyle().Foreground(lipgloss.Color("#C5C8C6"))

	b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#7AA2F7")).Bold(true).Render(" Commit "))
	b.WriteString("\n\n")
	b.WriteString(fmt.Sprintf("%s  %s\n", label.Render("Hash    "), value.Render(c.Hash[:12])))
	b.WriteString(fmt.Sprintf("%s  %s\n", label.Render("Author  "), value.Render(c.Author)))
	b.WriteString(fmt.Sprintf("%s  %s\n", label.Render("Date    "), value.Render(c.Timestamp.Format("02 Jan 2006 15:04"))))
	b.WriteString(fmt.Sprintf("%s  %s\n", label.Render("Message "), value.Render(c.MessageHead)))

	// Changed files
	b.WriteString("\n")
	b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#7AA2F7")).Bold(true).Render(" Files "))
	b.WriteString("\n\n")

	if len(m.changes) == 0 {
		b.WriteString(label.Render("  No files changed"))
	} else {
		for _, fc := range m.changes {
			badge := " "
			badgeSty := lipgloss.NewStyle().Foreground(lipgloss.Color("#565F89"))
			switch fc.Status {
			case types.FileStatusModified:
				badge = "M"
				badgeSty = lipgloss.NewStyle().Foreground(lipgloss.Color("#E5C07B"))
			case types.FileStatusAdded:
				badge = "A"
				badgeSty = lipgloss.NewStyle().Foreground(lipgloss.Color("#98C379"))
			case types.FileStatusDeleted:
				badge = "D"
				badgeSty = lipgloss.NewStyle().Foreground(lipgloss.Color("#E06C75"))
			case types.FileStatusUntracked:
				badge = "?"
				badgeSty = lipgloss.NewStyle().Foreground(lipgloss.Color("#565F89"))
			}
			b.WriteString(fmt.Sprintf("  %s  %s\n", badgeSty.Render(badge), value.Render(fc.Path)))
		}
	}

	return b.String()
}

func countGraphUpTo(rows []*types.GraphRow, target int) int {
	count := 0
	for i, row := range rows {
		if row.IsCommit {
			if count >= target {
				return i
			}
			count++
		}
	}
	return 0
}
