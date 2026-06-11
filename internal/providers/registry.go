package providers

import (
	"fmt"
	"sync"
)

// Registry manages all registered Git providers.
type Registry struct {
	mu       sync.RWMutex
	registry map[string]GitProvider
	active   string
}

// NewRegistry creates a new Registry with no registered providers.
// Providers must be registered via Register() before use.
func NewRegistry() *Registry {
	return &Registry{
		registry: make(map[string]GitProvider),
		active:   "github",
	}
}

// Register adds a provider to the registry.
func (r *Registry) Register(p GitProvider) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.registry[p.Name()] = p
}

// Get returns a provider by name.
func (r *Registry) Get(name string) (GitProvider, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	p, ok := r.registry[name]
	if !ok {
		return nil, fmt.Errorf("provider %q não encontrado", name)
	}
	return p, nil
}

// Active returns the currently active provider.
func (r *Registry) Active() GitProvider {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.registry[r.active]
}

// SetActive switches the active provider.
func (r *Registry) SetActive(name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.registry[name]; !ok {
		return fmt.Errorf("provider %q não encontrado", name)
	}
	r.active = name
	return nil
}

// List returns all registered provider names.
func (r *Registry) List() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	names := make([]string, 0, len(r.registry))
	for name := range r.registry {
		names = append(names, name)
	}
	return names
}

// All returns all registered providers.
func (r *Registry) All() []GitProvider {
	r.mu.RLock()
	defer r.mu.RUnlock()
	providers := make([]GitProvider, 0, len(r.registry))
	for _, p := range r.registry {
		providers = append(providers, p)
	}
	return providers
}
