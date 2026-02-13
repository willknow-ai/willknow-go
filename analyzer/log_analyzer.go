package analyzer

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/willknow-ai/willknow-go/provider"
	"github.com/willknow-ai/willknow-go/tools"
)

const analyzeSystemPrompt = `You are a code analyzer. Your task is to find log file paths in the application code.

Look for:
1. Log configuration in code (e.g., logrus.SetOutput, zap configuration, log.SetOutput)
2. File paths passed to logging libraries
3. Common log file locations (/var/log/*.log, ./logs/*.log, etc.)

Search the code and return ONLY a JSON array of log file paths found, or common paths if nothing is found.

Example response:
["/var/log/app.log", "/var/log/error.log"]

If you cannot find any log configuration, return common paths:
["/var/log/app.log"]`

// DetectLogFiles uses AI to analyze source code and detect log file paths
func DetectLogFiles(aiProvider provider.Provider, toolRegistry *tools.Registry, sourcePath string) ([]string, error) {
	// Create initial message asking AI to find log files
	messages := []provider.Message{
		{
			Role: "user",
			Content: []provider.ContentBlock{
				{
					Type: "text",
					Text: "Please analyze the application source code and find the log file paths. Search for log configuration in the code using the grep tool to find logging setup (search for patterns like 'log', 'SetOutput', 'logrus', 'zap', etc.). Return a JSON array of log file paths.",
				},
			},
		},
	}

	// Get tool definitions
	toolDefs := toolRegistry.GetToolDefinitions()

	// Call Claude API with tools (allow up to 3 turns for AI to use tools)
	maxTurns := 3
	for turn := 0; turn < maxTurns; turn++ {
		log.Printf("[Analyzer] Turn %d: Calling AI API...", turn+1)

		response, err := aiProvider.SendMessage(messages, toolDefs, analyzeSystemPrompt)
		if err != nil {
			return nil, fmt.Errorf("failed to call AI API: %w", err)
		}

		// Check stop reason
		if response.StopReason == "end_turn" {
			// AI finished, extract log paths from response
			for _, block := range response.Content {
				if block.Type == "text" {
					return extractLogPaths(block.Text)
				}
			}
			return nil, fmt.Errorf("AI did not return log paths")
		}

		// Handle tool use
		if response.StopReason == "tool_use" {
			// Execute tools and add results to conversation
			var toolResults []provider.ContentBlock

			for _, block := range response.Content {
				if block.Type == "tool_use" {
					log.Printf("[Analyzer] Executing tool: %s", block.Name)
					result, err := toolRegistry.Execute(block.Name, block.Input)
					if err != nil {
						result = fmt.Sprintf("Error: %v", err)
					}

					toolResults = append(toolResults, provider.ContentBlock{
						Type:      "tool_result",
						ToolUseID: block.ID,
						Content:   result,
					})
				}
			}

			// Add assistant's response to messages
			messages = append(messages, provider.Message{
				Role:    "assistant",
				Content: response.Content,
			})

			// Add tool results
			messages = append(messages, provider.Message{
				Role:    "user",
				Content: toolResults,
			})

			// Continue to next turn
			continue
		}

		return nil, fmt.Errorf("unexpected stop reason: %s", response.StopReason)
	}

	// If we exhausted turns, try to extract from last response
	return []string{"/var/log/app.log"}, nil // Fallback
}

// extractLogPaths extracts log file paths from AI response text
func extractLogPaths(text string) ([]string, error) {
	// Try to find JSON array in the text
	start := strings.Index(text, "[")
	end := strings.LastIndex(text, "]")

	if start == -1 || end == -1 || start >= end {
		// No JSON found, return default
		return []string{"/var/log/app.log"}, nil
	}

	jsonStr := text[start : end+1]

	var paths []string
	if err := json.Unmarshal([]byte(jsonStr), &paths); err != nil {
		// Failed to parse, return default
		return []string{"/var/log/app.log"}, nil
	}

	if len(paths) == 0 {
		return []string{"/var/log/app.log"}, nil
	}

	return paths, nil
}
