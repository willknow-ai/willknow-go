package main

import (
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"strings"
	"time"

	aiassistant "github.com/willknow-ai/willknow-go"
)

// Logger writes logs to file
type Logger struct {
	file *os.File
}

func NewLogger(path string) (*Logger, error) {
	// Create log directory
	os.MkdirAll("/var/log", 0755)

	file, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return nil, err
	}

	return &Logger{file: file}, nil
}

func (l *Logger) Log(requestID, level, message string) {
	timestamp := time.Now().Format(time.RFC3339)
	logLine := fmt.Sprintf("[%s] [%s] [RequestID: %s] %s\n", timestamp, level, requestID, message)
	l.file.WriteString(logLine)
	fmt.Print(logLine) // Also print to console
}

func (l *Logger) Close() {
	l.file.Close()
}

var logger *Logger

func main() {
	// Initialize logger
	var err error
	logger, err = NewLogger("/var/log/app.log")
	if err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}
	defer logger.Close()

	logger.Log("STARTUP", "INFO", "Application starting...")

	// Start AI Assistant
	go func() {
		apiKey := os.Getenv("CLAUDE_API_KEY")
		if apiKey == "" {
			log.Println("[WARNING] CLAUDE_API_KEY not set. AI Assistant will not work properly.")
			log.Println("[INFO] Set it with: export CLAUDE_API_KEY=your-key")
		}

		assistant, err := aiassistant.New(aiassistant.Config{
			SourcePath:   "/app/source",
			LogFiles:     []string{"/var/log/app.log"}, // Explicitly set for demo
			Port:         8888,
			ClaudeAPIKey: apiKey,
		})

		if err != nil {
			log.Printf("[ERROR] Failed to initialize AI Assistant: %v", err)
			return
		}

		if err := assistant.Start(); err != nil {
			log.Printf("[ERROR] AI Assistant failed: %v", err)
		}
	}()

	// Setup HTTP handlers
	http.HandleFunc("/", handleHome)
	http.HandleFunc("/api/users", handleUsers)
	http.HandleFunc("/api/error", handleError)

	logger.Log("STARTUP", "INFO", "Server starting on :8080")
	log.Println(strings.Repeat("=", 50))
	log.Println("🚀 Example App is running!")
	log.Println("  - Main app: http://localhost:8080")
	log.Println("  - AI Assistant: http://localhost:8888")
	log.Println(strings.Repeat("=", 50))

	if err := http.ListenAndServe(":8080", nil); err != nil {
		logger.Log("FATAL", "ERROR", fmt.Sprintf("Server failed: %v", err))
		log.Fatal(err)
	}
}

func handleHome(w http.ResponseWriter, r *http.Request) {
	requestID := generateRequestID()
	logger.Log(requestID, "INFO", "Home page accessed")

	html := `<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>Example App</title>
    <style>
        body {
            font-family: Arial, sans-serif;
            max-width: 800px;
            margin: 50px auto;
            padding: 20px;
            background: #f5f5f5;
        }
        .card {
            background: white;
            padding: 30px;
            border-radius: 10px;
            box-shadow: 0 2px 10px rgba(0,0,0,0.1);
            margin-bottom: 20px;
        }
        h1 { color: #333; }
        a {
            display: inline-block;
            margin: 10px 10px 10px 0;
            padding: 12px 24px;
            background: #667eea;
            color: white;
            text-decoration: none;
            border-radius: 5px;
        }
        a:hover { background: #5568d3; }
        .ai-link { background: #f59e0b; }
        .ai-link:hover { background: #d97706; }
        code {
            background: #f0f0f0;
            padding: 2px 6px;
            border-radius: 3px;
        }
    </style>
</head>
<body>
    <div class="card">
        <h1>🎉 Example Application</h1>
        <p>This is a demo web application with AI Assistant integration.</p>
        <h2>Try these endpoints:</h2>
        <a href="/api/users">GET /api/users</a>
        <a href="/api/error">GET /api/error (will fail)</a>
        <a href="http://localhost:8888" target="_blank" class="ai-link">🤖 Open AI Assistant</a>
    </div>
    <div class="card">
        <h2>How to use AI Assistant:</h2>
        <ol>
            <li>Click "Open AI Assistant" above</li>
            <li>Try triggering an error by clicking "/api/error"</li>
            <li>Note the <code>RequestID</code> from the error message</li>
            <li>Ask the AI: "RequestID xxx出错了，帮我分析"</li>
            <li>Watch the AI analyze logs and code!</li>
        </ol>
    </div>
</body>
</html>`

	w.Header().Set("Content-Type", "text/html")
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(html))
}

func handleUsers(w http.ResponseWriter, r *http.Request) {
	requestID := generateRequestID()
	logger.Log(requestID, "INFO", "Fetching users")

	users := []string{"Alice", "Bob", "Charlie"}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Request-ID", requestID)
	fmt.Fprintf(w, `{"requestId": "%s", "users": ["%s"]}`, requestID, users)

	logger.Log(requestID, "INFO", fmt.Sprintf("Returned %d users", len(users)))
}

func handleError(w http.ResponseWriter, r *http.Request) {
	requestID := generateRequestID()
	logger.Log(requestID, "INFO", "Error endpoint called")

	// Simulate a database error
	logger.Log(requestID, "ERROR", "Database connection timeout: failed to connect to localhost:5432")
	logger.Log(requestID, "ERROR", "Stack trace: handleError() at main.go:150")
	logger.Log(requestID, "ERROR", "This error is caused by missing database configuration")

	w.Header().Set("X-Request-ID", requestID)
	w.WriteHeader(http.StatusInternalServerError)
	fmt.Fprintf(w, `{"error": "Internal Server Error", "requestId": "%s", "message": "Please report this RequestID to the AI Assistant"}`, requestID)
}

func generateRequestID() string {
	const chars = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, 8)
	for i := range b {
		b[i] = chars[rand.Intn(len(chars))]
	}
	return string(b)
}

func init() {
	rand.Seed(time.Now().UnixNano())
}
