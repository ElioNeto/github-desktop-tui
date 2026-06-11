package tui

import (
	"context"
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	gitlocal "github.com/ElioNeto/github-desktop-tui/internal/git"
	"github.com/ElioNeto/github-desktop-tui/internal/store"
	"github.com/ElioNeto/github-desktop-tui/pkg/types"
)

// ── Colors ──
var (
	purple    = lipgloss.Color("#a371f7")
	purpleDim = lipgloss.Color("#6e40c9")
	bg        = lipgloss.Color("#0d1117")
	surface   = lipgloss.Color("#161b22")
	surface2  = lipgloss.Color("#1c2333")
	border    = lipgloss.Color("#30363d")
	text      = lipgloss.Color("#e6edf3")
	muted     = lipgloss.Color("#8b949e")
	dim       = lipgloss.Color("#6e7681")
	green     = lipgloss.Color("#3fb950")
	red       = lipgloss.Color("#f85149")
	yellow    = lipgloss.Color("#d29922")
	blue      = lipgloss.Color("#58a6ff")
	selBg     = lipgloss.Color("#2a1a4a")
)

var graphColors = []string{"#a371f7", "#3fb950", "#d29922", "#58a6ff", "#f85149", "#56b6c2"}

// ── Tabs ──
type tabID int

const (
	tabGraph tabID = iota
	tabWorktree
	tabHistory
	tabCommit
	tabDiff
	tabCount
)

var tabNames = [tabCount]string{"GRAPH", "WORKTREE", "HISTORY", "COMMIT", "DIFF"}
var tabKeys = [tabCount]string{"1", "2", "3", "4", "5"}

// ── Model ──
type model struct {
	width, height int
	tab           tabID
	scroll        int
	ready         bool
	gitOps        gitlocal.GitOperations
	store         *store.Store

	branches     []*types.Branch
	commits      []*types.Commit
	graph        []*types.GraphRow
	changes      []*types.FileChange
	commitFiles  []*types.FileChange
	detailCommit int
	notification string
}

func New(gitOps gitlocal.GitOperations, st *store.Store) tea.Model {
	return &model{
		gitOps:       gitOps,
		store:        st,
		branches:     []*types.Branch{},
		commits:      []*types.Commit{},
		graph:        []*types.GraphRow{},
		changes:      []*types.FileChange{},
		commitFiles:  []*types.FileChange{},
		detailCommit: 0,
		tab:          tabGraph,
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
	return tea.Batch(tea.EnterAltScreen, m.loadData())
}

func (m *model) loadData() tea.Cmd {
	return func() tea.Msg {
		ctx := context.TODO()
		branches, e1 := m.gitOps.Branches(ctx)
		if e1 != nil {
			return dataLoadedMsg{err: fmt.Errorf("branches: %w", e1)}
		}
		commits, e2 := m.gitOps.Log(ctx, &gitlocal.LogOptions{Limit: 100})
		if e2 != nil {
			return dataLoadedMsg{err: fmt.Errorf("commits: %w", e2)}
		}
		graph, e3 := m.gitOps.GraphLog(ctx, &gitlocal.LogOptions{Limit: 100})
		if e3 != nil {
			return dataLoadedMsg{err: fmt.Errorf("graph: %w", e3)}
		}
		changes, _ := m.gitOps.Status(ctx)
		if changes == nil {
			changes = []*types.FileChange{}
		}
		return dataLoadedMsg{branches, commits, graph, changes, nil}
	}
}

func (m *model) loadCommitFiles() tea.Cmd {
	return func() tea.Msg {
		if m.detailCommit < 0 || m.detailCommit >= len(m.commits) {
			return commitFilesMsg{files: []*types.FileChange{}}
		}
		files, err := m.gitOps.GetCommitFiles(context.TODO(), m.commits[m.detailCommit].Hash)
		if err != nil {
			return commitFilesMsg{files: []*types.FileChange{}, err: err}
		}
		return commitFilesMsg{files: files}
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
		var cmds []tea.Cmd
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

// ── Keys ──
func (m *model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit
	case "1":
		m.tab = tabGraph; m.scroll = 0; return m, nil
	case "2":
		m.tab = tabWorktree; m.scroll = 0; return m, nil
	case "3":
		m.tab = tabHistory; m.scroll = 0; return m, nil
	case "4":
		m.tab = tabCommit; m.scroll = 0; return m, nil
	case "5":
		m.tab = tabDiff; m.scroll = 0; return m, nil
	case "left":
		if m.tab > 0 {
			m.tab--; m.scroll = 0
		}
		return m, nil
	case "right":
		if m.tab < tabCount-1 {
			m.tab++; m.scroll = 0
		}
		return m, nil
	case "up", "k":
		if m.scroll > 0 {
			m.scroll--
		}
		return m, nil
	case "down", "j":
		m.scroll++
		return m, nil
	case "pgup":
		m.scroll -= m.height / 3
		if m.scroll < 0 {
			m.scroll = 0
		}
		return m, nil
	case "pgdown":
		m.scroll += m.height / 3
		return m, nil
	case "r":
		m.notification = ""
		return m, m.loadData()
	case "c":
		return m, m.commitAll()
	case "p":
		return m, m.pushBranch()
	case "l":
		return m, m.pullBranch()
	}
	return m, nil
}

func (m *model) commitAll() tea.Cmd {
	return func() tea.Msg {
		ctx := context.TODO()
		for _, fc := range m.changes {
			if !fc.Staged {
				m.gitOps.Stage(ctx, fc.Path)
			}
		}
		if _, err := m.gitOps.Commit(ctx, "feat: WIP"); err != nil {
			return notifMsg{fmt.Sprintf("Commit error: %v", err)}
		}
		return m.loadData()()
	}
}

func (m *model) pushBranch() tea.Cmd {
	return func() tea.Msg {
		branch := "main"
		for _, b := range m.branches {
			if b.IsActive {
				branch = b.Name
				break
			}
		}
		if err := m.gitOps.Push(context.TODO(), "origin", branch, false); err != nil {
			return notifMsg{fmt.Sprintf("Push error: %v", err)}
		}
		return m.loadData()()
	}
}

func (m *model) pullBranch() tea.Cmd {
	return func() tea.Msg {
		if err := m.gitOps.Pull(context.TODO(), "origin", ""); err != nil {
			return notifMsg{fmt.Sprintf("Pull error: %v", err)}
		}
		return m.loadData()()
	}
}

// ══════════════════════════════════════════════════════════════════
//  VIEW  —  NO BORDERS, FULL WIDTH
// ══════════════════════════════════════════════════════════════════

func (m *model) View() string {
	if !m.ready {
		return stl(muted, bg).Render(" Loading...")
	}

	// ── TAB BAR (line 1) ──
	tabBar := ""
	for i, name := range tabNames {
		if i == int(m.tab) {
			tabBar += stl(purple, bg).Bold(true).Render(fmt.Sprintf("  %s  %s  ", tabKeys[i], name))
		} else {
			tabBar += stl(muted, bg).Render(fmt.Sprintf("  %s  %s  ", tabKeys[i], name))
		}
	}

	// ── INFO LINE (line 2) ──
	branchName := "main"
	for _, br := range m.branches {
		if br.IsActive {
			branchName = br.Name
			break
		}
	}
	infoLine := fmt.Sprintf("  ⎇ %s  │  %s  │  %d commits  │  %d branches  │  %d changes",
		branchName, tabNames[m.tab], len(m.commits), len(m.branches), len(m.changes))

	// ── CONTENT (no borders, full width) ──
	contentH := m.height - 5 // tab bar + info + separator + footer + notification
	if contentH < 2 {
		contentH = 2
	}

	var content string
	switch m.tab {
	case tabGraph:
		content = m.renderGraph(contentH)
	case tabWorktree:
		content = m.renderWorktree(contentH)
	case tabHistory:
		content = m.renderHistory(contentH)
	case tabCommit:
		content = m.renderCommit(contentH)
	case tabDiff:
		content = m.renderDiff(contentH)
	}

	// ── SEPARATOR ──
	sep := stl(border, bg).Render(strings.Repeat("─", m.width))

	// ── NOTIFICATION ──
	notif := ""
	if m.notification != "" {
		notif = "\n" + stl(red, bg).Render("  ⚠ "+m.notification)
	}

	// ── FOOTER ──
	footer := stl(dim, bg).Render(strings.Repeat("─", m.width)) + "\n" +
		m.renderFooter()

	return stl(text, bg).Render(tabBar) + "\n" +
		stl(muted, bg).Render(infoLine) + "\n" +
		sep + "\n" +
		content +
		notif + "\n" +
		footer
}

func stl(fg, bg lipgloss.Color) lipgloss.Style {
	return lipgloss.NewStyle().Foreground(fg).Background(bg)
}

// ── FOOTER ──
func (m *model) renderFooter() string {
	keys := []struct{ key, desc string }{
		{"1-5", "tabs"}, {"←→", "nav"}, {"↑↓", "scroll"},
		{"pgup/pgdn", "page"}, {"r", "refresh"},
		{"c", "commit"}, {"p", "push"}, {"l", "pull"}, {"q", "quit"},
	}
	var parts []string
	for _, k := range keys {
		parts = append(parts,
			stl(purple, bg).Bold(true).Render(" "+k.key+" ")+
				stl(muted, bg).Render(k.desc))
	}
	line := strings.Join(parts, "  ")
	if len(line) > m.width {
		line = line[:m.width]
	}
	return stl(bg, bg).Padding(0, 1).Width(m.width).Render(line)
}

// ══════════════════════════════════════════════════════════════════
//  GRAPH TAB — 3 colunas sem bordas
// ══════════════════════════════════════════════════════════════════

func (m *model) renderGraph(h int) string {
	w := m.width
	sideW := max(22, w/6)
	detW := max(28, w/5)
	midW := w - sideW - detW - 2 // 2 col separators

	// Build each column content
	left := m.renderSidebarList(sideW, h)
	mid := m.renderGraphList(midW, h)
	right := m.renderDetailList(detW, h)

	// Render side by side with thin vertical separators
	sepV := stl(border, bg).Render("┃")

	var out strings.Builder
	for line := 0; line < h; line++ {
		out.WriteString(getLine(left, line, sideW))
		out.WriteString(sepV)
		out.WriteString(getLine(mid, line, midW))
		out.WriteString(sepV)
		out.WriteString(getLine(right, line, detW))
		out.WriteString("\n")
	}
	return out.String()
}

func getLine(s string, line, w int) string {
	lines := strings.Split(s, "\n")
	if line >= len(lines) {
		return strings.Repeat(" ", w)
	}
	l := lines[line]
	if len(l) > w {
		return l[:w]
	}
	return l + strings.Repeat(" ", w-len(l))
}

func (m *model) renderSidebarList(w, h int) string {
	var b strings.Builder
	b.WriteString(purpleB.Render(" Branches "))
	b.WriteString("\n\n")
	for _, br := range m.branches {
		if br.IsActive {
			ab := ""
			if br.Ahead > 0 || br.Behind > 0 {
				ab = dimS.Render(fmt.Sprintf(" ↑%d↓%d", br.Ahead, br.Behind))
			}
			b.WriteString(fmt.Sprintf("  %s  %s%s\n", purpleB.Render("●"), purpleB.Render(br.Name), ab))
		}
	}
	for _, br := range m.branches {
		if !br.IsActive {
			b.WriteString(fmt.Sprintf("  %s  %s\n", muteS.Render("○"), muteS.Render(br.Name)))
		}
	}
	b.WriteString("\n"); b.WriteString(purpleB.Render(" Changes ")); b.WriteString("\n\n")
	if len(m.changes) == 0 {
		b.WriteString(muteS.Render("  Clean"))
	} else {
		b.WriteString(dimS.Render(fmt.Sprintf("  %d file(s)", len(m.changes))))
		b.WriteString("\n")
		for _, fc := range m.changes {
			bd, st := badge(fc.Status)
			b.WriteString(fmt.Sprintf("  %s %s\n", st.Render(bd), txtS.Render(trunc(fc.Path, w-6))))
		}
	}
	return b.String()
}

func (m *model) renderGraphList(w, h int) string {
	var b strings.Builder
	activeBranch := "main"
	for _, br := range m.branches {
		if br.IsActive {
			activeBranch = br.Name
			break
		}
	}
	b.WriteString(purpleB.Render(" " + activeBranch + " "))
	b.WriteString(dimS.Render(fmt.Sprintf("%d commits", len(m.commits))))
	b.WriteString("\n\n")

	if len(m.graph) == 0 {
		b.WriteString(muteS.Render("  No commits"))
		return b.String()
	}

	colColor := make(map[int]string)
	colIdx := 0
	maxRows := h - 4
	if maxRows < 2 {
		maxRows = 2
	}

	commitIdx := 0
	start := 0
	if m.detailCommit >= maxRows {
		start = countUp(m.graph, m.detailCommit)
	}
	end := start + maxRows
	if end > len(m.graph) {
		end = len(m.graph)
	}

	for i := start; i < end; i++ {
		row := m.graph[i]
		gs := graphLine(row.Graph, colColor, &colIdx)
		for len(gs) < 6 {
			gs += " "
		}
		if row.IsCommit {
			msg := trunc(row.Message, w-len(gs)-28)
			line := fmt.Sprintf("%s %s %s  %s  %s", gs,
				hashS.Render(row.Hash), txtS.Render(msg),
				muteS.Render(row.Author), muteS.Render(row.Time))
			if commitIdx == m.detailCommit {
				b.WriteString(sel.Render(line))
			} else {
				b.WriteString(line)
			}
			commitIdx++
		} else {
			b.WriteString(gs)
		}
		b.WriteString("\n")
	}
	if len(m.graph) > maxRows {
		b.WriteString(dimS.Render(fmt.Sprintf("  ↓ %d more", len(m.graph)-end)))
	}
	return b.String()
}

func (m *model) renderDetailList(w, h int) string {
	var b strings.Builder
	if m.detailCommit < 0 || m.detailCommit >= len(m.commits) {
		b.WriteString(muteS.Render(" No commit selected"))
		return b.String()
	}
	c := m.commits[m.detailCommit]
	b.WriteString(purpleB.Render(" Commit ")); b.WriteString("\n\n")
	b.WriteString(fmt.Sprintf("%s  %s\n", labS.Render("Hash"), valS.Render(c.Hash[:12])))
	b.WriteString(fmt.Sprintf("%s  %s\n", labS.Render("Author"), valS.Render(trunc(c.Author, w-10))))
	b.WriteString(fmt.Sprintf("%s  %s\n", labS.Render("Date"), valS.Render(c.Timestamp.Format("02 Jan 2006 15:04"))))
	b.WriteString(fmt.Sprintf("%s  %s\n", labS.Render("Msg"), valS.Render(trunc(c.MessageHead, w-8))))
	b.WriteString("\n"); b.WriteString(purpleB.Render(" Files ")); b.WriteString("\n\n")
	files := m.commitFiles
	if len(files) == 0 {
		b.WriteString(muteS.Render("  No files"))
	} else {
		for _, fc := range files {
			bd, st := badge(fc.Status)
			b.WriteString(fmt.Sprintf("  %s  %s\n", st.Render(bd), valS.Render(trunc(fc.Path, w-8))))
		}
	}
	return b.String()
}

// ══════════════════════════════════════════════════════════════════
//  WORKTREE TAB
// ══════════════════════════════════════════════════════════════════

func (m *model) renderWorktree(h int) string {
	var b strings.Builder
	b.WriteString(purpleB.Render(" Working Tree ")); b.WriteString("\n\n")
	if len(m.changes) == 0 {
		b.WriteString(muteS.Render("  Clean — no changes"))
		return b.String()
	}
	for _, fc := range m.changes {
		bd, st := badge(fc.Status)
		label := "unstaged"
		if fc.Staged {
			label = "staged"
		}
		b.WriteString(fmt.Sprintf("  %s  %s  %s\n", st.Render(bd), txtS.Render(fc.Path), dimS.Render(label)))
	}
	return b.String()
}

// ══════════════════════════════════════════════════════════════════
//  HISTORY TAB
// ══════════════════════════════════════════════════════════════════

func (m *model) renderHistory(h int) string {
	var b strings.Builder
	b.WriteString(purpleB.Render(" History ")); b.WriteString("\n\n")
	if len(m.commits) == 0 {
		b.WriteString(muteS.Render("  No commits"))
		return b.String()
	}
	maxItems := h - 3
	if maxItems < 2 {
		maxItems = 2
	}
	start := m.detailCommit - maxItems/2
	if start < 0 {
		start = 0
	}
	end := start + maxItems
	if end > len(m.commits) {
		end = len(m.commits)
	}
	for i := start; i < end; i++ {
		c := m.commits[i]
		line := fmt.Sprintf("  %s  %s  %s",
			hashS.Render(c.ShortHash),
			txtS.Render(trunc(c.MessageHead, 60)),
			muteS.Render(c.Timestamp.Format("02 Jan 15:04")))
		if i == m.detailCommit {
			b.WriteString(sel.Render(line))
		} else {
			b.WriteString(line)
		}
		b.WriteString("\n")
	}
	return b.String()
}

// ══════════════════════════════════════════════════════════════════
//  COMMIT TAB
// ══════════════════════════════════════════════════════════════════

func (m *model) renderCommit(h int) string {
	var b strings.Builder
	b.WriteString(purpleB.Render(" New Commit ")); b.WriteString("\n\n")
	b.WriteString(muteS.Render("  Files:")); b.WriteString("\n\n")
	if len(m.changes) == 0 {
		b.WriteString(muteS.Render("  No changes to commit"))
	} else {
		for _, fc := range m.changes {
			bd, st := badge(fc.Status)
			b.WriteString(fmt.Sprintf("  %s  %s\n", st.Render(bd), txtS.Render(fc.Path)))
		}
	}
	b.WriteString("\n" + dimS.Render("  Press c to commit all"))
	return b.String()
}

// ══════════════════════════════════════════════════════════════════
//  DIFF TAB
// ══════════════════════════════════════════════════════════════════

func (m *model) renderDiff(h int) string {
	var b strings.Builder
	b.WriteString(purpleB.Render(" Diff ")); b.WriteString("\n\n")
	if len(m.commitFiles) == 0 {
		b.WriteString(muteS.Render("  No files in selected commit"))
	} else {
		for _, fc := range m.commitFiles {
			bd, st := badge(fc.Status)
			b.WriteString(fmt.Sprintf("  %s  %s\n", st.Render(bd), txtS.Render(fc.Path)))
		}
	}
	b.WriteString("\n" + dimS.Render("  ↑↓ to select commit  d to view diff"))
	return b.String()
}

// ══════════════════════════════════════════════════════════════════
//  STYLES & HELPERS
// ══════════════════════════════════════════════════════════════════

var (
	purpleB = stl(purple, bg).Bold(true)
	txtS    = stl(text, bg)
	muteS   = stl(muted, bg)
	dimS    = stl(dim, bg)
	hashS   = stl(muted, bg)
	labS    = stl(muted, bg)
	valS    = stl(text, bg)
	sel     = stl(text, selBg)
)

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func trunc(s string, n int) string {
	if len(s) <= n || n < 3 {
		return s
	}
	return s[:n-1] + "…"
}

func countUp(rows []*types.GraphRow, target int) int {
	c := 0
	for i, row := range rows {
		if row.IsCommit {
			if c >= target {
				return i
			}
			c++
		}
	}
	return 0
}

func graphLine(graph string, colColor map[int]string, colIdx *int) string {
	var out strings.Builder
	for j, ch := range graph {
		switch {
		case ch == '*':
			if _, ok := colColor[j]; !ok {
				colColor[j] = graphColors[*colIdx%len(graphColors)]
				*colIdx++
			}
			out.WriteString(stl(lipgloss.Color(colColor[j]), bg).Bold(true).Render("●"))
		case ch == '|' || ch == '/' || ch == '\\' || ch == '_':
			c := dim
			if color, ok := colColor[j]; ok {
				c = lipgloss.Color(color)
			}
			out.WriteString(stl(c, bg).Render(string(ch)))
		default:
			out.WriteString(string(ch))
		}
	}
	return out.String()
}

func badge(status types.FileStatus) (string, lipgloss.Style) {
	switch status {
	case types.FileStatusModified:
		return "M", stl(yellow, bg).Bold(true)
	case types.FileStatusAdded:
		return "A", stl(green, bg).Bold(true)
	case types.FileStatusDeleted:
		return "D", stl(red, bg).Bold(true)
	case types.FileStatusUntracked:
		return "?", stl(muted, bg)
	default:
		return " ", stl(dim, bg)
	}
}
