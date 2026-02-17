package aiassistant

import (
	"fmt"
	"log"
	"time"

	"github.com/willknow-ai/willknow-go/analyzer"
	"github.com/willknow-ai/willknow-go/indexer"
	"github.com/willknow-ai/willknow-go/provider"
	"github.com/willknow-ai/willknow-go/tools"
)

// Assistant is the main AI assistant instance
type Assistant struct {
	config       Config
	provider     provider.Provider
	toolRegistry *tools.Registry
	authManager  *AuthManager
	codeIndex    *indexer.CodeIndex
}

// New creates a new AI Assistant instance
func New(config Config) (*Assistant, error) {
	config.setDefaults()

	// Validate config
	if config.APIKey == "" {
		return nil, fmt.Errorf("APIKey is required")
	}

	// Create provider
	aiProvider, err := provider.NewProvider(provider.ProviderType(config.Provider), config.APIKey, config.Model, config.BaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to create provider: %w", err)
	}

	// Create tool registry
	toolRegistry := tools.NewRegistry(config.SourcePath)

	// Initialize auth manager
	authManager := newAuthManager(config.Auth)

	assistant := &Assistant{
		config:       config,
		provider:     aiProvider,
		toolRegistry: toolRegistry,
		authManager:  authManager,
	}

	// Auto-detect log files if not provided
	if len(config.LogFiles) == 0 {
		log.Println("[AI Assistant] No log files configured, attempting auto-detection...")
		logFiles, err := analyzer.DetectLogFiles(aiProvider, toolRegistry, config.SourcePath)
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

	// Build or load code index (if enabled)
	if config.EnableCodeIndex {
		const indexPath = "./code_index.json"
		const maxAge = 24 * time.Hour

		if indexer.IsIndexRecent(indexPath, maxAge) {
			log.Println("[AI Assistant] Loading existing code index...")
			codeIndex, err := indexer.LoadIndex(indexPath)
			if err != nil {
				log.Printf("[AI Assistant] Warning: Failed to load code index: %v", err)
				log.Println("[AI Assistant] Will build new index...")
			} else {
				assistant.codeIndex = codeIndex
				log.Printf("[AI Assistant] Code index loaded: %d files indexed", len(codeIndex.Files))
			}
		}

		// Build new index if not loaded
		if assistant.codeIndex == nil {
			log.Println("[AI Assistant] Building code index (this may take a few minutes)...")
			codeIndex, err := indexer.BuildCodeIndex(config.SourcePath, aiProvider)
			if err != nil {
				log.Printf("[AI Assistant] Warning: Failed to build code index: %v", err)
			} else {
				assistant.codeIndex = codeIndex
				log.Printf("[AI Assistant] Code index built: %d files indexed", len(codeIndex.Files))

				// Save index for future use
				if err := indexer.SaveIndex(indexPath, codeIndex); err != nil {
					log.Printf("[AI Assistant] Warning: Failed to save code index: %v", err)
				}
			}
		}

		// Register code index search tool if index is available
		if assistant.codeIndex != nil {
			toolRegistry.RegisterCodeIndexTool(assistant.codeIndex)
		}
	}

	return assistant, nil
}

// Start starts the AI Assistant web server
func (a *Assistant) Start() error {
	log.Printf("[AI Assistant] Starting on port %d...", a.config.Port)
	log.Printf("[AI Assistant] Source path: %s", a.config.SourcePath)
	log.Printf("[AI Assistant] Log files: %v", a.config.LogFiles)

	// Print auth startup message (password, open mode notice, etc.)
	a.authManager.printStartupMessage(a.config.Port)

	return startServer(a)
}
