package openapi

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// ExecuteTool executes an API tool call by making an HTTP request to the host
func ExecuteTool(tool *APITool, params map[string]interface{}, baseURL, authHeader string) (string, error) {
	// Build path with injected path parameters
	path := tool.Path
	queryParams := make(map[string]interface{})
	bodyParams := make(map[string]interface{})

	// Separate params into path, query, and body
	pathParamNames := make(map[string]bool)
	queryParamNames := make(map[string]bool)
	for _, p := range tool.Parameters {
		if p.In == "path" {
			pathParamNames[p.Name] = true
		} else if p.In == "query" {
			queryParamNames[p.Name] = true
		}
	}

	for name, value := range params {
		if pathParamNames[name] {
			// Inject into path
			path = strings.ReplaceAll(path, "{"+name+"}", fmt.Sprintf("%v", value))
		} else if queryParamNames[name] {
			queryParams[name] = value
		} else if tool.RequestBody != nil {
			// Assume it's a body param
			bodyParams[name] = value
		}
	}

	// Build full URL
	url := strings.TrimRight(baseURL, "/") + path
	if len(queryParams) > 0 {
		var qParts []string
		for k, v := range queryParams {
			qParts = append(qParts, fmt.Sprintf("%s=%v", k, v))
		}
		url += "?" + strings.Join(qParts, "&")
	}

	// Build request body
	var bodyReader io.Reader
	if len(bodyParams) > 0 {
		bodyBytes, err := json.Marshal(bodyParams)
		if err != nil {
			return "", fmt.Errorf("failed to marshal request body: %w", err)
		}
		bodyReader = bytes.NewReader(bodyBytes)
	}

	// Create HTTP request
	req, err := http.NewRequest(tool.Method, url, bodyReader)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	if len(bodyParams) > 0 {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Accept", "application/json")
	if authHeader != "" {
		req.Header.Set("Authorization", authHeader)
	}

	// Execute request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("API call failed: %w", err)
	}
	defer resp.Body.Close()

	// Read response
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	// Format result
	statusOK := resp.StatusCode >= 200 && resp.StatusCode < 300
	if !statusOK {
		return fmt.Sprintf("API call failed with status %d: %s", resp.StatusCode, string(respBody)), nil
	}

	// Pretty-print JSON response if possible
	var prettyJSON interface{}
	if json.Unmarshal(respBody, &prettyJSON) == nil {
		if pretty, err := json.MarshalIndent(prettyJSON, "", "  "); err == nil {
			return string(pretty), nil
		}
	}

	return string(respBody), nil
}

// FindTool looks up an APITool by name
func FindTool(tools []*APITool, name string) *APITool {
	for _, t := range tools {
		if t.Name == name {
			return t
		}
	}
	return nil
}
