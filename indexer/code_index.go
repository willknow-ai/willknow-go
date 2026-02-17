package indexer

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/willknow-ai/willknow-go/provider"
)

// CodeIndex represents an index of code files with their summaries
type CodeIndex struct {
	Files      map[string]FileSummary `json:"files"`
	CreatedAt  time.Time              `json:"created_at"`
	SourcePath string                 `json:"source_path"`
}

// FileSummary contains metadata and summary for a source file
type FileSummary struct {
	Path        string `json:"path"`
	Summary     string `json:"summary"`
	Size        int64  `json:"size"`
	LastIndexed string `json:"last_indexed"`
}

// BuildCodeIndex scans the source directory and generates summaries using LLM
func BuildCodeIndex(sourcePath string, llm provider.Provider) (*CodeIndex, error) {
	files, err := scanGoFiles(sourcePath)
	if err != nil {
		return nil, fmt.Errorf("failed to scan files: %w", err)
	}

	index := &CodeIndex{
		Files:      make(map[string]FileSummary),
		CreatedAt:  time.Now(),
		SourcePath: sourcePath,
	}

	// Summarize each file using LLM
	for _, file := range files {
		summary, err := summarizeFile(file, llm)
		if err != nil {
			// Log error but continue with other files
			fmt.Printf("[Code Index] Warning: failed to summarize %s: %v\n", file, err)
			continue
		}

		fileInfo, _ := os.Stat(file)
		relativePath := strings.TrimPrefix(file, sourcePath+"/")

		index.Files[relativePath] = FileSummary{
			Path:        relativePath,
			Summary:     summary,
			Size:        fileInfo.Size(),
			LastIndexed: time.Now().Format(time.RFC3339),
		}
	}

	return index, nil
}

// scanGoFiles recursively scans for .go files in the source directory
func scanGoFiles(sourcePath string) ([]string, error) {
	var files []string

	err := filepath.Walk(sourcePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip vendor and hidden directories
		if info.IsDir() {
			name := info.Name()
			if name == "vendor" || name == ".git" || strings.HasPrefix(name, ".") {
				return filepath.SkipDir
			}
			return nil
		}

		// Only index .go files
		if filepath.Ext(path) == ".go" {
			files = append(files, path)
		}

		return nil
	})

	return files, err
}

// summarizeFile reads a file and asks LLM to summarize its purpose
func summarizeFile(filePath string, llm provider.Provider) (string, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return "", err
	}

	// Truncate very large files to avoid token limits
	const maxChars = 8000
	fileContent := string(content)
	if len(fileContent) > maxChars {
		fileContent = fileContent[:maxChars] + "\n... [truncated]"
	}

	prompt := fmt.Sprintf(`请用一句话（不超过 50 字）总结这个 Go 源文件的核心功能。

只返回摘要文字，不要加任何前缀或解释。

文件内容：
%s`, fileContent)

	messages := []provider.Message{
		{
			Role: "user",
			Content: []provider.ContentBlock{
				{Type: "text", Text: prompt},
			},
		},
	}

	response, err := llm.SendMessage(messages, nil, "")
	if err != nil {
		return "", err
	}

	// Extract text from response
	var summary string
	for _, block := range response.Content {
		if block.Type == "text" {
			summary += block.Text
		}
	}

	return strings.TrimSpace(summary), nil
}

// LoadIndex loads an existing index from a file
func LoadIndex(indexPath string) (*CodeIndex, error) {
	data, err := os.ReadFile(indexPath)
	if err != nil {
		return nil, err
	}

	var index CodeIndex
	if err := json.Unmarshal(data, &index); err != nil {
		return nil, err
	}

	return &index, nil
}

// SaveIndex saves the index to a file
func SaveIndex(indexPath string, index *CodeIndex) error {
	data, err := json.MarshalIndent(index, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(indexPath, data, 0644)
}

// IsIndexRecent checks if an index file exists and was created recently
func IsIndexRecent(indexPath string, maxAge time.Duration) bool {
	info, err := os.Stat(indexPath)
	if err != nil {
		return false
	}

	age := time.Since(info.ModTime())
	return age < maxAge
}

// Search finds files matching the query based on summary content
func (idx *CodeIndex) Search(query string, limit int) []FileSummary {
	query = strings.ToLower(query)
	var results []FileSummary

	for _, file := range idx.Files {
		summaryLower := strings.ToLower(file.Summary)
		pathLower := strings.ToLower(file.Path)

		// Simple relevance scoring: check if query appears in summary or path
		if strings.Contains(summaryLower, query) || strings.Contains(pathLower, query) {
			results = append(results, file)
		}
	}

	// Limit results
	if len(results) > limit {
		results = results[:limit]
	}

	return results
}
