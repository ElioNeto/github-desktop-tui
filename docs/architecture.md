# Arquitetura do GitHub Desktop TUI

## Visão Geral

**github-desktop-tui** é um cliente Git multi-provedor rodando no terminal,
similar ao GitHub Desktop, mas compatível com **GitHub, GitLab, Bitbucket,
Gitea/Forgejo e servidores Git genéricos**.

Ele oferece uma interface de três painéis (explorer, conteúdo, detalhes)
com navegação total por teclado, suporte a temas e operações Git completas.

---

## Stack Tecnológico

| Camada        | Tecnologia                                              |
|---------------|---------------------------------------------------------|
| **Linguagem** | Go 1.22+                                                |
| **TUI**       | [Bubble Tea](https://github.com/charmbracelet/bubbletea) |
| **Estilo**    | [Lip Gloss](https://github.com/charmbracelet/lipgloss)   |
| **Componentes** | [Bubbles](https://github.com/charmbracelet/bubbles)   |
| **Git Local** | [go-git](https://github.com/go-git/go-git)               |
| **Config**    | Viper + JSON/YAML                                        |
| **Keychain**  | [go-keyring](https://github.com/zalando/go-keyring)      |
| **Build**     | Go toolchain + Makefile                                  |
| **Testes**    | testing + testify                                        |

### Por que Go + Bubble Tea?

- **Binário único** — sem runtime, sem dependências para o usuário final
- **Goroutines** — operações Git e chamadas de API em paralelo sem bloquear a TUI
- **Tipagem forte** — segurança em operações complexas de estado
- **Ecosystema Charm** — Bubble Tea, Lip Gloss e Bubbles são maturos e ativamente mantidos
- **Cross-platform** — Linux, macOS, Windows (WSL2+)

---

## Estrutura de Diretórios

```
github-desktop-tui/
├── cmd/
│   └── github-desktop-tui/
│       └── main.go              # Entry point (thin)
│
├── internal/
│   ├── app/
│   │   ├── app.go               # Bootstrap: config, providers, TUI
│   │   └── config.go            # Carrega configurações iniciais
│   │
│   ├── tui/
│   │   ├── tui.go               # Model raiz do Bubble Tea
│   │   ├── layout.go            # Gerenciamento de layout (3 painéis)
│   │   │
│   │   ├── views/               # Cada view é um componente Bubble Tea
│   │   │   ├── repositories/    # Lista de repositórios (painel esquerdo)
│   │   │   ├── commitlog/       # Histórico de commits (painel central)
│   │   │   ├── diffviewer/      # Visualizador de diff (painel central)
│   │   │   ├── branchlist/      # Gerenciamento de branches
│   │   │   ├── details/         # Detalhes do repositório (painel direito)
│   │   │   ├── filestree/       # Árvore de arquivos (painel direito)
│   │   │   ├── search/          # Busca/filtro global
│   │   │   └── help/            # Overlay de ajuda
│   │   │
│   │   ├── components/          # Widgets reutilizáveis
│   │   │   ├── spinner/         # Loading spinner
│   │   │   ├── table/           # Tabela genérica
│   │   │   ├── textinput/       # Campo de texto
│   │   │   ├── confirm/         # Diálogo de confirmação
│   │   │   └── statusbar/       # Barra de status inferior
│   │   │
│   │   ├── keybindings/         # Definições centralizadas de teclas
│   │   └── theme/               # Sistema de temas
│   │
│   ├── git/
│   │   ├── git.go               # Interface GitOperations
│   │   ├── local.go             # Operações Git locais (go-git)
│   │   ├── worktree.go          # Operações de worktree
│   │   ├── staging.go           # Stage/unstage/discard
│   │   └── commit.go            # Commit logic
│   │
│   ├── providers/
│   │   ├── interface.go         # Interface GitProvider
│   │   ├── registry.go          # Registro de provedores
│   │   ├── github/
│   │   │   ├── client.go        # API client GitHub (REST/GraphQL)
│   │   │   ├── types.go         # Tipos específicos do GitHub
│   │   │   ├── auth.go          # OAuth + token
│   │   │   └── provider.go      # Implementação GitProvider
│   │   ├── gitlab/
│   │   │   └── ...              # Mesma estrutura
│   │   ├── bitbucket/
│   │   │   └── ...
│   │   └── gitea/
│   │       └── ...
│   │
│   ├── store/
│   │   ├── store.go             # Store central (singleton)
│   │   ├── repositories.go     # Estado de repositórios
│   │   ├── commits.go          # Estado de commits
│   │   ├── branches.go         # Estado de branches
│   │   └── settings.go         # Preferências do usuário
│   │
│   ├── auth/
│   │   ├── auth.go             # Interface AuthManager
│   │   ├── keychain.go         # Keychain do SO
│   │   └── token.go            # Gerenciamento de tokens
│   │
│   └── config/
│       ├── config.go           # Leitura/escrita de config
│       └── defaults.go         # Valores padrão
│
├── pkg/
│   └── types/                   # Tipos compartilhados
│       ├── repository.go
│       ├── commit.go
│       ├── branch.go
│       ├── pullrequest.go
│       └── issue.go
│
├── docs/
│   ├── architecture.md         # Este documento
│   └── contributing.md
│
├── .teamcode/                   # Configuração de agentes
├── go.mod
├── Makefile
└── README.md
```

---

## Arquitetura em Camadas

```
┌─────────────────────────────────────────────────────┐
│                    TUI LAYER                         │
│  (Bubble Tea Model/View/Update)                     │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐          │
│  │ Explorer  │  │ Content  │  │ Details  │          │
│  │ (esq. 25%)│  │ (centro  │  │ (dir.    │          │
│  │ repos,    │  │  45%)    │  │  30%)    │          │
│  │ providers │  │ commits, │  │ file     │          │
│  │ favoritos │  │ diff     │  │ tree,    │          │
│  │           │  │ branches │  │ preview  │          │
│  └────┬──────┘  └────┬──────┘  └────┬──────┘        │
│       │              │              │                │
│  ┌────┴──────────────┴──────────────┴────┐           │
│  │           Status Bar                  │           │
│  │  branch │ provider │ sync │ clock     │           │
│  └────────────────────────────────────────┘           │
└──────────────────────┬──────────────────────────────┘
                       │
┌──────────────────────┴──────────────────────────────┐
│                   STORE LAYER                        │
│  (Estado central da aplicação)                      │
│  ┌────────────┐ ┌──────────┐ ┌──────────────────┐   │
│  │ ReposState │ │ GitState │ │ SettingsState     │   │
│  │ .repos[]   │ │ .branch  │ │ .theme           │   │
│  │ .selected  │ │ .commits │ │ .keybindings     │   │
│  │ .provider  │ │ .changes │ │ .active_provider │   │
│  └────────────┘ └──────────┘ └──────────────────┘   │
└──────────────────────┬──────────────────────────────┘
                       │
┌──────────────────────┴──────────────────────────────┐
│               GIT / PROVIDER LAYER                   │
│                                                      │
│  ┌──────────────────────────────────────────────┐   │
│  │          GitOperations (interface)           │   │
│  │  commit, push, pull, branch, diff, status    │   │
│  └──────────┬───────────────────────────────────┘   │
│             │                                        │
│  ┌──────────┴──────────┐  ┌──────────────────────┐  │
│  │   LocalGitAdapter   │  │   ProviderRegistry   │  │
│  │   (go-git / shell)  │  │                      │  │
│  └─────────────────────┘  │  ┌────┐ ┌────┐     │  │
│                           │  │ GH │ │ GL │ ... │  │
│                           │  └────┘ └────┘     │  │
│                           └──────────────────────┘  │
└──────────────────────────────────────────────────────┘
```

---

## Fluxo de Dados (Arquitetura Elm)

O Bubble Tea segue o padrão **Elm Architecture** (Model-Update-View):

```
                  ┌──────────┐
                  │  Msg     │ ← User input, timers, API responses
                  └────┬─────┘
                       │
              ┌────────┴────────┐
              │                 │
         ┌────┴────┐      ┌────┴────┐
         │ Update  │ ──── │ Model   │
         │ (msg)   │      │ (state) │
         └────┬────┘      └────┬────┘
              │                │
              └────┬───────────┘
                   │
              ┌────┴────┐
              │  View   │ → Renderização no terminal
              └─────────┘
```

### Fluxo Típico: Usuário faz um commit

```
1. Usuário digita 'c' (commit)
       │
2. TUI envia CommitKeyMsg
       │
3. Update() recebe CommitKeyMsg
       │
4. ├─ Model verifica se há mudanças stage
   ├─ Se não → envia ErrorMsg("Nada para commitar")
   ├─ Se sim → envia RequestCommitMsg
       │
5. Update() recebe RequestCommitMsg
       │
6. ├─ Abre textinput para mensagem
   ├─ Usuário digita mensagem + Enter
   ├─ Envia ExecuteCommitMsg{message}
       │
7. Update() recebe ExecuteCommitMsg
       │
8. ├─ Chama git.Local.Commit(message)
   ├─ Se erro → envia ErrorMsg
   ├─ Se sucesso →
       │  ├─ Atualiza store (limpa staging, atualiza log)
       │  ├─ Envia RefreshMsg
       │  └─ Envia SuccessMsg("Commit realizado")
       │
9. View() re-renderiza com estado atualizado
```

---

## Sistema de Provedores (Strategy Pattern)

Cada provedor Git implementa a interface `GitProvider`:

```go
// internal/providers/interface.go
type GitProvider interface {
    // Identidade
    Name() string
    DisplayName() string
    Icon() string

    // Autenticação
    AuthType() AuthType
    IsAuthenticated() bool
    Authenticate(ctx context.Context) (*AuthResult, error)

    // Repositórios
    ListRepositories(ctx context.Context) ([]*types.Repository, error)
    GetRepository(ctx context.Context, owner, name string) (*types.Repository, error)

    // Pull Requests / Merge Requests
    ListPullRequests(ctx context.Context, repo *types.Repository, state PRState) ([]*types.PullRequest, error)
    GetPullRequest(ctx context.Context, repo *types.Repository, id int) (*types.PullRequest, error)

    // Issues
    ListIssues(ctx context.Context, repo *types.Repository, state IssueState) ([]*types.Issue, error)

    // Branches
    ListBranches(ctx context.Context, repo *types.Repository) ([]*types.Branch, error)

    // Commits
    ListCommits(ctx context.Context, repo *types.Repository, opts *CommitListOptions) ([]*types.Commit, error)
}
```

### Registry

```go
// internal/providers/registry.go
type Registry struct {
    providers map[string]GitProvider
    active    string
}

func NewRegistry() *Registry {
    r := &Registry{providers: make(map[string]GitProvider)}
    r.Register(github.NewProvider())
    r.Register(gitlab.NewProvider())
    r.Register(bitbucket.NewProvider())
    r.Register(gitea.NewProvider())
    return r
}

func (r *Registry) Register(p GitProvider) {
    r.providers[p.Name()] = p
}

func (r *Registry) Active() GitProvider {
    return r.providers[r.active]
}
```

---

## Gerenciamento de Estado (Store)

O store segue o padrão de **event sourcing leve** — toda mutação de estado
dispara uma mensagem que o Bubble Tea processa no `Update()`.

```go
// internal/store/store.go
type Store struct {
    Repositories *RepositoryStore
    Commits      *CommitStore
    Branches     *BranchStore
    Settings     *SettingsStore
}

func New() *Store {
    return &Store{
        Repositories: NewRepositoryStore(),
        Commits:      NewCommitStore(),
        Branches:     NewBranchStore(),
        Settings:     NewSettingsStore(),
    }
}
```

Cada sub-store é thread-safe (sync.RWMutex) e emite mensagens via channel.

---

## Layout da TUI (3 Painéis)

O layout é baseado no especificado em `.teamcode/tui.json`:

```
┌─────────────┬────────────────────────┬─────────────────┐
│  EXPLORER   │       CONTENT         │    DETAILS      │
│  (25%)      │       (45%)           │    (30%)        │
│             │                        │                 │
│  ┌────────┐ │  ┌──────────────────┐ │  ┌───────────┐  │
│  │ Repo 1 │ │  │ * main           │ │  │ Owner     │  │
│  │ Repo 2 │ │  │ 2024-01-15 Fix.. │ │  │ Language  │  │
│  │ Repo 3 │ │  │ 2024-01-14 Add.. │ │  │ Stars     │  │
│  │ ...    │ │  │ 2024-01-13 Ini.. │ │  │ Forks     │  │
│  └────────┘ │  └──────────────────┘ │  │           │  │
│             │                        │  │ File tree │  │
│  Providers  │  [branch selector]     │  │ ┌───────┐ │  │
│  ○ GitHub   │                        │  │ │src/   │ │  │
│  ○ GitLab   │                        │  │ │docs/  │ │  │
│  ○ Bitbucket│                        │  │ │...    │ │  │
│             │                        │  │ └───────┘ │  │
└─────────────┴────────────────────────┴─────────────────┘
│ branch: main  │ provider: GitHub  │ sync: ↑0 ↓0 │ 14:30 │
└──────────────────────────────────────────────────────────┘
```

### Gerenciamento de Resize

```go
type Layout struct {
    LeftWidth   int
    CenterWidth int
    RightWidth  int
}

func CalculateLayout(width, height int, ratios [3]float64) Layout {
    // width * ratio, respeitando min_width
}
```

---

## Mensagens (Bubble Tea Messages)

```go
// Mensagens de navegação
type FocusChangeMsg struct{ Panel PanelID }
type ViewChangeMsg struct{ View ViewID }

// Mensagens de repositório
type RepoSelectMsg struct{ Repo *types.Repository }
type RepoRefreshMsg struct{}
type RepoListLoadedMsg struct{ Repos []*types.Repository }

// Mensagens de Git
type GitStatusMsg struct{ Status *git.Status }
type GitCommitMsg struct{ Hash string }
type GitPushMsg struct{ Success bool }
type GitPullMsg struct{ Changes int }
type GitBranchSwitchMsg struct{ Branch string }
type GitDiffMsg struct{ Diff string }

// Mensagens de provedor
type ProviderSwitchMsg struct{ Provider string }
type AuthRequiredMsg struct{ Provider string }
type AuthCompleteMsg struct{ Success bool }

// Mensagens de sistema
type ErrorMsg struct{ Err error }
type SuccessMsg struct{ Message string }
type NotificationMsg struct{ Level string; Message string }
```

---

## Tema

Cores definidas em JSON, carregadas via configuração:

```json
{
  "bg": "#1a1a2e",
  "surface": "#16213e",
  "accent": "#0f3460",
  "text": "#e0e0e0",
  "muted": "#a0a0a0",
  "success": "#4caf50",
  "warning": "#ff9800",
  "error": "#f44336"
}
```

O tema é aplicado via Lip Gloss:

```go
type Theme struct {
    Primary   lipgloss.Color
    Background lipgloss.Color
    Surface   lipgloss.Color
    Text      lipgloss.Color
    Muted     lipgloss.Color
    Success   lipgloss.Color
    Warning   lipgloss.Color
    Error     lipgloss.Color
    Info      lipgloss.Color
}
```

---

## Segurança

- **Tokens**: armazenados no keychain do SO (macOS Keychain, Linux secret-service,
  Windows Credential Manager)
- **Fallback**: arquivo `.github-desktop-tui/credentials.json` criptografado (AES-GCM)
- **Nunca** logar tokens ou credenciais
- **Suporte** a variáveis de ambiente: `GITHUB_TOKEN`, `GITLAB_TOKEN`, etc.

---

## CLI de Desenvolvimento

```bash
make dev          # go run ./cmd/github-desktop-tui
make build        # go build -o dist/github-desktop-tui ./cmd/github-desktop-tui
make test         # go test ./...
make lint         # golangci-lint run
make clean        # rm -rf dist/
```

---

## Roadmap (Fases)

### Fase 1 — Fundação (agora)
- [x] Configuração do projeto (Go modules, Makefile)
- [x] Arquitetura documentada
- [ ] Entry point funcional
- [ ] TUI com layout de 3 painéis
- [ ] Store central de estado
- [ ] Sistema de tema
- [ ] Provedor GitHub (autenticação + listar repos)
- [ ] Status bar
- [ ] Navegação entre painéis

### Fase 2 — Git Local
- [ ] Git operations wrapper (status, add, commit, push, pull)
- [ ] Commit log view
- [ ] Diff viewer
- [ ] Branch management
- [ ] File tree explorer

### Fase 3 — Multi-Provedor
- [ ] GitLab provider
- [ ] Bitbucket provider
- [ ] Gitea/Forgejo provider
- [ ] Provider switching
- [ ] OAuth flow (device code)

### Fase 4 — Polimento
- [ ] Search/filter
- [ ] Help overlay
- [ ] Temas customizáveis
- [ ] Performance optimization
- [ ] Testes E2E no terminal
- [ ] Documentação de usuário
