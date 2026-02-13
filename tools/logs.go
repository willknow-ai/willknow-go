package tools

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// LogQueryTool implements log querying functionality
type LogQueryTool struct {
	logFiles []string
}

// Execute queries logs for a search pattern
func (t *LogQueryTool) Execute(params map[string]interface{}) (string, error) {
	query, ok := params["query"].(string)
	if !ok {
		return "", fmt.Errorf("query parameter is required")
	}

	contextLines := 5
	if cl, ok := params["context_lines"].(float64); ok {
		contextLines = int(cl)
	}

	if len(t.logFiles) == 0 {
		return "", fmt.Errorf("no log files configured")
	}

	var allMatches []string
	totalMatches := 0

	// Search in each log file
	for _, logFile := range t.logFiles {
		matches, err := t.searchLogFile(logFile, query, contextLines)
		if err != nil {
			// Log error but continue with other files
			allMatches = append(allMatches, fmt.Sprintf("Error reading %s: %v", logFile, err))
			continue
		}

		if len(matches) > 0 {
			allMatches = append(allMatches, fmt.Sprintf("\n=== Log file: %s ===", logFile))
			allMatches = append(allMatches, matches...)
			totalMatches += len(matches)
		}
	}

	if totalMatches == 0 {
		return fmt.Sprintf("No log entries found for query: %s", query), nil
	}

	result := fmt.Sprintf("Found %d log entries for query: %s\n%s\n%s",
		totalMatches,
		query,
		strings.Repeat("-", 80),
		strings.Join(allMatches, "\n"))

	return result, nil
}

// searchLogFile searches a single log file for the query
func (t *LogQueryTool) searchLogFile(logFile, query string, contextLines int) ([]string, error) {
	file, err := os.Open(logFile)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var matches []string
	var lines []string
	scanner := bufio.NewScanner(file)

	// Read all lines into memory (for context)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	// Search for matches
	for i, line := range lines {
		if t.matchesQuery(line, query) {
			// Add context
			start := i - contextLines
			if start < 0 {
				start = 0
			}
			end := i + contextLines + 1
			if end > len(lines) {
				end = len(lines)
			}

			// Build context block
			var contextBlock []string
			for j := start; j < end; j++ {
				prefix := "  "
				if j == i {
					prefix = "> " // Mark the matching line
				}
				contextBlock = append(contextBlock, fmt.Sprintf("%s%s", prefix, lines[j]))
			}

			matches = append(matches, strings.Join(contextBlock, "\n"))
			matches = append(matches, "") // Empty line between matches

			// Limit results
			if len(matches) >= 50 {
				matches = append(matches, "... (showing first ~50 matches)")
				break
			}
		}
	}

	return matches, nil
}

// matchesQuery checks if a log line matches the query
func (t *LogQueryTool) matchesQuery(line, query string) bool {
	// Try simple text search first
	if strings.Contains(strings.ToLower(line), strings.ToLower(query)) {
		return true
	}

	// Try to parse as JSON and search in fields
	var logEntry map[string]interface{}
	if err := json.Unmarshal([]byte(line), &logEntry); err == nil {
		// Successfully parsed as JSON
		// Search in common fields
		for _, value := range logEntry {
			if strValue, ok := value.(string); ok {
				if strings.Contains(strings.ToLower(strValue), strings.ToLower(query)) {
					return true
				}
			}
		}
	}

	return false
}
