package aiassistant

// Config holds the configuration for the AI Assistant
type Config struct {
	// SourcePath is the path to the application source code
	// Default: /app/source
	SourcePath string

	// LogFiles are the paths to log files
	// If empty, the assistant will try to auto-detect log files on startup
	LogFiles []string

	// Port is the port to run the web UI on
	// Default: 8888
	Port int

	// Provider is the AI provider to use
	// Supported: anthropic, openai, deepseek, qwen, moonshot, glm, xai, minimax, baichuan, 01ai, groq, together, siliconflow, custom
	// Default: anthropic
	Provider string

	// APIKey is the API key for the provider
	APIKey string

	// Model is the model to use
	// If empty, uses the provider's default model
	Model string

	// BaseURL is the custom API endpoint (for custom or self-hosted providers)
	// If empty, uses the provider's default endpoint
	// Required for Provider="custom"
	BaseURL string

	// Auth configures authentication for the AI assistant.
	// See AuthConfig for details on the three supported modes.
	Auth AuthConfig
}

// setDefaults sets default values for unspecified config fields
func (c *Config) setDefaults() {
	if c.SourcePath == "" {
		c.SourcePath = "/app/source"
	}
	if c.Port == 0 {
		c.Port = 8888
	}
	if c.Provider == "" {
		c.Provider = "anthropic"
	}
	// Model defaults are set by the provider if not specified
}
