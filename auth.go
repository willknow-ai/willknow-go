package aiassistant

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"sync"
	"time"
)

// GetUserFunc is the function signature for custom user authentication.
// Provided by the host application to integrate with its existing auth system.
type GetUserFunc func(r *http.Request) (*User, error)

// User represents an authenticated user
type User struct {
	ID    string
	Name  string
	Email string
}

// NoAuth is a sentinel GetUserFunc that explicitly disables authentication.
// Use this when you want to allow unrestricted access to the assistant.
//
// Example:
//
//	assistant, _ := aiassistant.New(aiassistant.Config{
//	    Auth: aiassistant.AuthConfig{GetUser: aiassistant.NoAuth},
//	})
var NoAuth GetUserFunc = func(r *http.Request) (*User, error) {
	return &User{ID: "anonymous", Name: "Anonymous"}, nil
}

// AuthConfig configures authentication for the AI assistant.
//
// Three modes (in priority order):
//
//  1. GetUser set to custom function → host system auth (highest priority)
//  2. GetUser set to NoAuth → fully open, no authentication required
//  3. GetUser not set (nil) → password protection mode (default)
type AuthConfig struct {
	// GetUser is called on every request to identify the current user.
	// - Set to a custom function to integrate with your existing auth system.
	//   If the function returns an error, the request is rejected with 401.
	// - Set to aiassistant.NoAuth to explicitly disable authentication.
	// - Leave nil (default) to use built-in password protection.
	GetUser GetUserFunc

	// Password is used only in password protection mode (when GetUser is nil).
	// If empty, willknow generates a random password and prints it to the console.
	// If set, uses this password.
	Password string
}

// authMode represents the active authentication mode
type authMode int

const (
	authModePassword authMode = iota // default: password protection
	authModeCustom                   // host system provides GetUser
	authModeOpen                     // NoAuth: fully open
)

// authSession represents an authenticated session in password mode
type authSession struct {
	userID    string
	createdAt time.Time
}

// AuthManager handles authentication logic
type AuthManager struct {
	mode     authMode
	config   AuthConfig
	password string // resolved password (generated or from config)

	// sessions is used only in password mode
	sessions sync.Map // token string → *authSession
}

// newAuthManager creates and initializes an AuthManager from config
func newAuthManager(config AuthConfig) *AuthManager {
	am := &AuthManager{config: config}

	if config.GetUser != nil {
		// Check if it's the NoAuth sentinel by comparing function pointers via a call
		// We detect NoAuth by calling it with a nil request and checking for the sentinel response
		testUser, err := config.GetUser(nil)
		if err == nil && testUser != nil && testUser.ID == "anonymous" && testUser.Name == "Anonymous" {
			am.mode = authModeOpen
		} else {
			am.mode = authModeCustom
		}
	} else {
		am.mode = authModePassword
		// Resolve password
		if config.Password != "" {
			am.password = config.Password
		} else {
			am.password = generatePassword()
		}
	}

	return am
}

// generatePassword creates a random 8-character alphanumeric password
func generatePassword() string {
	b := make([]byte, 5)
	rand.Read(b)
	// Use hex, then take first 8 chars for a readable password
	return hex.EncodeToString(b)[:8]
}

// printStartupMessage prints auth info to console on startup
func (am *AuthManager) printStartupMessage(port int) {
	switch am.mode {
	case authModePassword:
		fmt.Printf("\n")
		fmt.Printf("╔══════════════════════════════════════════════╗\n")
		fmt.Printf("║         AI Assistant - Access Password         ║\n")
		fmt.Printf("╠══════════════════════════════════════════════╣\n")
		fmt.Printf("║  Password: %-34s ║\n", am.password)
		fmt.Printf("║  URL:      http://localhost:%-17d ║\n", port)
		fmt.Printf("╚══════════════════════════════════════════════╝\n")
		fmt.Printf("\n")
	case authModeOpen:
		fmt.Printf("[AI Assistant] WARNING: Authentication is disabled. Anyone can access the assistant.\n")
	case authModeCustom:
		fmt.Printf("[AI Assistant] Authentication: using host system auth (GetUser callback)\n")
	}
}

// authenticateRequest verifies a request and returns the authenticated user.
// Returns nil user and nil error for open mode.
// Returns nil user and non-nil error if authentication fails.
func (am *AuthManager) authenticateRequest(r *http.Request) (*User, error) {
	switch am.mode {
	case authModeOpen:
		return &User{ID: "anonymous", Name: "Anonymous"}, nil

	case authModeCustom:
		user, err := am.config.GetUser(r)
		if err != nil {
			return nil, fmt.Errorf("authentication failed: %w", err)
		}
		if user == nil {
			return nil, fmt.Errorf("authentication failed: no user returned")
		}
		return user, nil

	case authModePassword:
		cookie, err := r.Cookie("willknow_session")
		if err != nil {
			return nil, fmt.Errorf("no session cookie")
		}
		session, ok := am.sessions.Load(cookie.Value)
		if !ok {
			return nil, fmt.Errorf("invalid or expired session")
		}
		s := session.(*authSession)
		return &User{ID: s.userID}, nil
	}

	return nil, fmt.Errorf("unknown auth mode")
}

// verifyPassword checks if the provided password is correct and creates a session.
// Returns the session token on success.
func (am *AuthManager) verifyPassword(password string) (string, error) {
	if am.mode != authModePassword {
		return "", fmt.Errorf("not in password mode")
	}
	if password != am.password {
		return "", fmt.Errorf("incorrect password")
	}

	// Create session
	tokenBytes := make([]byte, 16)
	rand.Read(tokenBytes)
	token := hex.EncodeToString(tokenBytes)

	am.sessions.Store(token, &authSession{
		userID:    "user",
		createdAt: time.Now(),
	})

	return token, nil
}

// isPasswordMode returns true if using built-in password protection
func (am *AuthManager) isPasswordMode() bool {
	return am.mode == authModePassword
}

// isOpenMode returns true if authentication is fully disabled
func (am *AuthManager) isOpenMode() bool {
	return am.mode == authModeOpen
}
