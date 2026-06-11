package auth

import (
	"fmt"
	"os"
	"sync"
)

// AuthMethod indica como o token é obtido.
type AuthMethod string

const (
	AuthMethodDirect   AuthMethod = "direct"   // Token digitado diretamente
	AuthMethodEnvVar   AuthMethod = "envvar"   // Referência a variável de ambiente
	AuthMethodKeychain AuthMethod = "keychain" // Keychain do SO (futuro)
)

// ProviderAuth guarda as credenciais de um provedor.
type ProviderAuth struct {
	Provider string
	Method   AuthMethod
	Token    string  // Token direto OU nome da env var
	Username string
	BaseURL  string
}

// Manager gerencia autenticação de múltiplos provedores.
type Manager struct {
	mu        sync.RWMutex
	providers map[string]*ProviderAuth
}

// NewManager cria um novo gerenciador de autenticação.
func NewManager() *Manager {
	return &Manager{
		providers: make(map[string]*ProviderAuth),
	}
}

// SetToken armazena um token para o provedor.
// Se method for "envvar", value é o NOME da variável de ambiente.
func (m *Manager) SetToken(provider string, method AuthMethod, value string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if existing, ok := m.providers[provider]; ok {
		existing.Method = method
		existing.Token = value
	} else {
		m.providers[provider] = &ProviderAuth{
			Provider: provider,
			Method:   method,
			Token:    value,
		}
	}
}

// GetToken recupera o token resolvido para o provedor.
// Se o método for envvar, lê a variável de ambiente real via os.Getenv.
// Se o método for direct, retorna o token diretamente.
func (m *Manager) GetToken(provider string) (string, error) {
	m.mu.RLock()
	pa, ok := m.providers[provider]
	m.mu.RUnlock()

	if !ok {
		return "", fmt.Errorf("auth: no token configured for provider %q", provider)
	}

	switch pa.Method {
	case AuthMethodDirect:
		if pa.Token == "" {
			return "", fmt.Errorf("auth: token for provider %q is empty", provider)
		}
		return pa.Token, nil

	case AuthMethodEnvVar:
		if pa.Token == "" {
			return "", fmt.Errorf("auth: env var name not configured for provider %q", provider)
		}
		token := os.Getenv(pa.Token)
		if token == "" {
			return "", fmt.Errorf("auth: environment variable %q is empty or not set for provider %q", pa.Token, provider)
		}
		return token, nil

	case AuthMethodKeychain:
		return "", fmt.Errorf("auth: keychain method not implemented yet for provider %q", provider)

	default:
		return "", fmt.Errorf("auth: unknown auth method %q for provider %q", pa.Method, provider)
	}
}

// IsAuthenticated verifica se o provedor tem token válido.
// Tenta resolver o token (lendo env var se necessário) e retorna true se
// o token for não-vazio.
func (m *Manager) IsAuthenticated(provider string) bool {
	token, err := m.GetToken(provider)
	return err == nil && token != ""
}

// HasTokenConfig verifica se o provedor tem configuração de token (sem resolver).
// Retorna true se o token foi configurado (via SetToken), independentemente de
// ser um valor direto, nome de env var, etc. Não resolve/env var.
func (m *Manager) HasTokenConfig(provider string) bool {
	m.mu.RLock()
	_, ok := m.providers[provider]
	m.mu.RUnlock()
	return ok
}

// GetMethod retorna o método de autenticação configurado.
func (m *Manager) GetMethod(provider string) (AuthMethod, error) {
	m.mu.RLock()
	pa, ok := m.providers[provider]
	m.mu.RUnlock()

	if !ok {
		return "", fmt.Errorf("auth: no token configured for provider %q", provider)
	}
	return pa.Method, nil
}

// RemoveToken remove o token de um provedor.
func (m *Manager) RemoveToken(provider string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.providers, provider)
}

// ListProviders retorna lista de providers com token configurado.
func (m *Manager) ListProviders() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	providers := make([]string, 0, len(m.providers))
	for name := range m.providers {
		providers = append(providers, name)
	}
	return providers
}

// KnownEnvVars retorna uma lista de nomes de env vars conhecidos para autocomplete.
func KnownEnvVars(provider string) []string {
	switch provider {
	case "github":
		return []string{"GITHUB_TOKEN", "GH_TOKEN", "GITHUB_ENTERPRISE_TOKEN"}
	case "gitlab":
		return []string{"GITLAB_TOKEN", "GITLAB_ACCESS_TOKEN"}
	case "bitbucket":
		return []string{"BITBUCKET_TOKEN", "BITBUCKET_ACCESS_TOKEN"}
	case "gitea":
		return []string{"GITEA_TOKEN", "FORGEJO_TOKEN"}
	default:
		return nil
	}
}
