package datamonkey

import (
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// TokenConfig holds configuration for JWT token generation
type TokenConfig struct {
	// JWT token generation parameters
	KeyPath        string        // Path to the JWT key file
	Username       string        // Username for JWT token
	ExpirationSecs int64         // Expiration time in seconds for JWT token
	RefreshInterval time.Duration // How often to refresh the token
}

// TokenService provides JWT token generation and validation
type TokenService struct {
	Config TokenConfig
	mu     sync.RWMutex // Mutex to protect token operations
}

// NewTokenService creates a new TokenService instance
func NewTokenService(config TokenConfig) *TokenService {
	// Set default token refresh interval if not specified
	if config.RefreshInterval == 0 {
		config.RefreshInterval = 12 * time.Hour // Default to 12 hours
	}

	// Set default JWT expiration if not specified
	if config.ExpirationSecs == 0 {
		config.ExpirationSecs = 86400 // Default to 24 hours
	}

	return &TokenService{
		Config: config,
	}
}

// GenerateToken generates a JWT token
func (s *TokenService) GenerateToken(claims map[string]interface{}) (string, error) {
	// Check if key path is set
	if s.Config.KeyPath == "" {
		return "", fmt.Errorf("JWT key path not set")
	}

	// Read the JWT key file
	keyData, err := os.ReadFile(s.Config.KeyPath)
	if err != nil {
		return "", fmt.Errorf("failed to read JWT key file: %v", err)
	}

	// Create the JWT claims
	now := time.Now()
	jwtClaims := jwt.MapClaims{
		"iat": now.Unix(),
		"exp": now.Add(time.Duration(s.Config.ExpirationSecs) * time.Second).Unix(),
	}

	// Add custom claims
	for k, v := range claims {
		jwtClaims[k] = v
	}

	// Create and sign the token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwtClaims)
	signedToken, err := token.SignedString(keyData)
	if err != nil {
		return "", fmt.Errorf("failed to sign JWT token: %v", err)
	}

	log.Printf("Generated JWT token for claims: %v", claims)
	return signedToken, nil
}

// GenerateUserToken generates a token for a user
func (s *TokenService) GenerateUserToken(userId string) (string, error) {
	claims := map[string]interface{}{
		"sub": userId,
		"type": "user",
	}
	
	return s.GenerateToken(claims)
}

// ValidateToken validates a JWT token and returns its claims
func (s *TokenService) ValidateToken(tokenString string) (jwt.MapClaims, error) {
	// Check if key path is set
	if s.Config.KeyPath == "" {
		return nil, fmt.Errorf("JWT key path not set")
	}

	// Read the JWT key file
	keyData, err := os.ReadFile(s.Config.KeyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read JWT key file: %v", err)
	}

	// Parse the token
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		// Validate the signing method
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return keyData, nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %v", err)
	}

	// Check if the token is valid
	if !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}

	// Extract claims
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, fmt.Errorf("invalid claims")
	}

	return claims, nil
}
