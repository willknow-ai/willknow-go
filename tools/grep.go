package tools

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// GrepTool implements code search functionality
type GrepTool struct {
	sourcePath string
}

// Execute searches for a pattern in source files
func (t *GrepTool) Execute(params map[string]interface{}) (string, error) {
	pattern, ok := params["pattern"].(string)
	if !ok {
		return "", fmt.Errorf("pattern parameter is required")
	}

	// Get optional parameters
	filePattern := "**/*"
	if fp, ok := params["file_pattern"].(string); ok {
		filePattern = fp
	}

	ignoreCase := false
	if ic, ok := params["ignore_case"].(bool); ok {
		ignoreCase = ic
	}

	// Compile regex
	flags := ""
	if ignoreCase {
		flags = "(?i)"
	}
	regex, err := regexp.Compile(flags + pattern)
	if err != nil {
		return "", fmt.Errorf("invalid regex pattern: %w", err)
	}

	// Find files to search
	var filesToSearch []string
	err = filepath.Walk(t.sourcePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			// Skip common directories
			if info.Name() == ".git" || info.Name() == "node_modules" || info.Name() == "vendor" {
				return filepath.SkipDir
			}
			return nil
		}

		// Check if file matches the file pattern
		relPath, _ := filepath.Rel(t.sourcePath, path)
		matched, _ := filepath.Match(filePattern, filepath.Base(path))
		if matched || filePattern == "**/*" {
			// Also check for common code file extensions
			ext := filepath.Ext(path)
			if ext == ".go" || ext == ".js" || ext == ".ts" || ext == ".py" ||
			   ext == ".java" || ext == ".rb" || ext == ".php" || ext == ".c" ||
			   ext == ".cpp" || ext == ".h" || ext == ".rs" || ext == ".md" ||
			   ext == ".yaml" || ext == ".yml" || ext == ".json" || ext == ".xml" {
				filesToSearch = append(filesToSearch, relPath)
			}
		}
		return nil
	})

	if err != nil {
		return "", fmt.Errorf("error walking source directory: %w", err)
	}

	// Search in files
	var matches []string
	matchCount := 0

	for _, relPath := range filesToSearch {
		fullPath := filepath.Join(t.sourcePath, relPath)
		file, err := os.Open(fullPath)
		if err != nil {
			continue
		}

		scanner := bufio.NewScanner(file)
		lineNum := 1
		for scanner.Scan() {
			line := scanner.Text()
			if regex.MatchString(line) {
				matches = append(matches, fmt.Sprintf("%s:%d: %s", relPath, lineNum, line))
				matchCount++
				if matchCount >= 100 {
					// Limit results to prevent overwhelming output
					matches = append(matches, fmt.Sprintf("\n... (showing first 100 matches)"))
					file.Close()
					goto done
				}
			}
			lineNum++
		}
		file.Close()
	}

done:
	if len(matches) == 0 {
		return fmt.Sprintf("No matches found for pattern: %s", pattern), nil
	}

	result := fmt.Sprintf("Found %d matches for pattern: %s\n%s\n%s",
		matchCount,
		pattern,
		strings.Repeat("-", 80),
		strings.Join(matches, "\n"))

	return result, nil
}
