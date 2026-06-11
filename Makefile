.PHONY: dev build test lint clean tidy

# Nome do binário
BINARY := github-desktop-tui

# Diretório de saída
DIST_DIR := dist

# Flags de build
LDFLAGS := -ldflags="-s -w"

## dev: Executa o programa em modo desenvolvimento
dev:
	go run ./cmd/$(BINARY)

## build: Compila o binário para produção
build:
	go build $(LDFLAGS) -o $(DIST_DIR)/$(BINARY) ./cmd/$(BINARY)

## test: Executa todos os testes
test:
	go test ./... -v -count=1

## lint: Executa o linter (golangci-lint)
lint:
	golangci-lint run ./...

## tidy: Limpa e atualiza as dependências
tidy:
	go mod tidy
	go mod verify

## clean: Remove artefatos de build
clean:
	rm -rf $(DIST_DIR)/
	go clean -cache

## install: Instala o binário no GOPATH
install:
	go install ./cmd/$(BINARY)

## help: Mostra esta mensagem de ajuda
help:
	@echo "Uso: make <comando>"
	@echo ""
	@echo "Comandos:"
	@grep -E '^## ' $(MAKEFILE_LIST) | sed 's/## /  /' | sort
