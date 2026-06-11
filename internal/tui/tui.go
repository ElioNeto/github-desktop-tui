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

// ── Purple Identity ──
var (
	purple     = lipgloss.Color("#a371f7")
	purpleDim  = lipgloss.Color("#6e40c9")
	purpleBg   = lipgloss.Color("#2a1a4a")
	bg         = lipgloss.Color("#0d1117")
	surface    = lipgloss.Color("#161b22")
	surfaceAlt = lipgloss.Color("#1c2333")
	border     = lipgloss.Color("#30363d")
	text       = lipgloss.Color("#e6edf3")
	muted      = lipgloss.Color("#8b949e")
	dim        = lipgloss.Color("#6e7681")
	green      = lipgloss.Color("#3fb950")
	red        = lipgloss.Color("#f85149")
	yellow     = lipgloss.Color("#d29922")
	blue       = lipgloss.Color("#58a6ff")
	selBg      = lipgloss.Color("#2a1a4a")
)

var graphDots = []string{"#a371f7", "#3fb950", "#d29922", "#58a6ff", "#f85149", "#56b6c2"}

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

// ── Focus ──
type focusArea int

const (
	focusSidebar focusArea = iota
	focusHistory
	focusDetails
)

// ── Model ──
type model struct {
	width, height int
	tab           tabID
	focus         focusArea
	ready         bool
	scroll        int // lines scrolled

	gitOps gitlocal.GitOperations
	store  *store.Store

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
		focus:        focusHistory,
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

// ── Keys ──
func (m *model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit

	// Tab switching
	case "1":
		m.tab = tabGraph
		m.scroll = 0
		return m, nil
	case "2":
		m.tab = tabWorktree
		m.scroll = 0
		return m, nil
	case "3":
		m.tab = tabHistory
		m.scroll = 0
		return m, nil
	case "4":
		m.tab = tabCommit
		m.scroll = 0
		return m, nil
	case "5":
		m.tab = tabDiff
		m.scroll = 0
		return m, nil

	case "left", "h":
		if m.tab > 0 {
			m.tab--
			m.scroll = 0
		}
		return m, nil
	case "right":
		if m.tab < tabCount-1 {
			m.tab++
			m.scroll = 0
		}
		return m, nil

	// Focus
	case "tab":
		if m.tab == tabGraph {
			m.focus = (m.focus + 1) % 3
		}
		return m, nil
	case "shift+tab":
		if m.tab == tabGraph {
			m.focus = (m.focus - 1 + 3) % 3
		}
		return m, nil

	// Navigation & scroll
	case "up", "k":
		if m.tab == tabHistory || m.tab == tabWorktree || m.tab == tabDiff {
			if m.detailCommit > 0 {
				m.detailCommit--
				return m, m.loadCommitFiles()
			}
		}
		if m.scroll > 0 {
			m.scroll--
		}
		return m, nil
	case "down", "j":
		if m.tab == tabHistory || m.tab == tabWorktree || m.tab == tabDiff {
			if m.detailCommit < len(m.commits)-1 {
				m.detailCommit++
				return m, m.loadCommitFiles()
			}
		}
		m.scroll++
		return m, nil
	case "pgup":
		m.scroll -= m.height / 2
		if m.scroll < 0 {
			m.scroll = 0
		}
		return m, nil
	case "pgdown":
		m.scroll += m.height / 2
		return m, nil

	// Actions
	case "r":
		m.notification = ""
		return m, m.loadData()
	case "c":
		return m, m.doCommit()
	case "p":
		return m, m.doPush()
	case "l":
		return m, m.doPull()
	}
	return m, nil
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

// ══════════════════════════════════════════════════════════════════
//  VIEW
// ══════════════════════════════════════════════════════════════════

func (m *model) View() string {
	if !m.ready {
		return lipgloss.NewStyle().Foreground(muted).Background(bg).Render(" Loading...")
	}

	screen := lipgloss.JoinVertical(lipgloss.Left,
		m.renderHeader(),
		m.renderContent(),
		m.renderFooter(),
	)

	return screen
}

// ── Header: 5-line tab bar ──
func (m *model) renderHeader() string {
	var b strings.Builder

	// Line 1-2: tab bar
	tabStyle := lipgloss.NewStyle().Padding(0, 2).Foreground(muted).Background(bg)
	tabActive := lipgloss.NewStyle().Padding(0, 2).Foreground(purple).Bold(true).Background(bg)

	for i, name := range tabNames {
		if i == int(m.tab) {
			b.WriteString(tabActive.Render(fmt.Sprintf(" [%s] %s ", tabKeys[i], name)))
		} else {
			b.WriteString(tabStyle.Render(fmt.Sprintf("  %s  %s  ", tabKeys[i], name)))
		}
	}
	b.WriteString("\n")

	// Line 2: separator line with purple tint under active tab
	tabStart := 0
	for i := 0; i < int(m.tab); i++ {
		tabStart += len(tabNames[i]) + 8
	}
	tabLen := len(tabNames[m.tab]) + 8
	before := strings.Repeat("─", tabStart)
	activeLine := lipgloss.NewStyle().Foreground(purple).Background(bg).Render(strings.Repeat("─", tabLen))
	after := strings.Repeat("─", max(0, m.width-tabStart-tabLen))

	b.WriteString(before + activeLine + after)
	b.WriteString("\n")

	// Line 3: branch info
	branchName := "main"
	for _, br := range m.branches {
		if br.IsActive {
			branchName = br.Name
			break
		}
	}
	b.WriteString(lipgloss.NewStyle().Foreground(text).Background(bg).Padding(0, 2).
		Render(fmt.Sprintf(" ⎇ %s  ◆  %s", branchName, tabNames[m.tab])))
	b.WriteString("\n")

	// Line 4: context info for active tab
	info := ""
	switch m.tab {
	case tabGraph:
		info = fmt.Sprintf("%d commits  •  %d branches  •  %d files changed",
			len(m.commits), len(m.branches), len(m.changes))
	case tabWorktree:
		info = fmt.Sprintf("%d file(s) modified", len(m.changes))
	case tabHistory:
		info = fmt.Sprintf("%d commits total", len(m.commits))
	case tabCommit:
		info = "Stage files and write commit message"
	case tabDiff:
		info = "View file diffs"
	}
	b.WriteString(lipgloss.NewStyle().Foreground(muted).Background(bg).Padding(0, 2).
		Render(info))
	b.WriteString("\n")

	// Line 5: thin separator
	b.WriteString(lipgloss.NewStyle().Foreground(border).Background(bg).
		Render(strings.Repeat("─", m.width)))

	return b.String()
}

// ── Content ──
func (m *model) renderContent() string {
	contentH := m.height - 7 // header(5) + footer(1) + notif(1)
	if contentH < 3 {
		contentH = 3
	}

	switch m.tab {
	case tabGraph:
		return m.renderGraphTab(contentH)
	case tabWorktree:
		return m.renderWorktreeTab(contentH)
	case tabHistory:
		return m.renderHistoryTab(contentH)
	case tabCommit:
		return m.renderCommitTab(contentH)
	case tabDiff:
		return m.renderDiffTab(contentH)
	}
	return ""
}

// ── Footer ──
func (m *model) renderFooter() string {
	shortkeys := []string{
		"1-5 tabs", "← → nav",
		"↑↓ scroll", "pgup/pgdn page",
		"tab focus", "r refresh",
		"c commit", "p push", "l pull",
		"q quit",
	}
	var parts []string
	for _, k := range shortkeys {
		parts = append(parts, fmt.Sprintf("%s %s",
			lipgloss.NewStyle().Foreground(purple).Bold(true).Background(bg).Render(k[:1]),
			lipgloss.NewStyle().Foreground(muted).Background(bg).Render(k[1:])))
	}
	line := strings.Join(parts, "  ")
	if len(line) > m.width {
		line = line[:m.width]
	}
	return lipgloss.NewStyle().Foreground(border).Background(bg).
		Render(strings.Repeat("─", m.width)) + "\n" +
		lipgloss.NewStyle().Background(bg).Padding(0, 1).Width(m.width).Render(line)
}

// ══════════════════════════════════════════════════════════════════
//  GRAPH TAB (3 panels)
// ══════════════════════════════════════════════════════════════════

func (m *model) renderGraphTab(h int) string {
	sidebarW := max(24, m.width/5)
	detailsW := max(30, m.width/4)
	centerW := m.width - sidebarW - detailsW - 4

	panel := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		Padding(0, 1).
		Background(bg)

	active := panel.BorderForeground(purple)
	inactive := panel.BorderForeground(border)

	mkPanel := func(w int, isActive bool, content string) string {
		s := inactive.Width(w).Height(h)
		if isActive {
			s = active.Width(w).Height(h)
		}
		return s.Render(content)
	}

	return lipgloss.JoinHorizontal(lipgloss.Top,
		mkPanel(sidebarW, m.focus == focusSidebar, m.renderSidebar()),
		mkPanel(centerW, m.focus == focusHistory, m.renderCommitGraph(h)),
		mkPanel(detailsW, m.focus == focusDetails, m.renderDetails()),
	)
}

func (m *model) renderSidebar() string {
	var b strings.Builder
	b.WriteString(purpleStyle.Render(" Branches ")); b.WriteString("\n\n")
	for _, br := range m.branches {
		if br.IsActive {
			ab := ""
			if br.Ahead > 0 || br.Behind > 0 {
				ab = dimStyle.Render(fmt.Sprintf(" ↑%d↓%d", br.Ahead, br.Behind))
			}
			b.WriteString(fmt.Sprintf("  %s  %s%s\n", purpleDot.Render("●"), branchActive.Render(br.Name), ab))
		}
	}
	for _, br := range m.branches {
		if !br.IsActive {
			b.WriteString(fmt.Sprintf("  %s  %s\n", mutedStyle.Render("○"), mutedStyle.Render(br.Name)))
		}
	}
	b.WriteString("\n"); b.WriteString(purpleStyle.Render(" Changes ")); b.WriteString("\n\n")
	if len(m.changes) == 0 {
		b.WriteString(mutedStyle.Render("  Clean working tree"))
	} else {
		staged := 0
		for _, fc := range m.changes {
			if fc.Staged {
				staged++
			}
		}
		b.WriteString(dimStyle.Render(fmt.Sprintf("  %d file(s) (%d staged)", len(m.changes), staged)))
		b.WriteString("\n")
		for _, fc := range m.changes {
			badge, st := changeBadge(fc.Status)
			b.WriteString(fmt.Sprintf("  %s %s\n", st.Render(badge), textStyle.Render(fc.Path)))
		}
	}
	return b.String()
}

func (m *model) renderCommitGraph(h int) string {
	var b strings.Builder
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
		b.WriteString(mutedStyle.Render("  No commits"))
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
		start = countGraphUpTo(m.graph, m.detailCommit)
	}
	end := start + maxRows
	if end > len(m.graph) {
		end = len(m.graph)
	}

	for i := start; i < end; i++ {
		row := m.graph[i]
		graphStr := renderGraphLine(row.Graph, colColor, &colIdx)
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
			line := fmt.Sprintf("%s %s %s  %s  %s", graphStr,
				hashStyle.Render(row.Hash), textStyle.Render(msg),
				mutedStyle.Render(row.Author), mutedStyle.Render(row.Time))
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

func (m *model) renderDetails() string {
	var b strings.Builder
	if m.detailCommit < 0 || m.detailCommit >= len(m.commits) {
		b.WriteString(mutedStyle.Render(" No commit selected"))
		return b.String()
	}
	c := m.commits[m.detailCommit]
	b.WriteString(purpleStyle.Render(" Commit ")); b.WriteString("\n\n")
	b.WriteString(fmt.Sprintf("%s  %s\n", labelStyle.Render("Hash    "), valueStyle.Render(c.Hash[:12])))
	b.WriteString(fmt.Sprintf("%s  %s\n", labelStyle.Render("Author  "), valueStyle.Render(c.Author)))
	b.WriteString(fmt.Sprintf("%s  %s\n", labelStyle.Render("Date    "), valueStyle.Render(c.Timestamp.Format("02 Jan 2006 15:04"))))
	b.WriteString(fmt.Sprintf("%s  %s\n", labelStyle.Render("Message "), valueStyle.Render(c.MessageHead)))
	b.WriteString("\n"); b.WriteString(purpleStyle.Render(" Files ")); b.WriteString("\n\n")
	files := m.commitFiles
	if len(files) == 0 {
		b.WriteString(mutedStyle.Render("  No files changed"))
	} else {
		for _, fc := range files {
			badge, st := changeBadge(fc.Status)
			b.WriteString(fmt.Sprintf("  %s  %s\n", st.Render(badge), valueStyle.Render(fc.Path)))
		}
	}
	return b.String()
}

// ══════════════════════════════════════════════════════════════════
//  WORKTREE TAB
// ══════════════════════════════════════════════════════════════════

func (m *model) renderWorktreeTab(h int) string {
	var b strings.Builder
	b.WriteString(purpleStyle.Render(" Working Tree ")); b.WriteString("\n\n")
	if len(m.changes) == 0 {
		b.WriteString(mutedStyle.Render("  Clean — no changes"))
		return b.String()
	}
	for _, fc := range m.changes {
		badge, st := changeBadge(fc.Status)
		label := "unstaged"
		if fc.Staged {
			label = "staged  "
		}
		b.WriteString(fmt.Sprintf("  %s  %s  %s\n", st.Render(badge), valueStyle.Render(fc.Path), dimStyle.Render(label)))
	}
	return b.String()
}

// ══════════════════════════════════════════════════════════════════
//  HISTORY TAB
// ══════════════════════════════════════════════════════════════════

func (m *model) renderHistoryTab(h int) string {
	var b strings.Builder
	b.WriteString(purpleStyle.Render(" History ")); b.WriteString("\n\n")
	if len(m.commits) == 0 {
		b.WriteString(mutedStyle.Render("  No commits"))
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
			hashStyle.Render(c.ShortHash),
			textStyle.Render(c.MessageHead),
			mutedStyle.Render(c.Timestamp.Format("02 Jan 15:04")))
		if i == m.detailCommit {
			b.WriteString(selBgStyle.Render(line))
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

func (m *model) renderCommitTab(h int) string {
	var b strings.Builder
	b.WriteString(purpleStyle.Render(" New Commit ")); b.WriteString("\n\n")
	b.WriteString(mutedStyle.Render("  Files to commit:")); b.WriteString("\n\n")
	if len(m.changes) == 0 {
		b.WriteString(mutedStyle.Render("  No changes to commit"))
	} else {
		for _, fc := range m.changes {
			badge, st := changeBadge(fc.Status)
			b.WriteString(fmt.Sprintf("  %s  %s\n", st.Render(badge), textStyle.Render(fc.Path)))
		}
	}
	b.WriteString("\n"); b.WriteString(dimStyle.Render("  Press c to commit all (WIP message)"))
	return b.String()
}

// ══════════════════════════════════════════════════════════════════
//  DIFF TAB
// ══════════════════════════════════════════════════════════════════

func (m *model) renderDiffTab(h int) string {
	var b strings.Builder
	b.WriteString(purpleStyle.Render(" Diff ")); b.WriteString("\n\n")
	if m.detailCommit < 0 || m.detailCommit >= len(m.commits) {
		b.WriteString(mutedStyle.Render("  Select a commit to view diff"))
		return b.String()
	}
	if len(m.commitFiles) == 0 {
		b.WriteString(mutedStyle.Render("  No files in this commit"))
	} else {
		for _, fc := range m.commitFiles {
			badge, st := changeBadge(fc.Status)
			b.WriteString(fmt.Sprintf("  %s  %s\n", st.Render(badge), textStyle.Render(fc.Path)))
		}
	}
	b.WriteString("\n"); b.WriteString(dimStyle.Render("  ↑↓ select commit  d view diff"))
	return b.String()
}

// ══════════════════════════════════════════════════════════════════
//  HELPERS
// ══════════════════════════════════════════════════════════════════

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
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

func renderGraphLine(graph string, colColor map[int]string, colIdx *int) string {
	var out strings.Builder
	for j, ch := range graph {
		switch {
		case ch == '*':
			if _, ok := colColor[j]; !ok {
				colColor[j] = graphDots[*colIdx%len(graphDots)]
				*colIdx++
			}
			out.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color(colColor[j])).Bold(true).Render("●"))
		case ch == '|' || ch == '/' || ch == '\\' || ch == '_':
			c := dim
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

func changeBadge(status types.FileStatus) (string, lipgloss.Style) {
	switch status {
	case types.FileStatusModified:
		return "M", lipgloss.NewStyle().Foreground(yellow).Bold(true)
	case types.FileStatusAdded:
		return "A", lipgloss.NewStyle().Foreground(green).Bold(true)
	case types.FileStatusDeleted:
		return "D", lipgloss.NewStyle().Foreground(red).Bold(true)
	case types.FileStatusUntracked:
		return "?", lipgloss.NewStyle().Foreground(muted)
	default:
		return " ", lipgloss.NewStyle().Foreground(dim)
	}
}

// Pre-built styles
var (
	purpleStyle  = lipgloss.NewStyle().Foreground(purple).Bold(true)
	purpleDot    = lipgloss.NewStyle().Foreground(purple).Bold(true)
	branchActive = lipgloss.NewStyle().Foreground(purple).Bold(true)
	textStyle    = lipgloss.NewStyle().Foreground(text)
	mutedStyle   = lipgloss.NewStyle().Foreground(muted)
	dimStyle     = lipgloss.NewStyle().Foreground(dim)
	hashStyle    = lipgloss.NewStyle().Foreground(muted)
	labelStyle   = lipgloss.NewStyle().Foreground(muted)
	valueStyle   = lipgloss.NewStyle().Foreground(text)
	selBgStyle   = lipgloss.NewStyle().Background(selBg)
)
