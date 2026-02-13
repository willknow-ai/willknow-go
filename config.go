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

	// ClaudeAPIKey is the API key for Claude
	ClaudeAPIKey string

	// Model is the Claude model to use
	// Default: claude-sonnet-4-5-20250929
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
	if c.Model == "" {
		c.Model = "claude-sonnet-4-5-20250929"
	}
}
