# Contributing to GitHub Desktop TUI

Thank you for your interest in contributing! This document provides guidelines for contributing to the project.

## Code of Conduct

Please be respectful and constructive in all interactions.

## Getting Started

1. Fork the repository
2. Clone your fork: `git clone https://github.com/YOUR_USERNAME/github-desktop-tui.git`
3. Create a feature branch: `git checkout -b feat/your-feature`
4. Install Go 1.22+
5. Run `make dev` to start the development server

## Development Workflow

```bash
# Run in development mode
make dev

# Build for production
make build

# Run tests
make test

# Run linter
make lint

# Clean build artifacts
make clean
```

## Project Structure

```
github-desktop-tui/
├── cmd/           # Entry point
├── internal/      # Core packages
│   ├── app/       # Application bootstrap
│   ├── tui/       # Terminal UI (views, components, theme)
│   ├── git/       # Git operations (go-git)
│   ├── providers/ # Git provider interfaces + implementations
│   ├── store/     # State management
│   ├── auth/      # Authentication
│   └── config/    # Configuration
├── pkg/           # Shared types
└── docs/          # Documentation
```

## Pull Request Guidelines

1. Keep changes focused and surgical
2. Write clear commit messages following conventional commits
3. Add tests for new functionality
4. Ensure `make test` and `make lint` pass
5. Update documentation if needed
6. Reference related issues in the PR description

## Commit Message Format

```
<type>: <description>

[optional body]

[optional footer]
```

Types:
- `feat`: New feature
- `fix`: Bug fix
- `refactor`: Code refactoring
- `docs`: Documentation changes
- `test`: Test additions/changes
- `style`: Code style changes
- `chore`: Build/config changes

## Issues

- Search existing issues before creating a new one
- Use the provided issue templates
- Label issues appropriately (bug, enhancement, etc.)
- For bugs, include reproduction steps and environment info

## Questions?

Open a [Discussion](https://github.com/ElioNeto/github-desktop-tui/discussions) or check existing issues.
