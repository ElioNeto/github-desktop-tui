package keybindings

import "github.com/charmbracelet/bubbles/key"

// KeyMap defines all keybindings for the application.
type KeyMap struct {
	Help           key.Binding
	Quit           key.Binding
	Refresh        key.Binding
	FocusNext      key.Binding
	FocusPrev      key.Binding
	Search         key.Binding
	Enter          key.Binding
	Escape         key.Binding
	Diff           key.Binding
	Stage          key.Binding
	Unstage        key.Binding
	Commit         key.Binding
	Push           key.Binding
	Pull           key.Binding
	Branch         key.Binding
	ProviderSwitch key.Binding
	Up             key.Binding
	Down           key.Binding
	PageUp         key.Binding
	PageDown       key.Binding
	Home           key.Binding
	End            key.Binding

	// F1.3 - Panel navigation
	Panel1 key.Binding
	Panel2 key.Binding
	Panel3 key.Binding

	// F1.1 - Notification history
	History key.Binding

	// F1.4 - Theme toggle
	ThemeToggle key.Binding

	// F1.2 - Repo management
	RepoAdd    key.Binding
	RepoScan   key.Binding
	RepoRemove key.Binding
	RepoFav    key.Binding

	// F2.2 - Branch management
	CreateBranch key.Binding
	DeleteBranch key.Binding
	MergeBranch  key.Binding
	RenameBranch key.Binding

	// F2.6 - File tree explorer
	FileTreeToggle key.Binding

	// F2.5 - Cherry-pick / Revert
	CherryPick key.Binding
	Revert     key.Binding
}

// DefaultKeyMap returns the default keybinding set.
func DefaultKeyMap() KeyMap {
	return KeyMap{
		Help: key.NewBinding(
			key.WithKeys("?"),
			key.WithHelp("?", "ajuda"),
		),
		Quit: key.NewBinding(
			key.WithKeys("q", "ctrl+c"),
			key.WithHelp("q", "sair"),
		),
		Refresh: key.NewBinding(
			key.WithKeys("r"),
			key.WithHelp("r", "atualizar"),
		),
		FocusNext: key.NewBinding(
			key.WithKeys("tab"),
			key.WithHelp("tab", "próximo painel"),
		),
		FocusPrev: key.NewBinding(
			key.WithKeys("shift+tab"),
			key.WithHelp("shift+tab", "painel anterior"),
		),
		Search: key.NewBinding(
			key.WithKeys("/"),
			key.WithHelp("/", "buscar"),
		),
		Enter: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "confirmar"),
		),
		Escape: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "voltar"),
		),
		Diff: key.NewBinding(
			key.WithKeys("d"),
			key.WithHelp("d", "ver diff"),
		),
		Stage: key.NewBinding(
			key.WithKeys("s"),
			key.WithHelp("s", "stage"),
		),
		Unstage: key.NewBinding(
			key.WithKeys("u"),
			key.WithHelp("u", "unstage"),
		),
		Commit: key.NewBinding(
			key.WithKeys("c"),
			key.WithHelp("c", "commitar"),
		),
		Push: key.NewBinding(
			key.WithKeys("p"),
			key.WithHelp("p", "push"),
		),
		Pull: key.NewBinding(
			key.WithKeys("l"),
			key.WithHelp("l", "pull"),
		),
		Branch: key.NewBinding(
			key.WithKeys("b"),
			key.WithHelp("b", "branch"),
		),
		ProviderSwitch: key.NewBinding(
			key.WithKeys("P"),
			key.WithHelp("P", "trocar provedor"),
		),
		Up: key.NewBinding(
			key.WithKeys("up", "k"),
			key.WithHelp("↑/k", "cima"),
		),
		Down: key.NewBinding(
			key.WithKeys("down", "j"),
			key.WithHelp("↓/j", "baixo"),
		),
		PageUp: key.NewBinding(
			key.WithKeys("pgup"),
			key.WithHelp("pgup", "página acima"),
		),
		PageDown: key.NewBinding(
			key.WithKeys("pgdown"),
			key.WithHelp("pgdn", "página abaixo"),
		),
		Home: key.NewBinding(
			key.WithKeys("home"),
			key.WithHelp("home", "início"),
		),
		End: key.NewBinding(
			key.WithKeys("end"),
			key.WithHelp("end", "fim"),
		),

		// Panel navigation (F1.3)
		Panel1: key.NewBinding(
			key.WithKeys("1"),
			key.WithHelp("1", "painel esquerdo"),
		),
		Panel2: key.NewBinding(
			key.WithKeys("2"),
			key.WithHelp("2", "painel central"),
		),
		Panel3: key.NewBinding(
			key.WithKeys("3"),
			key.WithHelp("3", "painel direito"),
		),

		// Notification history (F1.1)
		History: key.NewBinding(
			key.WithKeys("N"),
			key.WithHelp("N", "histórico de notificações"),
		),

		// Theme toggle (F1.4)
		ThemeToggle: key.NewBinding(
			key.WithKeys("T"),
			key.WithHelp("T", "alternar tema"),
		),

		// Repo management (F1.2)
		RepoAdd: key.NewBinding(
			key.WithKeys("a"),
			key.WithHelp("a", "adicionar repositório"),
		),
		RepoScan: key.NewBinding(
			key.WithKeys("A"),
			key.WithHelp("A", "scanear diretório"),
		),
		RepoRemove: key.NewBinding(
			key.WithKeys("x"),
			key.WithHelp("x", "remover repositório"),
		),
		RepoFav: key.NewBinding(
			key.WithKeys("f"),
			key.WithHelp("f", "favoritar"),
		),

		// F2.2 - Branch management
		CreateBranch: key.NewBinding(
			key.WithKeys("C"),
			key.WithHelp("C", "criar branch"),
		),
		DeleteBranch: key.NewBinding(
			key.WithKeys("D"),
			key.WithHelp("D", "deletar branch"),
		),
		MergeBranch: key.NewBinding(
			key.WithKeys("m"),
			key.WithHelp("m", "merge branch"),
		),
		RenameBranch: key.NewBinding(
			key.WithKeys("R"),
			key.WithHelp("R", "renomear branch"),
		),

		// F2.6 - File tree explorer
		FileTreeToggle: key.NewBinding(
			key.WithKeys("t"),
			key.WithHelp("t", "árvore de arquivos"),
		),

		// F2.5 - Cherry-pick / Revert
		CherryPick: key.NewBinding(
			key.WithKeys("y"),
			key.WithHelp("y", "cherry-pick commit"),
		),
		Revert: key.NewBinding(
			key.WithKeys("V"),
			key.WithHelp("V", "reverter commit"),
		),
	}
}
