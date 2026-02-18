package openapi

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/willknow-ai/willknow-go/provider"
)

// MaxTools is the maximum number of API tools loaded from the OpenAPI spec
const MaxTools = 50

// APITool represents a single API endpoint as a callable tool
type APITool struct {
	Name        string
	Summary     string
	Description string
	Method      string // GET, POST, PUT, PATCH, DELETE
	Path        string // e.g., /users/{userId}
	Parameters  []Parameter
	RequestBody *RequestBody
}

// Parameter represents a path or query parameter
type Parameter struct {
	Name        string
	In          string // "path" or "query"
	Description string
	Required    bool
	Type        string
}

// RequestBody represents the JSON body for POST/PUT/PATCH requests
type RequestBody struct {
	Description string
	Required    map[string]bool
	Properties  map[string]PropertySchema
}

// PropertySchema represents a single JSON property
type PropertySchema struct {
	Type        string
	Description string
}

// ParsedSpec holds the parsed OpenAPI specification
type ParsedSpec struct {
	Title       string
	Description string
	ServerURL   string
	Tools       []*APITool
}

// ParseSpec reads and parses an OpenAPI spec file (YAML or JSON)
func ParseSpec(specPath string) (*ParsedSpec, error) {
	data, err := os.ReadFile(specPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read spec file: %w", err)
	}

	var raw map[string]interface{}
	ext := strings.ToLower(filepath.Ext(specPath))
	if ext == ".yaml" || ext == ".yml" {
		if err := yaml.Unmarshal(data, &raw); err != nil {
			return nil, fmt.Errorf("failed to parse YAML spec: %w", err)
		}
	} else {
		if err := json.Unmarshal(data, &raw); err != nil {
			return nil, fmt.Errorf("failed to parse JSON spec: %w", err)
		}
	}

	return extractSpec(raw)
}

func extractSpec(raw map[string]interface{}) (*ParsedSpec, error) {
	spec := &ParsedSpec{}

	// Extract info
	if info, ok := raw["info"].(map[string]interface{}); ok {
		spec.Title = getString(info, "title")
		spec.Description = getString(info, "description")
	}

	// Extract server URL from first server entry
	if servers, ok := raw["servers"].([]interface{}); ok && len(servers) > 0 {
		if server, ok := servers[0].(map[string]interface{}); ok {
			spec.ServerURL = getString(server, "url")
		}
	}

	// Extract paths
	paths, ok := raw["paths"].(map[string]interface{})
	if !ok {
		return spec, nil
	}

	var tools []*APITool
	totalEndpoints := 0

	for path, pathItem := range paths {
		methods, ok := pathItem.(map[string]interface{})
		if !ok {
			continue
		}

		for method, operation := range methods {
			method = strings.ToUpper(method)
			if !isHTTPMethod(method) {
				continue
			}

			totalEndpoints++
			op, ok := operation.(map[string]interface{})
			if !ok {
				continue
			}

			if len(tools) >= MaxTools {
				continue
			}

			tool := extractTool(path, method, op)
			if tool != nil {
				tools = append(tools, tool)
			}
		}
	}

	if totalEndpoints > MaxTools {
		fmt.Printf("[Willknow] Warning: OpenAPI spec has %d endpoints, only the first %d are loaded. Future versions will support semantic tool selection.\n", totalEndpoints, MaxTools)
	}

	spec.Tools = tools
	return spec, nil
}

func extractTool(path, method string, op map[string]interface{}) *APITool {
	tool := &APITool{
		Method:  method,
		Path:    path,
		Summary: getString(op, "summary"),
	}

	// Name from operationId, fallback to generated name
	tool.Name = getString(op, "operationId")
	if tool.Name == "" {
		tool.Name = generateOperationID(method, path)
	}

	tool.Description = getString(op, "description")
	if tool.Description == "" {
		tool.Description = tool.Summary
	}
	if tool.Description == "" {
		tool.Description = fmt.Sprintf("%s %s", method, path)
	}

	// Extract path and query parameters
	if params, ok := op["parameters"].([]interface{}); ok {
		for _, p := range params {
			param, ok := p.(map[string]interface{})
			if !ok {
				continue
			}
			in := getString(param, "in")
			if in != "path" && in != "query" {
				continue // skip header, cookie params for simplicity
			}
			schema, _ := param["schema"].(map[string]interface{})
			tool.Parameters = append(tool.Parameters, Parameter{
				Name:        getString(param, "name"),
				In:          in,
				Description: getString(param, "description"),
				Required:    getBool(param, "required") || in == "path",
				Type:        getSchemaType(schema),
			})
		}
	}

	// Extract request body (JSON only)
	if rb, ok := op["requestBody"].(map[string]interface{}); ok {
		tool.RequestBody = extractRequestBody(rb)
	}

	return tool
}

func extractRequestBody(rb map[string]interface{}) *RequestBody {
	result := &RequestBody{
		Description: getString(rb, "description"),
		Required:    make(map[string]bool),
		Properties:  make(map[string]PropertySchema),
	}

	content, ok := rb["content"].(map[string]interface{})
	if !ok {
		return result
	}

	jsonContent, ok := content["application/json"].(map[string]interface{})
	if !ok {
		return result
	}

	schema, ok := jsonContent["schema"].(map[string]interface{})
	if !ok {
		return result
	}

	// Mark required fields
	for _, r := range getStringSlice(schema, "required") {
		result.Required[r] = true
	}

	props, ok := schema["properties"].(map[string]interface{})
	if !ok {
		return result
	}

	for propName, propDef := range props {
		prop, ok := propDef.(map[string]interface{})
		if !ok {
			continue
		}
		result.Properties[propName] = PropertySchema{
			Type:        getSchemaType(prop),
			Description: getString(prop, "description"),
		}
	}

	return result
}

// ToProviderTool converts an APITool to a provider.Tool definition for the LLM
func (t *APITool) ToProviderTool() provider.Tool {
	properties := make(map[string]interface{})
	var required []string

	// Path and query parameters
	for _, p := range t.Parameters {
		propType := p.Type
		if propType == "" {
			propType = "string"
		}
		properties[p.Name] = map[string]interface{}{
			"type":        propType,
			"description": p.Description,
		}
		if p.Required {
			required = append(required, p.Name)
		}
	}

	// Request body properties
	if t.RequestBody != nil {
		for name, schema := range t.RequestBody.Properties {
			propType := schema.Type
			if propType == "" {
				propType = "string"
			}
			properties[name] = map[string]interface{}{
				"type":        propType,
				"description": schema.Description,
			}
			if t.RequestBody.Required[name] {
				required = append(required, name)
			}
		}
	}

	inputSchema := map[string]interface{}{
		"type":       "object",
		"properties": properties,
	}
	if len(required) > 0 {
		inputSchema["required"] = required
	}

	return provider.Tool{
		Name:        t.Name,
		Description: t.Description,
		InputSchema: inputSchema,
	}
}

// Helper functions

func getString(m map[string]interface{}, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}

func getBool(m map[string]interface{}, key string) bool {
	if v, ok := m[key].(bool); ok {
		return v
	}
	return false
}

func getSchemaType(schema map[string]interface{}) string {
	if schema == nil {
		return "string"
	}
	t := getString(schema, "type")
	if t == "" {
		return "string"
	}
	return t
}

func getStringSlice(m map[string]interface{}, key string) []string {
	items, ok := m[key].([]interface{})
	if !ok {
		return nil
	}
	var result []string
	for _, item := range items {
		if s, ok := item.(string); ok {
			result = append(result, s)
		}
	}
	return result
}

func isHTTPMethod(method string) bool {
	switch method {
	case "GET", "POST", "PUT", "PATCH", "DELETE":
		return true
	}
	return false
}

func generateOperationID(method, path string) string {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	result := strings.ToLower(method)
	for _, part := range parts {
		part = strings.Trim(part, "{}")
		if len(part) > 0 {
			result += strings.ToUpper(part[:1]) + part[1:]
		}
	}
	return result
}
