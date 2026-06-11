<div align="center">
  <br />
  <h1>⌨️ GitHub Desktop TUI</h1>
  <p>
    <strong>Multi-provider Git client for the terminal</strong>
  </p>
  <p>
    <a href="https://github.com/ElioNeto/github-desktop-tui/actions">
      <img src="https://img.shields.io/github/actions/workflow/status/ElioNeto/github-desktop-tui/ci.yml?branch=main&style=flat-square" alt="CI Status" />
    </a>
    <a href="https://goreportcard.com/report/github.com/ElioNeto/github-desktop-tui">
      <img src="https://goreportcard.com/badge/github.com/ElioNeto/github-desktop-tui?style=flat-square" alt="Go Report Card" />
    </a>
    <a href="https://github.com/ElioNeto/github-desktop-tui/releases">
      <img src="https://img.shields.io/github/v/release/ElioNeto/github-desktop-tui?style=flat-square&include_prereleases" alt="Release" />
    </a>
    <a href="https://www.npmjs.com/package/github-desktop-tui">
      <img src="https://img.shields.io/npm/v/github-desktop-tui?style=flat-square" alt="npm" />
    </a>
    <a href="https://github.com/ElioNeto/github-desktop-tui/blob/main/LICENSE">
      <img src="https://img.shields.io/github/license/ElioNeto/github-desktop-tui?style=flat-square" alt="License" />
    </a>
    <a href="https://github.com/ElioNeto/github-desktop-tui/issues">
      <img src="https://img.shields.io/github/issues/ElioNeto/github-desktop-tui?style=flat-square" alt="Issues" />
    </a>
  </p>
  <p>
    <strong>English</strong> · <a href="docs/README-pt-BR.md">Português</a>
  </p>

  <pre>npx github-desktop-tui</pre>

  <img src="docs/screenshot.png" alt="GitHub Desktop TUI Screenshot" width="80%" />
</div>

---

**GitHub Desktop TUI** is a full-featured Git client that runs entirely in your terminal.
Like GitHub Desktop, but for the terminal — and supporting **multiple Git providers**.

### ✨ Features

| Feature | Description |
|---------|-------------|
| **📋 Staging** | Select files to stage/unstage with arrow keys, commit with message |
| **📜 Timeline** | Browse full commit history with dates, authors, and messages |
| **🔀 Branches** | List, create, switch, merge, and delete branches |
| **🚀 Push/Pull** | Synchronize with remotes; push and pull with auth support |
| **🔑 Auth** | Configure tokens directly or via environment variables |
| **🌐 Multi-Provider** | GitHub, GitLab, Bitbucket, Gitea/Forgejo *(in progress)* |
| **🎨 Themes** | Custom color palette with rich Lip Gloss styling |
| **⌨️ Keyboard** | Full keyboard navigation, vim-style keys, help overlay |

### 🖼️ Interface

```
┌──── Repos ────┬──── Commits ───────────┬──── Details ───┐
│  ● MyProject  │  ⎇ main                │  Status         │
│  ○ OtherRepo  │                        │  ● 2 staged     │
│               │  12 Jan 15:04 a1b2c3d  │  ● 1 modified   │
│  ── Providers ──  Fix login bug        │                 │
│  ● GitHub ✓   │                        │  Auth           │
│  ○ GitLab     │  11 Jan 09:30 e5f6g7h  │  ● GitHub       │
│               │  Add feature           │  ○ GitLab       │
│  ── Keys ──   │                        │                 │
│  c  commit    │  10 Jan 18:00 i9j0k1l  │                 │
│  s  stage     │  Initial commit        │                 │
│  p  push      │                        │                 │
├───────────────┴────────────────────────┴─────────────────┤
│  github ⎇ main  +2 ~1  ✓github                          │
└──────────────────────────────────────────────────────────┘
```

---

## 📦 Installation

### Via npm (recommended)

```bash
# Global install
npm install -g github-desktop-tui

# Run directly without installing
npx github-desktop-tui

# Run in a specific repository
npx github-desktop-tui /path/to/repo
```

### Via Go

```bash
go install github.com/ElioNeto/github-desktop-tui/cmd/github-desktop-tui@latest
```

### From source

```bash
git clone https://github.com/ElioNeto/github-desktop-tui.git
cd github-desktop-tui
make build
./dist/github-desktop-tui
```

### Binary releases

Download the latest binary for your platform from the
[Releases page](https://github.com/ElioNeto/github-desktop-tui/releases).

| Platform | Architecture | Download |
|----------|-------------|----------|
| Linux | amd64 / arm64 | `github-desktop-tui-linux-{arch}` |
| macOS | amd64 / arm64 | `github-desktop-tui-darwin-{arch}` |
| Windows | amd64 / arm64 | `github-desktop-tui-windows-{arch}.exe` |

---

## 🚀 Quick Start

```bash
# Run in the current directory
github-desktop-tui

# Run in a specific repository
github-desktop-tui ~/projects/my-project

# Show help
github-desktop-tui --help
```

### First-time setup

1. Press `P` to open authentication
2. Choose a provider (GitHub, GitLab, etc.)
3. Enter your token directly or reference an env var (`$GITHUB_TOKEN`)
4. Press `r` to refresh and load repositories
5. Press `c` to view changes and start staging

### Authentication

You can authenticate in two ways:

| Method | How | Example |
|--------|-----|---------|
| **Direct** | Type/paste your token directly | `ghp_xxxxxxxxxxxx` |
| **Env Var** | Reference an environment variable name | `$GITHUB_TOKEN` |

The application checks these environment variables automatically:

| Provider | Env Vars |
|----------|----------|
| GitHub | `GITHUB_TOKEN`, `GH_TOKEN`, `GITHUB_ENTERPRISE_TOKEN` |
| GitLab | `GITLAB_TOKEN`, `GITLAB_ACCESS_TOKEN` |
| Bitbucket | `BITBUCKET_TOKEN`, `BITBUCKET_ACCESS_TOKEN` |
| Gitea | `GITEA_TOKEN`, `FORGEJO_TOKEN` |

---

## ⌨️ Keybindings

| Key | Action | Context |
|-----|--------|---------|
| `Tab` / `Shift+Tab` | Next/previous panel | Global |
| `↑` / `↓` or `k` / `j` | Navigate lists | All lists |
| `c` | Open staging view | Global |
| `s` | Stage/unstage file | Staging view |
| `Enter` | Confirm / commit | Staging, inputs |
| `p` | Push | Staging view |
| `l` | Pull | Global |
| `b` | Toggle branch list | Global |
| `/` | Open timeline | Global |
| `d` | View file diff | Staging view |
| `P` | Open auth | Global |
| `r` | Refresh all | Global |
| `?` | Toggle help overlay | Global |
| `Esc` | Close overlay / back | All overlays |
| `q` | Quit | Global |

---

## 🎨 Theme

The default color palette is designed for low eye fatigue:

```
Background  #382a2a  Dark espresso
Surface     #4a3a3a  Lighter espresso (panels)
Accent      #ff3d3d  Vibrant red
Text        #e5ebbc  Warm cream
Muted       #8dc4b7  Soft teal
Success     #8dc4b7  Teal
Warning     #ff9d7d  Peach
Error       #ff3d3d  Red
```

To customize, create `~/.config/github-desktop-tui/theme.json`:

```json
{
  "bg": "#1a1b26",
  "surface": "#24283b",
  "accent": "#f7768e",
  "text": "#c0caf5",
  "muted": "#565f89",
  "success": "#9ece6a",
  "warning": "#e0af68",
  "error": "#f7768e",
  "info": "#7dcfff"
}
```

---

## 🏗️ Architecture

The application follows a **three-layer architecture** with the **Elm pattern** (Model-Update-View):

```
┌──────────────────────────────────────────┐
│          TUI LAYER (Bubble Tea)          │
│  Explorer(22%) │ Content(48%) │ Det(30%) │
├──────────────────────────────────────────┤
│          STORE LAYER (State)             │
│  Repos │ Commits │ Branches │ Settings   │
├──────────────────────────────────────────┤
│     GIT / PROVIDER LAYER (Strategy)      │
│  GitOperations + ProviderRegistry        │
└──────────────────────────────────────────┘
```

### Tech stack

| Layer | Technology |
|-------|-----------|
| Language | [Go 1.22+](https://go.dev/) |
| TUI | [Bubble Tea](https://github.com/charmbracelet/bubbletea) |
| Styling | [Lip Gloss](https://github.com/charmbracelet/lipgloss) |
| Components | [Bubbles](https://github.com/charmbracelet/bubbles) |
| Git | [go-git](https://github.com/go-git/go-git) |
| Binary size | ~15 MB (self-contained) |

---

## 🗺️ Roadmap

### Phase 1 — Foundation ✅
- [x] Three-panel TUI layout
- [x] Git operations (status, add, commit, push, pull, log)
- [x] Staging view (select files to commit)
- [x] Commit timeline viewer
- [x] Authentication (token + env vars)
- [x] Branch listing and switching
- [x] Remote management
- [x] Custom color theme

### Phase 2 — Git Power 🔨
- [ ] Diff viewer with syntax highlighting
- [ ] Full branch management (create, delete, merge)
- [ ] Merge conflict resolution UI
- [ ] Interactive rebase
- [ ] Cherry-pick and revert

### Phase 3 — Multi-Provider 🌐
- [ ] GitHub OAuth Device Flow
- [ ] GitLab provider (API v4)
- [ ] Bitbucket provider (Cloud + Server)
- [ ] Gitea/Forgejo provider
- [ ] Dynamic provider switching

### Phase 4 — Polish ✨
- [ ] Global search and filter
- [ ] Performance optimization + caching
- [ ] E2E tests and CI/CD
- [ ] Custom keybindings
- [ ] Internationalization (PT/BR + EN)
- [ ] Plugin system (MCP)

---

## 🤝 Contributing

Contributions are welcome! Please see our
[Contributing Guide](docs/contributing.md) for details.

1. Fork the repository
2. Create your feature branch: `git checkout -b feat/amazing-feature`
3. Commit your changes: `git commit -m 'feat: add amazing feature'`
4. Push: `git push origin feat/amazing-feature`
5. Open a Pull Request

### Development

```bash
# Run in development mode
make dev

# Build for production
make build

# Run tests
make test

# Run linter
make lint
```

---

## 📄 License

Distributed under the **MIT License**. See [LICENSE](LICENSE) for more information.

---

## 🙏 Acknowledgments

- [Bubble Tea](https://github.com/charmbracelet/bubbletea) — the amazing Go TUI framework
- [go-git](https://github.com/go-git/go-git) — pure Go Git implementation
- [GitHub Desktop](https://desktop.github.com/) — inspiration for the UX
- [LazyGit](https://github.com/jesseduffield/lazygit) — reference for terminal Git clients

---

<div align="center">
  <sub>Built with ❤️ and Go</sub>
</div>
