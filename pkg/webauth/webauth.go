package webauth

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"golang.org/x/crypto/bcrypt"
	"github.com/rs/zerolog/log"
)

// PasswordData stores the hashed password and metadata
type PasswordData struct {
	HashedPassword string    `json:"hashed_password"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// SessionData stores session information
type SessionData struct {
	SessionID string    `json:"session_id"`
	CreatedAt time.Time `json:"created_at"`
	ExpiresAt time.Time `json:"expires_at"`
}

// WebAuth manages web interface authentication
type WebAuth struct {
	passwordFile string
	sessions     map[string]SessionData
}

// NewWebAuth creates a new WebAuth instance
func NewWebAuth(passwordFile string) *WebAuth {
	return &WebAuth{
		passwordFile: passwordFile,
		sessions:     make(map[string]SessionData),
	}
}

// IsPasswordSet checks if a password is currently set
func (wa *WebAuth) IsPasswordSet() bool {
	_, err := os.Stat(wa.passwordFile)
	return err == nil
}

// SetPassword sets a new password (hashes and stores it)
func (wa *WebAuth) SetPassword(password string) error {
	if len(password) < 8 {
		return fmt.Errorf("password must be at least 8 characters long")
	}

	// Hash the password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	// Create password data
	passwordData := PasswordData{
		HashedPassword: string(hashedPassword),
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	// Save to file
	return wa.savePasswordData(passwordData)
}

// VerifyPassword checks if the provided password is correct
func (wa *WebAuth) VerifyPassword(password string) bool {
	passwordData, err := wa.loadPasswordData()
	if err != nil {
		log.Error().Err(err).Msg("Failed to load password data")
		return false
	}

	err = bcrypt.CompareHashAndPassword([]byte(passwordData.HashedPassword), []byte(password))
	return err == nil
}

// RemovePassword removes the password file (disables password protection)
func (wa *WebAuth) RemovePassword() error {
	if !wa.IsPasswordSet() {
		return fmt.Errorf("no password is currently set")
	}

	err := os.Remove(wa.passwordFile)
	if err != nil {
		return fmt.Errorf("failed to remove password file: %w", err)
	}

	// Clear all sessions
	wa.sessions = make(map[string]SessionData)
	
	log.Info().Msg("Password protection disabled")
	return nil
}

// CreateSession creates a new session for authenticated users
func (wa *WebAuth) CreateSession() (string, error) {
	// Generate random session ID
	sessionIDBytes := make([]byte, 32)
	if _, err := rand.Read(sessionIDBytes); err != nil {
		return "", fmt.Errorf("failed to generate session ID: %w", err)
	}
	sessionID := hex.EncodeToString(sessionIDBytes)

	// Create session data
	sessionData := SessionData{
		SessionID: sessionID,
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(24 * time.Hour), // 24 hour sessions
	}

	// Store session
	wa.sessions[sessionID] = sessionData

	return sessionID, nil
}

// ValidateSession checks if a session is valid and not expired
func (wa *WebAuth) ValidateSession(sessionID string) bool {
	sessionData, exists := wa.sessions[sessionID]
	if !exists {
		return false
	}

	// Check if session is expired
	if time.Now().After(sessionData.ExpiresAt) {
		delete(wa.sessions, sessionID)
		return false
	}

	return true
}

// InvalidateSession removes a session (logout)
func (wa *WebAuth) InvalidateSession(sessionID string) {
	delete(wa.sessions, sessionID)
}

// CleanupExpiredSessions removes expired sessions
func (wa *WebAuth) CleanupExpiredSessions() {
	now := time.Now()
	for sessionID, sessionData := range wa.sessions {
		if now.After(sessionData.ExpiresAt) {
			delete(wa.sessions, sessionID)
		}
	}
}

// loadPasswordData loads password data from file
func (wa *WebAuth) loadPasswordData() (PasswordData, error) {
	var passwordData PasswordData

	file, err := os.Open(wa.passwordFile)
	if err != nil {
		return passwordData, fmt.Errorf("failed to open password file: %w", err)
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	err = decoder.Decode(&passwordData)
	if err != nil {
		return passwordData, fmt.Errorf("failed to decode password data: %w", err)
	}

	return passwordData, nil
}

// savePasswordData saves password data to file
func (wa *WebAuth) savePasswordData(passwordData PasswordData) error {
	file, err := os.Create(wa.passwordFile)
	if err != nil {
		return fmt.Errorf("failed to create password file: %w", err)
	}
	defer file.Close()

	// Set file permissions to be readable only by owner
	err = file.Chmod(0600)
	if err != nil {
		return fmt.Errorf("failed to set password file permissions: %w", err)
	}

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	err = encoder.Encode(passwordData)
	if err != nil {
		return fmt.Errorf("failed to encode password data: %w", err)
	}

	return nil
}

// ConstantTimeCompare performs constant-time string comparison to prevent timing attacks
func ConstantTimeCompare(a, b string) bool {
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}