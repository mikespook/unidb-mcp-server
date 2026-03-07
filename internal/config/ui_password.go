package config

import (
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"fmt"
	"os"

	"golang.org/x/crypto/bcrypt"

	"github.com/mikespook/unidb-mcp/internal/store"
)

const uiPasswordHashKey = "ui_password_hash"

// UIPasswordConfig holds the UI password configuration.
type UIPasswordConfig struct {
	// Disabled means no password is required.
	Disabled bool
	// Store reference for session validation and password changes.
	Store *store.Store
}

// LoadUIPassword loads or generates the UI password, stores its bcrypt hash in SQLite.
// Returns UIPasswordConfig. If UI_PASSWORD=false, auth is disabled.
func LoadUIPassword(s *store.Store) (*UIPasswordConfig, error) {
	envVal := os.Getenv("UI_PASSWORD")

	if envVal == "false" {
		fmt.Println("UI_PASSWORD=disabled")
		return &UIPasswordConfig{Disabled: true, Store: s}, nil
	}

	if envVal != "" {
		// Explicit password — hash and store it
		fmt.Printf("UI_PASSWORD=%s\n", envVal)
		hash, err := bcrypt.GenerateFromPassword([]byte(envVal), bcrypt.DefaultCost)
		if err != nil {
			return nil, fmt.Errorf("failed to hash UI password: %w", err)
		}
		if err := s.SetSetting(uiPasswordHashKey, string(hash)); err != nil {
			return nil, fmt.Errorf("failed to store UI password hash: %w", err)
		}
		return &UIPasswordConfig{Store: s}, nil
	}

	// No env var — check if a hash already exists in SQLite
	_, err := s.GetSetting(uiPasswordHashKey)
	if err == nil {
		// Hash already stored — plaintext is gone, note it's set		
		return &UIPasswordConfig{Store: s}, nil
	}
	if err != sql.ErrNoRows {
		return nil, err
	}

	// Generate a random password
	buf := make([]byte, 12)
	if _, err := rand.Read(buf); err != nil {
		return nil, fmt.Errorf("failed to generate UI password: %w", err)
	}
	password := base64.RawURLEncoding.EncodeToString(buf)

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("failed to hash UI password: %w", err)
	}
	if err := s.SetSetting(uiPasswordHashKey, string(hash)); err != nil {
		return nil, fmt.Errorf("failed to store UI password hash: %w", err)
	}

	fmt.Printf("UI_PASSWORD not set. Generated password: %s\n", password)
	fmt.Println("To use a custom password, set: UI_PASSWORD=<value>")
	fmt.Println("To disable authentication, set: UI_PASSWORD=false")

	return &UIPasswordConfig{Store: s}, nil
}

// ResetUIPassword generates a new random password, stores its hash, and returns the plaintext.
func ResetUIPassword(s *store.Store) (string, error) {
	buf := make([]byte, 12)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("failed to generate password: %w", err)
	}
	password := base64.RawURLEncoding.EncodeToString(buf)

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("failed to hash password: %w", err)
	}
	if err := s.SetSetting(uiPasswordHashKey, string(hash)); err != nil {
		return "", fmt.Errorf("failed to store password hash: %w", err)
	}
	return password, nil
}
