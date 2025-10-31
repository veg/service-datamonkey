package tests

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"

	sw "github.com/d-callan/service-datamonkey/go"
)

// setupTestKey creates a temporary JWT key file for testing
func setupTestKey(t *testing.T) (string, func()) {
	tmpFile, err := os.CreateTemp("", "jwt_test_key_*.txt")
	if err != nil {
		t.Fatalf("Failed to create temp key file: %v", err)
	}

	// Write a test key
	testKey := []byte("test-secret-key-for-jwt-testing-12345")
	if _, err := tmpFile.Write(testKey); err != nil {
		tmpFile.Close()
		os.Remove(tmpFile.Name())
		t.Fatalf("Failed to write test key: %v", err)
	}
	tmpFile.Close()

	cleanup := func() {
		os.Remove(tmpFile.Name())
	}

	return tmpFile.Name(), cleanup
}

// TestNewTokenService tests token service creation
func TestNewTokenService(t *testing.T) {
	tests := []struct {
		name                string
		config              sw.TokenConfig
		wantRefreshInterval time.Duration
		wantExpirationSecs  int64
	}{
		{
			name: "Default values",
			config: sw.TokenConfig{
				KeyPath:  "/tmp/test.key",
				Username: "testuser",
			},
			wantRefreshInterval: 12 * time.Hour,
			wantExpirationSecs:  86400,
		},
		{
			name: "Custom values",
			config: sw.TokenConfig{
				KeyPath:         "/tmp/test.key",
				Username:        "testuser",
				RefreshInterval: 6 * time.Hour,
				ExpirationSecs:  3600,
			},
			wantRefreshInterval: 6 * time.Hour,
			wantExpirationSecs:  3600,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := sw.NewSessionService(tt.config, nil)
			if service.Config.RefreshInterval != tt.wantRefreshInterval {
				t.Errorf("RefreshInterval = %v, want %v", service.Config.RefreshInterval, tt.wantRefreshInterval)
			}
			if service.Config.ExpirationSecs != tt.wantExpirationSecs {
				t.Errorf("ExpirationSecs = %v, want %v", service.Config.ExpirationSecs, tt.wantExpirationSecs)
			}
		})
	}
}

// TestGenerateToken tests JWT token generation
func TestGenerateToken(t *testing.T) {
	keyPath, cleanup := setupTestKey(t)
	defer cleanup()

	service := sw.NewSessionService(sw.TokenConfig{
		KeyPath:        keyPath,
		Username:       "testuser",
		ExpirationSecs: 3600,
	}, nil)

	tests := []struct {
		name    string
		claims  map[string]interface{}
		wantErr bool
	}{
		{
			name: "Generate token with custom claims",
			claims: map[string]interface{}{
				"sub":  "user123",
				"role": "admin",
			},
			wantErr: false,
		},
		{
			name:    "Generate token with empty claims",
			claims:  map[string]interface{}{},
			wantErr: false,
		},
		{
			name: "Generate token with multiple claim types",
			claims: map[string]interface{}{
				"sub":     "user456",
				"role":    "user",
				"premium": true,
				"level":   5,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token, err := service.GenerateToken(tt.claims)
			if (err != nil) != tt.wantErr {
				t.Errorf("GenerateToken() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr {
				if token == "" {
					t.Error("GenerateToken() returned empty token")
				}
				// Verify token has 3 parts (header.payload.signature)
				parts := strings.Split(token, ".")
				if len(parts) != 3 {
					t.Errorf("Token should have 3 parts, got %d", len(parts))
				}
			}
		})
	}
}

// TestGenerateTokenNoKeyPath tests error when key path is not set
func TestGenerateTokenNoKeyPath(t *testing.T) {
	service := sw.NewSessionService(sw.TokenConfig{
		Username: "testuser",
	}, nil)

	_, err := service.GenerateToken(map[string]interface{}{"sub": "user123"})
	if err == nil {
		t.Error("GenerateToken() should return error when key path is not set")
	}
	if !strings.Contains(err.Error(), "key path not set") {
		t.Errorf("Error should mention key path, got: %v", err)
	}
}

// TestGenerateTokenInvalidKeyPath tests error when key file doesn't exist
func TestGenerateTokenInvalidKeyPath(t *testing.T) {
	service := sw.NewSessionService(sw.TokenConfig{
		KeyPath:  "/nonexistent/path/to/key.txt",
		Username: "testuser",
	}, nil)

	_, err := service.GenerateToken(map[string]interface{}{"sub": "user123"})
	if err == nil {
		t.Error("GenerateToken() should return error for invalid key path")
	}
	if !strings.Contains(err.Error(), "failed to read JWT key file") {
		t.Errorf("Error should mention failed to read key file, got: %v", err)
	}
}

// TestGenerateUserToken tests user token generation
func TestGenerateUserToken(t *testing.T) {
	keyPath, cleanup := setupTestKey(t)
	defer cleanup()

	service := sw.NewSessionService(sw.TokenConfig{
		KeyPath:        keyPath,
		Username:       "testuser",
		ExpirationSecs: 3600,
	}, nil)

	userID := "user-alice-123"
	token, err := service.GenerateUserToken(userID)
	if err != nil {
		t.Fatalf("GenerateUserToken() error = %v", err)
	}
	if token == "" {
		t.Error("GenerateUserToken() returned empty token")
	}

	// Validate the token to check claims
	claims, err := service.ValidateToken(token)
	if err != nil {
		t.Fatalf("Failed to validate generated token: %v", err)
	}

	// Check that sub claim matches user ID
	if sub, ok := claims["sub"].(string); !ok || sub != userID {
		t.Errorf("Token sub claim = %v, want %v", claims["sub"], userID)
	}

	// Check that type claim is set
	if tokenType, ok := claims["type"].(string); !ok || tokenType != "user" {
		t.Errorf("Token type claim = %v, want 'user'", claims["type"])
	}
}

// TestValidateToken tests token validation
func TestValidateToken(t *testing.T) {
	keyPath, cleanup := setupTestKey(t)
	defer cleanup()

	service := sw.NewSessionService(sw.TokenConfig{
		KeyPath:        keyPath,
		Username:       "testuser",
		ExpirationSecs: 3600,
	}, nil)

	// Generate a valid token
	claims := map[string]interface{}{
		"sub":  "user123",
		"role": "admin",
	}
	validToken, err := service.GenerateToken(claims)
	if err != nil {
		t.Fatalf("Failed to generate test token: %v", err)
	}

	tests := []struct {
		name      string
		token     string
		wantErr   bool
		errSubstr string
	}{
		{
			name:    "Valid token",
			token:   validToken,
			wantErr: false,
		},
		{
			name:      "Invalid token format",
			token:     "invalid.token.format",
			wantErr:   true,
			errSubstr: "failed to parse token",
		},
		{
			name:      "Empty token",
			token:     "",
			wantErr:   true,
			errSubstr: "failed to parse token",
		},
		{
			name:      "Malformed token",
			token:     "not-a-jwt-token",
			wantErr:   true,
			errSubstr: "failed to parse token",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			claims, err := service.ValidateToken(tt.token)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateToken() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && tt.errSubstr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.errSubstr) {
					t.Errorf("Expected error containing '%s', got: %v", tt.errSubstr, err)
				}
			}
			if !tt.wantErr && claims == nil {
				t.Error("ValidateToken() should return claims for valid token")
			}
		})
	}
}

// TestValidateTokenExpired tests expired token validation
func TestValidateTokenExpired(t *testing.T) {
	keyPath, cleanup := setupTestKey(t)
	defer cleanup()

	// Create service with very short expiration
	service := sw.NewSessionService(sw.TokenConfig{
		KeyPath:        keyPath,
		Username:       "testuser",
		ExpirationSecs: 1, // 1 second
	}, nil)

	// Generate token
	token, err := service.GenerateUserToken("user123")
	if err != nil {
		t.Fatalf("Failed to generate token: %v", err)
	}

	// Wait for token to expire
	time.Sleep(2 * time.Second)

	// Try to validate expired token
	_, err = service.ValidateToken(token)
	if err == nil {
		t.Error("ValidateToken() should return error for expired token")
	}
}

// TestValidateTokenWrongKey tests token validation with wrong key
func TestValidateTokenWrongKey(t *testing.T) {
	// Create first key
	tmpFile1, err := os.CreateTemp("", "jwt_test_key1_*.txt")
	if err != nil {
		t.Fatalf("Failed to create temp key file 1: %v", err)
	}
	defer os.Remove(tmpFile1.Name())
	testKey1 := []byte("test-secret-key-number-one-12345")
	tmpFile1.Write(testKey1)
	tmpFile1.Close()

	// Create second key (different)
	tmpFile2, err := os.CreateTemp("", "jwt_test_key2_*.txt")
	if err != nil {
		t.Fatalf("Failed to create temp key file 2: %v", err)
	}
	defer os.Remove(tmpFile2.Name())
	testKey2 := []byte("test-secret-key-number-two-67890")
	tmpFile2.Write(testKey2)
	tmpFile2.Close()

	// Generate token with first key
	service1 := sw.NewSessionService(sw.TokenConfig{
		KeyPath:        tmpFile1.Name(),
		ExpirationSecs: 3600,
	}, nil)
	token, err := service1.GenerateUserToken("user123")
	if err != nil {
		t.Fatalf("Failed to generate token: %v", err)
	}

	// Try to validate with second key
	service2 := sw.NewSessionService(sw.TokenConfig{
		KeyPath:        tmpFile2.Name(),
		ExpirationSecs: 3600,
	}, nil)
	_, err = service2.ValidateToken(token)
	if err == nil {
		t.Error("ValidateToken() should fail when using wrong key")
	}
}

// TestValidateUserToken tests user token validation from HTTP context
func TestValidateUserToken(t *testing.T) {
	keyPath, cleanup := setupTestKey(t)
	defer cleanup()

	service := sw.NewSessionService(sw.TokenConfig{
		KeyPath:        keyPath,
		ExpirationSecs: 3600,
	}, nil)

	// Generate a valid token
	validToken, err := service.GenerateUserToken("user-alice")
	if err != nil {
		t.Fatalf("Failed to generate test token: %v", err)
	}

	tests := []struct {
		name          string
		setupContext  func(*gin.Context)
		wantUserID    string
		wantErr       bool
		wantErrSubstr string
	}{
		{
			name: "Valid token in query parameter",
			setupContext: func(c *gin.Context) {
				c.Request, _ = http.NewRequest("GET", "/?user_token="+validToken, nil)
			},
			wantUserID: "user-alice",
			wantErr:    false,
		},
		{
			name: "Valid token in header",
			setupContext: func(c *gin.Context) {
				c.Request, _ = http.NewRequest("GET", "/", nil)
				c.Request.Header.Set("user_token", validToken)
			},
			wantUserID: "user-alice",
			wantErr:    false,
		},
		{
			name: "Token with whitespace",
			setupContext: func(c *gin.Context) {
				c.Request, _ = http.NewRequest("GET", "/?user_token= "+validToken+" ", nil)
			},
			wantUserID: "user-alice",
			wantErr:    false,
		},
		{
			name: "Missing token",
			setupContext: func(c *gin.Context) {
				c.Request, _ = http.NewRequest("GET", "/", nil)
			},
			wantErr:       true,
			wantErrSubstr: "no token provided",
		},
		{
			name: "Invalid token",
			setupContext: func(c *gin.Context) {
				c.Request, _ = http.NewRequest("GET", "/?user_token=invalid-token", nil)
			},
			wantErr:       true,
			wantErrSubstr: "invalid token",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			tt.setupContext(c)

			userID, err := service.GetSubject(c)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateUserToken() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && tt.wantErrSubstr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErrSubstr) {
					t.Errorf("Expected error containing '%s', got: %v", tt.wantErrSubstr, err)
				}
			}
			if !tt.wantErr && userID != tt.wantUserID {
				t.Errorf("ValidateUserToken() userID = %v, want %v", userID, tt.wantUserID)
			}
		})
	}
}

// TestValidateUserTokenNoService tests validation without token service
func TestValidateUserTokenNoService(t *testing.T) {
	// Create a service with empty config (no key path)
	service := sw.NewSessionService(sw.TokenConfig{}, nil)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest("GET", "/?user_token=some-token", nil)

	_, err := service.GetSubject(c)
	if err == nil {
		t.Error("GetSubject() should return error when no valid token")
	}
}

// Mock job tracker for testing
type mockJobTracker struct {
	jobs map[string]string // jobID -> userID
}

func (m *mockJobTracker) GetJobOwner(jobID string) (string, error) {
	if userID, ok := m.jobs[jobID]; ok {
		if userID == "" {
			return "", fmt.Errorf("job has no associated user")
		}
		return userID, nil
	}
	return "", fmt.Errorf("job ID not found in tracker")
}

func (m *mockJobTracker) StoreJobMapping(jobID string, schedulerJobID string) error {
	return nil
}

func (m *mockJobTracker) GetSchedulerJobID(jobID string) (string, error) {
	return "", nil
}

func (m *mockJobTracker) DeleteJobMapping(jobID string) error {
	return nil
}

func (m *mockJobTracker) StoreJobWithUser(jobID string, schedulerJobID string, userID string) error {
	return nil
}

func (m *mockJobTracker) GetSchedulerJobIDByUser(jobID string, userID string) (string, error) {
	return "", nil
}

func (m *mockJobTracker) DeleteJobMappingByUser(jobID string, userID string) error {
	return nil
}

func (m *mockJobTracker) ListJobsByUser(userID string) ([]string, error) {
	return nil, nil
}

func (m *mockJobTracker) StoreJobMetadata(jobID string, alignmentID string, treeID string, methodType string, status string) error {
	return nil
}

func (m *mockJobTracker) GetJobMetadata(jobID string) (string, string, string, string, error) {
	return "", "", "", "", nil
}

func (m *mockJobTracker) UpdateJobStatus(jobID string, status string) error {
	return nil
}

func (m *mockJobTracker) UpdateJobStatusByUser(jobID string, userID string, status string) error {
	return nil
}

func (m *mockJobTracker) ListJobsWithFilters(filters map[string]interface{}) ([]string, error) {
	return nil, nil
}

func (m *mockJobTracker) ListJobsByStatus(statuses []sw.JobStatusValue) ([]sw.JobInfo, error) {
	// This is a mock implementation and can be empty for these tests.
	return []sw.JobInfo{}, nil
}

// TestCheckJobAccess tests job access verification
func TestCheckJobAccess(t *testing.T) {
	keyPath, cleanup := setupTestKey(t)
	defer cleanup()

	service := sw.NewSessionService(sw.TokenConfig{
		KeyPath:        keyPath,
		ExpirationSecs: 3600,
	}, nil)

	// Generate tokens for different users
	aliceToken, _ := service.GenerateUserToken("user-alice")
	bobToken, _ := service.GenerateUserToken("user-bob")

	// Setup mock job tracker
	tracker := &mockJobTracker{
		jobs: map[string]string{
			"job-alice-1": "user-alice",
			"job-bob-1":   "user-bob",
			"job-public":  "", // No owner
		},
	}

	tests := []struct {
		name          string
		token         string
		jobID         string
		tracker       sw.JobTracker
		wantUserID    string
		wantErr       bool
		wantErrSubstr string
	}{
		{
			name:       "Alice accesses her own job",
			token:      aliceToken,
			jobID:      "job-alice-1",
			tracker:    tracker,
			wantUserID: "user-alice",
			wantErr:    false,
		},
		{
			name:       "Bob accesses his own job",
			token:      bobToken,
			jobID:      "job-bob-1",
			tracker:    tracker,
			wantUserID: "user-bob",
			wantErr:    false,
		},
		{
			name:          "Alice tries to access Bob's job",
			token:         aliceToken,
			jobID:         "job-bob-1",
			tracker:       tracker,
			wantErr:       true,
			wantErrSubstr: "does not have access",
		},
		{
			name:          "Access non-existent job",
			token:         aliceToken,
			jobID:         "job-nonexistent",
			tracker:       tracker,
			wantErr:       true,
			wantErrSubstr: "job not found",
		},
		{
			name:       "No tracker provided",
			token:      aliceToken,
			jobID:      "job-any",
			tracker:    nil,
			wantUserID: "user-alice",
			wantErr:    false,
		},
		{
			name:          "Invalid token",
			token:         "invalid-token",
			jobID:         "job-alice-1",
			tracker:       tracker,
			wantErr:       true,
			wantErrSubstr: "session tracker not available",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request, _ = http.NewRequest("GET", "/?user_token="+tt.token, nil)

			userID, err := service.CheckJobAccess(c, tt.jobID, tt.tracker)
			if (err != nil) != tt.wantErr {
				t.Errorf("CheckJobAccess() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && tt.wantErrSubstr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErrSubstr) {
					t.Errorf("Expected error containing '%s', got: %v", tt.wantErrSubstr, err)
				}
			}
			if !tt.wantErr && userID != tt.wantUserID {
				t.Errorf("CheckJobAccess() userID = %v, want %v", userID, tt.wantUserID)
			}
		})
	}
}

// TestValidateTokenWithCustomClaims tests that custom claims are preserved
func TestValidateTokenWithCustomClaims(t *testing.T) {
	keyPath, cleanup := setupTestKey(t)
	defer cleanup()

	service := sw.NewSessionService(sw.TokenConfig{
		KeyPath:        keyPath,
		ExpirationSecs: 3600,
	}, nil)

	claims := map[string]interface{}{
		"sub":     "user123",
		"role":    "admin",
		"premium": true,
		"level":   42,
	}

	token, err := service.GenerateToken(claims)
	if err != nil {
		t.Fatalf("Failed to generate token: %v", err)
	}

	validatedClaims, err := service.ValidateToken(token)
	if err != nil {
		t.Fatalf("Failed to validate token: %v", err)
	}

	// Check all custom claims are preserved
	if sub, ok := validatedClaims["sub"].(string); !ok || sub != "user123" {
		t.Errorf("sub claim = %v, want user123", validatedClaims["sub"])
	}
	if role, ok := validatedClaims["role"].(string); !ok || role != "admin" {
		t.Errorf("role claim = %v, want admin", validatedClaims["role"])
	}
	if premium, ok := validatedClaims["premium"].(bool); !ok || !premium {
		t.Errorf("premium claim = %v, want true", validatedClaims["premium"])
	}
	// JWT numbers are float64
	if level, ok := validatedClaims["level"].(float64); !ok || level != 42 {
		t.Errorf("level claim = %v, want 42", validatedClaims["level"])
	}

	// Check standard claims are present
	if _, ok := validatedClaims["iat"]; !ok {
		t.Error("iat claim should be present")
	}
	if _, ok := validatedClaims["exp"]; !ok {
		t.Error("exp claim should be present")
	}
}

// TestTokenClaimsWithoutSub tests validation of token without sub claim
func TestTokenClaimsWithoutSub(t *testing.T) {
	keyPath, cleanup := setupTestKey(t)
	defer cleanup()

	service := sw.NewSessionService(sw.TokenConfig{
		KeyPath:        keyPath,
		ExpirationSecs: 3600,
	}, nil)

	// Manually create a token without sub claim
	keyData, _ := os.ReadFile(keyPath)
	now := time.Now()
	jwtClaims := jwt.MapClaims{
		"iat":  now.Unix(),
		"exp":  now.Add(time.Hour).Unix(),
		"role": "admin", // No sub claim
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwtClaims)
	tokenString, _ := token.SignedString(keyData)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest("GET", "/?user_token="+tokenString, nil)

	_, err := service.GetSubject(c)
	if err == nil {
		t.Error("GetSubject() should return error for token without sub claim")
	}
	if !strings.Contains(err.Error(), "missing subject claim") {
		t.Errorf("Error should mention missing subject claim, got: %v", err)
	}
}

// Mock dataset tracker for testing
type mockDatasetTracker struct {
	datasets map[string]string // datasetID -> userID
}

func (m *mockDatasetTracker) GetOwner(datasetID string) (string, error) {
	if userID, ok := m.datasets[datasetID]; ok {
		if userID == "" {
			return "", fmt.Errorf("dataset has no associated user")
		}
		return userID, nil
	}
	return "", fmt.Errorf("dataset ID not found in tracker")
}

func (m *mockDatasetTracker) Store(dataset sw.DatasetInterface) error {
	return nil
}

func (m *mockDatasetTracker) Get(datasetID string) (sw.DatasetInterface, error) {
	return nil, nil
}

func (m *mockDatasetTracker) Delete(datasetID string) error {
	return nil
}

func (m *mockDatasetTracker) List() ([]sw.DatasetInterface, error) {
	return nil, nil
}

func (m *mockDatasetTracker) StoreWithUser(dataset sw.DatasetInterface, userID string) error {
	return nil
}

func (m *mockDatasetTracker) ListByUser(userID string) ([]sw.DatasetInterface, error) {
	return nil, nil
}

func (m *mockDatasetTracker) DeleteByUser(datasetID string, userID string) error {
	return nil
}

func (m *mockDatasetTracker) DeleteAll() error {
	return nil
}

func (m *mockDatasetTracker) GetDatasetDir() string {
	return ""
}

func (m *mockDatasetTracker) Update(id string, updates map[string]interface{}) error {
	return nil
}

func (m *mockDatasetTracker) GetByUser(datasetID string, userID string) (sw.DatasetInterface, error) {
	return nil, nil
}

func (m *mockDatasetTracker) UpdateByUser(id string, userID string, updates map[string]interface{}) error {
	return nil
}

// TestCheckDatasetAccess tests dataset access verification
func TestCheckDatasetAccess(t *testing.T) {
	keyPath, cleanup := setupTestKey(t)
	defer cleanup()

	service := sw.NewSessionService(sw.TokenConfig{
		KeyPath:        keyPath,
		ExpirationSecs: 3600,
	}, nil)

	// Generate tokens for different users
	aliceToken, _ := service.GenerateUserToken("user-alice")
	bobToken, _ := service.GenerateUserToken("user-bob")

	// Setup mock dataset tracker
	tracker := &mockDatasetTracker{
		datasets: map[string]string{
			"dataset-alice-1": "user-alice",
			"dataset-bob-1":   "user-bob",
			"dataset-public":  "", // No owner (public)
		},
	}

	tests := []struct {
		name          string
		token         string
		datasetID     string
		tracker       sw.DatasetTracker
		wantUserID    string
		wantErr       bool
		wantErrSubstr string
	}{
		{
			name:       "Alice accesses her own dataset",
			token:      aliceToken,
			datasetID:  "dataset-alice-1",
			tracker:    tracker,
			wantUserID: "user-alice",
			wantErr:    false,
		},
		{
			name:       "Bob accesses his own dataset",
			token:      bobToken,
			datasetID:  "dataset-bob-1",
			tracker:    tracker,
			wantUserID: "user-bob",
			wantErr:    false,
		},
		{
			name:          "Alice tries to access Bob's dataset",
			token:         aliceToken,
			datasetID:     "dataset-bob-1",
			tracker:       tracker,
			wantErr:       true,
			wantErrSubstr: "does not have access",
		},
		{
			name:          "Access non-existent dataset",
			token:         aliceToken,
			datasetID:     "dataset-nonexistent",
			tracker:       tracker,
			wantErr:       true,
			wantErrSubstr: "dataset not found",
		},
		{
			name:          "Access public dataset (no owner)",
			token:         aliceToken,
			datasetID:     "dataset-public",
			tracker:       tracker,
			wantErr:       true,
			wantErrSubstr: "dataset has no associated user", // Public datasets (no owner) can't be verified
		},
		{
			name:       "No tracker provided",
			token:      aliceToken,
			datasetID:  "dataset-any",
			tracker:    nil,
			wantUserID: "user-alice",
			wantErr:    false,
		},
		{
			name:          "Invalid token",
			token:         "invalid-token",
			datasetID:     "dataset-alice-1",
			tracker:       tracker,
			wantErr:       true,
			wantErrSubstr: "session tracker not available", // With invalid token, new session is created but tracker is nil
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request, _ = http.NewRequest("GET", "/?user_token="+tt.token, nil)

			userID, err := service.CheckDatasetAccess(c, tt.datasetID, tt.tracker)
			if (err != nil) != tt.wantErr {
				t.Errorf("CheckDatasetAccess() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && tt.wantErrSubstr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErrSubstr) {
					t.Errorf("Expected error containing '%s', got: %v", tt.wantErrSubstr, err)
				}
			}
			if !tt.wantErr && userID != tt.wantUserID {
				t.Errorf("CheckDatasetAccess() userID = %v, want %v", userID, tt.wantUserID)
			}
		})
	}
}

// Mock conversation tracker for testing
type mockConversationTracker struct {
	conversations map[string]*sw.ChatConversation
	owners        map[string]string // conversationID -> subject
}

func (m *mockConversationTracker) GetConversation(conversationID string) (*sw.ChatConversation, error) {
	if conv, ok := m.conversations[conversationID]; ok {
		return conv, nil
	}
	return nil, fmt.Errorf("conversation not found")
}

func (m *mockConversationTracker) CreateConversation(conversation *sw.ChatConversation, subject string) error {
	if m.owners == nil {
		m.owners = make(map[string]string)
	}
	m.owners[conversation.Id] = subject
	return nil
}

func (m *mockConversationTracker) UpdateConversation(conversationID string, updates map[string]interface{}) error {
	return nil
}

func (m *mockConversationTracker) DeleteConversation(conversationID string) error {
	return nil
}

func (m *mockConversationTracker) ListConversations(userToken string) ([]*sw.ChatConversation, error) {
	return nil, nil
}

func (m *mockConversationTracker) AddMessage(conversationId string, message *sw.ChatMessage) error {
	return nil
}

func (m *mockConversationTracker) ListUserConversations(userToken string) ([]*sw.ChatConversation, error) {
	return nil, nil
}

func (m *mockConversationTracker) GetConversationOwner(conversationID string) (string, error) {
	if owner, ok := m.owners[conversationID]; ok {
		return owner, nil
	}
	return "", fmt.Errorf("conversation not found")
}

func (m *mockConversationTracker) GetConversationMessages(conversationId string) ([]sw.ChatMessage, error) {
	return nil, nil
}

func (m *mockConversationTracker) Close() error {
	return nil
}

// TestCheckConversationAccess tests conversation access verification
func TestCheckConversationAccess(t *testing.T) {
	keyPath, cleanup := setupTestKey(t)
	defer cleanup()

	service := sw.NewSessionService(sw.TokenConfig{
		KeyPath:        keyPath,
		ExpirationSecs: 3600,
	}, nil)

	// Generate tokens for different users
	aliceToken, _ := service.GenerateUserToken("user-alice")
	bobToken, _ := service.GenerateUserToken("user-bob")

	// Setup mock conversation tracker
	tracker := &mockConversationTracker{
		conversations: map[string]*sw.ChatConversation{
			"conv-alice-1": {
				Id: "conv-alice-1",
			},
			"conv-bob-1": {
				Id: "conv-bob-1",
			},
		},
		owners: map[string]string{
			"conv-alice-1": "user-alice",
			"conv-bob-1":   "user-bob",
		},
	}

	tests := []struct {
		name           string
		token          string
		conversationID string
		tracker        sw.ConversationTracker
		wantUserID     string
		wantErr        bool
		wantErrSubstr  string
	}{
		{
			name:           "Alice accesses her own conversation",
			token:          aliceToken,
			conversationID: "conv-alice-1",
			tracker:        tracker,
			wantUserID:     "user-alice",
			wantErr:        false,
		},
		{
			name:           "Bob accesses his own conversation",
			token:          bobToken,
			conversationID: "conv-bob-1",
			tracker:        tracker,
			wantUserID:     "user-bob",
			wantErr:        false,
		},
		{
			name:           "Alice tries to access Bob's conversation",
			token:          aliceToken,
			conversationID: "conv-bob-1",
			tracker:        tracker,
			wantErr:        true,
			wantErrSubstr:  "does not have access",
		},
		{
			name:           "Access non-existent conversation",
			token:          aliceToken,
			conversationID: "conv-nonexistent",
			tracker:        tracker,
			wantErr:        true,
			wantErrSubstr:  "conversation not found",
		},
		{
			name:           "No tracker provided",
			token:          aliceToken,
			conversationID: "conv-any",
			tracker:        nil,
			wantUserID:     "user-alice",
			wantErr:        false,
		},
		{
			name:           "Invalid token",
			token:          "invalid-token",
			conversationID: "conv-alice-1",
			tracker:        tracker,
			wantErr:        true,
			wantErrSubstr:  "session tracker not available", // With invalid token, new session is created but tracker is nil
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request, _ = http.NewRequest("GET", "/?user_token="+tt.token, nil)

			userID, err := service.CheckConversationAccess(c, tt.conversationID, tt.tracker)
			if (err != nil) != tt.wantErr {
				t.Errorf("CheckConversationAccess() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && tt.wantErrSubstr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErrSubstr) {
					t.Errorf("Expected error containing '%s', got: %v", tt.wantErrSubstr, err)
				}
			}
			if !tt.wantErr && userID != tt.wantUserID {
				t.Errorf("CheckConversationAccess() userID = %v, want %v", userID, tt.wantUserID)
			}
		})
	}
}

// TestGetOrCreateSubject tests the GetOrCreateSubject method (which internally calls createNewSession)
func TestGetOrCreateSubject(t *testing.T) {
	keyPath, cleanup := setupTestKey(t)
	defer cleanup()

	// Setup database with session tracker
	tmpFile, err := os.CreateTemp("", "test_sessions_*.db")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	dbPath := tmpFile.Name()
	tmpFile.Close()
	defer os.Remove(dbPath)

	db, dbCleanup := setupTestDB(t, dbPath)
	defer dbCleanup()

	sessionTracker := sw.NewSQLiteSessionTracker(db.GetDB())

	service := sw.NewSessionService(sw.TokenConfig{
		KeyPath:        keyPath,
		ExpirationSecs: 3600,
	}, sessionTracker)

	tests := []struct {
		name          string
		setupToken    bool
		wantNewToken  bool
		wantErr       bool
		wantErrSubstr string
	}{
		{
			name:         "Valid existing token",
			setupToken:   true,
			wantNewToken: false,
			wantErr:      false,
		},
		{
			name:         "No token - create new session",
			setupToken:   false,
			wantNewToken: true,
			wantErr:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			if tt.setupToken {
				// Generate a valid token
				token, err := service.GenerateUserToken("test-user")
				if err != nil {
					t.Fatalf("Failed to generate token: %v", err)
				}
				c.Request, _ = http.NewRequest("GET", "/?user_token="+token, nil)
			} else {
				c.Request, _ = http.NewRequest("GET", "/", nil)
			}

			subject, err := service.GetOrCreateSubject(c)

			if tt.wantErr {
				if err == nil {
					t.Error("Expected error but got none")
				} else if tt.wantErrSubstr != "" && !strings.Contains(err.Error(), tt.wantErrSubstr) {
					t.Errorf("Expected error containing '%s', got: %v", tt.wantErrSubstr, err)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if subject == "" {
					t.Error("Expected non-empty subject")
				}
			}

			if tt.wantNewToken {
				token := w.Header().Get("X-Session-Token")
				if token == "" {
					t.Error("Expected new session token in header")
				}
			}
		})
	}
}

// TestGetOrCreateSubjectNoTracker tests GetOrCreateSubject without session tracker
func TestGetOrCreateSubjectNoTracker(t *testing.T) {
	keyPath, cleanup := setupTestKey(t)
	defer cleanup()

	service := sw.NewSessionService(sw.TokenConfig{
		KeyPath:        keyPath,
		ExpirationSecs: 3600,
	}, nil) // No session tracker

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest("GET", "/", nil)

	_, err := service.GetOrCreateSubject(c)
	if err == nil {
		t.Error("Expected error when session tracker is nil")
	}
	if !strings.Contains(err.Error(), "session tracker not available") {
		t.Errorf("Expected 'session tracker not available' error, got: %v", err)
	}
}
