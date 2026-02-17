package tools

import (
	"fmt"

	"github.com/willknow-ai/willknow-go/indexer"
	"github.com/willknow-ai/willknow-go/provider"
)

// ToolExecutor is an interface for tool execution
type ToolExecutor interface {
	Execute(params map[string]interface{}) (string, error)
}

// Registry manages all available tools
type Registry struct {
	sourcePath    string
	tools         map[string]ToolExecutor
	logTool       *LogQueryTool
	codeIndexTool *CodeIndexTool
}

// NewRegistry creates a new tool registry
func NewRegistry(sourcePath string) *Registry {
	return &Registry{
		sourcePath: sourcePath,
		tools:      make(map[string]ToolExecutor),
	}
}

// RegisterLogTool registers the log query tool with log file paths
func (r *Registry) RegisterLogTool(logFiles []string) {
	r.logTool = &LogQueryTool{
		logFiles: logFiles,
	}
}

// RegisterCodeIndexTool registers the code index search tool
func (r *Registry) RegisterCodeIndexTool(codeIndex *indexer.CodeIndex) {
	r.codeIndexTool = &CodeIndexTool{
		codeIndex: codeIndex,
	}
}

// Execute executes a tool by name
func (r *Registry) Execute(name string, params map[string]interface{}) (string, error) {
	switch name {
	case "read_file":
		tool := &ReadFileTool{sourcePath: r.sourcePath}
		return tool.Execute(params)
	case "grep":
		tool := &GrepTool{sourcePath: r.sourcePath}
		return tool.Execute(params)
	case "glob":
		tool := &GlobTool{sourcePath: r.sourcePath}
		return tool.Execute(params)
	case "read_logs":
		if r.logTool == nil {
			return "", fmt.Errorf("log tool not configured")
		}
		return r.logTool.Execute(params)
	case "search_code_index":
		if r.codeIndexTool == nil {
			return "", fmt.Errorf("code index not available")
		}
		return r.codeIndexTool.Execute(params)
	default:
		return "", fmt.Errorf("unknown tool: %s", name)
	}
}

// GetToolDefinitions returns provider API tool definitions
func (r *Registry) GetToolDefinitions() []provider.Tool {
	tools := []provider.Tool{
		{
			Name:        "read_file",
			Description: "Read the contents of a file from the source code directory. Returns the file content with line numbers.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"file_path": map[string]interface{}{
						"type":        "string",
						"description": "The path to the file to read, relative to the source directory",
					},
					"start_line": map[string]interface{}{
						"type":        "integer",
						"description": "Optional: The line number to start reading from (1-indexed)",
					},
					"end_line": map[string]interface{}{
						"type":        "integer",
						"description": "Optional: The line number to stop reading at (inclusive)",
					},
				},
				"required": []string{"file_path"},
			},
		},
		{
			Name:        "grep",
			Description: "Search for a pattern in source code files using regex. Returns matching lines with file paths and line numbers.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"pattern": map[string]interface{}{
						"type":        "string",
						"description": "The regex pattern to search for",
					},
					"file_pattern": map[string]interface{}{
						"type":        "string",
						"description": "Optional: Limit search to files matching this glob pattern (e.g., '*.go', '**/*.js')",
					},
					"ignore_case": map[string]interface{}{
						"type":        "boolean",
						"description": "Optional: Whether to ignore case when matching",
					},
				},
				"required": []string{"pattern"},
			},
		},
		{
			Name:        "glob",
			Description: "Find files matching a glob pattern in the source code directory. Returns a list of matching file paths.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"pattern": map[string]interface{}{
						"type":        "string",
						"description": "The glob pattern to match (e.g., '*.go', '**/*.js', 'handlers/**')",
					},
				},
				"required": []string{"pattern"},
			},
		},
	}

	// Add log query tool if configured
	if r.logTool != nil {
		tools = append(tools, provider.Tool{
			Name:        "read_logs",
			Description: "Query application logs by request ID or search pattern. Returns relevant log entries with context.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"query": map[string]interface{}{
						"type":        "string",
						"description": "The search query (e.g., request ID, error message, or any text to search for)",
					},
					"context_lines": map[string]interface{}{
						"type":        "integer",
						"description": "Optional: Number of context lines to show before and after each match (default: 5)",
					},
				},
				"required": []string{"query"},
			},
		})
	}

	// Add code index search tool if available
	if r.codeIndexTool != nil {
		tools = append(tools, provider.Tool{
			Name:        "search_code_index",
			Description: "Search the codebase using semantic search. Finds files based on their purpose and functionality using LLM-generated summaries. Use this when you need to find files related to specific features or concepts (e.g., 'authentication', 'database connection', 'API handlers').",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"query": map[string]interface{}{
						"type":        "string",
						"description": "The search query describing what you're looking for (e.g., 'authentication', 'logging', 'user management')",
					},
					"limit": map[string]interface{}{
						"type":        "integer",
						"description": "Optional: Maximum number of results to return (default: 10)",
					},
				},
				"required": []string{"query"},
			},
		})
	}

	return tools
}
