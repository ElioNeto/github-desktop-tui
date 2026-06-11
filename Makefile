.PHONY: dev build test lint clean tidy install help

BINARY := github-desktop-tui
DIST_DIR := dist
LDFLAGS := -ldflags="-s -w"

## dev: Executa o aplicativo desktop em modo desenvolvimento
dev:
	go run ./cmd/$(BINARY)

## build: Compila o binário para produção
build:
	go build $(LDFLAGS) -o $(DIST_DIR)/$(BINARY) ./cmd/$(BINARY)

## build-linux: Compila para Linux (AppImage-style)
build-linux:
	go build $(LDFLAGS) -o $(DIST_DIR)/$(BINARY)-linux-amd64 ./cmd/$(BINARY)

## build-mac: Compila para macOS (cross-compile)
build-mac:
	GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o $(DIST_DIR)/$(BINARY)-darwin-amd64 ./cmd/$(BINARY)

## build-win: Compila para Windows (cross-compile)
build-win:
	GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o $(DIST_DIR)/$(BINARY)-windows-amd64.exe ./cmd/$(BINARY)

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
