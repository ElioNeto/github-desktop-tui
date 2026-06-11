---
description: "Iniciar servidor de desenvolvimento com hot-reload para a TUI"
---

Iniciar o servidor de desenvolvimento para o projeto github-desktop-tui.

## Detecção automática de stack
- **Go (Bubble Tea)**: `air` (hot-reload) ou `go run .`
- **Rust (Ratatui)**: `cargo watch -x run`
- **Bun/Node (Ink)**: `bun run dev`

## Verificar antes
1. Veja se existe `package.json`, `go.mod` ou `Cargo.toml` na raiz
2. Veja se existe `dev` script no `package.json`
3. Execute o comando apropriado

```bash
# Go com air
air

# Rust
cargo watch -x run

# Node/Bun
bun run dev
```
