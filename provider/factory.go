package provider

import "fmt"

// ProviderType represents the type of AI provider
type ProviderType string

const (
	ProviderAnthropic ProviderType = "anthropic"
	ProviderDeepSeek  ProviderType = "deepseek"
)

// NewProvider creates a new provider instance based on the provider type
func NewProvider(providerType ProviderType, apiKey, model, baseURL string) (Provider, error) {
	// Special handling for Anthropic (non-OpenAI compatible)
	if providerType == ProviderAnthropic {
		return NewAnthropicProvider(apiKey, model), nil
	}

	// Get preset configuration
	preset, exists := Presets[providerType]
	if !exists {
		return nil, fmt.Errorf("unsupported provider type: %s", providerType)
	}

	// Use custom BaseURL if provided, otherwise use preset
	finalBaseURL := baseURL
	if finalBaseURL == "" {
		finalBaseURL = preset.BaseURL
	}

	// Use custom model if provided, otherwise use preset default
	finalModel := model
	if finalModel == "" {
		finalModel = preset.DefaultModel
	}

	// Validate BaseURL for custom provider
	if providerType == "custom" && finalBaseURL == "" {
		return nil, fmt.Errorf("BaseURL is required for custom provider")
	}

	// Create OpenAI-compatible provider
	return NewOpenAICompatibleProvider(apiKey, finalModel, finalBaseURL, preset.Name), nil
}
