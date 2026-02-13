package aiassistant

import (
	"fmt"
	"log"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
	"github.com/willknow-ai/willknow-go/provider"
)

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
	Type    string `json:"type"` // "text", "error", "done"
	Content string `json:"content"`
}

// Session manages a chat session
type Session struct {
	messages []provider.Message
	mu       sync.Mutex
}

func startServer(a *Assistant) error {
	// Create a new ServeMux for AI Assistant (independent from user's app)
	mux := http.NewServeMux()

	// Serve static files (for MVP, we'll create a simple HTML page)
	mux.HandleFunc("/", serveHome)
	mux.HandleFunc("/api/ws", func(w http.ResponseWriter, r *http.Request) {
		handleWebSocket(w, r, a)
	})

	addr := fmt.Sprintf(":%d", a.config.Port)
	return http.ListenAndServe(addr, mux)
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

        let ws;
        let isProcessing = false;

        function connect() {
            const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
            ws = new WebSocket(protocol + '//' + window.location.host + '/api/ws');

            ws.onopen = () => {
                addMessage('system', 'Connected to AI Assistant. How can I help you?');
            };

            ws.onmessage = (event) => {
                const response = JSON.parse(event.data);

                if (response.type === 'text') {
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

	session := &Session{
		messages: []provider.Message{},
	}

	for {
		var msg ChatMessage
		err := conn.ReadJSON(&msg)
		if err != nil {
			log.Printf("WebSocket read error: %v", err)
			break
		}

		// Add user message to session
		session.mu.Lock()
		session.messages = append(session.messages, provider.Message{
			Role: "user",
			Content: []provider.ContentBlock{
				{Type: "text", Text: msg.Content},
			},
		})
		session.mu.Unlock()

		// Process with Claude API (allow multiple tool use turns)
		err = processChat(conn, a, session)
		if err != nil {
			conn.WriteJSON(ChatResponse{
				Type:    "error",
				Content: fmt.Sprintf("Error: %v", err),
			})
		}

		// Send done signal
		conn.WriteJSON(ChatResponse{Type: "done"})
	}
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
			} else if block.Type == "tool_use" {
				hasToolUse = true
				assistantContent = append(assistantContent, block)

				// Execute tool
				log.Printf("Executing tool: %s", block.Name)
				result, err := a.toolRegistry.Execute(block.Name, block.Input)
				if err != nil {
					result = fmt.Sprintf("Error: %v", err)
				}

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
