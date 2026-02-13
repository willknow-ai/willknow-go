package provider

import "fmt"

// ProviderType represents the type of AI provider
type ProviderType string

const (
	ProviderAnthropic ProviderType = "anthropic"
	ProviderDeepSeek  ProviderType = "deepseek"
)

// NewProvider creates a new provider instance based on the provider type
func NewProvider(providerType ProviderType, apiKey, model string) (Provider, error) {
	switch providerType {
	case ProviderAnthropic:
		return NewAnthropicProvider(apiKey, model), nil
	case ProviderDeepSeek:
		return NewDeepSeekProvider(apiKey, model), nil
	default:
		return nil, fmt.Errorf("unsupported provider type: %s", providerType)
	}
}
