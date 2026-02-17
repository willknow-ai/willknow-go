package aiassistant

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/willknow-ai/willknow-go/provider"
)

// contextKey is used to store values in request context without collisions
type contextKey string

const userContextKey contextKey = "willknow_user"

const systemPrompt = `You are an AI debugging assistant embedded in a running application.

Your role:
- Help users diagnose and fix issues in their application
- Access the application's source code to understand the codebase
- Read application logs to understand what went wrong
- Provide clear, actionable solutions

Available tools:
- read_file: Read source code files
- grep: Search code for patterns
- glob: Find files by pattern
- read_logs: Query logs by request ID or keywords

When a user reports an error:
1. Use read_logs to find relevant log entries (if they provide a request ID or error details)
2. Use read_file to examine the code where the error occurred
3. Analyze the root cause
4. Suggest a fix with specific file and line numbers

Be concise, technical, and focus on solving the problem quickly. Always reference specific files and line numbers when suggesting fixes.`

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins for MVP
	},
}

// ChatMessage represents a chat message from the client
type ChatMessage struct {
	Content string `json:"content"`
}

// ChatResponse represents a response to the client
type ChatResponse struct {
	Type      string `json:"type"`    // "text", "error", "done", "session_info"
	Content   string `json:"content"` // text content
	SessionID string `json:"sessionId,omitempty"` // session identifier
}

// Session manages a chat session
type Session struct {
	ID       string
	User     *User
	messages []provider.Message
	logFile  *os.File
	mu       sync.Mutex
}

// generateSessionID creates a unique session identifier
func generateSessionID() string {
	bytes := make([]byte, 8)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

// initSessionLog creates a log file for the session
func initSessionLog(sessionID string) (*os.File, error) {
	// Create sessions directory
	logDir := "./sessions"
	os.MkdirAll(logDir, 0755)

	// Create log file with timestamp
	timestamp := time.Now().Format("20060102_150405")
	filename := fmt.Sprintf("%s_%s.jsonl", timestamp, sessionID)
	filepath := filepath.Join(logDir, filename)

	file, err := os.OpenFile(filepath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return nil, err
	}

	return file, nil
}

// logSessionEvent logs an event to the session log file
func (s *Session) logEvent(eventType string, data interface{}) {
	if s.logFile == nil {
		return
	}

	event := map[string]interface{}{
		"timestamp":  time.Now().Format(time.RFC3339),
		"session_id": s.ID,
		"type":       eventType,
		"data":       data,
	}

	jsonData, err := json.Marshal(event)
	if err != nil {
		log.Printf("Failed to marshal session event: %v", err)
		return
	}

	s.logFile.Write(jsonData)
	s.logFile.WriteString("\n")
}

func startServer(a *Assistant) error {
	// Create a new ServeMux for AI Assistant (independent from user's app)
	mux := http.NewServeMux()

	// Auth routes (no authentication required)
	mux.HandleFunc("/auth/login", func(w http.ResponseWriter, r *http.Request) {
		handleLogin(w, r, a)
	})
	mux.HandleFunc("/auth/logout", func(w http.ResponseWriter, r *http.Request) {
		handleLogout(w, r, a)
	})

	// Protected routes
	mux.HandleFunc("/", authMiddleware(serveHome, a))
	mux.HandleFunc("/api/ws", authMiddleware(func(w http.ResponseWriter, r *http.Request) {
		handleWebSocket(w, r, a)
	}, a))

	addr := fmt.Sprintf(":%d", a.config.Port)
	return http.ListenAndServe(addr, mux)
}

// authMiddleware wraps a handler with authentication checks.
// In password mode: redirects unauthenticated requests to /auth/login.
// In custom GetUser mode: returns 401 JSON if authentication fails.
// In open mode: always allows access.
func authMiddleware(next http.HandlerFunc, a *Assistant) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, err := a.authManager.authenticateRequest(r)
		if err != nil {
			if a.authManager.isPasswordMode() {
				http.Redirect(w, r, "/auth/login", http.StatusFound)
			} else {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				w.Write([]byte(`{"error":"unauthorized","message":"Please log in to your application first"}`))
			}
			return
		}
		ctx := context.WithValue(r.Context(), userContextKey, user)
		next(w, r.WithContext(ctx))
	}
}

// handleLogin handles GET (show form) and POST (verify password) for password mode.
func handleLogin(w http.ResponseWriter, r *http.Request, a *Assistant) {
	if !a.authManager.isPasswordMode() {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	if r.Method == http.MethodPost {
		password := r.FormValue("password")
		token, err := a.authManager.verifyPassword(password)
		if err != nil {
			serveLoginPage(w, "Incorrect password. Please try again.")
			return
		}
		http.SetCookie(w, &http.Cookie{
			Name:     "willknow_session",
			Value:    token,
			Path:     "/",
			HttpOnly: true,
			SameSite: http.SameSiteLaxMode,
		})
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	serveLoginPage(w, "")
}

// handleLogout clears the session cookie.
func handleLogout(w http.ResponseWriter, r *http.Request, a *Assistant) {
	http.SetCookie(w, &http.Cookie{
		Name:     "willknow_session",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
	})
	http.Redirect(w, r, "/auth/login", http.StatusFound)
}

// serveLoginPage renders the password login page with an optional error message.
func serveLoginPage(w http.ResponseWriter, errMsg string) {
	errHTML := ""
	if errMsg != "" {
		errHTML = `<p class="error">` + errMsg + `</p>`
	}
	html := `<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>AI Assistant - Login</title>
    <style>
        * { margin: 0; padding: 0; box-sizing: border-box; }
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
            height: 100vh;
            display: flex;
            align-items: center;
            justify-content: center;
            background: #f5f5f5;
        }
        .login-box {
            background: white;
            padding: 40px;
            border-radius: 12px;
            box-shadow: 0 4px 20px rgba(0,0,0,0.1);
            width: 360px;
        }
        .login-box h1 {
            font-size: 22px;
            margin-bottom: 6px;
            background: linear-gradient(135deg, #667eea, #764ba2);
            -webkit-background-clip: text;
            -webkit-text-fill-color: transparent;
        }
        .login-box p.subtitle { color: #666; font-size: 14px; margin-bottom: 28px; }
        label { display: block; font-size: 13px; font-weight: 600; color: #444; margin-bottom: 6px; }
        input[type=password] {
            width: 100%;
            padding: 10px 14px;
            border: 2px solid #e0e0e0;
            border-radius: 8px;
            font-size: 15px;
            outline: none;
            transition: border-color 0.2s;
        }
        input[type=password]:focus { border-color: #667eea; }
        button {
            width: 100%;
            padding: 12px;
            margin-top: 16px;
            background: linear-gradient(135deg, #667eea, #764ba2);
            color: white;
            border: none;
            border-radius: 8px;
            font-size: 15px;
            font-weight: 600;
            cursor: pointer;
        }
        button:hover { opacity: 0.9; }
        .error { color: #c62828; font-size: 13px; margin-top: 12px; }
    </style>
</head>
<body>
    <div class="login-box">
        <h1>AI Assistant</h1>
        <p class="subtitle">Enter the password to access the assistant</p>
        <form method="POST" action="/auth/login">
            <label for="password">Password</label>
            <input type="password" id="password" name="password" autofocus placeholder="Enter password" />
            <button type="submit">Sign In</button>
            ` + errHTML + `
        </form>
    </div>
</body>
</html>`
	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(html))
}

func serveHome(w http.ResponseWriter, r *http.Request) {
	// Simple HTML page with WebSocket client
	html := `<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>AI Assistant</title>
    <style>
        * { margin: 0; padding: 0; box-sizing: border-box; }
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
            height: 100vh;
            display: flex;
            flex-direction: column;
            background: #f5f5f5;
        }
        .header {
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            color: white;
            padding: 20px;
            box-shadow: 0 2px 10px rgba(0,0,0,0.1);
        }
        .header h1 { font-size: 24px; margin-bottom: 5px; }
        .header p { font-size: 14px; opacity: 0.9; }
        .container {
            flex: 1;
            display: flex;
            flex-direction: column;
            max-width: 1200px;
            width: 100%;
            margin: 0 auto;
            padding: 20px;
        }
        #messages {
            flex: 1;
            overflow-y: auto;
            padding: 20px;
            background: white;
            border-radius: 10px;
            box-shadow: 0 2px 10px rgba(0,0,0,0.05);
            margin-bottom: 20px;
        }
        .message {
            margin-bottom: 20px;
            padding: 15px;
            border-radius: 8px;
            line-height: 1.6;
        }
        .message.user {
            background: #e3f2fd;
            margin-left: 20%;
        }
        .message.assistant {
            background: #f5f5f5;
            margin-right: 20%;
        }
        .message.error {
            background: #ffebee;
            color: #c62828;
        }
        .message strong {
            display: block;
            margin-bottom: 8px;
            font-size: 12px;
            text-transform: uppercase;
            opacity: 0.7;
        }
        .message-content {
            line-height: 1.6;
        }
        .message-content p {
            margin: 0.5em 0;
        }
        .message-content h1, .message-content h2, .message-content h3 {
            margin: 1em 0 0.5em 0;
            font-weight: 600;
        }
        .message-content h1 { font-size: 1.5em; }
        .message-content h2 { font-size: 1.3em; }
        .message-content h3 { font-size: 1.1em; }
        .message-content ul, .message-content ol {
            margin: 0.5em 0;
            padding-left: 2em;
        }
        .message-content li {
            margin: 0.3em 0;
        }
        .message-content pre {
            background: #282c34;
            color: #abb2bf;
            padding: 15px;
            border-radius: 5px;
            overflow-x: auto;
            margin: 10px 0;
        }
        .message-content code {
            background: #282c34;
            color: #e06c75;
            padding: 2px 6px;
            border-radius: 3px;
            font-size: 13px;
            font-family: 'Courier New', monospace;
        }
        .message-content pre code {
            background: transparent;
            color: #abb2bf;
            padding: 0;
        }
        .message-content em {
            font-style: italic;
        }
        .input-area {
            display: flex;
            gap: 10px;
            background: white;
            padding: 20px;
            border-radius: 10px;
            box-shadow: 0 2px 10px rgba(0,0,0,0.05);
        }
        #messageInput {
            flex: 1;
            padding: 12px;
            border: 2px solid #e0e0e0;
            border-radius: 8px;
            font-size: 14px;
            font-family: inherit;
        }
        #messageInput:focus {
            outline: none;
            border-color: #667eea;
        }
        #sendButton {
            padding: 12px 30px;
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            color: white;
            border: none;
            border-radius: 8px;
            cursor: pointer;
            font-size: 14px;
            font-weight: 600;
        }
        #sendButton:hover { opacity: 0.9; }
        #sendButton:disabled {
            opacity: 0.5;
            cursor: not-allowed;
        }
        .typing {
            color: #666;
            font-style: italic;
            padding: 15px;
        }
    </style>
</head>
<body>
    <div class="header">
        <h1>🤖 AI Assistant</h1>
        <p>Your intelligent debugging companion</p>
        <p id="sessionInfo" style="font-size: 12px; opacity: 0.8; margin-top: 5px;"></p>
    </div>
    <div class="container">
        <div id="messages"></div>
        <div class="input-area">
            <input type="text" id="messageInput" placeholder="Ask me anything about your application..." />
            <button id="sendButton">Send</button>
        </div>
    </div>

    <script>
        const messagesDiv = document.getElementById('messages');
        const messageInput = document.getElementById('messageInput');
        const sendButton = document.getElementById('sendButton');
        const sessionInfo = document.getElementById('sessionInfo');

        let ws;
        let isProcessing = false;
        let currentSessionId = '';

        function connect() {
            const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
            ws = new WebSocket(protocol + '//' + window.location.host + '/api/ws');

            ws.onopen = () => {
                addMessage('system', 'Connected to AI Assistant. How can I help you?');
            };

            ws.onmessage = (event) => {
                const response = JSON.parse(event.data);

                if (response.type === 'session_info') {
                    // Store and display session ID
                    currentSessionId = response.sessionId;
                    sessionInfo.textContent = 'Session ID: ' + currentSessionId;
                } else if (response.type === 'text') {
                    // Remove typing indicator
                    const typing = document.querySelector('.typing');
                    if (typing) typing.remove();

                    // Add or append to assistant message
                    const lastMsg = messagesDiv.lastElementChild;
                    if (lastMsg && lastMsg.classList.contains('assistant') && !lastMsg.dataset.complete) {
                        // Append to existing message content
                        const contentDiv = lastMsg.querySelector('.message-content');
                        if (contentDiv) {
                            const currentText = contentDiv.dataset.rawText || '';
                            const newText = currentText + response.content;
                            contentDiv.dataset.rawText = newText;
                            contentDiv.innerHTML = formatMarkdown(newText);
                        }
                    } else {
                        addMessage('assistant', response.content);
                    }
                } else if (response.type === 'done') {
                    const lastMsg = messagesDiv.lastElementChild;
                    if (lastMsg && lastMsg.classList.contains('assistant')) {
                        lastMsg.dataset.complete = 'true';
                    }
                    isProcessing = false;
                    sendButton.disabled = false;
                } else if (response.type === 'error') {
                    addMessage('error', response.content);
                    isProcessing = false;
                    sendButton.disabled = false;
                }

                messagesDiv.scrollTop = messagesDiv.scrollHeight;
            };

            ws.onerror = () => {
                addMessage('error', 'Connection error. Please refresh the page.');
            };

            ws.onclose = () => {
                addMessage('error', 'Connection closed. Please refresh the page.');
            };
        }

        function addMessage(type, content) {
            const div = document.createElement('div');
            div.className = 'message ' + type;

            if (type === 'user') {
                div.innerHTML = '<strong>You</strong><div class="message-content">' + escapeHtml(content) + '</div>';
            } else if (type === 'assistant') {
                const contentDiv = document.createElement('div');
                contentDiv.className = 'message-content';
                contentDiv.dataset.rawText = content;
                contentDiv.innerHTML = formatMarkdown(content);
                
                div.innerHTML = '<strong>AI Assistant</strong>';
                div.appendChild(contentDiv);
            } else if (type === 'system') {
                div.innerHTML = '<strong>System</strong><div class="message-content">' + escapeHtml(content) + '</div>';
            } else if (type === 'error') {
                div.innerHTML = '<strong>Error</strong><div class="message-content">' + escapeHtml(content) + '</div>';
            }

            messagesDiv.appendChild(div);
            messagesDiv.scrollTop = messagesDiv.scrollHeight;
        }

        function formatMarkdown(text) {
            text = escapeHtml(text);
            
            var backtick = String.fromCharCode(96);
            var tripleBacktick = backtick + backtick + backtick;
            
            var codeBlockPattern = tripleBacktick + '(\\w+)?\\n?([\\s\\S]*?)' + tripleBacktick;
            var codeBlockRegex = new RegExp(codeBlockPattern, 'g');
            text = text.replace(codeBlockRegex, '<pre><code>$2</code></pre>');
            
            var inlineCodePattern = backtick + '([^' + backtick + ']+)' + backtick;
            var inlineCodeRegex = new RegExp(inlineCodePattern, 'g');
            text = text.replace(inlineCodeRegex, '<code>$1</code>');
            
            text = text.replace(/\*\*([^\*]+)\*\*/g, '<strong>$1</strong>');
            text = text.replace(/\*([^\*]+)\*/g, '<em>$1</em>');
            
            text = text.replace(/^### (.+)$/gm, '<h3>$1</h3>');
            text = text.replace(/^## (.+)$/gm, '<h2>$1</h2>');
            text = text.replace(/^# (.+)$/gm, '<h1>$1</h1>');
            
            text = text.replace(/^[\-\*] (.+)$/gm, '<li>$1</li>');
            text = text.replace(/(<li>.*<\/li>)/s, '<ul>$1</ul>');
            
            text = text.replace(/^\d+\. (.+)$/gm, '<li>$1</li>');
            
            text = text.replace(/\n\n/g, '</p><p>');
            text = text.replace(/\n/g, '<br>');
            
            if (!text.startsWith('<')) {
                text = '<p>' + text + '</p>';
            }
            
            return text;
        }

        function escapeHtml(text) {
            const div = document.createElement('div');
            div.textContent = text;
            return div.innerHTML;
        }

        function sendMessage() {
            const content = messageInput.value.trim();
            if (!content || isProcessing) return;

            addMessage('user', content);
            messageInput.value = '';

            // Add typing indicator
            const typing = document.createElement('div');
            typing.className = 'typing';
            typing.textContent = 'AI is thinking...';
            messagesDiv.appendChild(typing);

            isProcessing = true;
            sendButton.disabled = true;

            ws.send(JSON.stringify({ content }));
        }

        sendButton.onclick = sendMessage;
        messageInput.onkeypress = (e) => {
            if (e.key === 'Enter') sendMessage();
        };

        connect();
    </script>
</body>
</html>`

	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(html))
}

func handleWebSocket(w http.ResponseWriter, r *http.Request, a *Assistant) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade error: %v", err)
		return
	}
	defer conn.Close()

	// Generate unique session ID
	sessionID := generateSessionID()

	// Initialize session log
	logFile, err := initSessionLog(sessionID)
	if err != nil {
		log.Printf("Failed to create session log: %v", err)
		// Continue without logging
	}
	defer func() {
		if logFile != nil {
			logFile.Close()
		}
	}()

	session := &Session{
		ID:       sessionID,
		User:     r.Context().Value(userContextKey).(*User),
		messages: []provider.Message{},
		logFile:  logFile,
	}

	// Log session start with user info
	userID := session.User.ID
	userName := session.User.Name
	session.logEvent("session_start", map[string]interface{}{
		"timestamp":   time.Now().Format(time.RFC3339),
		"remote_addr": r.RemoteAddr,
		"user_id":     userID,
		"user_name":   userName,
	})

	// Send session info to client
	conn.WriteJSON(ChatResponse{
		Type:      "session_info",
		SessionID: sessionID,
		Content:   fmt.Sprintf("Session %s started", sessionID),
	})

	log.Printf("[Session %s] Started (user: %s)", sessionID, userID)

	for {
		var msg ChatMessage
		err := conn.ReadJSON(&msg)
		if err != nil {
			log.Printf("[Session %s] WebSocket read error: %v", sessionID, err)
			session.logEvent("session_end", map[string]interface{}{
				"reason": "connection_closed",
				"error": err.Error(),
			})
			break
		}

		// Log user message
		session.logEvent("user_message", map[string]interface{}{
			"content": msg.Content,
		})

		// Add user message to session
		session.mu.Lock()
		session.messages = append(session.messages, provider.Message{
			Role: "user",
			Content: []provider.ContentBlock{
				{Type: "text", Text: msg.Content},
			},
		})
		session.mu.Unlock()

		log.Printf("[Session %s] User: %s", sessionID, msg.Content)

		// Process with AI (allow multiple tool use turns)
		err = processChat(conn, a, session)
		if err != nil {
			log.Printf("[Session %s] Error: %v", sessionID, err)
			session.logEvent("error", map[string]interface{}{
				"error": err.Error(),
			})
			conn.WriteJSON(ChatResponse{
				Type:    "error",
				Content: fmt.Sprintf("Error: %v", err),
			})
		}

		// Send done signal
		conn.WriteJSON(ChatResponse{Type: "done"})
	}

	log.Printf("[Session %s] Ended", sessionID)
}

func processChat(conn *websocket.Conn, a *Assistant, session *Session) error {
	maxTurns := 10 // Allow multiple tool use turns

	for turn := 0; turn < maxTurns; turn++ {
		// Call AI API
		session.mu.Lock()
		messages := make([]provider.Message, len(session.messages))
		copy(messages, session.messages)
		session.mu.Unlock()

		tools := a.toolRegistry.GetToolDefinitions()
		response, err := a.provider.SendMessage(messages, tools, systemPrompt)
		if err != nil {
			return err
		}

		// Process response content
		var assistantContent []provider.ContentBlock
		hasToolUse := false

		for _, block := range response.Content {
			if block.Type == "text" {
				// Send text to client
				conn.WriteJSON(ChatResponse{
					Type:    "text",
					Content: block.Text,
				})
				assistantContent = append(assistantContent, block)

				// Log AI text response
				session.logEvent("assistant_message", map[string]interface{}{
					"content": block.Text,
				})
			} else if block.Type == "tool_use" {
				hasToolUse = true
				assistantContent = append(assistantContent, block)

				// Log tool use
				session.logEvent("tool_use", map[string]interface{}{
					"tool_name": block.Name,
					"tool_id":   block.ID,
					"input":     block.Input,
				})

				// Execute tool
				log.Printf("Executing tool: %s", block.Name)
				result, err := a.toolRegistry.Execute(block.Name, block.Input)
				if err != nil {
					result = fmt.Sprintf("Error: %v", err)
				}

				// Log tool result
				session.logEvent("tool_result", map[string]interface{}{
					"tool_name": block.Name,
					"tool_id":   block.ID,
					"result":    result,
					"error":     err != nil,
				})

				// Add tool result to next message
				session.mu.Lock()
				session.messages = append(session.messages, provider.Message{
					Role:    "assistant",
					Content: assistantContent,
				})
				session.messages = append(session.messages, provider.Message{
					Role: "user",
					Content: []provider.ContentBlock{
						{
							Type:      "tool_result",
							ToolUseID: block.ID,
							Content:   result,
						},
					},
				})
				session.mu.Unlock()
			}
		}

		// If no tool use, we're done
		if !hasToolUse {
			session.mu.Lock()
			session.messages = append(session.messages, provider.Message{
				Role:    "assistant",
				Content: assistantContent,
			})
			session.mu.Unlock()
			break
		}
	}

	return nil
}
