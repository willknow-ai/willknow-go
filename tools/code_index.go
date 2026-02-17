package tools

import (
	"fmt"
	"strings"

	"github.com/willknow-ai/willknow-go/indexer"
)

// CodeIndexTool implements semantic code search using LLM-generated summaries
type CodeIndexTool struct {
	codeIndex *indexer.CodeIndex
}

// Execute searches the code index for files matching the query
func (t *CodeIndexTool) Execute(params map[string]interface{}) (string, error) {
	query, ok := params["query"].(string)
	if !ok || query == "" {
		return "", fmt.Errorf("query parameter is required")
	}

	// Get optional limit parameter
	limit := 10
	if l, ok := params["limit"].(float64); ok {
		limit = int(l)
	}

	// Search the index
	results := t.codeIndex.Search(query, limit)

	if len(results) == 0 {
		return fmt.Sprintf("No files found matching query: %s\n\nTip: Try different keywords or use glob/grep tools for exact pattern matching.", query), nil
	}

	// Format results
	var output strings.Builder
	output.WriteString(fmt.Sprintf("Found %d file(s) matching '%s':\n", len(results), query))
	output.WriteString(strings.Repeat("-", 80))
	output.WriteString("\n\n")

	for i, file := range results {
		output.WriteString(fmt.Sprintf("%d. %s\n", i+1, file.Path))
		output.WriteString(fmt.Sprintf("   Summary: %s\n", file.Summary))
		output.WriteString(fmt.Sprintf("   Size: %d bytes, Last indexed: %s\n", file.Size, file.LastIndexed))
		output.WriteString("\n")
	}

	output.WriteString(strings.Repeat("-", 80))
	output.WriteString("\n")
	output.WriteString("💡 Use read_file to view the contents of relevant files.\n")

	return output.String(), nil
}
