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

	// EnableCodeIndex enables built-in code indexing using LLM-generated summaries.
	// When enabled, the assistant will scan source files at startup and build a searchable index.
	// The index is cached to ./code_index.json with 24-hour TTL.
	// Default: false (disabled)
	EnableCodeIndex bool

	// APISpec is the path to an OpenAPI spec file (YAML or JSON).
	// When configured, the assistant automatically becomes an AI agent capable of calling
	// the host system's APIs. This enables external AI systems to interact with the host
	// through a single natural-language chat interface (/willknow/chat).
	// Default: "" (disabled)
	APISpec string

	// HostBaseURL is the base URL for executing API calls when APISpec is configured.
	// Defaults to the first server URL in the OpenAPI spec.
	// Example: "http://localhost:8080"
	HostBaseURL string

	// AgentInfo describes this agent's identity for the /willknow/info discovery endpoint.
	// Defaults to values from the OpenAPI spec's info section.
	AgentInfo AgentInfo
}

// AgentInfo holds identity information for the agent discovery endpoint
type AgentInfo struct {
	// Name is the agent's display name
	// Default: OpenAPI spec's info.title
	Name string

	// Description describes what this agent can do
	// Default: OpenAPI spec's info.description
	Description string
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
	// EnableCodeIndex defaults to false (disabled)
	// Model defaults are set by the provider if not specified
}
