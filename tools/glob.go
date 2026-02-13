package tools

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// GlobTool implements file pattern matching functionality
type GlobTool struct {
	sourcePath string
}

// Execute finds files matching a glob pattern
func (t *GlobTool) Execute(params map[string]interface{}) (string, error) {
	pattern, ok := params["pattern"].(string)
	if !ok {
		return "", fmt.Errorf("pattern parameter is required")
	}

	var matches []string

	// Walk the source directory
	err := filepath.Walk(t.sourcePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip common directories
		if info.IsDir() {
			if info.Name() == ".git" || info.Name() == "node_modules" || info.Name() == "vendor" {
				return filepath.SkipDir
			}
			return nil
		}

		// Get relative path
		relPath, err := filepath.Rel(t.sourcePath, path)
		if err != nil {
			return err
		}

		// Check if path matches the pattern
		matched, err := filepath.Match(pattern, filepath.Base(relPath))
		if err != nil {
			return err
		}

		// Also check if full relative path matches (for patterns like **/*.go)
		fullMatched := false
		if strings.Contains(pattern, "**") {
			// Simple ** pattern support
			simplifiedPattern := strings.ReplaceAll(pattern, "**", "*")
			fullMatched, _ = filepath.Match(simplifiedPattern, relPath)
		} else if strings.Contains(pattern, "/") {
			fullMatched, _ = filepath.Match(pattern, relPath)
		}

		if matched || fullMatched {
			matches = append(matches, relPath)
		}

		return nil
	})

	if err != nil {
		return "", fmt.Errorf("error walking source directory: %w", err)
	}

	if len(matches) == 0 {
		return fmt.Sprintf("No files found matching pattern: %s", pattern), nil
	}

	// Limit results
	if len(matches) > 100 {
		matches = matches[:100]
		matches = append(matches, "... (showing first 100 matches)")
	}

	result := fmt.Sprintf("Found %d files matching pattern: %s\n%s\n%s",
		len(matches),
		pattern,
		strings.Repeat("-", 80),
		strings.Join(matches, "\n"))

	return result, nil
}
