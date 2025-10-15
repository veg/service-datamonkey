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
	// JWT token generation parameters
	KeyPath        string        // Path to the JWT key file
	Username       string        // Username for JWT token
	ExpirationSecs int64         // Expiration time in seconds for JWT token
	RefreshInterval time.Duration // How often to refresh the token
}

// TokenService provides JWT token generation and validation
type TokenService struct {
	Config TokenConfig
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

// UserTokenValidator provides validation functions for user tokens
type UserTokenValidator struct {
	TokenService *TokenService
}

// NewUserTokenValidator creates a new UserTokenValidator instance
func NewUserTokenValidator(tokenService *TokenService) *UserTokenValidator {
	return &UserTokenValidator{
		TokenService: tokenService,
	}
}

// ValidateUserToken validates a user token from query parameter or header
func (v *UserTokenValidator) ValidateUserToken(c *gin.Context) (string, error) {
	// Check if token service is available
	if v.TokenService == nil {
		log.Println("Token service not available")
		return "", fmt.Errorf("token service not available")
	}

	// Try to get token from query parameter first
	userToken := c.Query("user_token")
	
	// If not in query, try header
	if userToken == "" {
		userToken = c.GetHeader("user_token")
	}
	
	// Check if token is provided
	if userToken == "" {
		log.Println("Missing user token")
		return "", fmt.Errorf("missing user token")
	}
	
	// Validate the token
	// Trim any whitespace from the token
	userToken = strings.TrimSpace(userToken)
	
	claims, err := v.TokenService.ValidateToken(userToken)
	if err != nil {
		log.Printf("Invalid user token: %v", err)
		return "", fmt.Errorf("invalid user token: %v", err)
	}
	
	// Extract user ID from claims
	userID, ok := claims["sub"].(string)
	if !ok {
		log.Println("Invalid token claims: missing user ID")
		return "", fmt.Errorf("invalid token claims: missing user ID")
	}
	
	return userID, nil
}

// CheckJobAccess verifies if a user has access to a specific job
func (v *UserTokenValidator) CheckJobAccess(c *gin.Context, jobID string, jobTracker JobTracker) (string, error) {
	// Validate the user token
	userID, err := v.ValidateUserToken(c)
	if err != nil {
		return "", err
	}
	
	// If no job tracker is provided, we can't verify ownership
	if jobTracker == nil {
		log.Println("Warning: Job tracker not provided, skipping ownership check")
		return userID, nil
	}
	
	// Get the job owner from the tracker
	ownerID, err := jobTracker.GetJobOwner(jobID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return "", fmt.Errorf("job not found: %s", jobID)
		}
		if strings.Contains(err.Error(), "not supported") {
			// If user tracking is not supported, allow access
			log.Printf("Warning: User tracking not supported for job tracker, allowing access to job %s", jobID)
			return userID, nil
		}
		return "", fmt.Errorf("failed to check job ownership: %v", err)
	}
	
	// Check if the user owns the job
	if ownerID != userID {
		return "", fmt.Errorf("user does not have access to this job")
	}
	
	return userID, nil
}

// CheckDatasetAccess verifies if a user has access to a specific dataset
func (v *UserTokenValidator) CheckDatasetAccess(c *gin.Context, datasetID string, datasetTracker DatasetTracker) (string, error) {
	// Validate the user token
	userID, err := v.ValidateUserToken(c)
	if err != nil {
		return "", err
	}
	
	// If no dataset tracker is provided, we can't verify ownership
	if datasetTracker == nil {
		log.Println("Warning: Dataset tracker not provided, skipping ownership check")
		return userID, nil
	}
	
	// Get the dataset owner using the exposed interface method
	ownerID, err := datasetTracker.GetOwner(datasetID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return "", fmt.Errorf("dataset not found: %s", datasetID)
		}
		if strings.Contains(err.Error(), "not supported") {
			// If user tracking is not supported, allow access
			log.Printf("Warning: User tracking not supported for dataset tracker, allowing access to dataset %s", datasetID)
			return userID, nil
		}
		if strings.Contains(err.Error(), "no associated user") {
			// Public dataset - allow access
			log.Printf("Dataset %s has no user_id, assuming public access", datasetID)
			return userID, nil
		}
		return "", fmt.Errorf("failed to check dataset ownership: %v", err)
	}
	
	// Check if the user owns the dataset
	if ownerID != userID {
		return "", fmt.Errorf("user does not have access to this dataset")
	}
	
	return userID, nil
}

// CheckConversationAccess verifies if a user has access to a specific conversation
func (v *UserTokenValidator) CheckConversationAccess(c *gin.Context, conversationID string, conversationTracker ConversationTracker) (string, error) {
	// Validate the user token
	userID, err := v.ValidateUserToken(c)
	if err != nil {
		return "", err
	}
	
	// If no conversation tracker is provided, we can't verify ownership
	if conversationTracker == nil {
		log.Println("Warning: Conversation tracker not provided, skipping ownership check")
		return userID, nil
	}
	
	// Get the conversation using the exposed interface method
	conversation, err := conversationTracker.GetConversation(conversationID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return "", fmt.Errorf("conversation not found: %s", conversationID)
		}
		return "", fmt.Errorf("failed to get conversation: %v", err)
	}
	
	// Validate the user token from the conversation
	conversationClaims, err := v.TokenService.ValidateToken(conversation.UserToken)
	if err != nil {
		return "", fmt.Errorf("invalid conversation token: %v", err)
	}
	
	// Extract user ID from claims
	conversationUserID, ok := conversationClaims["sub"].(string)
	if !ok {
		return "", fmt.Errorf("invalid token claims: missing user ID in conversation token")
	}
	
	// Check if the user owns the conversation
	if conversationUserID != userID {
		return "", fmt.Errorf("user does not have access to this conversation")
	}
	
	return userID, nil
}
