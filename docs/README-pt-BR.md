# ⌨️ GitHub Desktop TUI

**Cliente Git multi-provedor para o terminal.**

Uma interface bonita e funcional para operações Git, diretamente no seu terminal.
Suporta **GitHub, GitLab, Bitbucket e Gitea/Forgejo**.

```bash
npx github-desktop-tui
```

---

## ✨ Funcionalidades

| Funcionalidade | Descrição |
|---------------|-----------|
| **📋 Stage** | Selecione arquivos para stage/unstage com setas, faça commit com mensagem |
| **📜 Timeline** | Navegue pelo histórico completo de commits |
| **🔀 Branches** | Liste, crie, alterne, mescle e delete branches |
| **🚀 Push/Pull** | Sincronize com remotes com suporte a autenticação |
| **🔑 Autenticação** | Configure tokens diretamente ou via variáveis de ambiente |
| **🌐 Multi-Provedor** | GitHub, GitLab, Bitbucket, Gitea/Forgejo *(em progresso)* |
| **🎨 Temas** | Paleta de cores customizável com estilos Lip Gloss |
| **⌨️ Teclado** | Navegação completa por teclado, atalhos estilo vim |

---

## 📦 Instalação

### Via npm (recomendado)

```bash
npm install -g github-desktop-tui
npx github-desktop-tui
```

### Via Go

```bash
go install github.com/ElioNeto/github-desktop-tui/cmd/github-desktop-tui@latest
```

### Via fonte

```bash
git clone https://github.com/ElioNeto/github-desktop-tui.git
cd github-desktop-tui
make build
./dist/github-desktop-tui
```

---

## 🚀 Primeiros Passos

1. **P** → Abrir autenticação
2. Escolha um provedor e insira o token (ou nome da env var)
3. **r** → Atualizar e carregar repositórios
4. **c** → Ver mudanças e começar a fazer stage

### ⌨️ Atalhos Principais

| Tecla | Ação |
|-------|------|
| `c` | Abrir stage para commit |
| `↑↓` | Navegar na lista |
| `s` | Stage/unstage arquivo |
| `Enter` | Confirmar/comitar |
| `p` | Push |
| `l` | Pull |
| `b` | Lista de branches |
| `/` | Timeline de commits |
| `P` | Autenticação |
| `?` | Ajuda |
| `Tab` | Próximo painel |
| `q` | Sair |

---

## 🎨 Paleta de Cores

```
Background  #382a2a
Surface     #4a3a3a
Accent      #ff3d3d
Text        #e5ebbc
Muted       #8dc4b7
Success     #8dc4b7
Warning     #ff9d7d
Error       #ff3d3d
```

---

## 📄 Licença

MIT
