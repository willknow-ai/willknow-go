package provider

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// OpenAICompatibleProvider implements the Provider interface for OpenAI-compatible APIs
type OpenAICompatibleProvider struct {
	apiKey     string
	model      string
	baseURL    string
	name       string
	httpClient *http.Client
}

// NewOpenAICompatibleProvider creates a new OpenAI-compatible provider
func NewOpenAICompatibleProvider(apiKey, model, baseURL, name string) *OpenAICompatibleProvider {
	return &OpenAICompatibleProvider{
		apiKey:     apiKey,
		model:      model,
		baseURL:    baseURL,
		name:       name,
		httpClient: &http.Client{},
	}
}

// GetName returns the provider name
func (p *OpenAICompatibleProvider) GetName() string {
	return p.name
}

// convertToOpenAITools converts provider tools to OpenAI format
func convertToOpenAITools(tools []Tool) []map[string]interface{} {
	if len(tools) == 0 {
		return nil
	}

	openAITools := make([]map[string]interface{}, 0, len(tools))
	for _, tool := range tools {
		openAITools = append(openAITools, map[string]interface{}{
			"type": "function",
			"function": map[string]interface{}{
				"name":        tool.Name,
				"description": tool.Description,
				"parameters":  tool.InputSchema,
			},
		})
	}
	return openAITools
}

// convertToOpenAIFormat converts provider messages to OpenAI format
func convertToOpenAIFormat(messages []Message) []map[string]interface{} {
	openAIMessages := make([]map[string]interface{}, 0, len(messages))

	for _, msg := range messages {
		// Handle tool_use and tool_result in content blocks
		var textContent string
		var toolCalls []map[string]interface{}
		var toolCallID string

		for _, block := range msg.Content {
			switch block.Type {
			case "text":
				textContent += block.Text
			case "tool_use":
				// Convert to OpenAI tool call format
				toolCalls = append(toolCalls, map[string]interface{}{
					"id":   block.ID,
					"type": "function",
					"function": map[string]interface{}{
						"name":      block.Name,
						"arguments": mustMarshalJSON(block.Input),
					},
				})
			case "tool_result":
				// Tool results are sent as tool messages in OpenAI format
				toolCallID = block.ToolUseID
				textContent = block.Content
			}
		}

		// Create message based on content
		if msg.Role == "assistant" && len(toolCalls) > 0 {
			// Assistant message with tool calls
			msgMap := map[string]interface{}{
				"role":       "assistant",
				"tool_calls": toolCalls,
			}
			if textContent != "" {
				msgMap["content"] = textContent
			}
			openAIMessages = append(openAIMessages, msgMap)
		} else if msg.Role == "user" && toolCallID != "" {
			// Tool result message
			openAIMessages = append(openAIMessages, map[string]interface{}{
				"role":         "tool",
				"tool_call_id": toolCallID,
				"content":      textContent,
			})
		} else if textContent != "" {
			// Regular text message
			openAIMessages = append(openAIMessages, map[string]interface{}{
				"role":    msg.Role,
				"content": textContent,
			})
		}
	}

	return openAIMessages
}

// mustMarshalJSON marshals data to JSON string, returns empty object on error
func mustMarshalJSON(data interface{}) string {
	if data == nil {
		return "{}"
	}
	b, err := json.Marshal(data)
	if err != nil {
		return "{}"
	}
	return string(b)
}

// convertFromOpenAIFormat converts OpenAI response to provider format
func convertFromOpenAIFormat(openAIResp map[string]interface{}) (*Response, error) {
	response := &Response{
		Type: "message",
		Role: "assistant",
	}

	if id, ok := openAIResp["id"].(string); ok {
		response.ID = id
	}

	if model, ok := openAIResp["model"].(string); ok {
		response.Model = model
	}

	// Extract choices
	if choices, ok := openAIResp["choices"].([]interface{}); ok && len(choices) > 0 {
		if choice, ok := choices[0].(map[string]interface{}); ok {
			if message, ok := choice["message"].(map[string]interface{}); ok {
				// Handle text content
				if content, ok := message["content"].(string); ok && content != "" {
					response.Content = append(response.Content, ContentBlock{
						Type: "text",
						Text: content,
					})
				}

				// Handle tool calls
				if toolCalls, ok := message["tool_calls"].([]interface{}); ok {
					for _, tc := range toolCalls {
						if toolCall, ok := tc.(map[string]interface{}); ok {
							if function, ok := toolCall["function"].(map[string]interface{}); ok {
								// Parse arguments JSON
								var input map[string]interface{}
								if argsStr, ok := function["arguments"].(string); ok {
									json.Unmarshal([]byte(argsStr), &input)
								}

								// Safely get ID and name
								id, _ := toolCall["id"].(string)
								name, _ := function["name"].(string)

								response.Content = append(response.Content, ContentBlock{
									Type:  "tool_use",
									ID:    id,
									Name:  name,
									Input: input,
								})
							}
						}
					}
				}
			}

			if finishReason, ok := choice["finish_reason"].(string); ok {
				// Map OpenAI finish reasons to Anthropic format
				switch finishReason {
				case "tool_calls":
					response.StopReason = "tool_use"
				case "stop":
					response.StopReason = "end_turn"
				default:
					response.StopReason = finishReason
				}
			}
		}
	}

	// Extract usage
	if usage, ok := openAIResp["usage"].(map[string]interface{}); ok {
		if promptTokens, ok := usage["prompt_tokens"].(float64); ok {
			response.Usage.InputTokens = int(promptTokens)
		}
		if completionTokens, ok := usage["completion_tokens"].(float64); ok {
			response.Usage.OutputTokens = int(completionTokens)
		}
	}

	return response, nil
}

// SendMessage sends a message and returns the response
func (p *OpenAICompatibleProvider) SendMessage(messages []Message, tools []Tool, system string) (*Response, error) {
	openAIMessages := convertToOpenAIFormat(messages)

	// Add system message if provided
	if system != "" {
		openAIMessages = append([]map[string]interface{}{
			{
				"role":    "system",
				"content": system,
			},
		}, openAIMessages...)
	}

	req := map[string]interface{}{
		"model":    p.model,
		"messages": openAIMessages,
	}

	// Add tools if provided
	if len(tools) > 0 {
		req["tools"] = convertToOpenAITools(tools)
	}

	reqBody, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	apiURL := p.baseURL + "/chat/completions"
	httpReq, err := http.NewRequest("POST", apiURL, bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+p.apiKey)

	resp, err := p.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var openAIResp map[string]interface{}
	if err := json.Unmarshal(body, &openAIResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return convertFromOpenAIFormat(openAIResp)
}

// SendMessageStream sends a message and returns a streaming response
func (p *OpenAICompatibleProvider) SendMessageStream(messages []Message, tools []Tool, system string) (io.ReadCloser, error) {
	openAIMessages := convertToOpenAIFormat(messages)

	// Add system message if provided
	if system != "" {
		openAIMessages = append([]map[string]interface{}{
			{
				"role":    "system",
				"content": system,
			},
		}, openAIMessages...)
	}

	req := map[string]interface{}{
		"model":    p.model,
		"messages": openAIMessages,
		"stream":   true,
	}

	// Add tools if provided
	if len(tools) > 0 {
		req["tools"] = convertToOpenAITools(tools)
	}

	reqBody, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	apiURL := p.baseURL + "/chat/completions"
	httpReq, err := http.NewRequest("POST", apiURL, bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+p.apiKey)
	httpReq.Header.Set("Accept", "text/event-stream")

	resp, err := p.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	return resp.Body, nil
}
