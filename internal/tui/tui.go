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

// ── GitHub Dark Mode Colors ──
var (
	ghBg       = lipgloss.Color("#0d1117")
	ghSurface  = lipgloss.Color("#161b22")
	ghBorder   = lipgloss.Color("#30363d")
	ghBorderHi = lipgloss.Color("#58a6ff") // active border
	ghText     = lipgloss.Color("#e6edf3")
	ghMuted    = lipgloss.Color("#8b949e")
	ghDim      = lipgloss.Color("#484f58")
	ghBlue     = lipgloss.Color("#58a6ff")
	ghGreen    = lipgloss.Color("#3fb950")
	ghRed      = lipgloss.Color("#f85149")
	ghYellow   = lipgloss.Color("#d29922")
	ghOrange   = lipgloss.Color("#db6d28")
	ghPurple   = lipgloss.Color("#bc8cff")
	ghCyan     = lipgloss.Color("#39d2c0")
	ghSelBg    = lipgloss.Color("#1f2d47")
	ghHeaderBg = lipgloss.Color("#161b22")
)

// Graph colors for commit dots (GitHub-inspired palette)
var graphDotColors = []string{
	"#58a6ff", "#3fb950", "#d29922", "#bc8cff",
	"#39d2c0", "#db6d28", "#f85149", "#58a6ff",
}

// ── Focus area ──
type focusArea int

const (
	focusSidebar focusArea = iota
	focusHistory
	focusDetails
)

// ── Model ──
type model struct {
	width, height int
	focus         focusArea
	ready         bool

	gitOps gitlocal.GitOperations
	store  *store.Store

	branches      []*types.Branch
	commits       []*types.Commit
	graph         []*types.GraphRow
	changes       []*types.FileChange // working tree changes
	commitFiles   []*types.FileChange // files from selected commit
	detailCommit  int
	notification  string
}

func New(gitOps gitlocal.GitOperations, st *store.Store) tea.Model {
	return &model{
		gitOps:      gitOps,
		store:       st,
		branches:    []*types.Branch{},
		commits:     []*types.Commit{},
		graph:       []*types.GraphRow{},
		changes:     []*types.FileChange{},
		commitFiles: []*types.FileChange{},
		detailCommit: 0,
		focus:       focusHistory,
	}
}

// ── Messages ──
type dataLoadedMsg struct {
	branches []*types.Branch
	commits  []*types.Commit
	graph    []*types.GraphRow
	changes  []*types.FileChange
	err      error
}

type notifMsg struct{ text string }
type commitFilesMsg struct {
	files []*types.FileChange
	err   error
}

// ── Init ──
func (m *model) Init() tea.Cmd {
	return tea.Batch(
		tea.EnterAltScreen,
		m.loadData(),
	)
}

func (m *model) loadData() tea.Cmd {
	return func() tea.Msg {
		ctx := context.TODO()

		branches, berr := m.gitOps.Branches(ctx)
		if berr != nil {
			return dataLoadedMsg{err: fmt.Errorf("branches: %w", berr)}
		}

		commits, cerr := m.gitOps.Log(ctx, &gitlocal.LogOptions{Limit: 100})
		if cerr != nil {
			return dataLoadedMsg{err: fmt.Errorf("commits: %w", cerr)}
		}

		graph, gerr := m.gitOps.GraphLog(ctx, &gitlocal.LogOptions{Limit: 100})
		if gerr != nil {
			return dataLoadedMsg{err: fmt.Errorf("graph: %w", gerr)}
		}

		changes, serr := m.gitOps.Status(ctx)
		if serr != nil {
			// Non-fatal: changes might be empty
			changes = []*types.FileChange{}
		}

		return dataLoadedMsg{
			branches: branches,
			commits:  commits,
			graph:    graph,
			changes:  changes,
		}
	}
}

// ── Update ──
func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.ready = true
		return m, nil

	case dataLoadedMsg:
		if msg.err != nil {
			m.notification = fmt.Sprintf("Error: %v", msg.err)
			return m, nil
		}
		m.branches = msg.branches
		m.commits = msg.commits
		m.graph = msg.graph
		m.changes = msg.changes
		if m.detailCommit >= len(m.commits) {
			m.detailCommit = 0
		}
		// Load files for selected commit
		cmds := []tea.Cmd{}
		if len(m.commits) > 0 {
			cmds = append(cmds, m.loadCommitFiles())
		}
		return m, tea.Batch(cmds...)

	case notifMsg:
		m.notification = msg.text
		return m, nil

	case commitFilesMsg:
		if msg.err != nil {
			m.notification = fmt.Sprintf("Files error: %v", msg.err)
		}
		m.commitFiles = msg.files
		return m, nil

	case tea.KeyMsg:
		return m.handleKey(msg)
	}

	return m, nil
}

// ── Key handling ──
func (m *model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit

	case "tab":
		m.focus = (m.focus + 1) % 3
		return m, nil
	case "shift+tab":
		m.focus = (m.focus - 1 + 3) % 3
		return m, nil

	case "up", "k":
		return m.navUp()
	case "down", "j":
		return m.navDown()

	case "r":
		m.notification = ""
		return m, m.loadData()
	case "c":
		return m, m.doCommit()
	case "p":
		return m, m.doPush()
	case "l":
		return m, m.doPull()
	case "d":
		if m.detailCommit < len(m.commits) {
			return m, m.doShowDiff()
		}
	}

	return m, nil
}

func (m *model) navUp() (tea.Model, tea.Cmd) {
	switch m.focus {
	case focusHistory:
		if m.detailCommit > 0 {
			m.detailCommit--
			return m, m.loadCommitFiles()
		}
	case focusSidebar:
	}
	return m, nil
}

func (m *model) navDown() (tea.Model, tea.Cmd) {
	switch m.focus {
	case focusHistory:
		if m.detailCommit < len(m.commits)-1 {
			m.detailCommit++
			return m, m.loadCommitFiles()
		}
	}
	return m, nil
}

func (m *model) loadCommitFiles() tea.Cmd {
	return func() tea.Msg {
		if m.detailCommit < 0 || m.detailCommit >= len(m.commits) {
			return commitFilesMsg{files: []*types.FileChange{}}
		}
		hash := m.commits[m.detailCommit].Hash
		files, err := m.gitOps.GetCommitFiles(context.TODO(), hash)
		if err != nil {
			return commitFilesMsg{files: []*types.FileChange{}, err: err}
		}
		return commitFilesMsg{files: files}
	}
}

func (m *model) doCommit() tea.Cmd {
	return func() tea.Msg {
		ctx := context.TODO()
		for _, fc := range m.changes {
			if !fc.Staged {
				if err := m.gitOps.Stage(ctx, fc.Path); err != nil {
					return notifMsg{fmt.Sprintf("Stage error: %v", err)}
				}
			}
		}
		if _, err := m.gitOps.Commit(ctx, "feat: WIP"); err != nil {
			return notifMsg{fmt.Sprintf("Commit error: %v", err)}
		}
		return m.loadData()()
	}
}

func (m *model) doPush() tea.Cmd {
	return func() tea.Msg {
		ctx := context.TODO()
		branch := "main"
		for _, b := range m.branches {
			if b.IsActive {
				branch = b.Name
				break
			}
		}
		if err := m.gitOps.Push(ctx, "origin", branch, false); err != nil {
			return notifMsg{fmt.Sprintf("Push error: %v", err)}
		}
		return m.loadData()()
	}
}

func (m *model) doPull() tea.Cmd {
	return func() tea.Msg {
		ctx := context.TODO()
		if err := m.gitOps.Pull(ctx, "origin", ""); err != nil {
			return notifMsg{fmt.Sprintf("Pull error: %v", err)}
		}
		return m.loadData()()
	}
}

func (m *model) doShowDiff() tea.Cmd {
	return func() tea.Msg {
		// TODO: implement diff view
		return notifMsg{"Diff view: press any key to close (not yet implemented)"}
	}
}

// ── View ──
func (m *model) View() string {
	if !m.ready {
		return lipgloss.NewStyle().Foreground(ghMuted).Background(ghBg).Render(" Loading...")
	}

	// Layout
	sidebarW := max(26, m.width/5)
	detailsW := max(32, m.width/4)
	centerW := m.width - sidebarW - detailsW - 4
	panelH := m.height - 3

	// ── Panel builder ──
	baseStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ghBorder).
		Padding(0, 1).
		Background(ghBg)

	activeStyle := baseStyle.BorderForeground(ghBorderHi)

	makePanel := func(w int, active bool, content string) string {
		s := baseStyle.Width(w).Height(panelH)
		if active {
			s = activeStyle.Width(w).Height(panelH)
		}
		return s.Render(content)
	}

	// ── Header ──
	header := lipgloss.NewStyle().
		Foreground(ghMuted).
		Background(ghHeaderBg).
		Padding(0, 2).
		Width(m.width).
		Render(" git-tui  ◆  tab:focus  ↑↓:nav  c:commit  p:push  l:pull  d:diff  r:refresh  q:quit ")

	// ── Panels ──
	left := makePanel(sidebarW, m.focus == focusSidebar, renderSidebar(m))
	center := makePanel(centerW, m.focus == focusHistory, renderHistory(m))
	right := makePanel(detailsW, m.focus == focusDetails, renderDetails(m))

	body := lipgloss.JoinHorizontal(lipgloss.Top, left, center, right)

	// ── Notification bar ──
	if m.notification != "" {
		notifStyle := lipgloss.NewStyle().
			Foreground(ghRed).
			Background(ghBg).
			Padding(0, 2).
			Width(m.width)
		return lipgloss.JoinVertical(lipgloss.Left, header, body,
			notifStyle.Render(" ⚠  "+m.notification))
	}

	return lipgloss.JoinVertical(lipgloss.Left, header, body)
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// ══════════════════════════════════════════════════════════════════════
//  SIDEBAR
// ══════════════════════════════════════════════════════════════════════

func renderSidebar(m *model) string {
	var b strings.Builder

	// ── Branch list ──
	b.WriteString(sectionColor.Render(" Branches "))
	b.WriteString("\n\n")

	for _, br := range m.branches {
		if br.IsActive {
			ab := ""
			if br.Ahead > 0 || br.Behind > 0 {
				ab = dimStyle.Render(fmt.Sprintf(" ↑%d↓%d", br.Ahead, br.Behind))
			}
			b.WriteString(fmt.Sprintf("  %s  %s%s\n",
				activeDot.Render("●"),
				branchActive.Render(br.Name),
				ab))
		}
	}

	for _, br := range m.branches {
		if !br.IsActive {
			b.WriteString(fmt.Sprintf("  %s  %s\n",
				mutedStyle.Render("○"),
				mutedStyle.Render(br.Name)))
		}
	}

	// ── Changes ──
	b.WriteString("\n")
	b.WriteString(sectionColor.Render(" Changes "))
	b.WriteString("\n\n")

	if len(m.changes) == 0 {
		b.WriteString(mutedStyle.Render("  Clean working tree"))
	} else {
		staged := 0
		for _, fc := range m.changes {
			if fc.Staged {
				staged++
			}
		}
		summary := fmt.Sprintf("  %d file(s) (%d staged)", len(m.changes), staged)
		b.WriteString(dimStyle.Render(summary))
		b.WriteString("\n")

		for _, fc := range m.changes {
			badge, style := changeBadge(fc.Status)
			b.WriteString(fmt.Sprintf("  %s %s\n", style.Render(badge), textStyle.Render(fc.Path)))
		}
	}

	return b.String()
}

// ══════════════════════════════════════════════════════════════════════
//  HISTORY (COMMIT GRAPH)
// ══════════════════════════════════════════════════════════════════════

func renderHistory(m *model) string {
	var b strings.Builder

	// Branch header
	activeBranch := "main"
	for _, br := range m.branches {
		if br.IsActive {
			activeBranch = br.Name
			break
		}
	}
	b.WriteString(branchActive.Render(" "+activeBranch+" "))
	b.WriteString(dimStyle.Render(fmt.Sprintf("%d commits", len(m.commits))))
	b.WriteString("\n\n")

	if len(m.graph) == 0 {
		b.WriteString(mutedStyle.Render("  No commits found"))
		return b.String()
	}

	// Track column colors
	colColor := make(map[int]string)
	colIdx := 0

	maxRows := m.height - 6
	if maxRows < 3 {
		maxRows = 3
	}

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

		// Render graph
		graphStr := renderGraphLine(row.Graph, colColor, &colIdx)

		// Pad graph width
		for len(graphStr) < 6 {
			graphStr += " "
		}

		if row.IsCommit {
			msg := row.Message
			avail := m.width/3 - len(graphStr) - 30
			if avail < 5 {
				avail = 5
			}
			if len(msg) > avail {
				msg = msg[:avail-1] + "…"
			}
			line := fmt.Sprintf("%s %s %s  %s  %s",
				graphStr,
				hashStyle.Render(row.Hash),
				textStyle.Render(msg),
				mutedStyle.Render(row.Author),
				mutedStyle.Render(row.Time))

			if commitIdx == m.detailCommit {
				b.WriteString(selBgStyle.Render(line))
			} else {
				b.WriteString(line)
			}
			commitIdx++
		} else {
			b.WriteString(graphStr)
		}
		b.WriteString("\n")
	}

	if len(m.graph) > maxRows {
		b.WriteString(dimStyle.Render(fmt.Sprintf("  ↓ %d more", len(m.graph)-end)))
	}

	return b.String()
}

func renderGraphLine(graph string, colColor map[int]string, colIdx *int) string {
	var out strings.Builder
	for j, ch := range graph {
		switch {
		case ch == '*':
			if _, ok := colColor[j]; !ok {
				colColor[j] = graphDotColors[*colIdx%len(graphDotColors)]
				*colIdx++
			}
			out.WriteString(lipgloss.NewStyle().
				Foreground(lipgloss.Color(colColor[j])).
				Bold(true).
				Render("●"))
		case ch == '|' || ch == '/' || ch == '\\' || ch == '_':
			c := ghDim
			if color, ok := colColor[j]; ok {
				c = lipgloss.Color(color)
			}
			out.WriteString(lipgloss.NewStyle().Foreground(c).Render(string(ch)))
		default:
			out.WriteString(string(ch))
		}
	}
	return out.String()
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

// ══════════════════════════════════════════════════════════════════════
//  DETAILS
// ══════════════════════════════════════════════════════════════════════

func renderDetails(m *model) string {
	var b strings.Builder

	if m.detailCommit < 0 || m.detailCommit >= len(m.commits) {
		b.WriteString(mutedStyle.Render(" No commit selected"))
		return b.String()
	}

	c := m.commits[m.detailCommit]

	b.WriteString(sectionColor.Render(" Commit "))
	b.WriteString("\n\n")
	b.WriteString(fmt.Sprintf("%s  %s\n", labelStyle.Render("Hash    "), valueStyle.Render(c.Hash[:12])))
	b.WriteString(fmt.Sprintf("%s  %s\n", labelStyle.Render("Author  "), valueStyle.Render(c.Author)))
	b.WriteString(fmt.Sprintf("%s  %s\n", labelStyle.Render("Date    "), valueStyle.Render(c.Timestamp.Format("02 Jan 2006 15:04"))))
	b.WriteString(fmt.Sprintf("%s  %s\n", labelStyle.Render("Message "), valueStyle.Render(c.MessageHead)))

	// Changed files (from selected commit)
	b.WriteString("\n")
	b.WriteString(sectionColor.Render(" Files "))
	b.WriteString("\n\n")

	files := m.commitFiles
	if len(files) == 0 {
		b.WriteString(mutedStyle.Render("  No files changed"))
	} else {
		for _, fc := range files {
			badge, style := changeBadge(fc.Status)
			b.WriteString(fmt.Sprintf("  %s  %s\n", style.Render(badge), textStyle.Render(fc.Path)))
		}
	}

	return b.String()
}

// ══════════════════════════════════════════════════════════════════════
//  STYLE HELPERS
// ══════════════════════════════════════════════════════════════════════

var (
	sectionColor = lipgloss.NewStyle().Foreground(ghBlue).Bold(true)
	branchActive = lipgloss.NewStyle().Foreground(ghBlue).Bold(true)
	activeDot    = lipgloss.NewStyle().Foreground(ghBlue).Bold(true)
	textStyle    = lipgloss.NewStyle().Foreground(ghText)
	mutedStyle   = lipgloss.NewStyle().Foreground(ghMuted)
	dimStyle     = lipgloss.NewStyle().Foreground(ghDim)
	hashStyle    = lipgloss.NewStyle().Foreground(ghMuted)
	labelStyle   = lipgloss.NewStyle().Foreground(ghMuted)
	valueStyle   = lipgloss.NewStyle().Foreground(ghText)
	selBgStyle   = lipgloss.NewStyle().Background(ghSelBg)
)

func changeBadge(status types.FileStatus) (string, lipgloss.Style) {
	switch status {
	case types.FileStatusModified:
		return "M", lipgloss.NewStyle().Foreground(ghYellow).Bold(true)
	case types.FileStatusAdded:
		return "A", lipgloss.NewStyle().Foreground(ghGreen).Bold(true)
	case types.FileStatusDeleted:
		return "D", lipgloss.NewStyle().Foreground(ghRed).Bold(true)
	case types.FileStatusUntracked:
		return "?", lipgloss.NewStyle().Foreground(ghMuted)
	default:
		return " ", lipgloss.NewStyle().Foreground(ghDim)
	}
}
