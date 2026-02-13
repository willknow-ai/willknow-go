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

	// Provider is the AI provider to use (anthropic, deepseek)
	// Default: anthropic
	Provider string

	// APIKey is the API key for the provider
	APIKey string

	// Model is the model to use
	// Default: claude-3-5-sonnet-20241022 for anthropic, deepseek-chat for deepseek
	Model string
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
