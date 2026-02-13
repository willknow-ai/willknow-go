package aiassistant

import (
	"fmt"
	"log"

	"github.com/dear/go-ai-assistant/analyzer"
	"github.com/dear/go-ai-assistant/claude"
	"github.com/dear/go-ai-assistant/tools"
)

// Assistant is the main AI assistant instance
type Assistant struct {
	config       Config
	claudeClient *claude.Client
	toolRegistry *tools.Registry
}

// New creates a new AI Assistant instance
func New(config Config) (*Assistant, error) {
	config.setDefaults()

	// Validate config
	if config.ClaudeAPIKey == "" {
		return nil, fmt.Errorf("ClaudeAPIKey is required")
	}

	// Create Claude client
	claudeClient := claude.NewClient(config.ClaudeAPIKey, config.Model)

	// Create tool registry
	toolRegistry := tools.NewRegistry(config.SourcePath)

	assistant := &Assistant{
		config:       config,
		claudeClient: claudeClient,
		toolRegistry: toolRegistry,
	}

	// Auto-detect log files if not provided
	if len(config.LogFiles) == 0 {
		log.Println("[AI Assistant] No log files configured, attempting auto-detection...")
		logFiles, err := analyzer.DetectLogFiles(claudeClient, toolRegistry, config.SourcePath)
		if err != nil {
			log.Printf("[AI Assistant] Warning: Failed to auto-detect log files: %v", err)
			log.Println("[AI Assistant] You may need to manually configure log files")
		} else {
			assistant.config.LogFiles = logFiles
			log.Printf("[AI Assistant] Auto-detected log files: %v", logFiles)
		}
	}

	// Register log query tool with detected log files
	toolRegistry.RegisterLogTool(assistant.config.LogFiles)

	return assistant, nil
}

// Start starts the AI Assistant web server
func (a *Assistant) Start() error {
	log.Printf("[AI Assistant] Starting on port %d...", a.config.Port)
	log.Printf("[AI Assistant] Source path: %s", a.config.SourcePath)
	log.Printf("[AI Assistant] Log files: %v", a.config.LogFiles)
	log.Printf("[AI Assistant] Web UI will be available at http://localhost:%d", a.config.Port)

	return startServer(a)
}
