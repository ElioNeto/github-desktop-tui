---
description: "Compilar o projeto TUI para produção"
---

Compilar o projeto github-desktop-tui para produção.

## Por stack

### Go
```bash
go build -ldflags="-s -w" -o dist/github-desktop-tui .
```

### Rust
```bash
cargo build --release
# binário em target/release/github-desktop-tui
```

### Bun/Node
```bash
bun run build
# saída em dist/
```

## Output
O binário compilado vai para `dist/` ou `target/release/`.
