// Package providers provides LLM provider abstraction for icooclaw.
package providers

import (
	"fmt"
	"log/slog"
	"sync"

	"icooclaw/pkg/consts"
	"icooclaw/pkg/providers/platforms"
	"icooclaw/pkg/storage"
)

// ProviderFactory creates a provider from configuration.
type ProviderFactory func(cfg *storage.Provider) Provider

type providerFactoryKey struct {
	providerType consts.ProviderType
	protocol     consts.ProviderProtocol
}

// Manager manages provider factories and instances.
type Manager struct {
	storage   *storage.Storage
	factories map[providerFactoryKey]ProviderFactory
	providers map[string]Provider
	logger    *slog.Logger
	mu        sync.RWMutex
}

// NewManager creates a new provider manager.
func NewManager(s *storage.Storage, logger *slog.Logger) *Manager {
	if logger == nil {
		logger = slog.Default()
	}

	m := &Manager{
		storage:   s,
		factories: make(map[providerFactoryKey]ProviderFactory),
		providers: make(map[string]Provider),
		logger:    logger,
	}
	m.RegisterBuiltins()
	return m
}

// RegisterBuiltins registers all built-in provider factories.
func (m *Manager) RegisterBuiltins() {
	m.RegisterFactory(consts.ProviderOpenAI, consts.ProtocolOpenAI, platforms.NewOpenAIProvider)
	m.RegisterFactory(consts.ProviderAnthropic, consts.ProtocolAnthropic, platforms.NewAnthropicProvider)
	m.RegisterFactory(consts.ProviderMiniMax, consts.ProtocolOpenAI, platforms.NewMiniMaxOpenAIProvider)
	m.RegisterFactory(consts.ProviderMiniMax, consts.ProtocolAnthropic, platforms.NewMiniMaxAnthropicProvider)
	m.RegisterFactory(consts.ProviderDeepSeek, consts.ProtocolOpenAI, platforms.NewDeepSeekProvider)
	m.RegisterFactory(consts.ProviderOpenRouter, consts.ProtocolOpenAI, platforms.NewOpenRouterProvider)
	m.RegisterFactory(consts.ProviderGemini, consts.ProtocolOpenAI, platforms.NewGeminiProvider)
	m.RegisterFactory(consts.ProviderMistral, consts.ProtocolOpenAI, platforms.NewMistralProvider)
	m.RegisterFactory(consts.ProviderGroq, consts.ProtocolOpenAI, platforms.NewGroqProvider)
	m.RegisterFactory(consts.ProviderAzure, consts.ProtocolOpenAI, platforms.NewAzureOpenAIProvider)
	m.RegisterFactory(consts.ProviderOllama, consts.ProtocolOpenAI, platforms.NewOllamaProvider)
	m.RegisterFactory(consts.ProviderMoonshot, consts.ProtocolOpenAI, platforms.NewMoonshotProvider)
	m.RegisterFactory(consts.ProviderZhipu, consts.ProtocolOpenAI, platforms.NewZhipuProvider)
	m.RegisterFactory(consts.ProviderQwen, consts.ProtocolOpenAI, platforms.NewQwenProvider)
	m.RegisterFactory(consts.ProviderSiliconFlow, consts.ProtocolOpenAI, platforms.NewSiliconFlowProvider)
	m.RegisterFactory(consts.ProviderGrok, consts.ProtocolOpenAI, platforms.NewGrokProvider)
}

// RegisterFactory registers a provider factory.
func (m *Manager) RegisterFactory(
	providerType consts.ProviderType,
	protocol consts.ProviderProtocol,
	factory ProviderFactory,
) {
	m.mu.Lock()
	defer m.mu.Unlock()

	key := providerFactoryKey{providerType: providerType, protocol: protocol}
	m.factories[key] = factory
	m.logger.Debug("provider factory registered", "type", providerType, "protocol", protocol)
}

// CreateProvider creates a provider from configuration.
func (m *Manager) CreateProvider(cfg *storage.Provider) (Provider, error) {
	if cfg == nil {
		return nil, fmt.Errorf("provider config is nil")
	}
	if cfg.Type == "" {
		return nil, fmt.Errorf("provider type is required")
	}
	if cfg.Protocol == "" {
		return nil, fmt.Errorf("provider protocol is required for type %s", cfg.Type)
	}

	key := providerFactoryKey{providerType: cfg.Type, protocol: cfg.Protocol}

	m.mu.RLock()
	factory, ok := m.factories[key]
	m.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("unsupported provider combination: type=%s protocol=%s", cfg.Type, cfg.Protocol)
	}

	provider := factory(cfg)
	if provider == nil {
		return nil, fmt.Errorf("provider factory returned nil: type=%s protocol=%s", cfg.Type, cfg.Protocol)
	}
	return provider, nil
}

// Register registers a provider instance.
func (m *Manager) Register(name string, provider Provider) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.providers[name] = provider
	m.logger.Debug("provider registered", "name", name, "type", provider.GetName())
}

// Get gets a provider by name. It loads from storage on cache miss.
func (m *Manager) Get(name string) (Provider, error) {
	m.mu.RLock()
	if provider, ok := m.providers[name]; ok {
		m.mu.RUnlock()
		return provider, nil
	}
	m.mu.RUnlock()

	if m.storage == nil {
		return nil, fmt.Errorf("provider not found: %s", name)
	}

	cfg, err := m.storage.Provider().GetByName(name)
	if err != nil {
		return nil, fmt.Errorf("provider %s not found: %w", name, err)
	}

	provider, err := m.CreateProvider(cfg)
	if err != nil {
		return nil, err
	}

	m.mu.Lock()
	m.providers[name] = provider
	m.mu.Unlock()
	return provider, nil
}

// List lists all registered provider names.
func (m *Manager) List() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	names := make([]string, 0, len(m.providers))
	for name := range m.providers {
		names = append(names, name)
	}
	return names
}

// LoadFromConfig loads providers from configuration.
func (m *Manager) LoadFromConfig(configs []*storage.Provider) error {
	for _, cfg := range configs {
		provider, err := m.CreateProvider(cfg)
		if err != nil {
			m.logger.Warn("failed to create provider", "name", cfg.Name, "error", err)
			continue
		}
		m.Register(cfg.Name, provider)
	}
	return nil
}

// ProviderInfo contains information about a provider.
type ProviderInfo struct {
	Name     string                  `json:"name"`
	Type     consts.ProviderType     `json:"type"`
	Protocol consts.ProviderProtocol `json:"protocol,omitempty"`
	Model    string                  `json:"model"`
	Models   []string                `json:"models,omitempty"`
}

// GetInfo returns information about a provider.
func (m *Manager) GetInfo(name string) (*ProviderInfo, error) {
	provider, err := m.Get(name)
	if err != nil {
		return nil, err
	}

	return &ProviderInfo{
		Name:  name,
		Type:  consts.ProviderType(provider.GetName()),
		Model: provider.GetModel(),
	}, nil
}

// ListInfo returns information about all cached providers.
func (m *Manager) ListInfo() []*ProviderInfo {
	m.mu.RLock()
	defer m.mu.RUnlock()

	infos := make([]*ProviderInfo, 0, len(m.providers))
	for name, provider := range m.providers {
		infos = append(infos, &ProviderInfo{
			Name:  name,
			Type:  consts.ProviderType(provider.GetName()),
			Model: provider.GetModel(),
		})
	}

	return infos
}
