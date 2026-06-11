package github

import (
	"context"
	"fmt"
	"os"

	"github.com/ElioNeto/github-desktop-tui/internal/providers"
)

// Auth handles GitHub authentication.
type Auth struct {
	token string
}

// NewAuth creates a new GitHub auth handler.
func NewAuth() *Auth {
	return &Auth{}
}

// GetToken retrieves the GitHub token from environment, keychain, or prompt.
func (a *Auth) GetToken(ctx context.Context) (string, error) {
	// 1. Try environment variable
	if token := os.Getenv("GITHUB_TOKEN"); token != "" {
		a.token = token
		return token, nil
	}

	// 2. Try GITHUB_ENTERPRISE_TOKEN (for Enterprise users)
	if token := os.Getenv("GITHUB_ENTERPRISE_TOKEN"); token != "" {
		a.token = token
		return token, nil
	}

	// 3. Try GH_TOKEN (GitHub CLI compatible)
	if token := os.Getenv("GH_TOKEN"); token != "" {
		a.token = token
		return token, nil
	}

	return "", fmt.Errorf("token do GitHub não encontrado. Defina GITHUB_TOKEN, GH_TOKEN ou use 'gh auth login'")
}

// SetToken sets the token programmatically.
func (a *Auth) SetToken(token string) {
	a.token = token
}

// Validate checks if the token is valid by calling the GitHub API.
func (a *Auth) Validate(ctx context.Context) error {
	if a.token == "" {
		return fmt.Errorf("token não configurado")
	}

	client := NewClient(a.token, "")
	// Simple validation: try to get the authenticated user
	_, err := client.ListRepos(ctx)
	if err != nil {
		return fmt.Errorf("token inválido: %w", err)
	}

	return nil
}

// Authenticate performs the authentication flow.
func (a *Auth) Authenticate(ctx context.Context) (*providers.AuthResult, error) {
	token, err := a.GetToken(ctx)
	if err != nil {
		return &providers.AuthResult{
			Success: false,
			Error:   err,
		}, nil
	}

	client := NewClient(token, "")
	if err := a.Validate(ctx); err != nil {
		return &providers.AuthResult{
			Success: false,
			Error:   err,
		}, nil
	}

	a.token = token
	_ = client // client is ready to use

	return &providers.AuthResult{
		Success: true,
		Token:   token,
	}, nil
}
