package datamonkey

import (
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

// TokenConfig holds configuration for JWT token generation
type TokenConfig struct {
	KeyPath         string        // Path to the JWT key file
	Username        string        // Username for JWT token
	ExpirationSecs  int64         // Expiration time in seconds for JWT token
	RefreshInterval time.Duration // How often to refresh the token
}

// SessionService provides session management and JWT token operations
type SessionService struct {
	Config         TokenConfig
	SessionTracker SessionTracker
}

// NewSessionService creates a new SessionService instance
func NewSessionService(config TokenConfig, sessionTracker SessionTracker) *SessionService {
	// Set default token refresh interval if not specified
	if config.RefreshInterval == 0 {
		config.RefreshInterval = 12 * time.Hour
	}

	// Set default JWT expiration if not specified
	if config.ExpirationSecs == 0 {
		config.ExpirationSecs = 86400 // Default to 24 hours
	}

	return &SessionService{
		Config:         config,
		SessionTracker: sessionTracker,
	}
}

// GenerateToken generates a JWT token
func (s *SessionService) GenerateToken(claims map[string]interface{}) (string, error) {
	if s.Config.KeyPath == "" {
		return "", fmt.Errorf("JWT key path not set")
	}

	keyData, err := os.ReadFile(s.Config.KeyPath)
	if err != nil {
		return "", fmt.Errorf("failed to read JWT key file: %v", err)
	}

	now := time.Now()
	jwtClaims := jwt.MapClaims{
		"iat": now.Unix(),
		"exp": now.Add(time.Duration(s.Config.ExpirationSecs) * time.Second).Unix(),
	}

	for k, v := range claims {
		jwtClaims[k] = v
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwtClaims)
	signedToken, err := token.SignedString(keyData)
	if err != nil {
		return "", fmt.Errorf("failed to sign JWT token: %v", err)
	}

	log.Printf("Generated JWT token for claims: %v", claims)
	return signedToken, nil
}

// GenerateUserToken generates a token for a user
func (s *SessionService) GenerateUserToken(userId string) (string, error) {
	claims := map[string]interface{}{
		"sub":  userId,
		"type": "user",
	}
	return s.GenerateToken(claims)
}

// ValidateToken validates a JWT token and returns its claims
func (s *SessionService) ValidateToken(tokenString string) (jwt.MapClaims, error) {
	if s.Config.KeyPath == "" {
		return nil, fmt.Errorf("JWT key path not set")
	}

	keyData, err := os.ReadFile(s.Config.KeyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read JWT key file: %v", err)
	}

	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return keyData, nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %v", err)
	}

	if !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, fmt.Errorf("invalid claims")
	}

	return claims, nil
}

// ExtractToken extracts JWT token from gin context (header or query param)
func (s *SessionService) ExtractToken(c *gin.Context) string {
	// Try Authorization header first
	authHeader := c.GetHeader("Authorization")
	if authHeader != "" {
		parts := strings.Split(authHeader, " ")
		if len(parts) == 2 && parts[0] == "Bearer" {
			return strings.TrimSpace(parts[1])
		}
	}

	// Try custom user_token header
	if token := c.GetHeader("user_token"); token != "" {
		return strings.TrimSpace(token)
	}

	// Try query parameter
	if token := c.Query("user_token"); token != "" {
		return strings.TrimSpace(token)
	}

	return ""
}

// GetOrCreateSubject gets subject from token or creates new session
// This is the main method handlers should call
func (s *SessionService) GetOrCreateSubject(c *gin.Context) (string, error) {
	// Try to extract token
	token := s.ExtractToken(c)

	if token != "" {
		// Validate existing token
		claims, err := s.ValidateToken(token)
		if err != nil {
			log.Printf("Invalid token: %v, creating new session", err)
			return s.createNewSession(c)
		}

		// Extract subject from claims
		if sub, ok := claims["sub"].(string); ok {
			// Update last seen
			if s.SessionTracker != nil {
				s.SessionTracker.UpdateLastSeen(sub)
			}
			return sub, nil
		}

		log.Println("Token missing subject claim, creating new session")
		return s.createNewSession(c)
	}

	// No token provided, create new session
	log.Println("No token provided, creating new session")
	return s.createNewSession(c)
}

// createNewSession creates a new session and returns the subject
// Also adds the token to response headers
func (s *SessionService) createNewSession(c *gin.Context) (string, error) {
	var subject string

	// Create session in tracker
	if s.SessionTracker != nil {
		session, err := s.SessionTracker.CreateSession()
		if err != nil {
			log.Printf("Failed to create session in tracker: %v", err)
			return "", fmt.Errorf("failed to create session: %v", err)
		}
		subject = session.Subject
	} else {
		return "", fmt.Errorf("session tracker not available")
	}

	// Generate JWT token for this session
	claims := map[string]interface{}{
		"sub": subject,
	}
	token, err := s.GenerateToken(claims)
	if err != nil {
		log.Printf("Failed to generate token for session: %v", err)
		return "", fmt.Errorf("failed to generate token: %v", err)
	}

	// Add token to response header
	c.Header("X-Session-Token", token)
	c.Header("Access-Control-Expose-Headers", "X-Session-Token")

	log.Printf("Created new session: subject=%s", subject)
	return subject, nil
}

// GetSubject gets the subject from an existing token (does not create new session)
// Returns error if no valid token provided
func (s *SessionService) GetSubject(c *gin.Context) (string, error) {
	token := s.ExtractToken(c)
	if token == "" {
		return "", fmt.Errorf("no token provided")
	}

	claims, err := s.ValidateToken(token)
	if err != nil {
		return "", fmt.Errorf("invalid token: %v", err)
	}

	sub, ok := claims["sub"].(string)
	if !ok {
		return "", fmt.Errorf("token missing subject claim")
	}

	// Update last seen
	if s.SessionTracker != nil {
		s.SessionTracker.UpdateLastSeen(sub)
	}

	return sub, nil
}

// CheckJobAccess verifies if a user has access to a specific job
func (s *SessionService) CheckJobAccess(c *gin.Context, jobID string, jobTracker JobTracker) (string, error) {
	// Get subject (create session if needed)
	subject, err := s.GetOrCreateSubject(c)
	if err != nil {
		return "", err
	}

	// If no job tracker is provided, we can't verify ownership
	if jobTracker == nil {
		log.Println("Warning: Job tracker not provided, skipping ownership check")
		return subject, nil
	}

	// Get the job owner from the tracker
	ownerID, err := jobTracker.GetJobOwner(jobID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return "", fmt.Errorf("job not found: %s", jobID)
		}
		if strings.Contains(err.Error(), "not supported") {
			log.Printf("Warning: User tracking not supported for job tracker, allowing access to job %s", jobID)
			return subject, nil
		}
		return "", fmt.Errorf("failed to check job ownership: %v", err)
	}

	// Check if the user owns the job
	if ownerID != subject {
		return "", fmt.Errorf("user does not have access to this job")
	}

	return subject, nil
}

// CheckDatasetAccess verifies if a user has access to a specific dataset
func (s *SessionService) CheckDatasetAccess(c *gin.Context, datasetID string, datasetTracker DatasetTracker) (string, error) {
	// Get subject (create session if needed)
	subject, err := s.GetOrCreateSubject(c)
	if err != nil {
		return "", err
	}

	// If no dataset tracker is provided, we can't verify ownership
	if datasetTracker == nil {
		log.Println("Warning: Dataset tracker not provided, skipping ownership check")
		return subject, nil
	}

	// Get the dataset owner
	ownerID, err := datasetTracker.GetOwner(datasetID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return "", fmt.Errorf("dataset not found: %s", datasetID)
		}
		if strings.Contains(err.Error(), "not supported") {
			log.Printf("Warning: User tracking not supported for dataset tracker, allowing access to dataset %s", datasetID)
			return subject, nil
		}
		return "", fmt.Errorf("failed to check dataset ownership: %v", err)
	}

	// Check if the user owns the dataset
	if ownerID != subject {
		return "", fmt.Errorf("user does not have access to this dataset")
	}

	return subject, nil
}

// CheckConversationAccess verifies if a user has access to a specific conversation
func (s *SessionService) CheckConversationAccess(c *gin.Context, conversationID string, conversationTracker ConversationTracker) (string, error) {
	// Get subject (create session if needed)
	subject, err := s.GetOrCreateSubject(c)
	if err != nil {
		return "", err
	}

	// If no conversation tracker is provided, we can't verify ownership
	if conversationTracker == nil {
		log.Println("Warning: Conversation tracker not provided, skipping ownership check")
		return subject, nil
	}

	// Check if the user owns the conversation
	owner, err := conversationTracker.GetConversationOwner(conversationID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return "", fmt.Errorf("conversation not found: %s", conversationID)
		}
		return "", fmt.Errorf("failed to get conversation owner: %v", err)
	}

	if owner != subject {
		return "", fmt.Errorf("user does not have access to this conversation")
	}

	return subject, nil
}

// StartSessionCleanup starts a background goroutine to cleanup expired sessions
func (s *SessionService) StartSessionCleanup(interval time.Duration, maxAge time.Duration) {
	if s.SessionTracker == nil {
		log.Println("Session tracker not available, skipping cleanup")
		return
	}

	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for range ticker.C {
			count, err := s.SessionTracker.CleanupExpiredSessions(maxAge)
			if err != nil {
				log.Printf("Error cleaning up expired sessions: %v", err)
			} else if count > 0 {
				log.Printf("Cleaned up %d expired sessions", count)
			}
		}
	}()

	log.Printf("Started session cleanup task (interval: %v, max age: %v)", interval, maxAge)
}
