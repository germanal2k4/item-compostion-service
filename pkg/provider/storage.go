package provider

import (
	"fmt"
	"sync"
)

type ProviderStorage struct {
	mu        sync.RWMutex
	providers map[string]Provider
}

func NewProviderStorage() *ProviderStorage {
	return &ProviderStorage{
		providers: make(map[string]Provider),
	}
}
func (p *ProviderStorage) GetProvider(providerName string) (Provider, error) {
	p.mu.RLock()
	provider, exists := p.providers[providerName]
	p.mu.RUnlock()
	if !exists {
		return nil, fmt.Errorf("provider %s not found", providerName)
	}

	return provider, nil
}

func (p *ProviderStorage) RegisterProvider(provider Provider) {
	p.mu.Lock()
	p.providers[provider.GetName()] = provider
	p.mu.Unlock()
}
