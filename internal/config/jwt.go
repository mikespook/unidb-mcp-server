package config

import (
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"fmt"
	"os"

	"github.com/mikespook/unidb-mcp/internal/store"
)

// JWTConfig holds JWT authentication configuration
type JWTConfig struct {
	Secret string
}

// DefaultDevSecret is the default JWT secret for development mode
const DefaultDevSecret = "your-secure-secret-key-here"

// LoadJWTConfig loads JWT configuration from environment variables or auto-generates one.
func LoadJWTConfig(s *store.Store) (*JWTConfig, error) {
	secret := os.Getenv("JWT_SECRET")

	if secret != "" {
		fmt.Printf("JWT_SECRET=%s\n", secret)
		return &JWTConfig{Secret: secret}, nil
	}

	// In development mode, use default secret if JWT_SECRET is not set
	if os.Getenv("DEV_MODE") == "true" {
		fmt.Printf("JWT_SECRET=%s\n", DefaultDevSecret)
		return &JWTConfig{Secret: DefaultDevSecret}, nil
	}

	// Try to load from SQLite settings
	stored, err := s.GetSetting("jwt_secret")
	if err == nil {
		fmt.Printf("JWT_SECRET=%s\n", stored)
		return &JWTConfig{Secret: stored}, nil
	}
	if err != sql.ErrNoRows {
		return nil, err
	}

	// Generate a new random secret
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return nil, fmt.Errorf("failed to generate JWT secret: %w", err)
	}
	secret = base64.RawURLEncoding.EncodeToString(buf)

	if err := s.SetSetting("jwt_secret", secret); err != nil {
		return nil, fmt.Errorf("failed to persist JWT secret: %w", err)
	}

	fmt.Println("JWT_SECRET not set. Generated a random secret stored in SQLite.")
	fmt.Printf("To pin this secret across deployments, set: JWT_SECRET=%s\n", secret)

	return &JWTConfig{Secret: secret}, nil
}
