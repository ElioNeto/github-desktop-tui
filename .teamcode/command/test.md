---
description: "Executar testes da TUI multi-provider Git"
---

Executar testes para o projeto github-desktop-tui.

## Stack

### Go
```bash
go test ./... -v
```

### Rust
```bash
cargo test
```

### Bun/Node
```bash
bun test
```

## Teste específico
```bash
# Go
go test ./src/provider/github/ -v

# Rust
cargo test test_name

# Bun
bun test src/provider/github.test.ts
```
