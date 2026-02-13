package tools

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ReadFileTool implements file reading functionality
type ReadFileTool struct {
	sourcePath string
}

// Execute reads a file and returns its contents
func (t *ReadFileTool) Execute(params map[string]interface{}) (string, error) {
	filePath, ok := params["file_path"].(string)
	if !ok {
		return "", fmt.Errorf("file_path parameter is required")
	}

	// Build full path
	fullPath := filepath.Join(t.sourcePath, filePath)

	// Open file
	file, err := os.Open(fullPath)
	if err != nil {
		return "", fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	// Read file with line numbers
	scanner := bufio.NewScanner(file)
	var lines []string
	lineNum := 1

	// Get optional start and end line numbers
	startLine := 1
	endLine := -1 // -1 means read to end

	if sl, ok := params["start_line"].(float64); ok {
		startLine = int(sl)
	}
	if el, ok := params["end_line"].(float64); ok {
		endLine = int(el)
	}

	for scanner.Scan() {
		if lineNum >= startLine && (endLine == -1 || lineNum <= endLine) {
			lines = append(lines, fmt.Sprintf("%4d | %s", lineNum, scanner.Text()))
		}
		lineNum++
		if endLine != -1 && lineNum > endLine {
			break
		}
	}

	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("error reading file: %w", err)
	}

	if len(lines) == 0 {
		return "", fmt.Errorf("no lines found in specified range")
	}

	result := fmt.Sprintf("File: %s\n%s\n%s",
		filePath,
		strings.Repeat("-", 80),
		strings.Join(lines, "\n"))

	return result, nil
}
